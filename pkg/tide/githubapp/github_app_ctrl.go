package githubapp

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/go-scm/scm/transport"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/io"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	"github.com/jenkins-x/lighthouse/pkg/prow/gitprovider"
	"github.com/jenkins-x/lighthouse/pkg/tide"
	"github.com/jenkins-x/lighthouse/pkg/tide/history"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// GithubServer the default github server URL
	GithubServer = "https://github.com"
)

type gitHubAppTideController struct {
	controllers        []tide.Controller
	ownerTokenFinder   *OwnerTokensDir
	gitServer          string
	githubAppSecretDir string
	configAgent        *config.Agent
	botName            string
	gitKind            string
	maxRecordsPerPool  int
	opener             io.Opener
	historyURI         string
	statusURI          string
	logger             *logrus.Entry
	m                  sync.Mutex
}

// NewGitHubAppTideController creates a GitHub App style controller which needs to process each github owner
// using a separate git provider client due to the way GitHub App tokens work
func NewGitHubAppTideController(githubAppSecretDir string, configAgent *config.Agent, botName string, gitKind string, maxRecordsPerPool int, opener io.Opener, historyURI string, statusURI string) (tide.Controller, error) {

	gitServer := GithubServer
	return &gitHubAppTideController{
		ownerTokenFinder:  NewOwnerTokensDir(gitServer, githubAppSecretDir),
		gitServer:         gitServer,
		configAgent:       configAgent,
		botName:           botName,
		gitKind:           gitKind,
		maxRecordsPerPool: maxRecordsPerPool,
		opener:            opener,
		historyURI:        historyURI,
		statusURI:         statusURI,
		logger:            logrus.NewEntry(logrus.StandardLogger()),
	}, nil

}

func (g *gitHubAppTideController) Sync() error {
	// lets iterate through the config and create a controller for each
	err := g.createOwnerControllers()
	if err != nil {
		return err
	}
	// now lets sync them all
	errs := []error{}
	for _, c := range g.controllers {
		err := c.Sync()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return util.CombineErrors(errs...)
}

func (g *gitHubAppTideController) Shutdown() {
	for _, c := range g.controllers {
		c.Shutdown()
	}
	g.controllers = nil
}

func (g *gitHubAppTideController) GetPools() []tide.Pool {
	g.m.Lock()
	defer g.m.Unlock()
	pools := []tide.Pool{}
	for _, c := range g.controllers {
		cp := c.GetPools()
		pools = append(pools, cp...)
	}
	return pools
}

func (g *gitHubAppTideController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pools := g.GetPools()
	b, err := json.Marshal(pools)
	if err != nil {
		g.logger.WithError(err).Error("Encoding JSON.")
		b = []byte("[]")
	}
	if _, err = w.Write(b); err != nil {
		g.logger.WithError(err).Error("Writing JSON response.")
	}
}

func (g *gitHubAppTideController) GetHistory() *history.History {
	answer, err := history.New(g.maxRecordsPerPool, g.opener, g.historyURI)
	if err != nil {
		return answer
	}
	for _, c := range g.controllers {
		h := c.GetHistory()
		answer.Merge(h)
	}
	return answer
}

func (g *gitHubAppTideController) createOwnerControllers() error {
	// lets zap any old controllers
	g.Shutdown()
	g.controllers = nil

	errs := []error{}

	cfg := g.configAgent.Config()
	if cfg == nil {
		return errors.New("no config")
	}

	oqs := SplitTideQueries(cfg.Tide.Queries)
	for owner, queries := range oqs {
		// create copy of config with different queries
		ocfg := *cfg
		ocfg.Tide.Queries = queries
		configGetter := func() *config.Config {
			return &ocfg
		}

		c, err := g.createOwnerController(owner, configGetter)
		if err != nil {
			errs = append(errs, err)
		} else {
			g.controllers = append(g.controllers, c)
		}
	}
	return util.CombineErrors(errs...)
}

func (g *gitHubAppTideController) createOwnerController(owner string, configGetter config.Getter) (tide.Controller, error) {
	token, err := g.ownerTokenFinder.FindToken(owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find GitHub App token for %s", owner)
	}
	if token == "" {
		return nil, errors.Errorf("no GitHub App token for %s", owner)
	}

	scmClient, err := createTideGitHubAppScmClient(g.gitServer, token)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create SCM client")
	}
	gitproviderClient := gitprovider.ToClient(scmClient, g.botName)
	gitClient, err := git.NewClient(g.gitServer, g.gitKind)
	if err != nil {
		return nil, errors.Wrap(err, "creating git client")
	}
	gitClient.SetCredentials(g.botName, func() []byte {
		return []byte(token)
	})
	tektonClient, jxClient, _, ns, err := clients.GetClientsAndNamespace()
	if err != nil {
		return nil, errors.Wrap(err, "Error creating kubernetes resource clients.")
	}
	plumberClient, err := plumber.NewPlumber(jxClient, ns)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting Plumber client.")
	}
	clientFactory := jxfactory.NewFactory()
	mpClient, err := plumber.NewMetaPipelineClient(clientFactory)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting Kubernetes client.")
	}
	c, err := tide.NewController(gitproviderClient, gitproviderClient, plumberClient, mpClient, tektonClient, ns, configGetter, gitClient, g.maxRecordsPerPool, g.opener, g.historyURI, g.statusURI, nil)
	return c, err
}

func createTideGitHubAppScmClient(gitServer string, token string) (*scm.Client, error) {
	client, err := factory.NewClient("github", gitServer, "")
	defaultScmTransport(client)
	auth := &transport.Authorization{
		Base:        http.DefaultTransport,
		Scheme:      "token",
		Credentials: token,
	}
	tr := &transport.Custom{Base: auth,
		Before: func(r *http.Request) {
			r.Header.Set("Accept", "application/vnd.github.machine-man-preview+json")
		},
	}
	client.Client.Transport = tr

	return client, err
}

func defaultScmTransport(scmClient *scm.Client) {
	if scmClient.Client == nil {
		scmClient.Client = http.DefaultClient
	}
	if scmClient.Client.Transport == nil {
		scmClient.Client.Transport = http.DefaultTransport
	}
}
