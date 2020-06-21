package record

import (
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ActivityRecord is a struct for reporting information on a pipeline, build, or other activity triggered by Lighthouse
type ActivityRecord struct {
	Name            string                 `json:"name"`
	Owner           string                 `json:"owner,omitempty"`
	Repo            string                 `json:"repo,omitempty"`
	Branch          string                 `json:"branch,omitempty"`
	BuildIdentifier string                 `json:"buildId,omitempty"`
	Context         string                 `json:"context,omitempty"`
	GitURL          string                 `json:"gitURL,omitempty"`
	LogURL          string                 `json:"logURL,omitempty"`
	LinkURL         string                 `json:"linkURL,omitempty"`
	Status          v1alpha1.PipelineState `json:"status,omitempty"`
	BaseSHA         string                 `json:"baseSHA,omitempty"`
	LastCommitSHA   string                 `json:"lastCommitSHA,omitempty"`
	StartTime       *metav1.Time           `json:"startTime,omitempty"`
	CompletionTime  *metav1.Time           `json:"completionTime,omitempty"`
	Stages          []*ActivityStageOrStep `json:"stages,omitempty"`
	Steps           []*ActivityStageOrStep `json:"steps,omitEmpty"`
}

// ActivityStageOrStep represents a stage of an activity
type ActivityStageOrStep struct {
	Name           string                 `json:"name"`
	Status         v1alpha1.PipelineState `json:"status"`
	StartTime      *metav1.Time           `json:"startTime,omitempty"`
	CompletionTime *metav1.Time           `json:"completionTime,omitempty"`
	Stages         []*ActivityStageOrStep `json:"stages,omitempty"`
	Steps          []*ActivityStageOrStep `json:"steps,omitempty"`
}

// RunningStages returns the list of stages currently running
func (a *ActivityRecord) RunningStages() []string {
	var running []string

	for _, stage := range a.Stages {
		if stage.Status == v1alpha1.RunningState {
			running = append(running, stage.Name)
		}
	}
	return running
}
