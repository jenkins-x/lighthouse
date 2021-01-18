package inrepo

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/config/job"

	"github.com/h2non/gock"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeConfig(t *testing.T) {
	owner := "myorg"
	repo := "loadtest"
	ref := "master"
	fullName := scm.Join(owner, repo)

	scmClient, fakeData := fake.NewDefault()
	scmProvider := scmprovider.ToClient(scmClient, "my-bot")

	fakeData.Repositories = []*scm.Repository{
		{
			Namespace: owner,
			Name:      repo,
			FullName:  fullName,
			Branch:    "master",
		},
	}

	cfg := &config.Config{}
	pluginCfg := &plugins.Configuration{}
	fileBrowser := filebrowser.NewFileBrowserFromScmClient(scmProvider)
	flag, err := MergeTriggers(cfg, pluginCfg, fileBrowser, owner, repo, ref)
	require.NoError(t, err, "failed to merge configs")
	assert.True(t, flag, "did not return merge flag")

	LogConfig(t, cfg)

	r := owner + "/" + repo
	assert.Len(t, cfg.Presubmits[r], 2, "presubmits for repo %s", r)
	assert.Len(t, cfg.Postsubmits[r], 1, "postsubmits for repo %s", r)
}

func TestInvalidConfigs(t *testing.T) {
	scmClient, _ := fake.NewDefault()
	scmProvider := scmprovider.ToClient(scmClient, "my-bot")
	fileBrowser := filebrowser.NewFileBrowserFromScmClient(scmProvider)

	invalidRepos := []string{"duplicate-presubmit", "duplicate-postsubmit"}
	for _, repo := range invalidRepos {
		owner := "myorg"
		ref := "master"
		_, err := LoadTriggerConfig(fileBrowser, owner, repo, ref)
		require.Errorf(t, err, "should have failed to load triggers from repo %s/%s with ref %s", owner, repo, ref)

		t.Logf("got expected error loading invalid configuration on repo %s of: %s", repo, err.Error())
	}
}

func TestLoadJobFromURL(t *testing.T) {
	defer gock.Off()

	gock.New("https://raw.githubusercontent.com").
		Get("/rawlingsj/test/master/foo.yaml").
		Reply(200).
		Type("application/json").
		File("test_data/load_url/foo.yaml")

	j := &job.Base{}
	err := loadJobBaseFromSourcePath(nil, j, "", "", "https://raw.githubusercontent.com/rawlingsj/test/master/foo.yaml", "")
	assert.NoError(t, err, "should not have an error returned")
	assert.Equal(t, "jenkinsxio/chuck:0.0.1", j.PipelineRunSpec.PipelineSpec.Tasks[0].TaskSpec.Steps[0].Image, "image name for task is not correct")
}
