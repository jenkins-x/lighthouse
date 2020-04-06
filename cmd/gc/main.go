package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	lhclient "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/prow/logrusutil"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type options struct {
	namespace string
	maxAge    time.Duration
}

func (o *options) Validate() error {
	if o.namespace == "" {
		return fmt.Errorf("no --namespace given")
	}
	return nil
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	logrusutil.ComponentInit("lighthouse-gc-jobs")

	var o options
	fs.DurationVar(&o.maxAge, "max-age", 7*24*time.Hour, "Maximum age to keep LighthouseJobs.")
	fs.StringVar(&o.namespace, "namespace", "", "The namespace to listen in")

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

	cfg, err := jxfactory.NewFactory().CreateKubeConfig()
	if err != nil {
		logrus.WithError(err).Fatal("Could not create kubeconfig")
	}
	lhClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create Lighthouse API client")
	}

	lhInterface := lhClient.LighthouseV1alpha1().LighthouseJobs(o.namespace)

	jobList, err := lhInterface.List(metav1.ListOptions{})
	if err != nil {
		logrus.WithError(err).Fatalf("Could not list LighthouseJobs in namespace %s", o.namespace)
	}

	now := time.Now()

	for _, j := range jobList.Items {
		completionTime := j.Status.CompletionTime
		if completionTime != nil && completionTime.Add(o.maxAge).Before(now) {
			// The job completed at least maxAge ago, so delete it.
			err = deleteLighthouseJob(lhInterface, &j)
			if err != nil {
				logrus.WithError(err).Fatalf("Failed to delete LighthouseJob %s", j.Name)
			}
		} else if completionTime == nil && j.Status.StartTime.Add(o.maxAge).Before(now) {
			// The job never completed, but was created at least maxAge ago, so delete it.
			err = deleteLighthouseJob(lhInterface, &j)
			if err != nil {
				logrus.WithError(err).Fatalf("Failed to delete LighthouseJob %s", j.Name)
			}
		}
	}
}

func deleteLighthouseJob(lhInterface lhclient.LighthouseJobInterface, lhJob *v1alpha1.LighthouseJob) error {
	logrus.Infof("Deleting LighthouseJob %s", lhJob.Name)
	return lhInterface.Delete(lhJob.Name, metav1.NewDeleteOptions(0))
}
