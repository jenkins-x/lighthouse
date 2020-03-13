package tide

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pipelinev1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	untypedcorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	kpgapis "knative.dev/pkg/apis"
)

const (
	// tektonAPIVersion the APIVersion for using Tekton
	tektonAPIVersion = "tekton.dev/v1alpha1"

	// labelFailedAndRerun is added to PipelineRuns to replace their existing labels when the run fails for rerunnable reasons
	labelFailedAndRerun = "lighthouse-failed-and-rerun"
)

var (
	// pipelineRunShouldRetryMessages are the error messages that show up in the PipelineRun status for race conditions that should lead to the run being retried
	pipelineRunShouldRetryMessages = []string{
		"can't be found:pipeline.tekton.dev",
		"it contains Tasks that don't exist",
		"Error retrieving pipeline for pipelinerun",
	}
)

func rerunPipelineRunsWithRaceConditionFailure(tektonClient tektonclient.Interface, ns string, logger *logrus.Entry) error {
	// Get all PipelineRuns without the label indicating we've already rerun it.
	notRerunLabelSelector := fmt.Sprintf("!%s", labelFailedAndRerun)
	runs, err := tektonClient.TektonV1alpha1().PipelineRuns(ns).List(metav1.ListOptions{
		LabelSelector: notRerunLabelSelector,
	})
	if err != nil {
		return errors.Wrapf(err, "listing PipelineRuns in %s", ns)
	}

	// Filter through the runs to find failed ones
	for _, run := range runs.Items {
		runCondition := run.Status.GetCondition(kpgapis.ConditionSucceeded)
		if runCondition != nil && runCondition.Status == untypedcorev1.ConditionFalse {
			// If a run has failed, check its message (or reason if the message is empty) for one of the known rerun cases
			failDescription := runCondition.Message
			if failDescription == "" {
				failDescription = runCondition.Reason
			}
			if pipelineRunShouldRetry(failDescription) {
				if logger != nil {
					logger.Infof("PipelineRun %s failed with '%s', rerunning", run.Name, failDescription)
				}
				// Patch the labels on the existing run to remove all existing ones and replace with the label indicating we're rerunning it.
				updatedFailedRun := run.DeepCopy()
				updatedFailedRun.Labels = map[string]string{
					labelFailedAndRerun: "true",
				}
				if err := patchPipelineRun(tektonClient, ns, updatedFailedRun, logger); err != nil {
					return errors.Wrapf(err, "removing existing labels from failed PipelineRun %s", run.Name)
				}

				// Launch a new otherwise identical PipelineRun to replace the failing one.
				newRun := createReplacementPipelineRun(&run)
				_, err := tektonClient.TektonV1alpha1().PipelineRuns(ns).Create(newRun)
				if err != nil {
					return errors.Wrapf(err, "creating new PipelineRun %s to replace failed PipelineRun %s", newRun.Name, run.Name)
				}
				if logger != nil {
					logger.Infof("Replacement PipelineRun %s created", newRun.Name)
				}
			}
		}
	}

	return nil
}

func createReplacementPipelineRun(originalRun *pipelinev1alpha1.PipelineRun) *pipelinev1alpha1.PipelineRun {
	// Fall back on appending a random string to the original name, but preferably use the name of run's Pipeline with an appended random string
	newRunName := originalRun.Name + "-" + rand.String(5)
	if originalRun.Spec.PipelineRef.Name != "" {
		newRunName = originalRun.Spec.PipelineRef.Name + "-" + rand.String(5)
	}

	newRunLabels := make(map[string]string)
	for k, v := range originalRun.Labels {
		newRunLabels[k] = v
	}

	return &pipelinev1alpha1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			APIVersion: tektonAPIVersion,
			Kind:       "PipelineRun",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      newRunName,
			Labels:    newRunLabels,
			Namespace: originalRun.Namespace,
		},
		Spec: originalRun.Spec,
	}
}

func pipelineRunShouldRetry(msg string) bool {
	for _, retryMsg := range pipelineRunShouldRetryMessages {
		if strings.Contains(msg, retryMsg) {
			return true
		}
	}
	return false
}

func patchPipelineRun(tektonClient tektonclient.Interface, namespace string, newPr *pipelinev1alpha1.PipelineRun, logger *logrus.Entry) error {
	pr, err := tektonClient.TektonV1alpha1().PipelineRuns(namespace).Get(newPr.GetName(), metav1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "getting PipelineRun/%s", newPr.GetName())
	}
	// Skip updating the resource version to avoid conflicts
	newPr.ObjectMeta.ResourceVersion = pr.ObjectMeta.ResourceVersion
	newprData, err := json.Marshal(newPr)
	if err != nil {
		return errors.Wrapf(err, "marshaling the new PipelineRun/%s", newPr.GetName())
	}
	prData, err := json.Marshal(pr)
	if err != nil {
		return errors.Wrapf(err, "marshaling the PipelineRun/%s", pr.GetName())
	}
	patch, err := jsonpatch.CreateMergePatch(prData, newprData)
	if err != nil {
		return errors.Wrapf(err, "creating merge patch for PipelineRun/%s", pr.GetName())
	}
	if len(patch) == 0 {
		return nil
	}
	if logger != nil {
		logger.Infof("Created merge patch: %v", string(patch))
	}
	patched, err := tektonClient.TektonV1alpha1().PipelineRuns(namespace).Patch(pr.Name, types.MergePatchType, patch)
	if err != nil {
		return errors.Wrapf(err, "applying merge patch for PipelineRun/%s", pr.Name)
	}
	if !reflect.DeepEqual(patched.ObjectMeta.Labels, newPr.ObjectMeta.Labels) || !reflect.DeepEqual(patched.ObjectMeta.Annotations, newPr.ObjectMeta.Annotations) {
		patched.ObjectMeta.Labels = newPr.ObjectMeta.Labels
		patched.ObjectMeta.Annotations = newPr.ObjectMeta.Annotations
		_, err := tektonClient.TektonV1alpha1().PipelineRuns(namespace).Update(patched)
		if err != nil {
			return errors.Wrapf(err, "removing labels and annotations from PipelineRun/%s", pr.Name)
		}
	}
	return nil
}
