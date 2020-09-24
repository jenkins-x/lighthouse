/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jenkins

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/jenkins-x/lighthouse/pkg/util"

	"github.com/pkg/errors"

	"github.com/jenkins-x/lighthouse/pkg/config/job"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	client "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/lighthouse"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"k8s.io/apimachinery/pkg/types"

	"github.com/bwmarrin/snowflake"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
)

type lighthouseJobClient interface {
	Create(*v1alpha1.LighthouseJob) (*v1alpha1.LighthouseJob, error)
	List(metav1.ListOptions) (*v1alpha1.LighthouseJobList, error)
	UpdateStatus(*v1alpha1.LighthouseJob) (*v1alpha1.LighthouseJob, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.LighthouseJob, err error)
}

type jenkinsClient interface {
	Build(*v1alpha1.LighthouseJob, string) error
	ListBuilds(jobs []BuildQueryParams) (map[string]Build, error)
	Abort(job string, build *Build) error
}

type syncFn func(v1alpha1.LighthouseJob, map[string]Build) error

// Controller manages LighthouseJobs on a Jenkins server.
type Controller struct {
	lighthouseClient lighthouseJobClient
	jenkinsClient    jenkinsClient
	log              *logrus.Entry
	cfg              config.Getter
	node             *snowflake.Node
	// if skip report job results to github
	skipReport bool
	// selector that will be applied on Lighthouse jobs.
	selector string

	lock sync.RWMutex
	// pendingJobs is a short-lived cache that helps in limiting
	// the maximum concurrency of jobs.
	pendingJobs map[string]int

	jobLock sync.RWMutex
	// shared across the controller and a goroutine that gathers metrics.
	jobs  []v1alpha1.LighthouseJob
	clock clock.Clock
}

// NewController creates a new Controller from the provided clients.
func NewController(lighthouseClient client.LighthouseJobInterface, jenkinsClient *Client, logger *logrus.Entry, cfg config.Getter, selector string) (*Controller, error) {
	n, err := snowflake.NewNode(1)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	return &Controller{
		lighthouseClient: lighthouseClient,
		jenkinsClient:    jenkinsClient,
		log:              logger,
		cfg:              cfg,
		selector:         selector,
		node:             n,
		pendingJobs:      make(map[string]int),
		clock:            clock.RealClock{},
	}, nil
}

func (c *Controller) config() lighthouse.Controller {
	operators := c.cfg().Jenkinses
	if len(operators) == 1 {
		return operators[0].Controller
	}
	configured := make([]string, 0, len(operators))
	for _, cfg := range operators {
		if cfg.LabelSelectorString == c.selector {
			return cfg.Controller
		}
		configured = append(configured, cfg.LabelSelectorString)
	}
	if len(c.selector) == 0 {
		c.log.Panicf("You need to specify a non-empty --label-selector (existing selectors: %v).", configured)
	} else {
		c.log.Panicf("No config exists for --label-selector=%s.", c.selector)
	}
	return lighthouse.Controller{}
}

// canExecuteConcurrently checks whether the provided LighthouseJob can
// be executed concurrently.
func (c *Controller) canExecuteConcurrently(job *v1alpha1.LighthouseJob) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if max := c.config().MaxConcurrency; max > 0 {
		var running int
		for _, num := range c.pendingJobs {
			running += num
		}
		if running >= max {
			c.log.WithFields(jobutil.LighthouseJobFields(job)).Debugf("Not starting another job, already %d running.", running)
			return false
		}
	}

	if job.Spec.MaxConcurrency == 0 {
		c.pendingJobs[job.Spec.Job]++
		return true
	}

	numPending := c.pendingJobs[job.Spec.Job]
	if numPending >= job.Spec.MaxConcurrency {
		c.log.WithFields(jobutil.LighthouseJobFields(job)).Debugf("Not starting another instance of %s, already %d running.", job.Spec.Job, numPending)
		return false
	}
	c.pendingJobs[job.Spec.Job]++
	return true
}

// incrementNumPendingJobs increments the amount of
// pending LighthouseJob for the given job identifier
func (c *Controller) incrementNumPendingJobs(job string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.pendingJobs[job]++
}

// Sync does one sync iteration.
func (c *Controller) Sync() error {
	jobList, err := c.lighthouseClient.List(metav1.ListOptions{LabelSelector: c.selector})
	if err != nil {
		return fmt.Errorf("error listing Lighthouse jobList: %v", err)
	}
	// Share what we have for gathering metrics.
	c.jobLock.Lock()
	c.jobs = jobList.Items
	c.jobLock.Unlock()

	// TODO: Replace the following filtering with a field selector once CRDs support field selectors.
	// https://github.com/kubernetes/kubernetes/issues/53459
	var jenkinsJobs []v1alpha1.LighthouseJob
	for _, j := range jobList.Items {
		if j.Spec.Agent == job.JenkinsAgent {
			jenkinsJobs = append(jenkinsJobs, j)
		}
	}

	jenkinsBuilds, err := c.jenkinsClient.ListBuilds(getJenkinsJobs(jenkinsJobs))
	if err != nil {
		return fmt.Errorf("error listing jenkins builds: %v", err)
	}

	var syncErrs []error
	if err := c.terminateDupes(jenkinsJobs, jenkinsBuilds); err != nil {
		syncErrs = append(syncErrs, err)
	}

	pendingCh, triggeredCh, abortedCh := jobutil.PartitionActive(jenkinsJobs)
	errCh := make(chan error, len(jenkinsJobs))

	// Re-instantiate on every re-sync of the controller instead of trying
	// to keep this in sync with the state of the world.
	c.pendingJobs = make(map[string]int)
	// Sync pending jobList first so we can determine what is the maximum
	// number of new jobList we can trigger when syncing the non-pending jobs.
	maxSyncRoutines := c.config().MaxGoroutines
	c.log.Debugf("Handling %d pending lighthouse jobs", len(pendingCh))
	syncLighthouseJobs(c.log, c.syncPendingJob, maxSyncRoutines, pendingCh, errCh, jenkinsBuilds)
	c.log.Debugf("Handling %d triggered lighthouse jobs", len(triggeredCh))
	syncLighthouseJobs(c.log, c.syncTriggeredJob, maxSyncRoutines, triggeredCh, errCh, jenkinsBuilds)
	c.log.Debugf("Handling %d aborted lighthouse jobs", len(abortedCh))
	syncLighthouseJobs(c.log, c.syncAbortedJob, maxSyncRoutines, abortedCh, errCh, jenkinsBuilds)

	close(errCh)

	for err := range errCh {
		syncErrs = append(syncErrs, err)
	}

	if len(syncErrs) == 0 {
		return nil
	}
	return fmt.Errorf("errors syncing: %v", syncErrs)
}

// getJenkinsJobs returns all the Jenkins jobs for all active
// lighthouse jobs from the provided list. It handles deduplication.
func getJenkinsJobs(lighthouseJobs []v1alpha1.LighthouseJob) []BuildQueryParams {
	jenkinsJobs := []BuildQueryParams{}

	for _, lighthouseJob := range lighthouseJobs {
		if lighthouseJob.Complete() {
			continue
		}

		jenkinsJobs = append(jenkinsJobs, BuildQueryParams{
			JobName:         getJobName(&lighthouseJob.Spec),
			LighthouseJobID: lighthouseJob.Name,
		})
	}

	return jenkinsJobs
}

// terminateDupes aborts presubmits that have a newer version. It modifies jobs
// in-place when it aborts.
func (c *Controller) terminateDupes(lighthouseJobs []v1alpha1.LighthouseJob, jenkinsJobs map[string]Build) error {
	// "lighthouseJob org/repo#number" -> newest lighthouseJob
	dupes := make(map[string]int)
	for i, lighthouseJob := range lighthouseJobs {
		if lighthouseJob.Complete() || lighthouseJob.Spec.Type != job.PresubmitJob {
			continue
		}
		n := fmt.Sprintf("%s %s/%s#%d", lighthouseJob.Spec.Job, lighthouseJob.Spec.Refs.Org, lighthouseJob.Spec.Refs.Repo, lighthouseJob.Spec.Refs.Pulls[0].Number)
		prev, ok := dupes[n]
		if !ok {
			dupes[n] = i
			continue
		}
		cancelIndex := i
		if (&lighthouseJobs[prev].Status.StartTime).Before(&lighthouseJob.Status.StartTime) {
			cancelIndex = prev
			dupes[n] = i
		}
		toCancel := lighthouseJobs[cancelIndex]

		// Abort presubmit jobs for commits that have been superseded by
		// newer commits in GitHub pull requests.
		build, buildExists := jenkinsJobs[toCancel.ObjectMeta.Name]
		// Avoid cancelling enqueued builds.
		if buildExists && build.IsEnqueued() {
			continue
		}
		// Otherwise, abort it.
		if buildExists {
			if err := c.jenkinsClient.Abort(getJobName(&toCancel.Spec), &build); err != nil {
				c.log.WithError(err).WithFields(jobutil.LighthouseJobFields(&toCancel)).Warn("Cannot cancel Jenkins build")
			}
		}

		prevState := toCancel.Status.State
		toCancel.Status.State = v1alpha1.AbortedState
		toCancel.SetComplete()
		c.addActivity(&toCancel)
		c.log.WithFields(jobutil.LighthouseJobFields(&toCancel)).
			WithField("from", prevState).
			WithField("to", toCancel.Status.State).Info("Transitioning states.")

		updatedJob, err := c.lighthouseClient.UpdateStatus(&toCancel)
		if err != nil {
			return errors.Wrapf(err, "unable to update LighthouseJob status")
		}
		lighthouseJobs[cancelIndex] = *updatedJob
	}
	return nil
}

func syncLighthouseJobs(l *logrus.Entry, syncFn syncFn, maxSyncRoutines int, lighthouseJobs <-chan v1alpha1.LighthouseJob, syncErrors chan<- error, jenkinsBuilds map[string]Build) {
	goroutines := maxSyncRoutines
	if goroutines > len(lighthouseJobs) {
		goroutines = len(lighthouseJobs)
	}
	wg := &sync.WaitGroup{}
	wg.Add(goroutines)
	l.Debugf("Firing up %d goroutines", goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for lighthouseJob := range lighthouseJobs {
				if err := syncFn(lighthouseJob, jenkinsBuilds); err != nil {
					syncErrors <- err
				}
			}
		}()
	}
	wg.Wait()
}

func (c *Controller) syncPendingJob(lighthouseJob v1alpha1.LighthouseJob, jenkinsBuilds map[string]Build) error {
	// Record last known state so we can patch
	originalLighthouseJob := lighthouseJob.DeepCopy()

	jenkinsJob, jbExists := jenkinsBuilds[lighthouseJob.ObjectMeta.Name]
	if !jbExists {
		lighthouseJob.SetComplete()
		lighthouseJob.Status.State = v1alpha1.ErrorState
		lighthouseJob.Status.Description = "Error finding Jenkins job."
	} else {
		switch {
		case jenkinsJob.IsEnqueued():
			// Still in queue.
			c.incrementNumPendingJobs(lighthouseJob.Spec.Job)
			return nil

		case jenkinsJob.IsRunning():
			// Build still going.
			c.incrementNumPendingJobs(lighthouseJob.Spec.Job)
			if lighthouseJob.Status.Description == "Jenkins job running." {
				return nil
			}
			lighthouseJob.Status.Description = "Jenkins job running."

		case jenkinsJob.IsSuccess():
			// Build is complete.
			lighthouseJob.SetComplete()
			lighthouseJob.Status.State = v1alpha1.SuccessState
			lighthouseJob.Status.Description = "Jenkins job succeeded."

		case jenkinsJob.IsFailure():
			lighthouseJob.Status.State = v1alpha1.FailureState
			lighthouseJob.Status.Description = "Jenkins job failed."
			lighthouseJob.SetComplete()

		case jenkinsJob.IsAborted():
			lighthouseJob.Status.State = v1alpha1.AbortedState
			lighthouseJob.Status.Description = "Jenkins job aborted."
			lighthouseJob.SetComplete()
		}
		// Construct the status URL that will be used in reports.
		lighthouseJob.Status.ActivityName = jenkinsJob.BuildID()
		var b bytes.Buffer
		if err := c.config().JobURLTemplate.Execute(&b, &lighthouseJob); err != nil {
			c.log.WithFields(jobutil.LighthouseJobFields(&lighthouseJob)).Errorf("error executing URL template: %v", err)
		} else {
			lighthouseJob.Status.ReportURL = b.String()
		}
	}

	var err error
	if originalLighthouseJob.Status.State != lighthouseJob.Status.State || originalLighthouseJob.Status.ReportURL != lighthouseJob.Status.ReportURL {
		c.log.WithFields(jobutil.LighthouseJobFields(&lighthouseJob)).
			WithField("from", originalLighthouseJob.Status.State).
			WithField("to", lighthouseJob.Status.State).Info("Transitioning states.")
		// make sure to set an activity record this job state update
		c.addActivity(&lighthouseJob)
		_, err = c.lighthouseClient.UpdateStatus(&lighthouseJob)
	}
	return err
}

func (c *Controller) syncAbortedJob(lighthouseJob v1alpha1.LighthouseJob, jenkinsBuilds map[string]Build) error {
	if lighthouseJob.Status.State != v1alpha1.AbortedState || lighthouseJob.Complete() {
		return nil
	}

	if build, exists := jenkinsBuilds[lighthouseJob.Name]; exists {
		if err := c.jenkinsClient.Abort(getJobName(&lighthouseJob.Spec), &build); err != nil {
			return fmt.Errorf("failed to abort Jenkins build: %v", err)
		}
	}

	lighthouseJob.SetComplete()
	c.addActivity(&lighthouseJob)

	_, err := c.lighthouseClient.UpdateStatus(&lighthouseJob)
	return err
}

func (c *Controller) syncTriggeredJob(lighthouseJob v1alpha1.LighthouseJob, jenkinsBuilds map[string]Build) error {
	// Record last known state so we can patch
	originalLighthouseJob := lighthouseJob.DeepCopy()

	if _, exists := jenkinsBuilds[lighthouseJob.ObjectMeta.Name]; !exists {
		// Do not start more jobs than specified.
		if !c.canExecuteConcurrently(&lighthouseJob) {
			return nil
		}
		buildID, err := c.getBuildID()
		if err != nil {
			return fmt.Errorf("error getting build ID: %v", err)
		}
		// Start the Jenkins job.
		if err := c.jenkinsClient.Build(&lighthouseJob, buildID); err != nil {
			c.log.WithError(err).WithFields(jobutil.LighthouseJobFields(&lighthouseJob)).Warn("Cannot start Jenkins build")
			lighthouseJob.Status.State = v1alpha1.ErrorState
			lighthouseJob.Status.Description = "Error starting Jenkins job."
			lighthouseJob.SetComplete()
		} else {
			lighthouseJob.Status.State = v1alpha1.PendingState
			lighthouseJob.Status.Description = "Jenkins job enqueued."
			lighthouseJob.Status.StartTime = metav1.Now()
		}
	} else {
		// If a Jenkins build already exists for this job, advance the LighthouseJob to Pending and
		// it should be handled by syncPendingJob in the next sync.
		lighthouseJob.Status.State = v1alpha1.PendingState
		lighthouseJob.Status.Description = "Jenkins job enqueued."
	}

	var err error
	if originalLighthouseJob.Status.State != lighthouseJob.Status.State {
		c.log.WithFields(jobutil.LighthouseJobFields(&lighthouseJob)).
			WithField("from", originalLighthouseJob.Status.State).
			WithField("to", lighthouseJob.Status.State).Info("Transitioning states.")
		// make sure to set an activity record this job state update
		c.addActivity(&lighthouseJob)
		_, err = c.lighthouseClient.UpdateStatus(&lighthouseJob)
	}

	return err
}

func (c *Controller) addActivity(lighthouseJob *v1alpha1.LighthouseJob) {
	activity := v1alpha1.ActivityRecord{
		Name:            lighthouseJob.Name,
		Status:          lighthouseJob.Status.State,
		StartTime:       &lighthouseJob.Status.StartTime,
		CompletionTime:  lighthouseJob.Status.CompletionTime,
		Owner:           lighthouseJob.Labels[util.OrgLabel],
		Repo:            lighthouseJob.Labels[util.RepoLabel],
		GitURL:          lighthouseJob.Annotations[util.CloneURIAnnotation],
		LastCommitSHA:   lighthouseJob.Labels[util.LastCommitSHALabel],
		Branch:          lighthouseJob.Labels[util.BranchLabel],
		BuildIdentifier: lighthouseJob.Labels[util.BuildNumLabel],
		Context:         lighthouseJob.Labels[util.ContextLabel],
	}

	lighthouseJob.Status.Activity = &activity
}

func (c *Controller) getBuildID() (string, error) {
	return c.node.Generate().String(), nil
}
