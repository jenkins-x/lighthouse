package status

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	jxv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	jxinformers "github.com/jenkins-x/jx/pkg/client/informers/externalversions/jenkins.io/v1"
	jxlisters "github.com/jenkins-x/jx/pkg/client/listers/jenkins.io/v1"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	lhinformers "github.com/jenkins-x/lighthouse/pkg/client/informers/externalversions/lighthouse/v1alpha1"
	lhlisters "github.com/jenkins-x/lighthouse/pkg/client/listers/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/prow/pjutil"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/yaml"
)

const (
	controllerName           = "lighthouse-status"
	defaultTargetURLTemplate = "{{ .BaseURL }}/teams/{{ .Namespace }}/projects/{{ .Owner }}/{{ .Repository }}/{{ .Branch }}/{{ .Build }}"
)

// Controller listens for changes to PipelineActivitys and updates the corresponding LighthouseJobs and provider commit statuses.
type Controller struct {
	jxClient jxclient.Interface
	lhClient clientset.Interface

	activityLister jxlisters.PipelineActivityLister
	activitySynced cache.InformerSynced

	lhLister lhlisters.LighthouseJobLister
	lhSynced cache.InformerSynced
	// queue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	queue workqueue.RateLimitingInterface

	logger *logrus.Entry
	ns     string
}

// NewController returns a new controller for syncing PipelineActivity updates to LighthouseJobs and commit statuses
func NewController(jxClient jxclient.Interface, lhClient clientset.Interface, activityInformer jxinformers.PipelineActivityInformer,
	lhInformer lhinformers.LighthouseJobInformer, ns string, logger *logrus.Entry) *Controller {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName)
	}

	controller := &Controller{
		jxClient:       jxClient,
		lhClient:       lhClient,
		activityLister: activityInformer.Lister(),
		activitySynced: activityInformer.Informer().HasSynced,
		lhLister:       lhInformer.Lister(),
		lhSynced:       lhInformer.Informer().HasSynced,
		logger:         logger,
		ns:             ns,
		queue:          RateLimiter(),
	}

	activityInformer.Informer()
	logger.Info("Setting up event handlers")
	activityInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newAct := newObj.(*jxv1.PipelineActivity)
			oldAct := oldObj.(*jxv1.PipelineActivity)
			// Skip updates solely triggered by resyncs. We only care if they're actually different.
			if oldAct.ResourceVersion == newAct.ResourceVersion {
				return
			}
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				// TODO: REMOVE
				logger.Warnf("UPDATE FUNC FOR %s", key)
				controller.queue.AddRateLimited(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				// TODO: REMOVE
				logger.Warnf("DELETE FUNC FOR %s", key)
				controller.queue.AddRateLimited(key)
			}
		},
	})

	return controller
}

// Run actually runs the controller
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	c.logger.Info("Starting controller")

	// Wait for the caches to be synced before starting workers
	c.logger.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.activitySynced, c.lhSynced); !ok {
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

	// Get the PipelineActivity resource with this namespace/name
	activity, err := c.activityLister.PipelineActivities(namespace).Get(name)
	if err != nil {
		// The PipelineActivity resource may no longer exist, in which case we delete the associated LH job
		// TODO: Actually delete.
		if errors.IsNotFound(err) {
			c.logger.Warnf("activity '%s' in work queue no longer exists", key)
			return nil
		}

		return err
	}

	var job *v1alpha1.LighthouseJob

	// Get all LighthouseJobs with the same owner/repo/branch/build/context
	labelSelector, err := createLabelSelectorFromActivity(activity)
	possibleJobs, err := c.lhLister.LighthouseJobs(namespace).List(labelSelector)
	if err != nil {
		return err
	}
	if len(possibleJobs) == 0 {
		// Let's try to guess and create a LH job if this is a postsubmit, since that could have been created via
		// jx start pipeline. We're going to assume that we should only do this if it's on master.
		// TODO: this probably is not good per se and needs work or removing (apb)
		if activity.Spec.GitBranch == "master" {
			convertedJob := launcher.ToLighthouseJob(activity)
			newLabels, newAnnotations := pjutil.LabelsAndAnnotationsForJob(convertedJob)
			newJob := pjutil.NewLighthouseJob(convertedJob.Spec, newLabels, newAnnotations)
			newJob.Status = v1alpha1.LighthouseJobStatus{
				State:           v1alpha1.TriggeredState,
				ActivityName:    util.ToValidName(activity.Name),
				StartTime:       *activity.Spec.StartedTimestamp,
				LastReportState: string(scm.StateUnknown),
			}
			createdJob, err := c.lhClient.LighthouseV1alpha1().LighthouseJobs(namespace).Create(&newJob)
			if err != nil {
				c.logger.WithError(err).Warnf("could not create LighthouseJob for postsubmit PipelineActivity %s", activity.Name)
				return nil
			}
			possibleJobs = append(possibleJobs, createdJob)
		} else {
			c.logger.Warnf("no LighthouseJobs found matching label selector %s", labelSelector.String())
			return nil
		}
	}

	// To be safe, find the job with the activity's name in its status.
	for _, j := range possibleJobs {
		if j.Status.ActivityName == activity.Name {
			job = j
		}
	}

	if job == nil {
		return nil
	}

	// Update the job's status for the activity.
	jobCopy := job.DeepCopy()
	c.updateJobStatusForActivity(activity, jobCopy)
	c.reportStatus(namespace, activity, jobCopy)

	currentJob, err := c.lhLister.LighthouseJobs(namespace).Get(jobCopy.Name)
	if err != nil {
		c.logger.WithError(err).Errorf("couldn't get the orig of job %s", jobCopy.Name)
		return err
	}
	if !reflect.DeepEqual(currentJob.Status, jobCopy.Status) {
		currentJob.Status = jobCopy.Status
		_, err = c.lhClient.LighthouseV1alpha1().LighthouseJobs(namespace).UpdateStatus(currentJob)
		if err != nil {
			c.logger.WithError(err).Errorf("error updating status for job %s", currentJob.Name)
			return err
		}
	}
	return nil
}

func (c *Controller) updateJobStatusForActivity(activity *jxv1.PipelineActivity, job *v1alpha1.LighthouseJob) {
	activityState := v1alpha1.ToPipelineState(activity.Spec.Status)
	if activityState != job.Status.State {
		job.Status.State = activityState
	}
	if activity.Spec.LastCommitSHA != job.Status.LastCommitSHA {
		job.Status.LastCommitSHA = activity.Spec.LastCommitSHA
	}
	if activity.Spec.CompletedTimestamp != nil && activity.Spec.CompletedTimestamp != job.Status.CompletionTime {
		job.Status.CompletionTime = activity.Spec.CompletedTimestamp
	}
}

// RateLimiter creates a ratelimiting queue for the status controller.
func RateLimiter() workqueue.RateLimitingInterface {
	rl := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 120*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(1000), 50000)},
	)
	return workqueue.NewNamedRateLimitingQueue(rl, controllerName)
}

func createLabelSelectorFromActivity(activity *jxv1.PipelineActivity) (labels.Selector, error) {
	var selectors []string

	if owner, ok := activity.Labels[util.ActivityOwnerLabel]; ok {
		selectors = append(selectors, fmt.Sprintf("%s=%s", util.OrgLabel, owner))
	}
	if repo, ok := activity.Labels[util.ActivityRepositoryLabel]; ok {
		selectors = append(selectors, fmt.Sprintf("%s=%s", util.RepoLabel, repo))
	}
	if branch, ok := activity.Labels[util.ActivityBranchLabel]; ok {
		selectors = append(selectors, fmt.Sprintf("%s=%s", util.BranchLabel, branch))
	}
	if buildNum, ok := activity.Labels[util.ActivityBuildLabel]; ok {
		selectors = append(selectors, fmt.Sprintf("%s=%s", util.BuildNumLabel, buildNum))
	}
	if context, ok := activity.Labels[util.ActivityContextLabel]; ok {
		selectors = append(selectors, fmt.Sprintf("%s=%s", util.ContextLabel, context))
	}

	return labels.Parse(strings.Join(selectors, ","))
}

func (c *Controller) reportStatus(ns string, activity *jxv1.PipelineActivity, job *v1alpha1.LighthouseJob) {
	sha := activity.Spec.LastCommitSHA
	if sha == "" && activity.Labels != nil {
		sha = activity.Labels[jxv1.LabelLastCommitSha]
	}

	owner := activity.Spec.GitOwner
	repo := activity.Spec.GitRepository
	gitURL := activity.Spec.GitURL
	activityStatus := activity.Spec.Status
	statusInfo := toScmStatusDescriptionRunningStages(activity)

	fields := map[string]interface{}{
		"name":        activity.Name,
		"status":      activityStatus,
		"gitOwner":    owner,
		"gitRepo":     repo,
		"gitSHA":      sha,
		"gitURL":      gitURL,
		"gitBranch":   activity.Spec.GitBranch,
		"gitStatus":   statusInfo.scmStatus,
		"buildNumber": activity.Spec.Build,
		"duration":    durationString(activity.Spec.StartedTimestamp, activity.Spec.CompletedTimestamp),
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

	switch job.Status.LastReportState {
	// already completed - avoid reporting again if a promotion happens after a PR has merged and the pipeline updates status
	case string(jxv1.ActivityStatusTypeSucceeded), string(jxv1.ActivityStatusTypeAborted), string(jxv1.ActivityStatusTypeFailed):
		return
	}

	// Check if state and running stages haven't changed and return if they haven't
	if job.Status.LastReportState == string(activityStatus) &&
		job.Status.Description == statusInfo.description {
		return
	}

	pipelineContext := activity.Spec.Context
	if pipelineContext == "" {
		pipelineContext = "jenkins-x"
	}

	gitRepoStatus := &scm.StatusInput{
		State: statusInfo.scmStatus,
		Label: pipelineContext,
		Desc:  statusInfo.description,
	}
	urlBase := c.getReportURLBase()
	if urlBase != "" {
		targetURL := c.createReportTargetURL(defaultTargetURLTemplate, ReportParams{
			Owner:      owner,
			Repository: repo,
			Branch:     activity.Spec.GitBranch,
			Build:      activity.Spec.Build,
			Context:    pipelineContext,
			// TODO: Need to get the job URL base in here somehow. (apb)
			BaseURL:   strings.TrimRight(urlBase, "/"),
			Namespace: ns,
		})

		if strings.HasPrefix(targetURL, "http://") || strings.HasPrefix(targetURL, "https://") {
			gitRepoStatus.Target = targetURL
		}
	}
	scmClient, _, _, err := c.createSCMClient()
	if err != nil {
		c.logger.WithFields(fields).WithError(err).Warnf("failed to create SCM client")
		return
	}

	_, err = scmClient.CreateStatus(owner, repo, sha, gitRepoStatus)
	if err != nil {
		c.logger.WithFields(fields).WithError(err).Warnf("failed to report git status")
		// TODO: Need something here to prevent infinite attempts to create status from just bombing us. (apb)
		return
	}

	c.logger.WithFields(fields).Info("reported git status")
	job.Status.Description = statusInfo.description
	job.Status.LastReportState = string(statusInfo.scmStatus)
}

// getReportURLBase gets the base report URL from the environment
func (c *Controller) getReportURLBase() string {
	return os.Getenv("LIGHTHOUSE_REPORT_URL_BASE")
}

// ReportParams contains the parameters for target URL templates
type ReportParams struct {
	BaseURL, Owner, Repository, Branch, Build, Context, Namespace string
}

// createReportTargetURL creates the target URL for pipeline results/logs from a template
func (c *Controller) createReportTargetURL(templateText string, params ReportParams) string {
	templateData, err := toObjectMap(params)
	if err != nil {
		c.logger.WithError(err).Warnf("failed to convert git ReportParams to a map for %#v", params)
		return ""
	}

	tmpl, err := template.New("target_url.tmpl").Option("missingkey=error").Parse(templateText)
	if err != nil {
		c.logger.WithError(err).Warnf("failed to parse git ReportsParam template: %s", templateText)
		return ""
	}

	var buf strings.Builder
	err = tmpl.Execute(&buf, templateData)
	if err != nil {
		c.logger.WithError(err).Warnf("failed to evaluate git ReportsParam template: %s due to: %s", templateText, err.Error())
		return ""
	}
	return buf.String()
}

type reportStatusInfo struct {
	scmStatus     scm.State
	description   string
	runningStages string
}

func toScmStatusDescriptionRunningStages(activity *jxv1.PipelineActivity) reportStatusInfo {
	info := reportStatusInfo{
		description:   "",
		runningStages: "",
		scmStatus:     scm.StateUnknown,
	}
	switch activity.Spec.Status {
	case jxv1.ActivityStatusTypeSucceeded:
		info.scmStatus = scm.StateSuccess
		info.description = "Pipeline successful"
	case jxv1.ActivityStatusTypeRunning, jxv1.ActivityStatusTypePending:
		info.scmStatus = scm.StatePending
		info.description = "Pipeline running"
	case jxv1.ActivityStatusTypeError:
		info.scmStatus = scm.StateError
		info.description = "Error executing pipeline"
	default:
		info.scmStatus = scm.StateFailure
		info.description = "Pipeline failed"
	}
	stagesByStatus := activity.StagesByStatus()

	if len(stagesByStatus[jxv1.ActivityStatusTypeRunning]) > 0 {
		info.runningStages = strings.Join(stagesByStatus[jxv1.ActivityStatusTypeRunning], ",")
		info.description = fmt.Sprintf("Pipeline running stage(s): %s", strings.Join(stagesByStatus[jxv1.ActivityStatusTypeRunning], ", "))
		if len(info.description) > 63 {
			info.description = info.description[:59] + "..."
		}
	}
	return info
}

// toObjectMap converts the given object into a map of strings/maps using YAML marshalling
func toObjectMap(object interface{}) (map[string]interface{}, error) {
	answer := map[string]interface{}{}
	data, err := yaml.Marshal(object)
	if err != nil {
		return answer, err
	}
	err = yaml.Unmarshal(data, &answer)
	return answer, err
}

// durationString returns the duration between start and end time as string
func durationString(start *metav1.Time, end *metav1.Time) string {
	if start == nil || end == nil {
		return ""
	}
	return end.Sub(start.Time).Round(time.Second).String()
}

func (c *Controller) createSCMClient() (scmprovider.SCMClient, string, string, error) {
	kind := c.gitKind()
	serverURL := os.Getenv("GIT_SERVER")

	token, err := c.createSCMToken(kind)
	if err != nil {
		return nil, serverURL, token, err
	}
	client, err := factory.NewClient(kind, serverURL, token)
	scmClient := scmprovider.ToClient(client, c.GetBotName())
	return scmClient, serverURL, token, err
}

func (c *Controller) gitKind() string {
	kind := os.Getenv("GIT_KIND")
	if kind == "" {
		kind = "github"
	}
	return kind
}

// GetBotName returns the bot name
func (c *Controller) GetBotName() string {
	botName := os.Getenv("GIT_USER")
	if botName == "" {
		botName = "jenkins-x-bot"
	}
	return botName
}

func (c *Controller) createSCMToken(gitKind string) (string, error) {
	envName := "GIT_TOKEN"
	value := os.Getenv(envName)
	if value == "" {
		return value, fmt.Errorf("No token available for git kind %s at environment variable $%s", gitKind, envName)
	}
	return value, nil
}
