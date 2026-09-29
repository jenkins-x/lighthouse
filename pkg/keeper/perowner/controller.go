package perowner

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/keeper"
	"github.com/jenkins-x/lighthouse/pkg/keeper/history"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/scmclients"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type perOwnerKeeperController struct {
	controllers       []keeper.Controller
	scmClients        *scmclients.Clients
	configAgent       *config.Agent
	maxRecordsPerPool int
	historyURI        string
	statusURI         string
	ns                string
	logger            *logrus.Entry
	m                 sync.Mutex
}

// NewPerOwnerKeeperController creates a controller that runs one keeper per owner.
func NewPerOwnerKeeperController(scmClients *scmclients.Clients, configAgent *config.Agent, maxRecordsPerPool int, historyURI string, statusURI string, ns string) (keeper.Controller, error) {
	return &perOwnerKeeperController{
		scmClients:        scmClients,
		configAgent:       configAgent,
		maxRecordsPerPool: maxRecordsPerPool,
		historyURI:        historyURI,
		statusURI:         statusURI,
		ns:                ns,
		logger:            logrus.NewEntry(logrus.StandardLogger()),
	}, nil
}

func (g *perOwnerKeeperController) Sync() error {
	// lets iterate through the config and create a controller for each
	err := g.createOwnerControllers()
	if err != nil {
		return err
	}
	// now lets sync them all
	var errs *multierror.Error
	for _, c := range g.controllers {
		err := c.Sync()
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs.ErrorOrNil()
}

func (g *perOwnerKeeperController) Shutdown() {
	for _, c := range g.controllers {
		c.Shutdown()
	}
	g.controllers = nil
}

func (g *perOwnerKeeperController) GetPools() []keeper.Pool {
	g.m.Lock()
	defer g.m.Unlock()
	pools := []keeper.Pool{}
	for _, c := range g.controllers {
		cp := c.GetPools()
		pools = append(pools, cp...)
	}
	return pools
}

func (g *perOwnerKeeperController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func (g *perOwnerKeeperController) GetHistory() *history.History {
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

func (g *perOwnerKeeperController) createOwnerControllers() error {
	// lets zap any old controllers
	g.Shutdown()
	g.controllers = nil

	var errs *multierror.Error

	cfg := g.configAgent.Config()
	if cfg == nil {
		return errors.New("no config")
	}

	oqs := splitKeeperQueries(cfg.Keeper.Queries)
	for owner, queries := range oqs {
		// create copy of config with different queries
		ocfg := *cfg
		ocfg.Keeper.Queries = queries
		configGetter := func() *config.Config {
			return &ocfg
		}

		c, err := g.createOwnerController(owner, configGetter)
		if err != nil {
			errs = multierror.Append(errs, err)
		} else {
			g.controllers = append(g.controllers, c)
		}
	}
	return errs.ErrorOrNil()
}

func (g *perOwnerKeeperController) createOwnerController(owner string, configGetter config.Getter) (keeper.Controller, error) {
	ownerClients, err := g.scmClients.ForOwner(owner)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot create SCM client for owner %s", owner)
	}

	if err := ownerClients.CheckCredential(); err != nil {
		return nil, err
	}

	gitClient, err := ownerClients.Git()
	if err != nil {
		return nil, err
	}

	fileBrowsers, err := ownerClients.Browsers()
	if err != nil {
		return nil, err
	}

	tektonClient, _, lhClient, _, err := clients.GetAPIClients()
	if err != nil {
		return nil, errors.Wrap(err, "Error creating kubernetes resource clients.")
	}
	launcherClient := launcher.NewLauncher(lhClient, g.ns)

	return keeper.NewController(ownerClients.Provider, ownerClients.Provider, fileBrowsers,
		launcherClient, tektonClient, lhClient, g.ns, configGetter, gitClient,
		g.maxRecordsPerPool, g.historyURI, g.statusURI, nil)
}
