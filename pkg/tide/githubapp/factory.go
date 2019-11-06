package githubapp

import (
	"os"

	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/io"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	"github.com/jenkins-x/lighthouse/pkg/prow/gitprovider"
	"github.com/jenkins-x/lighthouse/pkg/tide"
	"github.com/pkg/errors"
)

func NewTideController(configAgent *config.Agent, botName string, gitClient git.Client, maxRecordsPerPool int, opener io.Opener, historyURI string, statusURI string) (tide.TideController, error) {
	githubAppSecretDir := os.Getenv("GITHUB_APP_SECRET_DIR")
	if githubAppSecretDir != "" {
		return NewGitHubAppTideController(githubAppSecretDir, configAgent, botName, gitClient, maxRecordsPerPool, opener, historyURI, statusURI)
	}

	scmClient, err := factory.NewClientFromEnvironment()
	if err != nil {
		return nil, errors.Wrap(err, "cannot create SCM client")
	}
	gitproviderClient := gitprovider.ToClient(scmClient, botName)
	_, jxClient, _, ns, err := clients.GetClientsAndNamespace()
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
	c, err := tide.NewController(gitproviderClient, gitproviderClient, plumberClient, mpClient, configAgent.Config, gitClient, maxRecordsPerPool, opener, historyURI, statusURI, nil)
	return c, err
}
