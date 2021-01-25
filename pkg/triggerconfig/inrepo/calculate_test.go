package inrepo_test

import (
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	fakescm "github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/config/lighthouse"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/inrepo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculate(t *testing.T) {
	owner := "myorg"
	repo := "myrepo"
	ref := "master"
	fullName := scm.Join(owner, repo)

	scmClient, fakeData := fakescm.NewDefault()
	scmProvider := scmprovider.ToClient(scmClient, "my-bot")
	fakeData.Repositories = []*scm.Repository{
		{
			Namespace: owner,
			Name:      repo,
			FullName:  fullName,
			Branch:    "master",
		},
	}

	enabled := true
	sharedConfig := &config.Config{
		ProwConfig: config.ProwConfig{
			InRepoConfig: lighthouse.InRepoConfig{
				Enabled: map[string]*bool{
					fullName: &enabled,
				},
			},
		},
	}
	sharedPluginConfig := &plugins.Configuration{}

	fileBrowsers, err := filebrowser.NewFileBrowsers(filebrowser.GitHubURL, filebrowser.NewFileBrowserFromScmClient(scmProvider))
	require.NoError(t, err, "failed to create filebrowsers")

	cfg, pluginsCfg, err := inrepo.Generate(fileBrowsers, sharedConfig, sharedPluginConfig, owner, repo, ref)
	require.NoError(t, err, "failed to calculate in repo config")

	require.NoError(t, err, "failed to invoke getClientAndTrigger")

	// lets verify we've loaded in repository global configuration
	require.NotNil(t, cfg, "no client.Config found")
	require.NotNil(t, pluginsCfg, "no client.PluginConfig found")

	// lets verify we've loaded in repository trigger configuration
	require.Len(t, cfg.Presubmits[fullName], 2, "presubmits for repo %s", fullName)
	var presubmit *job.Presubmit
	for _, p := range cfg.Presubmits[fullName] {
		if p.Name == "test" {
			pCopy := p
			presubmit = &pCopy
			break
		}
	}
	assert.NotNil(t, presubmit, "Couldn't find presubmit 'test' for repo %s", fullName)
	assert.NotNil(t, presubmit.PipelineRunSpec, "cfg.Presubmits[0].PipelineRunSpec for repo %s", fullName)
	assert.Equal(t, job.TektonPipelineAgent, presubmit.Agent, "cfg.Presubmits[0].Agent for repo %s", fullName)

	assert.NotNil(t, cfg.Presubmits[fullName][1].PipelineRunSpec, "cfg.Presubmits[1].PipelineRunSpec for repo %s", fullName)

	// lets verify we have a valid PipelineRunSpec and PipelineRunParams
	if assert.NotNil(t, presubmit.PipelineRunSpec.PipelineSpec, "cfg.Presubmits[0].PipelineRunSpec.PipelineSpec for repo %s", fullName) {
		t.Logf("repo %s has Presubmits[0] with %d tasks", fullName, len(presubmit.PipelineRunSpec.PipelineSpec.Tasks))
	}

	pipelineRunParams := presubmit.PipelineRunParams
	if assert.Len(t, pipelineRunParams, 1, "cfg.Presubmits[0].PipelineRunParams for repo %s", fullName) {
		t.Logf("repo %s has Presubmits[0] with PipelineRun Params %s = %s", fullName, pipelineRunParams[0].Name, pipelineRunParams[0].ValueTemplate)
	}

	require.Len(t, cfg.Postsubmits[fullName], 1, "postsubmits for repo %s", fullName)
	postsubmit := cfg.Postsubmits[fullName][0]
	assert.NotNil(t, postsubmit.PipelineRunSpec, "cfg.Postsubmits[0].PipelineRunSpec for repo %s", fullName)
	assert.Equal(t, job.TektonPipelineAgent, postsubmit.Agent, "cfg.Postsubmits[0].Agent for repo %s", fullName)

	// lets test we've a trigger configuration for this repository
	trigger := pluginsCfg.TriggerFor(owner, repo)
	require.NotNil(t, trigger, "no trigger found")
	require.Len(t, trigger.Repos, 1, "trigger.Repos")
	assert.Equal(t, trigger.Repos[0], fullName, "trigger.Repos[0]")
}

func TestTriggersInBranchMergeToMaster(t *testing.T) {
	owner := "myorg"
	repo := "branchtest"
	ref := "mybranch"
	fullName := scm.Join(owner, repo)

	scmClient, fakeData := fakescm.NewDefault()
	scmProvider := scmprovider.ToClient(scmClient, "my-bot")

	fakeData.Repositories = []*scm.Repository{
		{
			Namespace: owner,
			Name:      repo,
			FullName:  fullName,
			Branch:    "master",
		},
	}

	enabled := true
	sharedConfig := &config.Config{
		ProwConfig: config.ProwConfig{
			InRepoConfig: lighthouse.InRepoConfig{
				Enabled: map[string]*bool{
					fullName: &enabled,
				},
			},
		},
	}
	sharedPluginConfig := &plugins.Configuration{}

	fileBrowsers, err := filebrowser.NewFileBrowsers(filebrowser.GitHubURL, filebrowser.NewFileBrowserFromScmClient(scmProvider))
	require.NoError(t, err, "failed to create filebrowsers")

	cfg, _, err := inrepo.Generate(fileBrowsers, sharedConfig, sharedPluginConfig, owner, repo, ref)
	require.NoError(t, err, "failed to calculate in repo config")

	presubmits := cfg.Presubmits[fullName]
	presubmitNames := []string{}
	for _, ps := range presubmits {
		presubmitNames = append(presubmitNames, ps.Name)
	}
	t.Logf("repo %s has presubit names %v\n", fullName, presubmitNames)

	assert.Len(t, presubmits, 2, "presubmits for repo %s", fullName)
	assert.Contains(t, presubmitNames, "lint", "presubmits for repo %s", fullName)
	assert.Contains(t, presubmitNames, "newthingy", "presubmits for repo %s", fullName)

	assert.Len(t, cfg.Postsubmits[fullName], 1, "postsubmits for repo %s", fullName)

}
