package jx

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	jxv1 "github.com/jenkins-x/jx-api/pkg/apis/jenkins.io/v1"
	jxclient "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	jxinformers "github.com/jenkins-x/jx-api/pkg/client/informers/externalversions/jenkins.io/v1"
	jxlisters "github.com/jenkins-x/jx-api/pkg/client/listers/jenkins.io/v1"
	"github.com/jenkins-x/jx/v2/pkg/tekton/metapipeline"
	"github.com/jenkins-x/lighthouse-config/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	lhinformers "github.com/jenkins-x/lighthouse/pkg/client/informers/externalversions/lighthouse/v1alpha1"
	lhlisters "github.com/jenkins-x/lighthouse/pkg/client/listers/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	controllerName    = "jx-controller"
	activityKeyPrefix = "activity"
	jobKeyPrefix      = "job"
)

// Controller listens for changes to PipelineActivitys and updates the corresponding LighthouseJobs with their activity
type Controller struct {
	jxClient   jxclient.Interface
	lhClient   clientset.Interface
	kubeClient kubernetes.Interface

	mpClient metapipeline.Client

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

	configMapWatcher *watcher.ConfigMapWatcher

	jobConfig *config.Agent

	wg     *sync.WaitGroup
	logger *logrus.Entry
	ns     string
}

// NewController returns a new controller for syncing PipelineActivity updates to LighthouseJobs and commit statuses
func NewController(kubeClient kubernetes.Interface, jxClient jxclient.Interface, lhClient clientset.Interface, activityInformer jxinformers.PipelineActivityInformer,
	lhInformer lhinformers.LighthouseJobInformer, ns string, logger *logrus.Entry) (*Controller, error) {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName)
	}

	configAgent := &config.Agent{}

	onConfigYamlChange := func(text string) {
		if text != "" {
			cfg, err := config.LoadYAMLConfig([]byte(text))
			if err != nil {
				logrus.WithError(err).Error("Error processing the prow Config YAML")
			} else {
				logrus.Info("updating the prow core configuration")
				configAgent.Set(cfg)
			}
		}
	}

	callbacks := []watcher.ConfigMapCallback{
		&watcher.ConfigMapEntryCallback{
			Name:     util.ProwConfigMapName,
			Key:      util.ProwConfigFilename,
			Callback: onConfigYamlChange,
		},
	}
	configMapWatcher, err := watcher.NewConfigMapWatcher(kubeClient, ns, callbacks, stopper())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ConfigMap watcher")
	}

	activityLister := activityInformer.Lister()
	activitySynced := activityInformer.Informer().HasSynced

	mpClient, _, _, err := NewMetaPipelineClient(ns)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create metapipeline client")
	}

	controller := &Controller{
		jxClient:         jxClient,
		lhClient:         lhClient,
		mpClient:         mpClient,
		activityLister:   activityLister,
		activitySynced:   activitySynced,
		lhLister:         lhInformer.Lister(),
		lhSynced:         lhInformer.Informer().HasSynced,
		logger:           logger,
		ns:               ns,
		queue:            RateLimiter(),
		jobConfig:        configAgent,
		configMapWatcher: configMapWatcher,
		kubeClient:       kubeClient,
	}

	logger.Info("Setting up event handlers")
	lhInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := toKey(obj)
			if err == nil {
				controller.queue.AddRateLimited(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newJob := newObj.(*v1alpha1.LighthouseJob)
			oldJob := oldObj.(*v1alpha1.LighthouseJob)
			if oldJob.ResourceVersion == newJob.ResourceVersion {
				return
			}
			// Don't queue any job that isn't in the triggered state
			if newJob.Status.State != v1alpha1.TriggeredState {
				return
			}
			key, err := toKey(newObj)
			if err == nil {
				controller.queue.AddRateLimited(key)
			}
		},
	})
	activityInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := toKey(obj)
			if err == nil {
				controller.queue.AddRateLimited(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newAct := newObj.(*jxv1.PipelineActivity)
			oldAct := oldObj.(*jxv1.PipelineActivity)
			// Skip updates solely triggered by resyncs. We only care if they're actually different.
			if oldAct.ResourceVersion == newAct.ResourceVersion {
				return
			}
			key, err := toKey(newObj)
			if err == nil {
				controller.queue.AddRateLimited(key)
			}
		},
	})

	controller.wg = &sync.WaitGroup{}

	return controller, nil
}

func toKey(obj interface{}) (string, error) {
	baseKey, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return "", err
	}
	switch obj.(type) {
	case *v1alpha1.LighthouseJob:
		return jobKeyPrefix + ":::" + baseKey, nil
	case *jxv1.PipelineActivity:
		return activityKeyPrefix + ":::" + baseKey, nil
	default:
		return "", errors.New("unknown type, cannot enqueue")
	}
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
	// Get the type of the key
	keyParts := strings.Split(key, ":::")

	if len(keyParts) != 2 {
		return fmt.Errorf("no key type found in %s", key)
	}
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(keyParts[1])
	if err != nil {
		c.logger.Warnf("invalid resource key: %s", keyParts[1])
		return nil
	}

	if keyParts[0] == activityKeyPrefix {
		return c.syncActivity(namespace, name, key)
	}
	if keyParts[0] == jobKeyPrefix {
		return c.syncJob(namespace, name, key)
	}

	return fmt.Errorf("unknown key type: %s", keyParts[0])
}

func (c *Controller) syncJob(namespace, name, key string) error {
	// Get the LighthouseJob resource with this namespace/name
	origJob, err := c.lhLister.LighthouseJobs(namespace).Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			c.logger.Warnf("activity '%s' in work queue no longer exists", key)
			return nil
		}

		// Return an error here so that we requeue and retry.
		return err
	}

	spec := &origJob.Spec

	// Only launch for the appropriate agent types and for triggered state
	if origJob.Status.State == v1alpha1.TriggeredState && (spec.Agent == config.JenkinsXAgent || spec.Agent == config.LegacyDefaultAgent) {
		jobName := spec.Refs.Repo
		owner := spec.Refs.Org
		sourceURL := spec.Refs.CloneURI

		pullRefData := c.getPullRefs(sourceURL, spec)
		pullRefs := ""
		if len(spec.Refs.Pulls) > 0 {
			pullRefs = pullRefData.String()
		}

		branch := spec.GetBranch()
		if branch == "" {
			branch = "master"
		}
		if pullRefs == "" {
			pullRefs = branch + ":"
		}

		job := spec.Job
		var kind metapipeline.PipelineKind
		if len(spec.Refs.Pulls) > 0 {
			kind = metapipeline.PullRequestPipeline
		} else {
			kind = metapipeline.ReleasePipeline
		}

		l := logrus.WithFields(logrus.Fields(map[string]interface{}{
			"Owner":     owner,
			"Name":      jobName,
			"SourceURL": sourceURL,
			"Branch":    branch,
			"PullRefs":  pullRefs,
			"Job":       job,
		}))
		l.Info("about to start Jenkinx X meta pipeline")

		sa := os.Getenv("JX_SERVICE_ACCOUNT")
		if sa == "" {
			sa = "tekton-bot"
		}

		pipelineCreateParam := metapipeline.PipelineCreateParam{
			PullRef:      pullRefData,
			PipelineKind: kind,
			Context:      spec.Context,
			// No equivalent to https://github.com/jenkins-x/jx/blob/bb59278c2707e0e99b3c24be926745c324824388/pkg/cmd/controller/pipeline/pipelinerunner_controller.go#L236
			//   for getting environment variables from the prow job here, so far as I can tell (abayer)
			// Also not finding an equivalent to labels from the PipelineRunRequest
			ServiceAccount: sa,
			// I believe we can use an empty string default image?
			DefaultImage: os.Getenv("JX_DEFAULT_IMAGE"),
			EnvVariables: spec.GetEnvVars(),
		}

		activityKey, tektonCRDs, err := c.mpClient.Create(pipelineCreateParam)
		if err != nil {
			return errors.Wrap(err, "unable to create Tekton CRDs")
		}

		jobCopy := origJob.DeepCopy()

		// Add the build number from the activity key to the labels on the job
		jobCopy.Labels[util.BuildNumLabel] = activityKey.Build

		origJSON, err := json.Marshal(origJob)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal original job %s", origJob.Name)
		}
		copyJSON, err := json.Marshal(jobCopy)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal updated job %s", jobCopy.Name)
		}
		patch, err := jsonpatch.CreateMergePatch(origJSON, copyJSON)
		if err != nil {
			return errors.Wrapf(err, "failed to create JSON patch for job %s", jobCopy.Name)
		}

		appliedJob, err := c.lhClient.LighthouseV1alpha1().LighthouseJobs(c.ns).Patch(jobCopy.Name, types.MergePatchType, patch)
		if err != nil {
			return errors.Wrapf(err, "unable to set build number on LighthouseJob %s", jobCopy.Name)
		}

		// Set status on the job
		appliedJob.Status = v1alpha1.LighthouseJobStatus{
			State:        v1alpha1.PendingState,
			ActivityName: util.ToValidName(activityKey.Name),
			StartTime:    metav1.Now(),
		}
		_, err = c.lhClient.LighthouseV1alpha1().LighthouseJobs(c.ns).UpdateStatus(appliedJob)
		if err != nil {
			return errors.Wrapf(err, "unable to set status on LighthouseJob %s", appliedJob.Name)
		}

		err = c.mpClient.Apply(activityKey, tektonCRDs)
		if err != nil {
			return errors.Wrap(err, "unable to apply Tekton CRDs")
		}
	}
	return nil
}

func (c *Controller) syncActivity(namespace, name, key string) error {
	// Get the PipelineActivity resource with this namespace/name
	jxActivity, err := c.activityLister.PipelineActivities(namespace).Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			c.logger.Warnf("activity '%s' in work queue no longer exists", key)
			return nil
		}

		// Return an error here so that we requeue and retry.
		return err
	}
	activityRecord, err := ConvertPipelineActivity(jxActivity)
	if err != nil {
		return err
	}

	var job *v1alpha1.LighthouseJob

	// Get all LighthouseJobs with the same owner/repo/branch/build/context
	labelSelector, err := createLabelSelectorFromActivity(activityRecord)
	possibleJobs, err := c.lhLister.LighthouseJobs(namespace).List(labelSelector)
	if err != nil {
		return err
	}
	if len(possibleJobs) == 0 {
		// TODO: Something to handle jx start pipeline cases - my previous approach resulted in infinite creations of new jobs, which was...wrong. (apb)
		c.logger.Warnf("no LighthouseJobs found matching label selector %s", labelSelector.String())
		return nil
	}

	// To be safe, find the job with the activity's name in its status.
	for _, j := range possibleJobs {
		if j.Status.ActivityName == activityRecord.Name {
			job = j
		}
	}

	if job == nil {
		return nil
	}

	// Update the job's status for the activity.
	jobCopy := job.DeepCopy()
	jobCopy.Status.Activity = activityRecord

	currentJob, err := c.lhLister.LighthouseJobs(namespace).Get(jobCopy.Name)
	if err != nil {
		c.logger.WithError(err).Errorf("couldn't get the orig of job %s", jobCopy.Name)
		// Return an error here so we requeue and retry.
		return err
	}
	if !reflect.DeepEqual(currentJob.Status, jobCopy.Status) {
		c.logger.Infof("Recording updated activity for job %s", currentJob.Name)
		currentJob.Status = jobCopy.Status
		_, err = c.lhClient.LighthouseV1alpha1().LighthouseJobs(namespace).UpdateStatus(currentJob)
		if err != nil {
			c.logger.WithError(err).Errorf("error updating status with new activity for job %s", currentJob.Name)
			// Return an error here so we requeue and retry.
			return err
		}
	}
	return nil
}

// RateLimiter creates a ratelimiting queue for the foghorn controller.
func RateLimiter() workqueue.RateLimitingInterface {
	rl := workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 120*time.Second),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(1000), 50000)},
	)
	return workqueue.NewNamedRateLimitingQueue(rl, controllerName)
}

func createLabelSelectorFromActivity(activity *v1alpha1.ActivityRecord) (labels.Selector, error) {
	var selectors []string

	if activity.Owner != "" {
		selectors = append(selectors, fmt.Sprintf("%s=%s", util.OrgLabel, strings.ToLower(activity.Owner)))
	}
	if activity.Repo != "" {
		selectors = append(selectors, fmt.Sprintf("%s=%s", util.RepoLabel, activity.Repo))
	}
	if activity.Branch != "" {
		selectors = append(selectors, fmt.Sprintf("%s=%s", util.BranchLabel, activity.Branch))
	}
	if activity.BuildIdentifier != "" {
		selectors = append(selectors, fmt.Sprintf("%s=%s", util.BuildNumLabel, activity.BuildIdentifier))
	}
	if activity.Context != "" {
		selectors = append(selectors, fmt.Sprintf("%s=%s", util.ContextLabel, activity.Context))
	}

	return labels.Parse(strings.Join(selectors, ","))
}

func (c *Controller) getPullRefs(sourceURL string, spec *v1alpha1.LighthouseJobSpec) metapipeline.PullRef {
	var pullRef metapipeline.PullRef
	if len(spec.Refs.Pulls) > 0 {
		var prs []metapipeline.PullRequestRef
		for _, pull := range spec.Refs.Pulls {
			prs = append(prs, metapipeline.PullRequestRef{ID: strconv.Itoa(pull.Number), MergeSHA: pull.SHA})
		}

		pullRef = metapipeline.NewPullRefWithPullRequest(sourceURL, spec.Refs.BaseRef, spec.Refs.BaseSHA, prs...)
	} else {
		pullRef = metapipeline.NewPullRef(sourceURL, spec.Refs.BaseRef, spec.Refs.BaseSHA)
	}

	return pullRef
}

// stopper returns a channel that remains open until an interrupt is received.
func stopper() chan struct{} {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logrus.Warn("Interrupt received, attempting clean shutdown...")
		close(stop)
		<-c
		logrus.Error("Second interrupt received, force exiting...")
		os.Exit(1)
	}()
	return stop
}
