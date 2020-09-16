package inrepo_test

import (
	"os"
	"testing"

	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/inrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeConfigIntegration(t *testing.T) {
	token := os.Getenv("GIT_TOKEN")
	if token == "" {
		t.Logf("no $GIT_TOKEN defined so skipping this integration test")
		t.SkipNow()
		return
	}
	scmClient, _ := factory.NewClient("github", "https://github.com", token)
	scmProvider := scmprovider.ToClient(scmClient, "my-bot")

	repoOwner := "jenkins-x"
	repoName := "lighthouse-test-project"
	sha := ""
	cfg := &config.Config{}
	pluginCfg := &plugins.Configuration{}

	flag, err := inrepo.MergeTriggers(cfg, pluginCfg, scmProvider, repoOwner, repoName, sha)
	require.NoError(t, err, "failed to merge configs")
	assert.True(t, flag, "did not return merge flag")

	// lets display the context
	LogConfig(t, cfg)

	r := repoOwner + "/" + repoName
	assert.Len(t, cfg.Presubmits[r], 2, "presubmits for repo %s", r)
	assert.Len(t, cfg.Postsubmits[r], 1, "postsubmits for repo %s", r)
}

// LogConfig displays the generated config
func LogConfig(t *testing.T, cfg *config.Config) {
	for k, v := range cfg.Presubmits {
		t.Logf("presubmits for repository %s\n", k)

		for _, r := range v {
			t.Logf("  presubmit %s\n", r.Name)
		}
	}
	for k, v := range cfg.Postsubmits {
		t.Logf("postsubmits for repository %s\n", k)

		for _, r := range v {
			t.Logf("  postsubmits %s\n", r.Name)
		}
	}
}
