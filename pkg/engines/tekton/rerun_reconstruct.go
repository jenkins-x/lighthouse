package tekton

import (
	"fmt"
	"strconv"
	"strings"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	configjob "github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/util"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ErrMissingRequiredLabels is returned when a PipelineRun is missing labels
// required to reconstruct a LighthouseJob.
type ErrMissingRequiredLabels struct {
	Missing []string
}

func (e *ErrMissingRequiredLabels) Error() string {
	return fmt.Sprintf("missing required labels for reconstruction: %s", strings.Join(e.Missing, ", "))
}

// isPullRequestType returns true if the job type is associated with a pull request (presubmit or batch).
func isPullRequestType(jobType configjob.PipelineKind) bool {
	return jobType == configjob.PresubmitJob || jobType == configjob.BatchJob
}

// rerunSpecFromPipelineRun reconstructs a LighthouseJobSpec entirely from the metadata
// (labels, annotations, and Spec) of a rerun PipelineRun.
func rerunSpecFromPipelineRun(pr *pipelinev1.PipelineRun) (lighthousev1alpha1.LighthouseJobSpec, error) {
	labels := pr.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}

	requiredLabels := []string{
		configjob.LighthouseJobTypeLabel,
		util.ContextLabel,
		util.OrgLabel,
		util.RepoLabel,
		util.LastCommitSHALabel,
	}

	var missing []string
	for _, req := range requiredLabels {
		if _, ok := labels[req]; !ok {
			missing = append(missing, req)
		}
	}

	if len(missing) > 0 {
		return lighthousev1alpha1.LighthouseJobSpec{}, &ErrMissingRequiredLabels{Missing: missing}
	}

	annotations := pr.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	prSpecCopy := pr.Spec.DeepCopy()

	baseRef := labels[util.BranchLabel]
	if isPullRequestType(configjob.PipelineKind(labels[configjob.LighthouseJobTypeLabel])) {
		// Foghorn only needs BaseSHA and Pulls for PR reporting.
		// The PR branch name is stored in BranchLabel but BaseRef must remain empty for presubmit/batch
		// to match the original jobutil.LabelsAndAnnotationsForJob behavior.
		baseRef = ""
	}

	spec := lighthousev1alpha1.LighthouseJobSpec{
		Agent:           configjob.TektonPipelineAgent,
		Type:            configjob.PipelineKind(labels[configjob.LighthouseJobTypeLabel]),
		Namespace:       pr.Namespace,
		Context:         labels[util.ContextLabel],
		Job:             annotations[util.LighthouseJobAnnotation],
		PipelineRunSpec: prSpecCopy,
		Refs: &lighthousev1alpha1.Refs{
			Org:      labels[util.OrgLabel],
			Repo:     labels[util.RepoLabel],
			BaseSHA:  labels[util.BaseSHALabel],
			BaseRef:  baseRef,
			CloneURI: annotations[util.CloneURIAnnotation],
		},
	}

	if pullStr, ok := labels[util.PullLabel]; ok && pullStr != "" {
		if pullNumber, err := strconv.Atoi(pullStr); err == nil {
			spec.Refs.Pulls = []lighthousev1alpha1.Pull{
				{
					Number: pullNumber,
					SHA:    labels[util.LastCommitSHALabel],
				},
			}
		}
	}

	return spec, nil
}

// newReconstructedLighthouseJob creates a new LighthouseJob from a rerun PipelineRun and a reconstructed spec.
func newReconstructedLighthouseJob(pr *pipelinev1.PipelineRun, spec lighthousev1alpha1.LighthouseJobSpec) *lighthousev1alpha1.LighthouseJob {
	ljLabels := make(map[string]string)
	for k, v := range pr.GetLabels() {
		if strings.HasPrefix(k, "lighthouse.jenkins-x.io/") || k == configjob.CreatedByLighthouseLabel {
			ljLabels[k] = v
		}
	}

	return &lighthousev1alpha1.LighthouseJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "lighthouse.jenkins.io/v1alpha1",
			Kind:       "LighthouseJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pr.Name,
			Namespace: pr.Namespace,
			Labels:    ljLabels,
		},
		Spec:   spec,
		Status: lighthousev1alpha1.LighthouseJobStatus{},
	}
}
