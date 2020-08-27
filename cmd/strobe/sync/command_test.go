package sync

import (
	"encoding/json"
	"fmt"
	"testing"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/stretchr/testify/assert"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMakeCronJob(t *testing.T) {
	defaultNamespace := "default"
	one := int32(1)
	testCases := []struct {
		name           string
		namespace      string
		serviceAccount string
		periodic       job.Periodic
		tag            string
		expectedJob    lighthousev1alpha1.LighthouseJob
	}{
		{
			name:           "cronjob created",
			namespace:      "test",
			serviceAccount: "test-sa",
			periodic: job.Periodic{
				Base: job.Base{
					Name:      "test",
					Namespace: &defaultNamespace,
					Agent:     "tekton-pipeline",
					PipelineRunSpec: &pipelinev1beta1.PipelineRunSpec{
						PipelineRef: &pipelinev1beta1.PipelineRef{
							Name: "test",
						},
					},
				},
				Cron: "* * * * *",
			},
			tag: "test-tag",
			expectedJob: lighthousev1alpha1.LighthouseJob{
				TypeMeta: metav1.TypeMeta{
					Kind:       "LighthouseJob",
					APIVersion: "lighthouse.jenkins.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-",
					Namespace:    "test",
				},
				Spec: lighthousev1alpha1.LighthouseJobSpec{
					Agent:     "tekton-pipeline",
					Type:      "periodic",
					Job:       "test",
					Namespace: "default",
					PipelineRunSpec: &pipelinev1beta1.PipelineRunSpec{
						PipelineRef: &pipelinev1beta1.PipelineRef{
							Name: "test",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cronJob, err := makeCronJob(tc.namespace, tc.serviceAccount, tc.periodic, tc.tag)
			assert.NoError(t, err)
			assert.NotNil(t, cronJob)
			assert.Equal(t, tc.namespace, cronJob.Namespace)
			assert.Empty(t, cronJob.GenerateName)
			assert.Equal(t, tc.periodic.Name, cronJob.Name)
			assert.NotNil(t, cronJob.Labels)
			assert.NotNil(t, cronJob.Labels[periodicLabel])
			assert.Equal(t, periodicLabelValue, cronJob.Labels[periodicLabel])
			assert.Equal(t, v1beta1.ForbidConcurrent, cronJob.Spec.ConcurrencyPolicy)
			assert.Equal(t, &one, cronJob.Spec.SuccessfulJobsHistoryLimit)
			assert.Equal(t, &one, cronJob.Spec.FailedJobsHistoryLimit)
			expectedJob, _ := json.Marshal(tc.expectedJob)
			containers := []corev1.Container{{
				Name:  "strobe-start",
				Image: fmt.Sprintf("gcr.io/jenkinsxio/lighthouse-strobe:%s", tc.tag),
				Args:  []string{"start"},
				Env: []corev1.EnvVar{{
					Name:  "LHJOB",
					Value: string(expectedJob),
				}},
			}}
			assert.Equal(t, containers, cronJob.Spec.JobTemplate.Spec.Template.Spec.Containers)
		})
	}
}

func TestUpdateCronjobs(t *testing.T) {
	defaultNamespace := "default"
	periodic := job.Periodic{
		Base: job.Base{
			Name:      "test",
			Namespace: &defaultNamespace,
			Agent:     "tekton-pipeline",
			PipelineRunSpec: &pipelinev1beta1.PipelineRunSpec{
				PipelineRef: &pipelinev1beta1.PipelineRef{
					Name: "test",
				},
			},
		},
		Cron: "* * * * *",
	}
	cronjob, _ := makeCronJob(defaultNamespace, "test-sa", periodic, "test-tag")
	cronjob.Spec.Schedule = "1 1 1 1 1"
	testCases := []struct {
		name             string
		periodics        []job.Periodic
		cronjobs         []*v1beta1.CronJob
		expectedCronjobs map[string]string
	}{
		{
			name: "cronjob created",
			periodics: []job.Periodic{
				periodic,
			},
			cronjobs: []*v1beta1.CronJob{},
			expectedCronjobs: map[string]string{
				"test": "* * * * *",
			},
		},
		{
			name:      "cronjob delete",
			periodics: []job.Periodic{},
			cronjobs: []*v1beta1.CronJob{
				cronjob,
			},
			expectedCronjobs: map[string]string{},
		},
		{
			name: "cronjob updated",
			periodics: []job.Periodic{
				periodic,
			},
			cronjobs: []*v1beta1.CronJob{
				cronjob,
			},
			expectedCronjobs: map[string]string{
				"test": "* * * * *",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			objects := []runtime.Object{}
			for _, object := range tc.cronjobs {
				objects = append(objects, object)
			}
			client := fake.NewSimpleClientset(objects...)
			assert.NotNil(t, client)
			jobConfig := job.Config{
				Periodics: tc.periodics,
			}
			err := updateCronjobs(client, defaultNamespace, "test-sa", jobConfig, "test-tag")
			assert.NoError(t, err)
			for name, schedule := range tc.expectedCronjobs {
				cronjob, err := client.BatchV1beta1().CronJobs(defaultNamespace).Get(name, metav1.GetOptions{})
				assert.NoError(t, err)
				assert.NotNil(t, cronjob)
				assert.Equal(t, schedule, cronjob.Spec.Schedule)
			}
		})
	}
}
