package githubapp

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/go-scm/scm/transport"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	jxutil "github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/keeper"
	"github.com/jenkins-x/lighthouse/pkg/keeper/history"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)


type gitHubAppKeeperController struct {
	controllers        []keeper.Controller
	ownerTokenFinder   *util.OwnerTokensDir
	gitServer          string
	githubAppSecretDir string
	configAgent        *config.Agent
	botName            string
	gitKind            string
	maxRecordsPerPool  int
	historyURI         string
	statusURI          string
	logger             *logrus.Entry
	m                  sync.Mutex
}

// NewGitHubAppKeeperController creates a GitHub App style controller which needs to process each github owner
// using a separate git provider client due to the way GitHub App tokens work
func NewGitHubAppKeeperController(githubAppSecretDir string, configAgent *config.Agent, botName string, gitKind string, maxRecordsPerPool int, historyURI string, statusURI string) (keeper.Controller, error) {

	gitServer := util.GithubServer
	return &gitHubAppKeeperController{
		ownerTokenFinder:  util.NewOwnerTokensDir(gitServer, githubAppSecretDir),
		gitServer:         gitServer,
		configAgent:       configAgent,
		botName:           botName,
		gitKind:           gitKind,
		maxRecordsPerPool: maxRecordsPerPool,
		historyURI:        historyURI,
		statusURI:         statusURI,
		logger:            logrus.NewEntry(logrus.StandardLogger()),
	}, nil

}

func (g *gitHubAppKeeperController) Sync() error {
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
	return jxutil.CombineErrors(errs...)
}

func (g *gitHubAppKeeperController) Shutdown() {
	for _, c := range g.controllers {
		c.Shutdown()
	}
	g.controllers = nil
}

func (g *gitHubAppKeeperController) GetPools() []keeper.Pool {
	g.m.Lock()
	defer g.m.Unlock()
	pools := []keeper.Pool{}
	for _, c := range g.controllers {
		cp := c.GetPools()
		pools = append(pools, cp...)
	}
	return pools
}

func (g *gitHubAppKeeperController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (g *gitHubAppKeeperController) GetHistory() *history.History {
	answer, err := history.New(g.maxRecordsPerPool, g.historyURI)
	if err != nil {
		return answer
	}
	for _, c := range g.controllers {
		h := c.GetHistory()
		answer.Merge(h)
	}
	return answer
}

func (g *gitHubAppKeeperController) createOwnerControllers() error {
	// lets zap any old controllers
	g.Shutdown()
	g.controllers = nil

	errs := []error{}

	cfg := g.configAgent.Config()
	if cfg == nil {
		return errors.New("no config")
	}

	oqs := SplitKeeperQueries(cfg.Keeper.Queries)
	for owner, queries := range oqs {
		// create copy of config with different queries
		ocfg := *cfg
		ocfg.Keeper.Queries = queries
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
	return jxutil.CombineErrors(errs...)
}

func (g *gitHubAppKeeperController) createOwnerController(owner string, configGetter config.Getter) (keeper.Controller, error) {
	token, err := g.ownerTokenFinder.FindToken(owner)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find GitHub App token for %s", owner)
	}
	if token == "" {
		return nil, errors.Errorf("no GitHub App token for %s", owner)
	}

	scmClient, err := createKeeperGitHubAppScmClient(g.gitServer, token)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create SCM client")
	}
	util.AddAuthToSCMClient(scmClient, token, true)
	gitproviderClient := scmprovider.ToClient(scmClient, g.botName)
	gitClient, err := git.NewClient(g.gitServer, g.gitKind)
	if err != nil {
		return nil, errors.Wrap(err, "creating git client")
	}
	gitClient.SetCredentials(util.GitHubAppGitRemoteUsername, func() []byte {
		return []byte(token)
	})
	tektonClient, jxClient, _, lhClient, ns, err := clients.GetClientsAndNamespace(nil)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating kubernetes resource clients.")
	}
	launcherClient, err := launcher.NewLauncher(jxClient, lhClient, ns)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting PipelineLauncher client.")
	}
	clientFactory := jxfactory.NewFactory()
	mpClient, err := launcher.NewMetaPipelineClient(clientFactory)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting Kubernetes client.")
	}
	c, err := keeper.NewController(gitproviderClient, gitproviderClient, launcherClient, mpClient, tektonClient, lhClient, ns, configGetter, gitClient, g.maxRecordsPerPool, g.historyURI, g.statusURI, nil)
	return c, err
}

func createKeeperGitHubAppScmClient(gitServer string, token string) (*scm.Client, error) {
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
