package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	lhclient "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type options struct {
	namespace string
	maxAge    time.Duration
	verbose   bool
}

func (o *options) Validate() error {
	return nil
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	logrusutil.ComponentInit("lighthouse-gc-jobs")

	var o options
	fs.DurationVar(&o.maxAge, "max-age", 7*24*time.Hour, "Maximum age to keep LighthouseJobs.")
	fs.StringVar(&o.namespace, "namespace", "", "The namespace to listen in (If should listen in all just leave empty - but ClusterRole is needed)")
	fs.BoolVar(&o.verbose, "verbose", false, "Increase verbosity to verbose level")

	err := fs.Parse(args)
	if err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	return o
}

func main() {
	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}
	if o.verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	cfg, err := clients.GetConfig("", "")
	if err != nil {
		logrus.WithError(err).Fatal("Could not create kubeconfig")
	}
	lhClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create Lighthouse API client")
	}

	if !cleanUp(lhClient, o.namespace, o.maxAge) {
		logrus.Errorln("Failed to clean up at least one job, exiting with failure")
		os.Exit(1)
	}
}

func cleanUp(lhClient *clientset.Clientset, namespace string, maxAge time.Duration) bool {
	lhInterface := lhClient.LighthouseV1alpha1().LighthouseJobs(namespace)

	jobList, err := lhInterface.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Fatalf("Could not list LighthouseJobs in namespace '%s'", namespace)
	}

	now := time.Now()
	result := true

	for _, job := range jobList.Items {
		logrus.Debugf("Checking job name=%s from %s completed at %s", job.Name, job.Namespace, job.Status.CompletionTime)

		j := job
		completionTime := j.Status.CompletionTime
		if completionTime != nil && completionTime.Add(maxAge).Before(now) {
			// The job completed at least maxAge ago, so delete it.
			err = deleteLighthouseJob(lhInterface, &j)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to delete LighthouseJob %s/%s", j.Namespace, j.Name)
				result = false
			}
		} else if completionTime == nil && j.Status.StartTime.Add(maxAge).Before(now) {
			// The job never completed, but was created at least maxAge ago, so delete it.
			err = deleteLighthouseJob(lhInterface, &j)
			if err != nil {
				logrus.WithError(err).Errorf("Failed to delete LighthouseJob %s/%s", j.Namespace, j.Name)
				result = false
			}
		}
	}

	return result
}

func deleteLighthouseJob(lhInterface lhclient.LighthouseJobInterface, lhJob *v1alpha1.LighthouseJob) error {
	logrus.Infof("Deleting LighthouseJob %s/%s", lhJob.Namespace, lhJob.Name)
	return lhInterface.Delete(context.TODO(), lhJob.Name, *metav1.NewDeleteOptions(0))
}
