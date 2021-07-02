package githubapp

import (
	"net/url"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/git"
	gitv2 "github.com/jenkins-x/lighthouse/pkg/git/v2"
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
		scmClient, err = factory.NewClient(gitKind, serverURL, gitToken, factory.SetUsername(botName))
	} else {
		scmClient, err = factory.NewClient(gitKind, serverURL, "", factory.SetUsername(botName))
	}
	if err != nil {
		return nil, errors.Wrap(err, "cannot create SCM client")
	}
	util.AddAuthToSCMClient(scmClient, gitToken, false)
	gitproviderClient := scmprovider.ToClient(scmClient, botName)
	gitClient, err := git.NewClient(serverURL, gitKind)
	if err != nil {
		return nil, errors.Wrap(err, "creating git client")
	}
	gitClient.SetCredentials(botName, func() []byte {
		return []byte(gitToken)
	})

	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", serverURL)
	}

	gitCloneUser := botName

	configureOpts := func(opts *gitv2.ClientFactoryOpts) {
		opts.Token = func() []byte {
			return []byte(gitToken)
		}
		opts.GitUser = func() (name, email string, err error) {
			name = gitCloneUser
			return
		}
		opts.Username = func() (login string, err error) {
			login = gitCloneUser
			return
		}
		if u.Host != "" {
			opts.Host = u.Host
		}
		if u.Scheme != "" {
			opts.Scheme = u.Scheme
		}
	}
	gitFactory, err := gitv2.NewClientFactory(configureOpts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create git client factory for server %s", serverURL)
	}
	fb := filebrowser.NewFileBrowserFromGitClient(gitFactory)
	fileBrowsers, err := filebrowser.NewFileBrowsers(serverURL, fb)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create git file browser")
	}

	tektonClient, _, lhClient, _, err := clients.GetAPIClients()
	if err != nil {
		return nil, errors.Wrap(err, "Error creating kubernetes resource clients.")
	}
	launcherClient := launcher.NewLauncher(lhClient, ns)
	c, err := keeper.NewController(gitproviderClient, gitproviderClient, fileBrowsers, launcherClient, tektonClient, lhClient, ns, configAgent.Config, gitClient, maxRecordsPerPool, historyURI, statusURI, nil)
	return c, err
}
