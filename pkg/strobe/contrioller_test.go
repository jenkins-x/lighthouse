package strobe

import (
	"testing"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateLighthouseJob(t *testing.T) {
	namespace := "lighthouse"
	expectedLighthouseJob := &v1alpha1.LighthouseJob{
		ObjectMeta: metav1.ObjectMeta{
			// It is important for the LighthouseJob name to remain
			// deterministic across releases to prevent duplicate jobs from
			// being created for the same schedule time during a rolling update
			Name: "hello-world-27303840",
			Labels: map[string]string{
				"created-by-lighthouse":        "true",
				"lighthouse.jenkins-x.io/job":  "hello-world",
				"lighthouse.jenkins-x.io/type": "periodic",
			},
			Annotations: map[string]string{
				"lighthouse.jenkins-x.io/job": "hello-world",
			},
		},
		Spec: v1alpha1.LighthouseJobSpec{
			Type:      job.PeriodicJob,
			Agent:     job.TektonPipelineAgent,
			Namespace: namespace,
			Job:       "hello-world",
			PipelineRunSpec: &tektonv1beta1.PipelineRunSpec{
				PipelineSpec: &tektonv1beta1.PipelineSpec{
					Tasks: []tektonv1beta1.PipelineTask{
						{
							Name: "hello-world",
							TaskSpec: &tektonv1beta1.EmbeddedTask{
								TaskSpec: tektonv1beta1.TaskSpec{
									Steps: []tektonv1beta1.Step{
										{
											Container: v1.Container{
												Image: "busybox",
											},
											Script: "echo 'Hello World!'",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	periodicJobConfig := &job.Periodic{
		Cron: "*/1 * * * *",
		Base: job.Base{
			Name:      "hello-world",
			Namespace: &namespace,
			Agent:     job.TektonPipelineAgent,
			PipelineRunSpec: &tektonv1beta1.PipelineRunSpec{
				PipelineSpec: &tektonv1beta1.PipelineSpec{
					Tasks: []tektonv1beta1.PipelineTask{
						{
							Name: "hello-world",
							TaskSpec: &tektonv1beta1.EmbeddedTask{
								TaskSpec: tektonv1beta1.TaskSpec{
									Steps: []tektonv1beta1.Step{
										{
											Container: v1.Container{
												Image: "busybox",
											},
											Script: "echo 'Hello World!'",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	lighthouseJob := generateLighthouseJob(logrus.NewEntry(logrus.StandardLogger()), periodicJobConfig, time.Date(2022, 0, 0, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, expectedLighthouseJob, lighthouseJob)
}
