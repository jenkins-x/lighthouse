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

package main

import (
	"context"
	"flag"
	"os"
	"time"

	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/jenkins-x/lighthouse/pkg/strobe"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/sirupsen/logrus"
	"gopkg.in/robfig/cron.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
)

type options struct {
	namespace string
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options
	fs.StringVar(&o.namespace, "namespace", "", "The namespace to listen in")

	err := fs.Parse(args)
	if err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}
	return o
}

func main() {
	logrusutil.ComponentInit("strobe")

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)

	// Retrieve LighthouseJob client
	_, _, lighthouseClient, _, err := clients.GetAPIClients()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to create Lighthouse client")
	}

	// Create rate limited queue. We will add the names of periodic jobs to this
	// queue as they are updated in the Lighthouse config
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Subscribe to config changes
	configAgent := &config.Agent{}
	configCh := make(chan config.Delta)
	configAgent.Subscribe(configCh)

	// Start config watcher
	configMapWatcher, err := watcher.SetupConfigMapWatchers(o.namespace, configAgent, nil)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to start ConfigMap watcher")
	}
	defer configMapWatcher.Stop()

	// Enqueue periodic jobs for reconciliation as changes to the Lighthouse
	// config are received
	go o.enqueuePeriodicJobs(lighthouseClient, configCh, queue)

	// Create and start controller
	controller := strobe.NewLighthousePeriodicJobController(queue, lighthouseClient, configAgent)
	controller.Run(1, util.Stopper())
}

// enqueuePeriodicJobs enqueues all existing periodic jobs and then enqueues
// periodic jobs by watching for changes to the Lighthouse config
func (o options) enqueuePeriodicJobs(lighthouseClient clientset.Interface, configCh <-chan config.Delta, queue workqueue.RateLimitingInterface) {
	// List and enqueue all existing periodic LighthouseJobs. This allows us to
	// determine the periodic jobs that were scheduled in the past and recover
	// from a missed schedule time due to a crash or restart.
	//
	// Note that if there are no existing LighthouseJobs for a particular
	// periodic job then there is no way to know how long ago it was defined and
	// so in this case we do nothing until the next schedule time; in other
	// words, it is possible for the first schedule time for a particular
	// periodic job to be missed if this controller is unavailable at schedule
	// time.
	//
	// Note that CronJobs do not suffer from this problem since the creation
	// timestamp can always be used to determine the earliest schedule time:
	// https://github.com/kubernetes/kubernetes/blob/v1.24.6/pkg/controller/cronjob/utils.go#L71-L77
	lighthouseJobList, err := lighthouseClient.LighthouseV1alpha1().LighthouseJobs(o.namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Fatal("Failed to list LighthouseJobs")
	}
	for _, lighthouseJob := range lighthouseJobList.Items {
		if lighthouseJob.Spec.Type == job.PeriodicJob && len(lighthouseJob.Spec.Job) > 0 && len(lighthouseJob.Spec.Namespace) > 0 {
			key := ctrl.Request{NamespacedName: types.NamespacedName{Name: lighthouseJob.Spec.Job, Namespace: lighthouseJob.Spec.Namespace}}
			queue.Add(key)
			logrus.Infof("Periodic job %s enqueued!", key)
		}
	}

	// Watch for config changes and enqueue periodic jobs
	for configDelta := range configCh {
		logrus.Info("Lighthouse config updated")
		config := configDelta.After
		for _, periodic := range config.JobConfig.Periodics {
			if periodic.Namespace == nil || *periodic.Namespace == "" {
				// This should not be possible as long as configuration defaults
				// are being applied properly
				logrus.Infof("Periodic job configuration %s has missing Namespace, skipping...", periodic.Name)
				continue
			}

			// If a Namespace was specified then ignore periodic jobs that do not match
			if o.namespace != "" && *periodic.Namespace != o.namespace {
				logrus.Infof("Periodic job configuration %s specifies an external Namespace %s, skipping...", periodic.Name, periodic.Namespace)
				continue
			}

			// Define queue key
			key := ctrl.Request{NamespacedName: types.NamespacedName{Name: periodic.Name, Namespace: *periodic.Namespace}}

			// Parse cron schedule and calculate its next schedule time
			cron, err := cron.Parse(periodic.Cron)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to parse cron schedule for periodic job %s, skipping...", key)
				continue
			}
			now := time.Now()
			nextScheduleTime := cron.Next(now)

			// Enqueue periodic job at its next schedule time. This prevents
			// jobs from being scheduled as soon as they are defined
			queue.AddAfter(key, nextScheduleTime.Sub(now))
			logrus.Infof("Periodic job %s enqueued!", key)
		}
	}
}
