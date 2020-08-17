package tekton

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	lhinformers "github.com/jenkins-x/lighthouse/pkg/client/informers/externalversions/lighthouse/v1alpha1"
	lhlisters "github.com/jenkins-x/lighthouse/pkg/client/listers/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	resourcesv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	tektonclient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektoninformers "github.com/tektoncd/pipeline/pkg/client/informers/externalversions/pipeline/v1beta1"
	tektonlisters "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"golang.org/x/time/rate"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilrand "k8s.io/apimachinery/pkg/util/rand"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	controllerName = "tekton-controller"
	prKeyPrefix    = "pr"
	jobKeyPrefix   = "job"

	gitCloneCatalogTaskName = "git-clone"
	gitCloneURLParam        = "url"
	gitCloneRevisionParam   = "revision"
	gitMergeCatalogTaskName = "git-batch-merge"
	gitMergeBatchRefsParam  = "batchedRefs"
)

// Controller listens for changes to PipelineRuns and updates the corresponding LighthouseJobs with their activity
type Controller struct {
	tektonClient tektonclient.Interface
	lhClient     clientset.Interface
	kubeClient   kubernetes.Interface

	prLister tektonlisters.PipelineRunLister
	prSynced cache.InformerSynced

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

	wg           *sync.WaitGroup
	logger       *logrus.Entry
	ns           string
	dashboardURL string
}

// NewController returns a new controller for syncing PipelineRun updates to LighthouseJobs and commit statuses
func NewController(kubeClient kubernetes.Interface, tektonClient tektonclient.Interface, lhClient clientset.Interface, prInformer tektoninformers.PipelineRunInformer,
	lhInformer lhinformers.LighthouseJobInformer, ns string, dashboardURL string, logger *logrus.Entry) (*Controller, error) {
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName)
	}

	configAgent := &config.Agent{}

	configMapWatcher, err := watcher.SetupConfigMapWatchers(ns, configAgent, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ConfigMap watcher")
	}

	prLister := prInformer.Lister()
	prSynced := prInformer.Informer().HasSynced

	controller := &Controller{
		tektonClient:     tektonClient,
		lhClient:         lhClient,
		prLister:         prLister,
		prSynced:         prSynced,
		lhLister:         lhInformer.Lister(),
		lhSynced:         lhInformer.Informer().HasSynced,
		logger:           logger,
		ns:               ns,
		dashboardURL:     dashboardURL,
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
	prInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := toKey(obj)
			if err == nil {
				controller.queue.AddRateLimited(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newAct := newObj.(*tektonv1beta1.PipelineRun)
			oldAct := oldObj.(*tektonv1beta1.PipelineRun)
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
	case *tektonv1beta1.PipelineRun:
		return prKeyPrefix + ":::" + baseKey, nil
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
	if ok := cache.WaitForCacheSync(stopCh, c.prSynced, c.lhSynced); !ok {
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

	if keyParts[0] == prKeyPrefix {
		return c.syncPipelineRun(namespace, name, key)
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
			c.logger.Warnf("job '%s' in work queue no longer exists", key)
			return nil
		}

		// Return an error here so that we requeue and retry.
		return err
	}

	spec := &origJob.Spec

	// Only launch for the appropriate agent types and for triggered state
	if origJob.Status.State == v1alpha1.TriggeredState && spec.Agent == config.TektonPipelineAgent {
		jobCopy := origJob.DeepCopy()
		pr, err := c.makePipelineRun(*jobCopy)
		if err != nil {
			return errors.Wrapf(err, "failed to create PipelineRun for job %s", jobCopy.Name)
		}

		// Add the build ID from the pipelinerun to the labels on the job
		jobCopy.Labels[util.BuildNumLabel] = pr.Labels[util.BuildNumLabel]

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
			State:     v1alpha1.PendingState,
			StartTime: metav1.Now(),
		}
		_, err = c.lhClient.LighthouseV1alpha1().LighthouseJobs(c.ns).UpdateStatus(appliedJob)
		if err != nil {
			return errors.Wrapf(err, "unable to set status on LighthouseJob %s", appliedJob.Name)
		}

		_, err = c.tektonClient.TektonV1beta1().PipelineRuns(c.ns).Create(pr)
		if err != nil {
			return errors.Wrap(err, "unable to create PipelineRun")
		}
	}
	return nil
}

func (c *Controller) syncPipelineRun(namespace, name, key string) error {
	// Get the PipelineActivity resource with this namespace/name
	rawRun, err := c.prLister.PipelineRuns(namespace).Get(name)
	if err != nil {
		if kubeerrors.IsNotFound(err) {
			c.logger.Warnf("PipelineRun '%s' in work queue no longer exists", key)
			return nil
		}

		// Return an error here so that we requeue and retry.
		return err
	}
	activityRecord := ConvertPipelineRun(rawRun)

	var job *v1alpha1.LighthouseJob

	// TODO: For the moment, we could just do a get, but I think we're going to change the name of the PR eventually.
	// Get all LighthouseJobs with the same owner/repo/branch/build/context
	labelSelector, err := createLabelSelectorFromActivity(activityRecord)
	if err != nil {
		c.logger.Errorf("error creating label selector: %s", err)
		return err
	}

	possibleJobs, err := c.lhLister.LighthouseJobs(namespace).List(labelSelector)
	if err != nil {
		return err
	}
	if len(possibleJobs) == 0 {
		c.logger.Warnf("no LighthouseJobs found matching label selector %s", labelSelector.String())
		return nil
	}

	// To be safe, find the job with the activity's name in its status.
	for _, j := range possibleJobs {
		if j.Name == activityRecord.JobID {
			job = j
		}
	}

	if job == nil {
		return nil
	}

	// Update the job's status for the activity.
	jobCopy := job.DeepCopy()
	if c.dashboardURL != "" {
		jobCopy.Status.ReportURL = fmt.Sprintf("%s/#/namespaces/%s/pipelineruns/%s", trimDashboardURL(c.dashboardURL), c.ns, rawRun.Name)
	}
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

func trimDashboardURL(base string) string {
	return strings.TrimSuffix(strings.TrimSuffix(base, "#"), "/")
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

// generateBuildID generates a unique build ID for consistency with Prow behavior
func generateBuildID() string {
	return fmt.Sprintf("%d", utilrand.Int())
}

// makePipeline creates a PipelineRun and substitutes LighthouseJob managed pipeline resources with ResourceSpec instead of ResourceRef
// so that we don't have to take care of potentially dangling created pipeline resources.
func (c *Controller) makePipelineRun(lj v1alpha1.LighthouseJob) (*tektonv1beta1.PipelineRun, error) {
	// First validate.
	if lj.Spec.PipelineRunSpec == nil {
		return nil, errors.New("no PipelineSpec defined")
	}

	buildID := generateBuildID()
	if buildID == "" {
		return nil, errors.New("empty BuildID in status")
	}

	prLabels, annotations := jobutil.LabelsAndAnnotationsForJob(lj, buildID)
	specCopy := lj.Spec.PipelineRunSpec.DeepCopy()
	p := tektonv1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Annotations:  annotations,
			GenerateName: fmt.Sprintf("%s-", lj.Spec.Job),
			Namespace:    lj.Spec.Namespace,
			Labels:       prLabels,
		},
		Spec: *specCopy,
	}
	// Set a default timeout of 1 day if no timeout is specified
	if p.Spec.Timeout == nil {
		p.Spec.Timeout = &metav1.Duration{Duration: 24 * time.Hour}
	}
	p.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: lighthouse.GroupAndVersion,
		Kind:       "LighthouseJob",
		Name:       lj.Name,
		UID:        lj.UID,
	}}

	if err := validatePipelineRunSpec(lj.Spec.Type, lj.Spec.ExtraRefs, lj.Spec.PipelineRunSpec); err != nil {
		return nil, fmt.Errorf("invalid pipeline_run_spec: %v", err)
	}

	// Add parameters instead of env vars.
	env := lj.Spec.GetEnvVars()
	env[v1alpha1.BuildIDEnv] = buildID
	env[v1alpha1.RepoURLEnv] = lj.Spec.Refs.CloneURI
	var batchedRefsVals []string
	for _, pull := range lj.Spec.Refs.Pulls {
		if pull.Ref != "" {
			batchedRefsVals = append(batchedRefsVals, pull.Ref)
		}
	}
	if len(batchedRefsVals) > 0 {
		env[v1alpha1.PullPullRefEnv] = strings.Join(batchedRefsVals, " ")
	}
	if len(lj.Spec.PipelineRunParams) > 0 {
		for _, param := range lj.Spec.PipelineRunParams {
			parsedTemplate, err := template.New(param.Name).Parse(param.ValueTemplate)
			if err != nil {
				return nil, err
			}
			var msgBuffer bytes.Buffer
			err = parsedTemplate.Execute(&msgBuffer, lj.Spec.Refs)
			if err != nil {
				return nil, err
			}
			env[param.Name] = msgBuffer.String()
		}
	} else {
		paramNames, err := determineGitCloneOrMergeTaskParams(&p, c.tektonClient)
		if err != nil {
			return nil, err
		}
		if paramNames == nil {
			c.logger.Warnf("git-clone and/or git-batch-merge task parameters not found in Pipeline for PipelineRun, so skipping setting PipelineRun parameters for revision")
		} else {
			env[paramNames.urlParam] = lj.Spec.Refs.CloneURI
			if paramNames.revParam != "" {
				if len(lj.Spec.Refs.Pulls) > 0 {
					env[paramNames.revParam] = lj.Spec.Refs.Pulls[0].SHA
				} else {
					env[paramNames.revParam] = "master"
				}
			}
			if paramNames.baseRevisionParam != "" {
				env[paramNames.baseRevisionParam] = lj.Spec.Refs.BaseRef
			}
			if paramNames.batchedRefsParam != "" {
				env[paramNames.batchedRefsParam] = strings.Join(batchedRefsVals, " ")
			}
		}
	}
	for _, key := range sets.StringKeySet(env).List() {
		val := env[key]
		// TODO: make this handle existing values/substitutions.
		p.Spec.Params = append(p.Spec.Params, tektonv1beta1.Param{
			Name: key,
			Value: tektonv1beta1.ArrayOrString{
				Type:      tektonv1beta1.ParamTypeString,
				StringVal: val,
			},
		})
	}

	return &p, nil
}

type gitTaskParamNames struct {
	urlParam          string
	revParam          string
	batchedRefsParam  string
	baseRevisionParam string
}

func determineGitCloneOrMergeTaskParams(pr *tektonv1beta1.PipelineRun, tektonClient tektonclient.Interface) (*gitTaskParamNames, error) {
	if pr == nil {
		return nil, errors.New("provided PipelineRun is nil")
	}

	if pr.Spec.PipelineSpec == nil && pr.Spec.PipelineRef == nil {
		return nil, errors.New("neither PipelineSpec nor PipelineRef specified for PipelineRun")
	}
	var pipelineSpec *tektonv1beta1.PipelineSpec

	if pr.Spec.PipelineSpec != nil {
		pipelineSpec = pr.Spec.PipelineSpec
	} else {
		pipeline, err := tektonClient.TektonV1beta1().Pipelines(pr.Namespace).Get(pr.Spec.PipelineRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to find Pipeline %s for PipelineRun", pr.Spec.PipelineRef.Name)
		}
		pipelineSpec = &pipeline.Spec
	}

	paramNames := &gitTaskParamNames{}

	for _, task := range pipelineSpec.Tasks {
		if task.TaskRef != nil {
			if task.TaskRef.Name == gitCloneCatalogTaskName {
				for _, p := range task.Params {
					if p.Name == gitCloneURLParam && p.Value.Type == tektonv1beta1.ParamTypeString {
						paramNames.urlParam = extractPipelineParamFromTaskParamValue(p.Value.StringVal)
					}
					if p.Name == gitCloneRevisionParam && p.Value.Type == tektonv1beta1.ParamTypeString {
						paramNames.revParam = extractPipelineParamFromTaskParamValue(p.Value.StringVal)
					}
				}

				if paramNames.urlParam != "" && paramNames.revParam != "" {
					return paramNames, nil
				}
			}
			if task.TaskRef.Name == gitMergeCatalogTaskName {
				for _, p := range task.Params {
					if p.Name == gitCloneURLParam && p.Value.Type == tektonv1beta1.ParamTypeString {
						paramNames.urlParam = extractPipelineParamFromTaskParamValue(p.Value.StringVal)
					}
					if p.Name == gitCloneRevisionParam && p.Value.Type == tektonv1beta1.ParamTypeString {
						paramNames.baseRevisionParam = extractPipelineParamFromTaskParamValue(p.Value.StringVal)
					}
					if p.Name == gitMergeBatchRefsParam && p.Value.Type == tektonv1beta1.ParamTypeString {
						paramNames.batchedRefsParam = extractPipelineParamFromTaskParamValue(p.Value.StringVal)
					}
				}

				if paramNames.urlParam != "" && paramNames.batchedRefsParam != "" {
					return paramNames, nil
				}

			}
		}
	}

	return nil, nil
}

func extractPipelineParamFromTaskParamValue(taskParam string) string {
	if strings.HasPrefix(taskParam, "$(params.") && strings.HasSuffix(taskParam, ")") {
		return strings.TrimPrefix(strings.TrimSuffix(taskParam, ")"), "$(params.")
	}
	return ""
}

var reProwExtraRef = regexp.MustCompile(`PROW_EXTRA_GIT_REF_(\d+)`)

func validatePipelineRunSpec(jobType config.PipelineKind, extraRefs []v1alpha1.Refs, spec *tektonv1beta1.PipelineRunSpec) error {
	if spec == nil {
		return nil
	}
	// Validate that that the refs match what is requested by the job.
	// The implicit git ref is optional to use, but any extra refs specified must
	// be used or removed. (Specifying an unused extra ref must always be
	// unintentional so we want to warn the user.)
	extraIndexes := sets.NewInt()
	for _, resource := range spec.Resources {
		// Validate that periodic jobs don't request an implicit git ref
		if jobType == config.PeriodicJob && resource.ResourceRef.Name == ProwImplicitGitResource {
			return fmt.Errorf("periodic jobs do not have an implicit git ref to replace %s", ProwImplicitGitResource)
		}

		match := reProwExtraRef.FindStringSubmatch(resource.ResourceRef.Name)
		if len(match) != 2 {
			continue
		}
		if len(match[1]) > 1 && match[1][0] == '0' {
			return fmt.Errorf("resource %q: leading zeros are not allowed in PROW_EXTRA_GIT_REF_* indexes", resource.Name)
		}
		i, _ := strconv.Atoi(match[1]) // This can't error based on the regexp.
		extraIndexes.Insert(i)
	}
	for i := range extraRefs {
		if !extraIndexes.Has(i) {
			return fmt.Errorf("extra_refs[%d] is not used; some resource must reference PROW_EXTRA_GIT_REF_%d", i, i)
		}
	}
	if len(extraRefs) != extraIndexes.Len() {
		strs := make([]string, 0, extraIndexes.Len())
		for i := range extraIndexes {
			strs = append(strs, strconv.Itoa(i))
		}
		return fmt.Errorf(
			"%d extra_refs are specified, but the following PROW_EXTRA_GIT_REF_* indexes are used: %s",
			len(extraRefs),
			strings.Join(strs, ", "),
		)
	}
	return nil
}

// makePipelineGitResourceSpec creates a pipeline git resource spec from the LighthouseJob's refs
func makePipelineGitResourceSpec(refs v1alpha1.Refs) *resourcesv1alpha1.PipelineResourceSpec {
	// Pick source URL
	var sourceURL string
	switch {
	case refs.CloneURI != "":
		sourceURL = refs.CloneURI
	case refs.RepoLink != "":
		sourceURL = fmt.Sprintf("%s.git", refs.RepoLink)
	default:
		sourceURL = fmt.Sprintf("https://github.com/%s/%s.git", refs.Org, refs.Repo)
	}

	// Pick revision
	var revision string
	switch {
	case len(refs.Pulls) > 0:
		if refs.Pulls[0].SHA != "" {
			revision = refs.Pulls[0].SHA
		} else {
			revision = fmt.Sprintf("pull/%d/head", refs.Pulls[0].Number)
		}
	case refs.BaseSHA != "":
		revision = refs.BaseSHA
	default:
		revision = refs.BaseRef
	}

	spec := resourcesv1alpha1.PipelineResourceSpec{
		Type: resourcesv1alpha1.PipelineResourceTypeGit,
		Params: []resourcesv1alpha1.ResourceParam{
			{
				Name:  "url",
				Value: sourceURL,
			},
			{
				Name:  "revision",
				Value: revision,
			},
		},
	}

	return &spec
}
