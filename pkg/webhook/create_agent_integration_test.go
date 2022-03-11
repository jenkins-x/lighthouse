package webhook

import (
	"os"
	"testing"

	lru "github.com/hashicorp/golang-lru"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/git"
	gitv2 "github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCreateAgentIntegration(t *testing.T) {
	owner := os.Getenv("GIT_OWNER")
	repo := os.Getenv("GIT_REPO")
	ref := os.Getenv("GIT_REF")
	gitUser := os.Getenv("GIT_USERNAME")
	gitToken := os.Getenv("GIT_TOKEN")
	if owner == "" || repo == "" || ref == "" || gitUser == "" || gitToken == "" {
		t.Skipf("skipping integration test as missing $GIT_OWNER/$GIT_REPO/$GIT_REF/$GIT_USERNAME/$GIT_TOKEN")
		return
	}

	s := Server{}
	s.ConfigAgent = &config.Agent{}
	s.Plugins = &plugins.ConfigAgent{}

	cfg := s.ConfigAgent.Config

	_, err := watcher.SetupConfigMapWatchers("jx", s.ConfigAgent, s.Plugins)
	assert.NoError(t, err)

	_, scmClient, serverURL, _, err := util.GetSCMClient("", cfg)
	assert.NoError(t, err)

	_, kubeClient, lhClient, _, err := clients.GetAPIClients()
	assert.NoError(t, err)

	gitClient, err := git.NewClient(serverURL, util.GitKind(cfg))
	assert.NoError(t, err)

	configureOpts := func(opts *gitv2.ClientFactoryOpts) {
		opts.Token = func() []byte {
			return []byte(gitToken)

		}
		opts.GitUser = func() (name, email string, err error) {
			name = gitUser
			return
		}
		opts.Username = func() (login string, err error) {
			login = gitUser
			return
		}

	}

	gitFactory, err := gitv2.NewClientFactory(configureOpts)
	assert.NoError(t, err)

	fb := filebrowser.NewFileBrowserFromGitClient(gitFactory)

	s.FileBrowsers, err = filebrowser.NewFileBrowsers(serverURL, fb)
	assert.NoError(t, err)

	s.InRepoCache, err = lru.New(5000)
	assert.NoError(t, err)

	s.ClientAgent = &plugins.ClientAgent{
		BotName:           "test-bot",
		SCMProviderClient: scmClient,
		KubernetesClient:  kubeClient,
		GitClient:         gitClient,
		LighthouseClient:  lhClient.LighthouseV1alpha1().LighthouseJobs("jx"),
		LauncherClient:    launcher.NewLauncher(lhClient, "jx", "jx", "jx"),
	}

	l := logrus.WithField("Repository", scm.Join(owner, repo)+"@"+ref)
	_, err = s.CreateAgent(l, owner, repo, ref)
	assert.NoError(t, err)
}
