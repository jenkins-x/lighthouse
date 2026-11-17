package webhook

import (
	"os"
	"testing"

	lru "github.com/hashicorp/golang-lru"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmclients"
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

	scmClients, err := scmclients.New(cfg, scmclients.Options{BotName: gitUser})
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, scmClients.Clean())
	})

	_, kubeClient, lhClient, _, err := clients.GetAPIClients()
	assert.NoError(t, err)

	s.FileBrowsers = scmClients.FileBrowsers

	s.InRepoCache, err = lru.New(5000)
	assert.NoError(t, err)

	s.ClientAgent = &plugins.ClientAgent{
		BotName:           scmClients.BotName,
		SCMProviderClient: scmClients.SCMProviderClient.ToScmClient(),
		KubernetesClient:  kubeClient,
		GitClient:         scmClients.GitClient,
		LighthouseClient:  lhClient.LighthouseV1alpha1().LighthouseJobs("jx"),
		LauncherClient:    launcher.NewLauncher(lhClient, "jx"),
	}

	l := logrus.WithField("Repository", scm.Join(owner, repo)+"@"+ref)
	_, err = s.CreateAgent(l, owner, repo, ref)
	assert.NoError(t, err)
}
