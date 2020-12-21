package githubapp

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/keeper"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
)

// NewKeeperController creates a new controller; either regular or a GitHub App flavour
// depending on the $GITHUB_APP_SECRET_DIR environment variable
func NewKeeperController(configAgent *config.Agent, botName string, gitKind string, gitToken string, serverURL string, maxRecordsPerPool int, historyURI string, statusURI string, ns string) (keeper.Controller, error) {
	githubAppSecretDir := util.GetGitHubAppSecretDir()
	if githubAppSecretDir != "" {
		return NewGitHubAppKeeperController(githubAppSecretDir, configAgent, botName, gitKind, maxRecordsPerPool, historyURI, statusURI, ns)
	}

	var scmClient *scm.Client
	var err error
	if gitKind == "gitea" {
		// gitea returns 403 if the gitToken isn't passed here
		scmClient, err = factory.NewClient(gitKind, serverURL, gitToken)
	} else {
		scmClient, err = factory.NewClient(gitKind, serverURL, "")
	}
	if err != nil {
		return nil, errors.Wrap(err, "cannot create SCM client")
	}
	util.AddAuthToSCMClient(scmClient, gitToken, false)
	gitproviderClient := scmprovider.ToClient(scmClient, botName)
	gitClient, err := git.NewClient(serverURL, botName)
	if err != nil {
		return nil, errors.Wrap(err, "creating git client")
	}
	gitClient.SetCredentials(botName, func() []byte {
		return []byte(gitToken)
	})

	tektonClient, _, lhClient, _, err := clients.GetAPIClients()
	if err != nil {
		return nil, errors.Wrap(err, "Error creating kubernetes resource clients.")
	}
	launcherClient := launcher.NewLauncher(lhClient, ns)
	c, err := keeper.NewController(gitproviderClient, gitproviderClient, launcherClient, tektonClient, lhClient, ns, configAgent.Config, gitClient, maxRecordsPerPool, historyURI, statusURI, nil)
	return c, err
}
