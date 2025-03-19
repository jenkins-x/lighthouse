package keeper

import (
	"context"
	"strings"
	"testing"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	untypedcorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kpgapis "knative.dev/pkg/apis"
)

const (
	firstLabel  = "first-label"
	secondLabel = "second-label"
)

func TestRerunPipelineRunsWithRaceConditionFailure(t *testing.T) {
	tests := []struct {
		name           string
		originalStatus *kpgapis.Condition
		shouldRerun    bool
		alreadyRerun   bool
	}{
		{
			name: "pipeline can't be found",
			originalStatus: &kpgapis.Condition{
				Type:   kpgapis.ConditionSucceeded,
				Status: untypedcorev1.ConditionFalse,
				Message: "Pipeline jx/jenkins-x-jx-pr-5266-images-16 can't be found:pipeline.tekton.dev\n" +
					"\"jenkins-x-jx-pr-5266-images-16\" not found",
			},
			shouldRerun: true,
		},
		{
			name: "tasks that don't exist",
			originalStatus: &kpgapis.Condition{
				Type:   kpgapis.ConditionSucceeded,
				Status: untypedcorev1.ConditionFalse,
				Message: "Pipeline jx/jenkins-x-environment-tekton-we-xm2mg-13 can''t be Run;\n" +
					"it contains Tasks that don't exist: Couldn't retrieve Task \"jenkins-x-environment-tekton-we-xm2mg-release-13\":\n" +
					"task.tekton.dev \"jenkins-x-environment-tekton-we-xm2mg-release-13\"",
			},
			shouldRerun: true,
		},
		{
			name: "error retrieving pipeline",
			originalStatus: &kpgapis.Condition{
				Type:   kpgapis.ConditionSucceeded,
				Status: untypedcorev1.ConditionFalse,
				Message: "Error retrieving pipeline for pipelinerun jx/jenkins-x-jx-pr-5266-images-16:\n" +
					"error when listing pipelines for pipelineRun jenkins-x-jx-pr-5266-images-16:\n" +
					"pipeline.tekton.dev \"jenkins-x-jx-pr-5266-images-16\" not found",
			},
			shouldRerun: true,
		},
		{
			name: "other error shouldn't be rerun",
			originalStatus: &kpgapis.Condition{
				Type:    kpgapis.ConditionSucceeded,
				Status:  untypedcorev1.ConditionFalse,
				Message: "Any other error",
			},
			shouldRerun: false,
		},
		{
			name: "success should never be rerun",
			originalStatus: &kpgapis.Condition{
				Type:   kpgapis.ConditionSucceeded,
				Status: untypedcorev1.ConditionTrue,
				Message: "Error retrieving pipeline for pipelinerun jx/jenkins-x-jx-pr-5266-images-16:\n" +
					"error when listing pipelines for pipelineRun jenkins-x-jx-pr-5266-images-16:\n" +
					"pipeline.tekton.dev \"jenkins-x-jx-pr-5266-images-16\" not found",
			},
			shouldRerun: false,
		},
		{
			name: "unknown should never be rerun",
			originalStatus: &kpgapis.Condition{
				Type:   kpgapis.ConditionSucceeded,
				Status: untypedcorev1.ConditionUnknown,
				Message: "Error retrieving pipeline for pipelinerun jx/jenkins-x-jx-pr-5266-images-16:\n" +
					"error when listing pipelines for pipelineRun jenkins-x-jx-pr-5266-images-16:\n" +
					"pipeline.tekton.dev \"jenkins-x-jx-pr-5266-images-16\" not found",
			},
			shouldRerun: false,
		},
		{
			name: "already rerun",
			originalStatus: &kpgapis.Condition{
				Type:   kpgapis.ConditionSucceeded,
				Status: untypedcorev1.ConditionFalse,
				Message: "Pipeline jx/jenkins-x-jx-pr-5266-images-16 can't be found:pipeline.tekton.dev\n" +
					"\"jenkins-x-jx-pr-5266-images-16\" not found",
			},
			shouldRerun:  false,
			alreadyRerun: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ns := "some-namespace"
			baseRunName := "some-pipelinerun"
			pipelineName := "some-pipeline"
			saName := "some-sa"
			originalRun := makeTestPipelineRun(tc.originalStatus, baseRunName, pipelineName, saName, ns, tc.alreadyRerun)
			originalRunCopy := originalRun.DeepCopy()

			tektonClient := tektonfake.NewSimpleClientset(originalRun)

			err := rerunPipelineRunsWithRaceConditionFailure(tektonClient, ns, nil)
			assert.NoError(t, err)

			prList, err := tektonClient.TektonV1().PipelineRuns(ns).List(context.TODO(), metav1.ListOptions{})
			assert.NoError(t, err)

			if tc.shouldRerun {
				var updatedRun *pipelinev1.PipelineRun
				var rerunRun *pipelinev1.PipelineRun
				if len(prList.Items) != 2 {
					t.Fatalf("Expected 2 PipelineRuns, but there are %d", len(prList.Items))
				}
				for _, r := range prList.Items {
					if r.Name == baseRunName {
						updatedRun = (&r).DeepCopy()
					} else {
						rerunRun = (&r).DeepCopy()
					}
				}
				if updatedRun == nil {
					t.Fatalf("Updated original PipelineRun %s should exist", baseRunName)
				} else if rerunRun == nil {
					t.Fatal("There should be a second PipelineRun")
				} else {
					assert.True(t, strings.HasPrefix(rerunRun.Name, pipelineName+"-"), "New PipelineRun should start with %s-", pipelineName)
					rerunCondition := rerunRun.Status.GetCondition(kpgapis.ConditionSucceeded)
					assert.Nil(t, rerunCondition, "New PipelineRun should have an empty status")
					assert.Equal(t, "", rerunRun.ResourceVersion, "New PipelineRun should have had its ResourceVersion wiped out, but it is %s", rerunRun.ResourceVersion)

					rerunFailedLabel := rerunRun.Labels[labelFailedAndRerun]
					assert.Equal(t, "", rerunFailedLabel, "New PipelineRun shouldn't have a value for %s, but has %s", labelFailedAndRerun, rerunFailedLabel)

					if d := cmp.Diff(rerunRun.Spec, originalRunCopy.Spec); d != "" {
						t.Errorf("New PipelineRun spec did not match original: %s", d)
					}

					updatedFirstLabel := updatedRun.Labels[firstLabel]
					updatedSecondLabel := updatedRun.Labels[secondLabel]
					updatedFailedLabel := updatedRun.Labels[labelFailedAndRerun]
					assert.Equal(t, "true", updatedFailedLabel, "Faield PipelineRun should have a true value for %s, but has %s", labelFailedAndRerun, updatedFailedLabel)
					assert.Equal(t, "", updatedFirstLabel, "Failed PipelineRun shouldn't have a value for %s, but has %s", firstLabel, updatedFirstLabel)
					assert.Equal(t, "", updatedSecondLabel, "Failed PipelineRun shouldn't have a value for %s, but has %s", secondLabel, updatedSecondLabel)
				}
			} else {
				if len(prList.Items) != 1 {
					t.Fatalf("Expected 1 PipelineRun, but there are %d", len(prList.Items))
				}
				if !tc.alreadyRerun {
					updatedRun := prList.Items[0]
					updatedFailedLabel := updatedRun.Labels[labelFailedAndRerun]
					assert.Equal(t, "", updatedFailedLabel, "Unmodified PipelineRun should not have a value for %s, but has %s", labelFailedAndRerun, updatedFailedLabel)
				}
			}
		})
	}
}

func makeTestPipelineRun(condition *kpgapis.Condition, baseRunName string, pipelineName string, sa string, namespace string, alreadyRerun bool) *pipelinev1.PipelineRun {
	pr := &pipelinev1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: tektonAPIVersion,
			Kind:       "PipelineRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            baseRunName,
			Namespace:       namespace,
			ResourceVersion: "12345678",
		},
		Spec: pipelinev1.PipelineRunSpec{
			PipelineRef: &pipelinev1.PipelineRef{
				APIVersion: tektonAPIVersion,
				Name:       pipelineName,
			},
			TaskRunTemplate: pipelinev1.PipelineTaskRunTemplate{
				ServiceAccountName: sa,
			},
		},
		Status: pipelinev1.PipelineRunStatus{},
	}

	if alreadyRerun {
		pr.Labels = map[string]string{
			labelFailedAndRerun: "true",
		}
	} else {
		pr.Labels = map[string]string{
			firstLabel:  "foo",
			secondLabel: "bar",
		}
	}

	pr.Status.SetCondition(condition)

	return pr
}
