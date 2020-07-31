package main

import (
	"flag"
	"os"
	"time"

	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	lhinformers "github.com/jenkins-x/lighthouse/pkg/client/informers/externalversions"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/foghorn"
	"github.com/jenkins-x/lighthouse/pkg/interrupts"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

type options struct {
	namespace string

	dryRun bool
}

func (o *options) Validate() error {
	return nil
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options
	fs.BoolVar(&o.dryRun, "dry-run", true, "Whether to mutate any real-world state.")
	fs.StringVar(&o.namespace, "namespace", "", "The namespace to listen in")

	err := fs.Parse(args)
	if err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	return o
}

func main() {
	logrusutil.ComponentInit("lighthouse-foghorn")

	defer interrupts.WaitForGracefulShutdown()

	stopCh := util.Stopper()

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	cfg, err := clients.GetConfig("", "")
	if err != nil {
		logrus.WithError(err).Fatal("Could not create kubeconfig")
	}

	lhClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create Lighthouse API client")
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create Kubernetes API client")
	}
	lhInformerFactory := lhinformers.NewSharedInformerFactoryWithOptions(lhClient, time.Minute*30, lhinformers.WithNamespace(o.namespace))

	controller, err := foghorn.NewController(kubeClient,
		lhClient,
		lhInformerFactory.Lighthouse().V1alpha1().LighthouseJobs(),
		o.namespace,
		nil)

	if err != nil {
		logrus.WithError(err).Fatal("Error creating controller")
	}
	lhInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		logrus.WithError(err).Fatal("Error running controller")
	}
}
