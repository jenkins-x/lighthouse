package main

import (
	"flag"
	"os"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	tektonengine "github.com/jenkins-x/lighthouse/pkg/engines/tekton"
	"github.com/jenkins-x/lighthouse/pkg/interrupts"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/sirupsen/logrus"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

type options struct {
	namespace               string
	dashboardURL            string
	dashboardTemplate       string
	enableRerunStatusUpdate bool
}

func (o *options) Validate() error {
	return nil
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options
	fs.StringVar(&o.namespace, "namespace", "", "The namespace to listen in")
	fs.StringVar(&o.dashboardURL, "dashboard-url", "", "The base URL for the Tekton Dashboard to link to for build reports")
	fs.StringVar(&o.dashboardTemplate, "dashboard-template", "", "The template expression for generating the URL to the build report based on the PipelineRun parameters. If not specified defaults to $LIGHTHOUSE_DASHBOARD_TEMPLATE")
	fs.BoolVar(&o.enableRerunStatusUpdate, "enable-rerun-status-update", false, "Enable updating the status at the git provider when PipelineRuns are rerun")
	err := fs.Parse(args)
	if err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	return o
}

func main() {
	logrusutil.ComponentInit("lighthouse-tekton-controller")

	scheme := runtime.NewScheme()
	if err := lighthousev1alpha1.AddToScheme(scheme); err != nil {
		logrus.WithError(err).Fatal("Failed to register scheme")
	}
	if err := pipelinev1beta1.AddToScheme(scheme); err != nil {
		logrus.WithError(err).Fatal("Failed to register scheme")
	}

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	cfg, err := clients.GetConfig("", "")
	if err != nil {
		logrus.WithError(err).Fatal("Could not create kubeconfig")
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{Scheme: scheme, Namespace: o.namespace})
	if err != nil {
		logrus.WithError(err).Fatal("Unable to start manager")
	}

	lhJobReconciler := tektonengine.NewLighthouseJobReconciler(mgr.GetClient(), mgr.GetAPIReader(), mgr.GetScheme(), o.dashboardURL, o.dashboardTemplate, o.namespace)
	if err = lhJobReconciler.SetupWithManager(mgr); err != nil {
		logrus.WithError(err).Fatal("Unable to create controller")
	}

	if o.enableRerunStatusUpdate {
		rerunPipelineRunReconciler := tektonengine.NewRerunPipelineRunReconciler(mgr.GetClient(), mgr.GetScheme())
		if err = rerunPipelineRunReconciler.SetupWithManager(mgr); err != nil {
			logrus.WithError(err).Fatal("Unable to create RerunPipelineRun controller")
		}
	}

	defer interrupts.WaitForGracefulShutdown()
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		logrus.WithError(err).Fatal("Problem running manager")
	}
}
