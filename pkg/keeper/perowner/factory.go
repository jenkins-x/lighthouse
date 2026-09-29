// Package perowner builds keeper controllers, one per owner when credentials are scoped per owner.
package perowner

import (
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/keeper"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/scmclients"
	"github.com/pkg/errors"
)

// NewKeeperController creates a keeper controller for the credential mode of scmClients.
func NewKeeperController(scmClients *scmclients.Clients, configAgent *config.Agent, maxRecordsPerPool int, historyURI string, statusURI string, ns string) (keeper.Controller, error) {
	if scmClients.RequiresOwner() {
		return NewPerOwnerKeeperController(scmClients, configAgent, maxRecordsPerPool, historyURI, statusURI, ns)
	}

	ownerClients, err := scmClients.ForOwner("")
	if err != nil {
		return nil, errors.Wrap(err, "cannot create SCM client")
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
	launcherClient := launcher.NewLauncher(lhClient, ns)

	return keeper.NewController(ownerClients.Provider, ownerClients.Provider, fileBrowsers,
		launcherClient, tektonClient, lhClient, ns, configAgent.Config, gitClient,
		maxRecordsPerPool, historyURI, statusURI, nil)
}
