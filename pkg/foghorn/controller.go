package foghorn

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	lhinformers "github.com/jenkins-x/lighthouse/pkg/client/informers/externalversions/lighthouse/v1alpha1"
	lhlisters "github.com/jenkins-x/lighthouse/pkg/client/listers/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider/reporter"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	controllerName = "foghorn"
)

// Controller listens for changes to PipelineActivitys and updates the corresponding LighthouseJobs and provider commit statuses.
type Controller struct {
	lhClient   clientset.Interface
	kubeClient kubernetes.Interface

	lhLister lhlisters.LighthouseJobLister
	lhSynced cache.InformerSynced
	// queue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	queue workqueue.RateLimitingInterface

	configMapWatcher *watcher.ConfigMapWatcher

	jobConfig    *config.Agent
	pluginConfig *plugins.ConfigAgent

	wg     *sync.WaitGroup
	logger *logrus.Entry
	ns     string
}

// NewController returns a new controller for syncing LighthouseJobs and commit statuses
func NewController(kubeClient kubernetes.Interface, lhClient clientset.Interface, lhInformer lhinformers.LighthouseJobInformer, ns string, logger *logrus.Entry) (*Controller, error) {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName)
	}

	configAgent := &config.Agent{}
	pluginAgent := &plugins.ConfigAgent{}

	configMapWatcher, err := watcher.SetupConfigMapWatchers(ns, configAgent, pluginAgent)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ConfigMap watcher")
	}

	controller := &Controller{
		lhClient:         lhClient,
		lhLister:         lhInformer.Lister(),
		lhSynced:         lhInformer.Informer().HasSynced,
		logger:           logger,
		ns:               ns,
		queue:            RateLimiter(),
		jobConfig:        configAgent,
		pluginConfig:     pluginAgent,
		configMapWatcher: configMapWatcher,
		kubeClient:       kubeClient,
	}

	logger.Info("Setting up event handlers")
	lhInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newAct := newObj.(*v1alpha1.LighthouseJob)
			oldAct := oldObj.(*v1alpha1.LighthouseJob)
			// Skip updates solely triggered by resyncs. We only care if they're actually different.
			if oldAct.ResourceVersion == newAct.ResourceVersion {
				return
			}
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				controller.queue.AddRateLimited(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				controller.queue.AddRateLimited(key)
			}
		},
	})

	controller.wg = &sync.WaitGroup{}

	return controller, nil
}

// Run actually runs the controller
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	c.logger.Info("Starting controller")

	defer c.configMapWatcher.Stop()

	// Wait for the caches to be synced before starting workers
	c.logger.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.lhSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	c.logger.Info("Starting workers")
	// Launch the appropriate number of workers to process PipelineActivity resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	c.logger.Info("Started workers")
	<-stopCh
	c.logger.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.queue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.queue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.queue.Forget(obj)
			c.logger.Warnf("expected string in workqueue but got %#v", obj)
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// PipelineActivity resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.queue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.queue.Forget(obj)
		c.logger.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		c.logger.WithError(err).Error("failure reconciling")
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		c.logger.Warnf("invalid resource key: %s", key)
		return nil
	}

	job, err := c.lhLister.LighthouseJobs(namespace).Get(name)
	if err != nil {
		// The LighthouseJob resource may no longer exist
		if kubeerrors.IsNotFound(err) {
			c.logger.Warnf("activity '%s' in work queue no longer exists", key)
			return nil
		}

		// Return an error here so that we requeue and retry.
		return err
	}

	if job == nil {
		return nil
	}

	activityRecord := job.Status.Activity

	if activityRecord == nil {
		// There's no activity on the job, so there's nothing for us to do.
		return nil
	}

	// Update the job's status for the activity.
	jobCopy := job.DeepCopy()
	c.updateJobStatusForActivity(activityRecord, jobCopy)
	c.reportStatus(namespace, activityRecord, jobCopy)

	currentJob, err := c.lhLister.LighthouseJobs(namespace).Get(jobCopy.Name)
	if err != nil {
		c.logger.WithError(err).Errorf("couldn't get the orig of job %s", jobCopy.Name)
		// Return an error here so we requeue and retry.
		return err
	}
	if !reflect.DeepEqual(currentJob.Status, jobCopy.Status) {
		currentJob.Status = jobCopy.Status
		_, err = c.lhClient.LighthouseV1alpha1().LighthouseJobs(namespace).UpdateStatus(currentJob)
		if err != nil {
			c.logger.WithError(err).Errorf("error updating status for job %s", currentJob.Name)
			// Return an error here so we requeue and retry.
			return err
		}
	}
	return nil
}

func (c *Controller) updateJobStatusForActivity(activity *v1alpha1.ActivityRecord, job *v1alpha1.LighthouseJob) {
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

// RateLimiter creates a ratelimiting queue for the foghorn controller.
func RateLimiter() workqueue.RateLimitingInterface {
	rl := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 120*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(1000), 50000)},
	)
	return workqueue.NewNamedRateLimitingQueue(rl, controllerName)
}

func (c *Controller) reportStatus(ns string, activity *v1alpha1.ActivityRecord, job *v1alpha1.LighthouseJob) {
	sha := activity.LastCommitSHA

	owner := activity.Owner
	repo := activity.Repo
	gitURL := activity.GitURL
	activityStatus := activity.Status
	statusInfo := toScmStatusDescriptionRunningStages(activity, util.GitKind(c.jobConfig.Config))

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
		c.logger.WithFields(fields).Debugf("Cannot report pipeline %s as we have no git SHA", activity.Name)
		return

	}
	if sha == "" {
		c.logger.WithFields(fields).Debugf("Cannot report pipeline %s as we have no git SHA", activity.Name)
		return
	}
	if owner == "" {
		c.logger.WithFields(fields).Debugf("Cannot report pipeline %s as we have no git Owner", activity.Name)
		return
	}
	if repo == "" {
		c.logger.WithFields(fields).Debugf("Cannot report pipeline %s as we have no git repository name", activity.Name)
		return
	}

	if statusInfo.scmStatus == scm.StateUnknown {
		return
	}

	switch scm.ToState(job.Status.LastReportState) {
	// already completed - avoid reporting again if a promotion happens after a PR has merged and the pipeline updates status
	case scm.StateFailure, scm.StateError, scm.StateSuccess, scm.StateCanceled:
		return
	}

	c.logger.WithFields(fields).Warnf("last report: %s, current: %s, last desc: %s, current: %s", job.Status.LastReportState, statusInfo.scmStatus.String(),
		job.Status.Description, statusInfo.description)

	// Check if state and running stages haven't changed and return if they haven't
	if scm.ToState(job.Status.LastReportState) == statusInfo.scmStatus &&
		job.Status.Description == statusInfo.description {
		return
	}

	// Trigger external plugins if appropriate
	if external := util.ExternalPluginsForEvent(c.pluginConfig, util.LighthousePayloadTypeActivity, fmt.Sprintf("%s/%s", owner, repo)); len(external) > 0 {
		go util.CallExternalPluginsWithActivityRecord(c.logger, external, activity, util.HMACToken(), c.wg)
	}

	pipelineContext := activity.Context
	if pipelineContext == "" {
		pipelineContext = "jenkins-x"
	}

	gitRepoStatus := &scm.StatusInput{
		State:  statusInfo.scmStatus,
		Label:  pipelineContext,
		Desc:   statusInfo.description,
		Target: job.Status.ReportURL,
	}
	scmClient, _, _, _, err := util.GetSCMClient(owner, c.jobConfig.Config)
	if err != nil {
		c.logger.WithFields(fields).WithError(err).Warnf("failed to create SCM client")
		return
	}

	_, err = scmClient.CreateStatus(owner, repo, sha, gitRepoStatus)
	if err != nil {
		c.logger.WithFields(fields).WithError(err).Warnf("failed to report git status with target URL '%s'", gitRepoStatus.Target)
		// TODO: Need something here to prevent infinite attempts to create status from just bombing us. (apb)
		return
	}

	err = reporter.Report(scmClient, c.jobConfig.Config().Plank.ReportTemplate, job, []config.PipelineKind{config.PresubmitJob})
	if err != nil {
		// For now, we're just going to ignore failures here.
		c.logger.WithFields(fields).WithError(err).Warnf("failed to update comments on the PR")
	}
	c.logger.WithFields(fields).Info("reported git status")
	job.Status.Description = statusInfo.description
	job.Status.LastReportState = statusInfo.scmStatus.String()
}

type reportStatusInfo struct {
	scmStatus     scm.State
	description   string
	runningStages string
}

func toScmStatusDescriptionRunningStages(activity *v1alpha1.ActivityRecord, gitKind string) reportStatusInfo {
	info := reportStatusInfo{
		description:   "",
		runningStages: "",
		scmStatus:     scm.StateUnknown,
	}
	switch activity.Status {
	case v1alpha1.SuccessState:
		info.scmStatus = scm.StateSuccess
		info.description = "Pipeline successful"
	case v1alpha1.RunningState, v1alpha1.PendingState:
		info.scmStatus = scm.StateRunning
		info.description = "Pipeline running"
	case v1alpha1.AbortedState:
		info.scmStatus = scm.StateError
		info.description = "Error executing pipeline"
	case v1alpha1.FailureState:
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
