package tekton

import (
	"context"
	"fmt"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	configjob "github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const jobOwnerKey = ".metadata.controller"

var apiGVStr = lighthousev1alpha1.SchemeGroupVersion.String()

// LighthouseJobReconciler reconciles a LighthouseJob object
type LighthouseJobReconciler struct {
	client       client.Client
	apiReader    client.Reader
	logger       *logrus.Entry
	scheme       *runtime.Scheme
	dashboardURL string
	namespace    string
}

// NewLighthouseJobReconciler creates a LighthouseJob reconciler
func NewLighthouseJobReconciler(client client.Client, apiReader client.Reader, scheme *runtime.Scheme, dashboardURL string, namespace string) *LighthouseJobReconciler {
	return &LighthouseJobReconciler{
		client:       client,
		apiReader:    apiReader,
		logger:       logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName),
		scheme:       scheme,
		dashboardURL: dashboardURL,
		namespace:    namespace,
	}
}

// SetupWithManager sets up the reconcilier with it's manager
func (r *LighthouseJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&pipelinev1beta1.PipelineRun{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		obj := rawObj.(*pipelinev1beta1.PipelineRun)
		owner := metav1.GetControllerOf(obj)
		// TODO: would be nice to get kind from the type rather than a hard coded string
		if owner == nil || owner.APIVersion != apiGVStr || owner.Kind != "LighthouseJob" {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&lighthousev1alpha1.LighthouseJob{}).
		WithEventFilter(predicate.ResourceVersionChangedPredicate{}).
		Owns(&pipelinev1beta1.PipelineRun{}).
		Complete(r)
}

// Reconcile represents an iteration of the reconciliation loop
func (r *LighthouseJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()

	r.logger.Infof("Reconcile LighthouseJob %+v", req)

	// get lighthouse job
	var job lighthousev1alpha1.LighthouseJob
	if err := r.client.Get(ctx, req.NamespacedName, &job); err != nil {
		r.logger.Warningf("Unable to get LighthouseJob: %s", err)
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// filter on job agent
	if job.Spec.Agent != configjob.TektonPipelineAgent {
		return ctrl.Result{}, nil
	}

	// get job's pipeline runs
	var pipelineRunList pipelinev1beta1.PipelineRunList
	if err := r.client.List(ctx, &pipelineRunList, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		r.logger.Errorf("Failed list pipeline runs: %s", err)
		return ctrl.Result{}, err
	}

	// if pipeline run does not exist, create it
	if len(pipelineRunList.Items) == 0 {
		if job.Status.State == lighthousev1alpha1.TriggeredState {
			// construct a pipeline run
			pipelineRun, err := makePipelineRun(ctx, job, r.namespace, r.logger, r.apiReader)
			if err != nil {
				r.logger.Errorf("Failed to make pipeline run: %s", err)
				return ctrl.Result{}, err
			}
			// link it to the current lighthouse job
			if err := ctrl.SetControllerReference(&job, pipelineRun, r.scheme); err != nil {
				r.logger.Errorf("Failed to set owner reference: %s", err)
				return ctrl.Result{}, err
			}
			// TODO: changing the status should be a consequence of a pipeline run being created
			// update status
			job.Status = lighthousev1alpha1.LighthouseJobStatus{
				State:     lighthousev1alpha1.PendingState,
				StartTime: metav1.Now(),
			}
			if err := r.client.Status().Update(ctx, &job); err != nil {
				r.logger.Errorf("Failed to update LighthouseJob status: %s", err)
				return ctrl.Result{}, err
			}
			// create pipeline run
			if err := r.client.Create(ctx, pipelineRun); err != nil {
				r.logger.Errorf("Failed to create pipeline run: %s", err)
				return ctrl.Result{}, err
			}
		}
	} else if len(pipelineRunList.Items) == 1 {
		// if pipeline run exists, create it and update status
		pipelineRun := pipelineRunList.Items[0]
		r.logger.Infof("Reconcile PipelineRun %+v", pipelineRun)
		// update build id
		job.Labels[util.BuildNumLabel] = pipelineRun.Labels[util.BuildNumLabel]
		if err := r.client.Update(ctx, &job); err != nil {
			r.logger.Errorf("failed to update Project status: %s", err)
			return ctrl.Result{}, err
		}
		if r.dashboardURL != "" {
			job.Status.ReportURL = fmt.Sprintf("%s/#/namespaces/%s/pipelineruns/%s", trimDashboardURL(r.dashboardURL), r.namespace, pipelineRun.Name)
		}
		job.Status.Activity = ConvertPipelineRun(&pipelineRun)
		if err := r.client.Status().Update(ctx, &job); err != nil {
			r.logger.Errorf("Failed to update LighthouseJob status: %s", err)
			return ctrl.Result{}, err
		}
	} else {
		r.logger.Errorf("A lighthouse job should never have more than 1 pipeline run")
	}

	return ctrl.Result{}, nil
}
