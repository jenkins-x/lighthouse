package tekton

import (
	"context"
	"strings"

	"knative.dev/pkg/apis"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonversioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConvertPipelineRun translates a PipelineRun into an ActivityRecord
func ConvertPipelineRun(ctx context.Context, logger *logrus.Entry, tektonclient tektonversioned.Interface, pr *pipelinev1.PipelineRun, namespace string) (*v1alpha1.ActivityRecord, error) {
	if pr == nil {
		return nil, nil
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
		var stage *v1alpha1.ActivityStageOrStep
		var err error

		switch childReference.Kind {
		case "CustomRun":
			stage, err = convertCustomRunReference(ctx, tektonclient, childReference.Name, childReference.PipelineTaskName, namespace)
		case "TaskRun", "":
			stage, err = convertTaskRunReference(ctx, tektonclient, pr.Name, childReference.Name, childReference.PipelineTaskName, namespace)
		default:
			logger.Warnf("Unknown ChildReference kind '%s' for '%s', skipping", childReference.Kind, childReference.Name)
			continue
		}

		if err != nil {
			return nil, err
		}

		if stage != nil {
			record.Stages = append(record.Stages, stage)
		}
	}
	// log URL is definitely gonna wait

	return record, nil
}

// convertTaskRunReference fetches a TaskRun and converts it to an ActivityStageOrStep with its steps.
func convertTaskRunReference(ctx context.Context, tektonclient tektonversioned.Interface, prName, taskRunName, pipelineTaskName, namespace string) (*v1alpha1.ActivityStageOrStep, error) {
	taskrun, err := tektonclient.TektonV1().TaskRuns(namespace).Get(ctx, taskRunName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	stageName := pipelineTaskName
	if stageName == "" {
		cleanedUpTaskName := taskRunName
		if len(taskRunName) > 6 {
			cleanedUpTaskName = taskRunName[:len(taskRunName)-6]
		}
		stageName = strings.TrimPrefix(cleanedUpTaskName, prName+"-")
	}

	t := &v1alpha1.ActivityStageOrStep{
		Name:           stageName,
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

	return t, nil
}

// convertCustomRunReference fetches a CustomRun and converts it to an ActivityStageOrStep without steps.
// CustomRuns represent Custom Tasks and do not create Pods, so they have no steps.
func convertCustomRunReference(ctx context.Context, tektonclient tektonversioned.Interface, customRunName, pipelineTaskName, namespace string) (*v1alpha1.ActivityStageOrStep, error) {
	customRun, err := tektonclient.TektonV1beta1().CustomRuns(namespace).Get(ctx, customRunName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	stageName := pipelineTaskName
	if stageName == "" {
		stageName = customRunName
	}

	return &v1alpha1.ActivityStageOrStep{
		Name:           stageName,
		Status:         convertTektonStatus(customRun.Status.GetCondition(apis.ConditionSucceeded), customRun.Status.StartTime, customRun.Status.CompletionTime),
		StartTime:      customRun.Status.StartTime,
		CompletionTime: customRun.Status.CompletionTime,
	}, nil
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

func timePtrEqual(a, b *metav1.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(b)
}

// TerminalActivitySyncedWithPipelineRun returns true when the LighthouseJob's Activity already matches
// the terminal PipelineRun at the level ConvertPipelineRun would set without fetching TaskRuns (top-level
// record fields, stage count, optional report URL).
func TerminalActivitySyncedWithPipelineRun(job *v1alpha1.LighthouseJob, pr *pipelinev1.PipelineRun, expectedReportURL string, dashboardConfigured bool) bool {
	if job == nil || pr == nil {
		return false
	}
	cond := pr.Status.GetCondition(apis.ConditionSucceeded)
	if cond == nil || (!cond.IsTrue() && !cond.IsFalse()) {
		return false
	}
	act := job.Status.Activity
	if act == nil {
		return false
	}
	expectedStatus := convertTektonStatus(cond, pr.Status.StartTime, pr.Status.CompletionTime)
	if act.Status != expectedStatus {
		return false
	}
	if act.Name != pr.Name {
		return false
	}
	if !timePtrEqual(act.StartTime, pr.Status.StartTime) {
		return false
	}
	if !timePtrEqual(act.CompletionTime, pr.Status.CompletionTime) {
		return false
	}
	if len(act.Stages) != len(pr.Status.ChildReferences) {
		return false
	}
	if dashboardConfigured && job.Status.ReportURL != expectedReportURL {
		return false
	}
	return true
}
