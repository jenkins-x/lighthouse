package tekton

import (
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
)

// ConvertPipelineRun translates a PipelineRun into an ActivityRecord
func ConvertPipelineRun(pr *v1beta1.PipelineRun) *v1alpha1.ActivityRecord {
	if pr == nil {
		return nil
	}

	record := new(v1alpha1.ActivityRecord)

	record.Name = pr.Name

	record.BaseSHA = pr.Labels[util.BaseSHALabel]
	record.Repo = pr.Labels[util.RepoLabel]
	record.Context = pr.Labels[util.ContextLabel]
	record.Owner = pr.Labels[util.OrgLabel]
	record.Branch = pr.Labels[util.BranchLabel]
	record.BuildIdentifier = pr.Labels[util.BuildNumLabel]
	record.LastCommitSHA = pr.Labels[util.LastCommitSHALabel]

	record.GitURL = pr.Annotations[util.CloneURIAnnotation]
	record.StartTime = pr.Status.StartTime
	record.CompletionTime = pr.Status.CompletionTime

	cond := pr.Status.GetCondition(apis.ConditionSucceeded)

	record.Status = convertTektonStatus(cond, record.StartTime, record.CompletionTime)

	for taskName, task := range pr.Status.TaskRuns {
		t := &v1alpha1.ActivityStageOrStep{
			Name:           taskName,
			Status:         convertTektonStatus(task.Status.GetCondition(apis.ConditionSucceeded), task.Status.StartTime, task.Status.CompletionTime),
			StartTime:      task.Status.StartTime,
			CompletionTime: task.Status.CompletionTime,
		}

		for _, step := range task.Status.Steps {
			s := &v1alpha1.ActivityStageOrStep{
				Name: step.Name,
			}
			switch {
			case step.Terminated != nil:
				if step.Terminated.ExitCode != 0 {
					s.Status = v1alpha1.FailureState
				} else {
					s.Status = v1alpha1.SuccessState
				}
				s.StartTime = step.Terminated.StartedAt.DeepCopy()
				s.CompletionTime = step.Terminated.FinishedAt.DeepCopy()
			case step.Running != nil:
				s.Status = v1alpha1.RunningState
				s.StartTime = step.Running.StartedAt.DeepCopy()
			case step.Waiting != nil:
				s.Status = v1alpha1.PendingState
			default:
				s.Status = v1alpha1.TriggeredState
			}

			t.Steps = append(t.Steps, s)
		}

		record.Stages = append(record.Stages, t)
	}
	// log URL is definitely gonna wait

	return record
}

func convertTektonStatus(cond *apis.Condition, start, finished *metav1.Time) v1alpha1.PipelineState {
	switch {
	case cond.Status == corev1.ConditionTrue:
		return v1alpha1.SuccessState
	case cond.Status == corev1.ConditionFalse:
		return v1alpha1.FailureState
	case start.IsZero():
		return v1alpha1.TriggeredState
	case cond.Status == corev1.ConditionUnknown, finished.IsZero():
		return v1alpha1.RunningState
	default:
		return v1alpha1.PendingState
	}
}
