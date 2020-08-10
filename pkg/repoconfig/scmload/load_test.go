package scmload_test

import (
	"testing"

	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/repoconfig/scmload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeConfig(t *testing.T) {
	scmClient, _ := fake.NewDefault()
	repoOwner := "myorg"
	repoName := "myrepo"
	sha := "TODO"
	cfg := &config.Config{}
	pluginCfg := &plugins.Configuration{}
	flag, err := scmload.MergeConfig(cfg, pluginCfg, scmClient, repoOwner, repoName, sha)
	require.NoError(t, err, "failed to merge configs")
	assert.True(t, flag, "did not return merge flag")

	LogConfig(t, cfg)

	r := repoOwner + "/" + repoName
	assert.Len(t, cfg.Presubmits[r], 2, "presubmits for repo %s", r)
	assert.Len(t, cfg.Postsubmits[r], 1, "postsubmits for repo %s", r)
}
