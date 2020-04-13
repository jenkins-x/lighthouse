package githubapp

import (
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/jx/pkg/jxfactory"
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
func NewKeeperController(configAgent *config.Agent, botName string, gitKind string, gitToken string, serverURL string, maxRecordsPerPool int, historyURI string, statusURI string) (keeper.Controller, error) {
	clientFactory := jxfactory.NewFactory()
	mpClient, err := launcher.NewMetaPipelineClient(clientFactory)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting Kubernetes client.")
	}
	githubAppSecretDir := util.GetGitHubAppSecretDir()
	if githubAppSecretDir != "" {
		return NewGitHubAppKeeperController(githubAppSecretDir, configAgent, mpClient, botName, gitKind, maxRecordsPerPool, historyURI, statusURI)
	}

	scmClient, err := factory.NewClient(gitKind, serverURL, "")
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

	tektonClient, jxClient, _, lhClient, ns, err := clients.GetClientsAndNamespace(nil)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating kubernetes resource clients.")
	}
	launcherClient, err := launcher.NewLauncher(jxClient, lhClient, ns)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting PipelineLauncher client.")
	}
	c, err := keeper.NewController(gitproviderClient, gitproviderClient, launcherClient, mpClient, tektonClient, lhClient, ns, configAgent.Config, gitClient, maxRecordsPerPool, historyURI, statusURI, nil)
	return c, err
}
