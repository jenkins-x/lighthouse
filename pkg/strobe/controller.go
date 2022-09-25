package strobe

import (
	"context"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/sirupsen/logrus"
	"gopkg.in/robfig/cron.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	controllerName = "strobe"
)

type LighthousePeriodicJobController struct {
	logger           *logrus.Entry
	queue            workqueue.RateLimitingInterface
	lighthouseClient clientset.Interface
	configAgent      *config.Agent
}

func NewLighthousePeriodicJobController(queue workqueue.RateLimitingInterface, lighthouseClient clientset.Interface, configAgent *config.Agent) *LighthousePeriodicJobController {
	return &LighthousePeriodicJobController{
		logger:           logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName),
		queue:            queue,
		lighthouseClient: lighthouseClient,
		configAgent:      configAgent,
	}
}

func (c *LighthousePeriodicJobController) Run(workerCount int, stopCh chan struct{}) {
	c.logger.Info("Starting controller")
	defer c.queue.ShutDown()

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	c.logger.Info("Stopping controller")
}

func (c *LighthousePeriodicJobController) runWorker() {
	for c.processNextItem() {
	}
}

// processNextItem takes items from the queue and reconciles them
func (c *LighthousePeriodicJobController) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks
	// the key for other workers. This allows safe parallel processing because
	// the same key is never processed in parallel
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	reconcileAfter, err := c.reconcile(key.(ctrl.Request))

	// Handle the error if something went wrong with reconciliation
	c.handleErr(err, key)

	// Enqueue next job
	if reconcileAfter != time.Duration(0) {
		c.queue.AddAfter(key, reconcileAfter)
	}

	return true
}

// handleErr checks if an error happened and makes sure to retry later
func (c *LighthousePeriodicJobController) handleErr(err error, key interface{}) {
	if err == nil {
		c.logger.Infof("Periodic job %s reconciled successfully!", key)
		// Forget key on successful reconciliation
		c.queue.Forget(key)
		return
	}

	// Retry with backoff if there was a reconciliation error
	c.logger.WithError(err).Infof("Failed to reconcile periodic job %s", key)
	c.queue.AddRateLimited(key)
}

func (c *LighthousePeriodicJobController) findLighthousePeriodicJobConfig(req ctrl.Request) *job.Periodic {
	for _, periodic := range c.configAgent.Config().JobConfig.Periodics {
		if periodic.Name == req.Name && periodic.Namespace != nil && *periodic.Namespace == req.Namespace {
			c.logger.Infof("Found configuration for periodic job %s", req)
			return &periodic
		}
	}
	c.logger.Errorf("Failed to find configuration for periodic job %s", req)
	return nil
}

// reconcile contains the business logic of the controller
func (c *LighthousePeriodicJobController) reconcile(req ctrl.Request) (reconcileAfter time.Duration, err error) {
	c.logger.Infof("Reconciling periodic job %s...", req)

	// Find the periodic job configuration
	periodicJobConfig := c.findLighthousePeriodicJobConfig(req)
	if periodicJobConfig == nil {
		return reconcileAfter, nil
	}

	// Fix the current time to simplify calculations
	now := time.Now()

	// Parse cron schedule
	cron, err := cron.Parse(periodicJobConfig.Cron)
	if err != nil {
		c.logger.Info("Failed to parse cron schedule")
		return reconcileAfter, nil
	}
	nextScheduleTime := cron.Next(now)
	reconcileAfter = nextScheduleTime.Sub(now)

	// Find matching LighthouseJobs
	lighthouseJobList, err := c.lighthouseClient.LighthouseV1alpha1().LighthouseJobs(req.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		c.logger.Info("Failed to list LighthouseJobs")
		return reconcileAfter, err
	}
	var matchingLighthouseJobs []v1alpha1.LighthouseJob
	for _, lighthouseJob := range lighthouseJobList.Items {
		if lighthouseJob.Spec.Type == job.PeriodicJob &&
			lighthouseJob.Spec.Job == req.Name &&
			lighthouseJob.Spec.Namespace == req.Namespace {
			matchingLighthouseJobs = append(matchingLighthouseJobs, lighthouseJob)
		}
	}

	// Bail if we have reached the maximum concurrency for this job
	if periodicJobConfig.MaxConcurrency > 0 {
		var activeLighthouseJobs []v1alpha1.LighthouseJob
		for _, lighthouseJob := range matchingLighthouseJobs {
			if lighthouseJob.Status.CompletionTime == nil {
				activeLighthouseJobs = append(activeLighthouseJobs, lighthouseJob)
			}
		}
		if len(activeLighthouseJobs) > periodicJobConfig.MaxConcurrency {
			c.logger.Infof("Maximum concurrency limit for periodic job %s reached!", req)
			return reconcileAfter, nil
		}
	}

	// Determine last schedule time
	var lastScheduleTime time.Time
	for _, lighthouseJob := range matchingLighthouseJobs {
		scheduleTime := lighthouseJob.CreationTimestamp.Time
		if lastScheduleTime.IsZero() {
			lastScheduleTime = scheduleTime
		} else {
			if lastScheduleTime.Before(scheduleTime) {
				lastScheduleTime = scheduleTime
			}
		}
	}

	// If we have been unable to find the last schedule time or the last
	// schedule time is too far in the past then we set the last schedule time
	// to a recently passed schedule time
	nextNextScheduleTime := cron.Next(nextScheduleTime)
	// This is the time duration between schedules
	interScheduleDuration := nextNextScheduleTime.Sub(nextScheduleTime)
	// We look no more than two schedules back as that is enough to ensure at
	// least one expected schedule between then and now
	earliestScheduleTime := nextScheduleTime.Add(-2 * interScheduleDuration)
	if lastScheduleTime.IsZero() || lastScheduleTime.Before(earliestScheduleTime) {
		lastScheduleTime = earliestScheduleTime
	}

	// Calculate the last schedule time that we missed
	var lastMissedScheduleTime time.Time
	for t := cron.Next(lastScheduleTime); !t.After(now); t = cron.Next(t) {
		lastMissedScheduleTime = t
	}

	// If we haven't missed any schedule times then there is nothing to do
	if lastMissedScheduleTime.IsZero() {
		c.logger.Infof("No schedule times have been missed for periodic job %s", req)
		return reconcileAfter, nil
	}

	// Schedule a job for the last missed schedule. We use the last missed
	// schedule time to generate the job name to act as a lock to prevent
	// duplicate jobs from being created for the same time
	hasher := fnv.New32a()
	hasher.Write([]byte(req.Name + lastMissedScheduleTime.UTC().String()))
	hash := fmt.Sprint(hasher.Sum32())
	// The hash should only by of a certain length
	maxHashLength := 10
	if len(hash) > maxHashLength {
		hash = hash[0:maxHashLength]
	}
	suffix := "-" + hash
	// Kubernetes resource names have a maximum length:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-subdomain-names
	maxNameLength := 253
	lighthouseJobName := req.Name
	if len(lighthouseJobName) > maxNameLength-len(suffix) {
		lighthouseJobName = lighthouseJobName[0 : maxNameLength-len(suffix)]
	}
	lighthouseJobName += suffix

	// Generate LighthouseJob
	lighthouseJobSpec := jobutil.PeriodicSpec(c.logger, *periodicJobConfig)

	// Tekton Controller requires that `Refs` is not nil...
	// https://github.com/jenkins-x/lighthouse/blob/v1.6.5/pkg/engines/tekton/utils.go#L84
	if lighthouseJobSpec.Refs == nil {
		lighthouseJobSpec.Refs = &v1alpha1.Refs{}
	}
	// ...and `BaseRef` is used to generate the name of the PipelineRun
	// https://github.com/jenkins-x/lighthouse/blob/v1.6.5/pkg/jobutil/jobutil.go#L207
	lighthouseJobSpec.Refs.BaseRef = req.Name

	labels, annotations := jobutil.LabelsAndAnnotationsForSpec(lighthouseJobSpec, periodicJobConfig.Labels, periodicJobConfig.Annotations)
	lighthouseJob := &v1alpha1.LighthouseJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "lighthouse.jenkins.io/v1alpha1",
			Kind:       "LighthouseJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        lighthouseJobName,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: lighthouseJobSpec,
	}

	// Create LighthouseJob
	lighthouseJob, err = c.lighthouseClient.LighthouseV1alpha1().LighthouseJobs(req.Namespace).Create(context.TODO(), lighthouseJob, metav1.CreateOptions{})
	if err != nil {
		c.logger.Errorf("Failed to create periodic job %s", req)
		return reconcileAfter, err
	}
	c.logger.Infof("LighthouseJob %s created!", lighthouseJobName)
	// Upgrade LighthouseJob with triggered status
	lighthouseJob.Status = v1alpha1.LighthouseJobStatus{
		State: v1alpha1.TriggeredState,
	}
	_, err = c.lighthouseClient.LighthouseV1alpha1().LighthouseJobs(req.Namespace).UpdateStatus(context.TODO(), lighthouseJob, metav1.UpdateOptions{})
	if err != nil {
		c.logger.Errorf("Failed to upgrade periodic job %s", req)
		return reconcileAfter, err
	}
	c.logger.Infof("LighthouseJob %s updated!", lighthouseJobName)

	return reconcileAfter, nil
}
