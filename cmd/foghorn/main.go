package main

import (
	"flag"
	"os"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/foghorn"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type options struct {
	namespace                string
	skipTerminatedReconciles bool
	maxConcurrentReconciles  int
}

func (o *options) Validate() error {
	return nil
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options
	fs.StringVar(&o.namespace, "namespace", "", "The namespace to listen in")
	fs.BoolVar(&o.skipTerminatedReconciles, "skip-terminated-reconciles", false, "When true, add LighthouseJob watch predicates beyond resource-version changes (skip enqueue when activity is terminal and SCM status is already final). Default false matches historical behavior (resource-version filter only)")
	fs.IntVar(&o.maxConcurrentReconciles, "max-concurrent-reconciles", 1, "Parallel reconciles for the foghorn controller")

	err := fs.Parse(args)
	if err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	return o
}

func main() {
	logrusutil.ComponentInit("lighthouse-foghorn")

	// Wire zap from controller-runtime so client-go and controller-runtime
	// do not emit log.SetLogger was never called.
	ctrl.SetLogger(zap.New(zap.UseDevMode(false)))

	scheme := runtime.NewScheme()
	if err := lighthousev1alpha1.AddToScheme(scheme); err != nil {
		logrus.WithError(err).Fatal("Failed to register lighthousev1alpha1 scheme")
	}

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	cfg, err := clients.GetConfig("", "")
	if err != nil {
		logrus.WithError(err).Fatal("Could not create kubeconfig")
	}

	mgr, err := ctrl.NewManager(cfg, manager.Options{
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				o.namespace: {},
			},
		},
		Scheme: scheme,
	})
	if err != nil {
		logrus.WithError(err).Fatal("Unable to start manager")
	}

	reconciler, err := foghorn.NewLighthouseJobReconciler(mgr.GetClient(), mgr.GetScheme(), o.namespace, o.skipTerminatedReconciles, o.maxConcurrentReconciles)
	if err != nil {
		logrus.WithError(err).Fatal("Unable to instantiate reconciler")
	}
	if err = reconciler.SetupWithManager(mgr); err != nil {
		logrus.WithError(err).Fatal("Unable to create controller")
	}

	defer reconciler.ConfigMapWatcher.Stop()

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logrus.WithError(err).Fatal("Problem running manager")
	}
}
