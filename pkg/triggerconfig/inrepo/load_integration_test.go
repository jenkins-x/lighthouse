package inrepo

import (
	fbfake "github.com/jenkins-x/lighthouse/pkg/filebrowser/fake"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
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
	repoOwner := "jenkins-x"
	repoName := "lighthouse-test-project"
	sha := ""
	cfg := &config.Config{}
	pluginCfg := &plugins.Configuration{}

	fileBrowsers, err := filebrowser.NewFileBrowsers(filebrowser.GitHubURL, fbfake.NewFakeFileBrowser(filepath.Join("test_data"), true))
	require.NoError(t, err, "failed to create filebrowsers")

	fc := filebrowser.NewFetchCache()
	flag, err := MergeTriggers(cfg, pluginCfg, fileBrowsers, fc, NewResolverCache(), repoOwner, repoName, sha)
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
			t.Logf("  presubmit %s trigger: %s rerun_command: %s\n", r.Name, r.Trigger, r.RerunCommand)
		}
	}
	for k, v := range cfg.Postsubmits {
		t.Logf("postsubmits for repository %s\n", k)

		for _, r := range v {
			t.Logf("  postsubmits %s\n", r.Name)
		}
	}
}
