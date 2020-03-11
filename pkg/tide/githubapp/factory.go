package githubapp

import (
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/io"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	"github.com/jenkins-x/lighthouse/pkg/prow/gitprovider"
	"github.com/jenkins-x/lighthouse/pkg/tide"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
)

// NewTideController creates a new controller; either regular or a GitHub App flavour
// depending on the $GITHUB_APP_SECRET_DIR environment variable
func NewTideController(configAgent *config.Agent, botName string, gitKind string, gitToken string, serverURL string, maxRecordsPerPool int, opener io.Opener, historyURI string, statusURI string) (tide.Controller, error) {
	githubAppSecretDir := util.GetGitHubAppSecretDir()
	if githubAppSecretDir != "" {
		return NewGitHubAppTideController(githubAppSecretDir, configAgent, botName, gitKind, maxRecordsPerPool, opener, historyURI, statusURI)
	}

	scmClient, err := factory.NewClient(gitKind, serverURL, "")
	if err != nil {
		return nil, errors.Wrap(err, "cannot create SCM client")
	}
	util.AddAuthToSCMClient(scmClient, gitToken, false)
	gitproviderClient := gitprovider.ToClient(scmClient, botName)
	gitClient, err := git.NewClient(serverURL, botName)
	if err != nil {
		return nil, errors.Wrap(err, "creating git client")
	}
	gitClient.SetCredentials(botName, func() []byte {
		return []byte(gitToken)
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
	c, err := tide.NewController(gitproviderClient, gitproviderClient, plumberClient, mpClient, tektonClient, ns, configAgent.Config, gitClient, maxRecordsPerPool, opener, historyURI, statusURI, nil)
	return c, err
}
