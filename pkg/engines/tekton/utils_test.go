package tekton

import (
	"context"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestMakePeriodicPipelineRun(t *testing.T) {
	namespace := "lighthouse"
	lighthouseJob := v1alpha1.LighthouseJob{
		Spec: v1alpha1.LighthouseJobSpec{
			Type:      job.PeriodicJob,
			Job:       "hello-world",
			Namespace: namespace,
			Agent:     job.TektonPipelineAgent,
			PipelineRunSpec: &tektonv1beta1.PipelineRunSpec{
				PipelineSpec: &tektonv1beta1.PipelineSpec{},
			},
		},
	}

	pipelineRun, err := makePipelineRun(context.TODO(), lighthouseJob, []*v1alpha1.LighthouseBreakpoint{}, namespace, logrus.NewEntry(logrus.StandardLogger()), &epochBuildIDGenerator{}, nil)

	// Check there was no error generating periodic PipelineRun
	assert.NoError(t, err)

	// Check the PipelineRun name was generated correctly
	assert.Equal(t, "hello-world-", pipelineRun.GenerateName)
}
