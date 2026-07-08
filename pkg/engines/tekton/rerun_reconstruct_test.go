package tekton

import (
	"errors"
	"testing"

	configjob "github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func basePipelineRun() *pipelinev1.PipelineRun {
	return &pipelinev1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myorg-myrepo-main-abc12-2-rerun",
			Namespace: "jx",
			Labels: map[string]string{
				configjob.LighthouseJobTypeLabel: string(configjob.PostsubmitJob),
				util.ContextLabel:                "build",
				util.OrgLabel:                    "myorg",
				util.RepoLabel:                   "myrepo",
				util.LastCommitSHALabel:          "sha123",
				util.BaseSHALabel:                "basesha456",
				util.BranchLabel:                 "main",
			},
			Annotations: map[string]string{
				util.LighthouseJobAnnotation: "myorg-myrepo-build",
				util.CloneURIAnnotation:      "https://github.com/myorg/myrepo.git",
			},
		},
	}
}

func TestRerunSpecFromPipelineRun_Postsubmit(t *testing.T) {
	spec, err := rerunSpecFromPipelineRun(basePipelineRun())
	require.NoError(t, err)
	assert.Equal(t, configjob.TektonPipelineAgent, spec.Agent)
	assert.Equal(t, configjob.PostsubmitJob, spec.Type)
	assert.Equal(t, "build", spec.Context)
	assert.Equal(t, "myorg-myrepo-build", spec.Job)
	require.NotNil(t, spec.Refs)
	assert.Equal(t, "myorg", spec.Refs.Org)
	assert.Equal(t, "myrepo", spec.Refs.Repo)
	assert.Equal(t, "basesha456", spec.Refs.BaseSHA)
	assert.Equal(t, "main", spec.Refs.BaseRef, "postsubmit keeps the base ref")
	assert.Equal(t, "https://github.com/myorg/myrepo.git", spec.Refs.CloneURI)
	assert.Empty(t, spec.Refs.Pulls)
}

func TestRerunSpecFromPipelineRun_PresubmitClearsBaseRefAndSetsPulls(t *testing.T) {
	pr := basePipelineRun()
	pr.Labels[configjob.LighthouseJobTypeLabel] = string(configjob.PresubmitJob)
	pr.Labels[util.BranchLabel] = "PR-42"
	pr.Labels[util.PullLabel] = "42"

	spec, err := rerunSpecFromPipelineRun(pr)
	require.NoError(t, err)
	assert.Empty(t, spec.Refs.BaseRef, "presubmit must clear the base ref")
	require.Len(t, spec.Refs.Pulls, 1)
	assert.Equal(t, 42, spec.Refs.Pulls[0].Number)
	assert.Equal(t, "sha123", spec.Refs.Pulls[0].SHA)
}

func TestRerunSpecFromPipelineRun_BatchClearsBaseRef(t *testing.T) {
	pr := basePipelineRun()
	pr.Labels[configjob.LighthouseJobTypeLabel] = string(configjob.BatchJob)

	spec, err := rerunSpecFromPipelineRun(pr)
	require.NoError(t, err)
	assert.Empty(t, spec.Refs.BaseRef)
}

func TestRerunSpecFromPipelineRun_NonNumericPullIgnored(t *testing.T) {
	pr := basePipelineRun()
	pr.Labels[configjob.LighthouseJobTypeLabel] = string(configjob.PresubmitJob)
	pr.Labels[util.PullLabel] = "not-a-number"

	spec, err := rerunSpecFromPipelineRun(pr)
	require.NoError(t, err)
	assert.Empty(t, spec.Refs.Pulls)
}

func TestRerunSpecFromPipelineRun_MissingRequiredLabels(t *testing.T) {
	pr := basePipelineRun()
	delete(pr.Labels, util.ContextLabel)
	delete(pr.Labels, util.OrgLabel)

	_, err := rerunSpecFromPipelineRun(pr)
	require.Error(t, err)
	var missing *ErrMissingRequiredLabels
	require.True(t, errors.As(err, &missing))
	assert.ElementsMatch(t, []string{util.ContextLabel, util.OrgLabel}, missing.Missing)
}
