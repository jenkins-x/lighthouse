package jx_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	v1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/engines/jx"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestConvertPipelineActivity(t *testing.T) {
	workDir, err := os.Getwd()
	assert.NoError(t, err)
	activityBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/test_data/pipelineactivity.yaml", workDir))
	assert.NoError(t, err)

	activity := &v1.PipelineActivity{}
	err = yaml.Unmarshal(activityBytes, activity)
	assert.NoError(t, err)

	jobBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/test_data/lhjob.yaml", workDir))
	assert.NoError(t, err)
	job := &v1alpha1.LighthouseJob{}
	err = yaml.Unmarshal(jobBytes, job)
	assert.NoError(t, err)

	converted, err := jx.ConvertPipelineActivity(activity)
	assert.NoError(t, err)

	assert.Equal(t, job.Labels[util.OrgLabel], converted.Owner)
	assert.Equal(t, job.Labels[util.RepoLabel], converted.Repo)
	assert.Equal(t, job.Labels[util.BranchLabel], converted.Branch)
	assert.Equal(t, job.Labels[util.BuildNumLabel], converted.BuildIdentifier)
	assert.Equal(t, job.Labels[util.ContextLabel], converted.Context)

	expectedBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/test_data/record.yaml", workDir))
	assert.NoError(t, err)
	expectedRecord := &v1alpha1.ActivityRecord{}
	err = yaml.Unmarshal(expectedBytes, expectedRecord)

	assert.Equal(t, expectedRecord, converted)
}
