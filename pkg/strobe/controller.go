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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
)

type LighthousePeriodicJobController struct {
	logger           *logrus.Entry
	queue            workqueue.RateLimitingInterface
	lighthouseClient clientset.Interface
	configAgent      *config.Agent
}

func NewLighthousePeriodicJobController(queue workqueue.RateLimitingInterface, lighthouseClient clientset.Interface, configAgent *config.Agent) *LighthousePeriodicJobController {
	return &LighthousePeriodicJobController{
		logger:           logrus.NewEntry(logrus.StandardLogger()),
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

func generateLighthouseJob(logger *logrus.Entry, periodicJobConfig *job.Periodic, lastMissedScheduleTime time.Time) *v1alpha1.LighthouseJob {
	// We use the last missed schedule time to generate the job name to act as a
	// lock to prevent duplicate jobs from being created for the same time
	hasher := fnv.New32a()
	hasher.Write([]byte(periodicJobConfig.Name + lastMissedScheduleTime.UTC().String()))
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
	lighthouseJobName := periodicJobConfig.Name
	if len(lighthouseJobName) > maxNameLength-len(suffix) {
		lighthouseJobName = lighthouseJobName[0 : maxNameLength-len(suffix)]
	}
	lighthouseJobName += suffix

	lighthouseJobSpec := jobutil.PeriodicSpec(logger, *periodicJobConfig)
	labels, annotations := jobutil.LabelsAndAnnotationsForSpec(lighthouseJobSpec, periodicJobConfig.Labels, periodicJobConfig.Annotations)

	return &v1alpha1.LighthouseJob{
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
		c.logger.WithError(err).Error("Failed to parse cron schedule")
		// There is no point raising the error because we will still not be able
		// to parse the cron schedule on retry. Instead, we wait for the
		// operator to update the config with a valid cron
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

	// We now want to calculate the last schedule time that we missed to
	// determine whether we need to schedule a job. To prevent an incorrect
	// clock from eating up all the CPU and memory of this controller we want to
	// limit how far we look back. Firstly, we know that the last missed
	// schedule time will be after the 2 intervals before the next schedule...
	nextNextScheduleTime := cron.Next(nextScheduleTime)
	interval := nextNextScheduleTime.Sub(nextScheduleTime)
	earliestScheduleTime := nextScheduleTime.Add(-2 * interval)
	// ...and we also do not want to consider any time before this job was last
	// scheduled
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
	if earliestScheduleTime.Before(lastScheduleTime) {
		earliestScheduleTime = lastScheduleTime
	}
	// We are now ready to calculate the last schedule time that we missed
	var lastMissedScheduleTime time.Time
	for t := cron.Next(earliestScheduleTime); !t.After(now); t = cron.Next(t) {
		lastMissedScheduleTime = t
	}

	// If we haven't missed any schedule times then there is nothing to do
	if lastMissedScheduleTime.IsZero() {
		c.logger.Infof("No schedule times have been missed for periodic job %s", req)
		return reconcileAfter, nil
	}

	// Generate LighthouseJob
	lighthouseJob := generateLighthouseJob(c.logger, periodicJobConfig, lastMissedScheduleTime)

	// Create LighthouseJob
	lighthouseJob, err = c.lighthouseClient.LighthouseV1alpha1().LighthouseJobs(req.Namespace).Create(context.TODO(), lighthouseJob, metav1.CreateOptions{})
	// Note that we ignore the error if the LighthouseJob already exists
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		c.logger.Errorf("Failed to create periodic job %s", req)
		return reconcileAfter, err
	}
	c.logger.Infof("LighthouseJob %s created!", lighthouseJob.Name)

	// Finish reconciliation if the job has already been triggered
	if len(lighthouseJob.Status.State) > 0 {
		c.logger.Infof("LighthouseJob %s has already been triggered!", lighthouseJob.Name)
		return reconcileAfter, nil
	}

	// Update LighthouseJob with triggered status
	lighthouseJob.Status = v1alpha1.LighthouseJobStatus{
		State: v1alpha1.TriggeredState,
	}
	_, err = c.lighthouseClient.LighthouseV1alpha1().LighthouseJobs(req.Namespace).UpdateStatus(context.TODO(), lighthouseJob, metav1.UpdateOptions{})
	if err != nil {
		c.logger.Errorf("Failed to upgrade periodic job %s", req)
		return reconcileAfter, err
	}
	c.logger.Infof("LighthouseJob %s triggered!", lighthouseJob.Name)

	return reconcileAfter, nil
}
