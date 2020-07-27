package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	jxclient "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned"
	jxinformers "github.com/jenkins-x/jx-api/pkg/client/informers/externalversions"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	lhinformers "github.com/jenkins-x/lighthouse/pkg/client/informers/externalversions"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/engines/jx"
	"github.com/jenkins-x/lighthouse/pkg/interrupts"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
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

func main() {
	logrusutil.ComponentInit("lighthouse-jx-controller")

	defer interrupts.WaitForGracefulShutdown()

	stopCh := stopper()

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

	jxClient, err := jxclient.NewForConfig(cfg)
	if err != nil {
		logrus.WithError(err).Fatal("Could not create Jenkins X API client")
	}
	jxInformerFactory := jxinformers.NewSharedInformerFactoryWithOptions(jxClient, time.Minute*30, jxinformers.WithNamespace(o.namespace))
	paInformer := jxInformerFactory.Jenkins().V1().PipelineActivities()

	controller, err := jx.NewController(kubeClient,
		jxClient,
		lhClient,
		paInformer,
		lhInformerFactory.Lighthouse().V1alpha1().LighthouseJobs(),
		o.namespace,
		nil)

	if err != nil {
		logrus.WithError(err).Fatal("Error creating controller")
	}
	jxInformerFactory.Start(stopCh)
	lhInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		logrus.WithError(err).Fatal("Error running controller")
	}
}
