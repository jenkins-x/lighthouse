package tekton

import (
	"context"
	"strings"

	"knative.dev/pkg/apis"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/util"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConvertPipelineRun translates a PipelineRun into an ActivityRecord
func ConvertPipelineRun(pr *pipelinev1.PipelineRun) (*v1alpha1.ActivityRecord, error) {
	if pr == nil {
		return nil, nil
	}

	tektonclient, _, _, _, err := clients.GetAPIClients()
	if err != nil {
		return nil, err
	}

	record := new(v1alpha1.ActivityRecord)

	record.Name = pr.Name

	record.JobID = pr.Labels[job.LighthouseJobIDLabel]
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

	for _, childReference := range pr.Status.ChildReferences {
		taskrun, err := tektonclient.TektonV1().TaskRuns("jx").Get(context.TODO(), childReference.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		cleanedUpTaskName := strings.TrimPrefix(taskrun.Name[:len(taskrun.Name)-6], pr.Name+"-")
		t := &v1alpha1.ActivityStageOrStep{
			Name:           cleanedUpTaskName,
			Status:         convertTektonStatus(taskrun.Status.GetCondition(apis.ConditionSucceeded), taskrun.Status.StartTime, taskrun.Status.CompletionTime),
			StartTime:      taskrun.Status.StartTime,
			CompletionTime: taskrun.Status.CompletionTime,
		}

		for _, step := range taskrun.Status.Steps {
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

	return record, nil
}

func convertTektonStatus(cond *apis.Condition, start, finished *metav1.Time) v1alpha1.PipelineState {
	if cond == nil {
		return v1alpha1.PendingState
	}
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
