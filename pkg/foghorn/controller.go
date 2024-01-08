package foghorn

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider/reporter"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	controllerName = "foghorn"

	retryCount = 5
)

// LighthouseJobReconciler listens for changes to LighthouseJobs and updates the corresponding LighthouseJob status and provider commit statuses.
type LighthouseJobReconciler struct {
	// ConfigMapWatcher watches for changes in our relevant config maps and updates the reconciler's versions when required.
	ConfigMapWatcher *watcher.ConfigMapWatcher

	client client.Client
	logger *logrus.Entry
	scheme *runtime.Scheme

	jobConfig    *config.Agent
	pluginConfig *plugins.ConfigAgent

	wg *sync.WaitGroup
	ns string
}

// NewLighthouseJobReconciler returns a new controller for syncing LighthouseJobs and commit statuses
func NewLighthouseJobReconciler(client client.Client, scheme *runtime.Scheme, ns string) (*LighthouseJobReconciler, error) {
	return NewLighthouseJobReconcilerWithConfig(client, scheme, ns, nil, nil, nil)
}

// NewLighthouseJobReconcilerWithConfig takes returns a new controller for syncing LighthouseJobs and commit statuses using the provided config map watcher and configs
func NewLighthouseJobReconcilerWithConfig(client client.Client, scheme *runtime.Scheme, ns string, configMapWatcher *watcher.ConfigMapWatcher, jobConfig *config.Agent, pluginConfig *plugins.ConfigAgent) (*LighthouseJobReconciler, error) {
	logger := logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName)

	if jobConfig == nil {
		jobConfig = &config.Agent{}
	}
	if pluginConfig == nil {
		pluginConfig = &plugins.ConfigAgent{}
	}

	if configMapWatcher == nil {
		var err error
		configMapWatcher, err = watcher.SetupConfigMapWatchers(ns, jobConfig, pluginConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create ConfigMap watcher")
		}
	}

	return &LighthouseJobReconciler{
		client:           client,
		scheme:           scheme,
		logger:           logger,
		ns:               ns,
		jobConfig:        jobConfig,
		pluginConfig:     pluginConfig,
		ConfigMapWatcher: configMapWatcher,
		wg:               &sync.WaitGroup{},
	}, nil
}

// SetupWithManager sets up the reconciler with its manager
func (r *LighthouseJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&lighthousev1alpha1.LighthouseJob{}).
		WithEventFilter(predicate.ResourceVersionChangedPredicate{}).
		Complete(r)
}

// Reconcile represents an iteration of the reconciliation loop
func (r *LighthouseJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	r.logger.Infof("Reconcile LighthouseJob %+v", req)

	// get lighthouse job
	var job lighthousev1alpha1.LighthouseJob
	if err := r.client.Get(ctx, req.NamespacedName, &job); err != nil {
		r.logger.Warningf("Unable to get LighthouseJob: %s", err)
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	activityRecord := job.Status.Activity

	if activityRecord == nil {
		// There's no activity on the job, so there's nothing for us to do.
		return ctrl.Result{}, nil
	}

	// Update the job's status for the activity.
	jobCopy := job.DeepCopy()
	r.updateJobStatusForActivity(activityRecord, jobCopy)
	r.reportStatus(activityRecord, jobCopy)

	if !reflect.DeepEqual(job.Status, jobCopy.Status) {
		f := func(job *lighthousev1alpha1.LighthouseJob) error {
			job.Status = jobCopy.Status
			if err := r.client.Status().Update(ctx, job); err != nil {
				r.logger.Errorf("Failed to update LighthouseJob status: %s", err)
				return err
			}
			return nil
		}
		if err := r.retryModifyJob(ctx, req.NamespacedName, &job, f); err != nil {
			r.logger.Errorf("Failed to update LighthouseJob status: %s", err)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// retryModifyJob tries to modify the Job retrying if it fails
func (r *LighthouseJobReconciler) retryModifyJob(ctx context.Context, ns client.ObjectKey, job *lighthousev1alpha1.LighthouseJob, f func(job *lighthousev1alpha1.LighthouseJob) error) error {
	i := 0
	for {
		i++
		err := f(job)
		if err == nil {
			if i > 1 {
				r.logger.Infof("took %d attempts to update Job %s", i, job.Name)
			}
			return nil
		}
		if i >= retryCount {
			return errors.Wrapf(err, "failed to update Job %s after %d attempts", job.Name, retryCount)
		}

		if err := r.client.Get(ctx, ns, job); err != nil {
			r.logger.Warningf("Unable to get LighthouseJob %s due to: %s", job.Name, err)
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return client.IgnoreNotFound(err)
		}
	}
}

func (r *LighthouseJobReconciler) updateJobStatusForActivity(activity *lighthousev1alpha1.ActivityRecord, job *lighthousev1alpha1.LighthouseJob) {
	if activity.Status != job.Status.State {
		job.Status.State = activity.Status
	}
	if activity.LastCommitSHA != job.Status.LastCommitSHA {
		job.Status.LastCommitSHA = activity.LastCommitSHA
	}
	if activity.CompletionTime != nil && activity.CompletionTime != job.Status.CompletionTime {
		job.Status.CompletionTime = activity.CompletionTime
	}
}

func (r *LighthouseJobReconciler) reportStatus(activity *lighthousev1alpha1.ActivityRecord, j *lighthousev1alpha1.LighthouseJob) {
	sha := activity.LastCommitSHA

	owner := activity.Owner
	repo := activity.Repo
	gitURL := activity.GitURL
	activityStatus := activity.Status
	statusInfo := toScmStatusDescriptionRunningStages(activity, util.GitKind(r.jobConfig.Config))

	fields := map[string]interface{}{
		"name":        activity.Name,
		"status":      activityStatus,
		"gitOwner":    owner,
		"gitRepo":     repo,
		"gitSHA":      sha,
		"gitURL":      gitURL,
		"gitBranch":   activity.Branch,
		"gitStatus":   statusInfo.scmStatus.String(),
		"buildNumber": activity.BuildIdentifier,
		"duration":    durationString(activity.StartTime, activity.CompletionTime),
	}
	if gitURL == "" {
		r.logger.WithFields(fields).Debugf("Cannot report pipeline %s as we have no git SHA", activity.Name)
		return

	}
	if sha == "" {
		r.logger.WithFields(fields).Debugf("Cannot report pipeline %s as we have no git SHA", activity.Name)
		return
	}
	if owner == "" {
		r.logger.WithFields(fields).Debugf("Cannot report pipeline %s as we have no git Owner", activity.Name)
		return
	}
	if repo == "" {
		r.logger.WithFields(fields).Debugf("Cannot report pipeline %s as we have no git repository name", activity.Name)
		return
	}

	if statusInfo.scmStatus == scm.StateUnknown {
		return
	}

	switch scm.ToState(j.Status.LastReportState) {
	// already completed - avoid reporting again if a promotion happens after a PR has merged and the pipeline updates status
	case scm.StateFailure, scm.StateError, scm.StateSuccess, scm.StateCanceled:
		return
	}

	r.logger.WithFields(fields).Warnf("last report: %s, current: %s, last desc: %s, current: %s", j.Status.LastReportState, statusInfo.scmStatus.String(),
		j.Status.Description, statusInfo.description)

	// Check if state and running stages haven't changed and return if they haven't
	if scm.ToState(j.Status.LastReportState) == statusInfo.scmStatus &&
		j.Status.Description == statusInfo.description {
		return
	}

	// Trigger external plugins if appropriate
	if external := util.ExternalPluginsForEvent(r.pluginConfig, util.LighthousePayloadTypeActivity, fmt.Sprintf("%s/%s", owner, repo), nil); len(external) > 0 {
		go util.CallExternalPluginsWithActivityRecord(r.logger, external, activity, util.HMACToken(), r.wg)
	}

	pipelineContext := activity.Context
	if pipelineContext == "" {
		pipelineContext = "jenkins-x"
	}

	gitRepoStatus := &scm.StatusInput{
		State:  statusInfo.scmStatus,
		Label:  pipelineContext,
		Desc:   statusInfo.description,
		Target: j.Status.ReportURL,
	}
	scmClient, _, _, _, err := util.GetSCMClient(owner, r.jobConfig.Config)
	if err != nil {
		r.logger.WithFields(fields).WithError(err).Warnf("failed to create SCM client")
		return
	}

	_, err = scmClient.CreateStatus(owner, repo, sha, gitRepoStatus)
	if err != nil {
		r.logger.WithFields(fields).WithError(err).Warnf("failed to report git status with target URL '%s'", gitRepoStatus.Target)
		// TODO: Need something here to prevent infinite attempts to create status from just bombing us. (apb)
		return
	}

	err = reporter.Report(scmClient, r.jobConfig.Config().Plank.ReportTemplate, j, []job.PipelineKind{job.PresubmitJob})
	if err != nil {
		// For now, we're just going to ignore failures here.
		r.logger.WithFields(fields).WithError(err).Warnf("failed to update comments on the PR")
	}
	r.logger.WithFields(fields).Info("reported git status")
	j.Status.Description = statusInfo.description
	j.Status.LastReportState = statusInfo.scmStatus.String()
}

type reportStatusInfo struct {
	scmStatus     scm.State
	description   string
	runningStages string
}

func toScmStatusDescriptionRunningStages(activity *lighthousev1alpha1.ActivityRecord, gitKind string) reportStatusInfo {
	info := reportStatusInfo{
		description:   "",
		runningStages: "",
		scmStatus:     scm.StateUnknown,
	}
	switch activity.Status {
	case lighthousev1alpha1.SuccessState:
		info.scmStatus = scm.StateSuccess
		info.description = "Pipeline successful"
	case lighthousev1alpha1.RunningState, lighthousev1alpha1.PendingState:
		info.scmStatus = scm.StateRunning
		info.description = "Pipeline running"
	case lighthousev1alpha1.AbortedState:
		info.scmStatus = scm.StateError
		info.description = "Error executing pipeline"
	case lighthousev1alpha1.FailureState:
		info.scmStatus = scm.StateFailure
		info.description = "Pipeline failed"
	default:
		info.scmStatus = scm.StateUnknown
		info.description = "Pipeline in unknown state"
	}

	runningStages := activity.RunningStages()
	// GitLab does not currently support updating description without changing state, so we need simple descriptions there.
	// TODO: link to GitLab issue (apb)
	if len(runningStages) > 0 && gitKind != "gitlab" {
		info.runningStages = strings.Join(runningStages, ",")
		info.description = fmt.Sprintf("Pipeline running stage(s): %s", strings.Join(runningStages, ", "))
		if len(info.description) > 63 {
			info.description = info.description[:59] + "..."
		}
	}
	return info
}

// durationString returns the duration between start and end time as string
func durationString(start *metav1.Time, end *metav1.Time) string {
	if start == nil || end == nil {
		return ""
	}
	return end.Sub(start.Time).Round(time.Second).String()
}
