package tekton

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	configjob "github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonversioned "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const jobOwnerKey = ".metadata.controller"

var apiGVStr = lighthousev1alpha1.SchemeGroupVersion.String()

// LighthouseJobReconciler reconciles a LighthouseJob object
type LighthouseJobReconciler struct {
	client                   client.Client
	tektonclient             tektonversioned.Interface
	apiReader                client.Reader
	logger                   *logrus.Entry
	scheme                   *runtime.Scheme
	idGenerator              buildIDGenerator
	dashboardURL             string
	dashboardTemplate        string
	namespace                string
	disableLogging           bool
	skipTerminatedReconciles bool
	maxConcurrentReconciles  int
}

// NewLighthouseJobReconciler creates a LighthouseJob reconciler
func NewLighthouseJobReconciler(client client.Client, apiReader client.Reader, scheme *runtime.Scheme, tektonclient tektonversioned.Interface, dashboardURL string, dashboardTemplate string, namespace string, skipTerminatedReconciles bool, maxConcurrentReconciles int) *LighthouseJobReconciler {
	if dashboardTemplate == "" {
		dashboardTemplate = os.Getenv("LIGHTHOUSE_DASHBOARD_TEMPLATE")
	}
	return &LighthouseJobReconciler{
		client:                   client,
		apiReader:                apiReader,
		logger:                   logrus.NewEntry(logrus.StandardLogger()).WithField("controller", controllerName),
		scheme:                   scheme,
		dashboardURL:             dashboardURL,
		dashboardTemplate:        dashboardTemplate,
		namespace:                namespace,
		tektonclient:             tektonclient,
		idGenerator:              &epochBuildIDGenerator{},
		skipTerminatedReconciles: skipTerminatedReconciles,
		maxConcurrentReconciles:  maxConcurrentReconciles,
	}
}

var tektonControllerIndexFunc = func(rawObj client.Object) []string {
	obj := rawObj.(*pipelinev1.PipelineRun)
	owner := metav1.GetControllerOf(obj)
	// TODO: would be nice to get kind from the type rather than a hard coded string
	if owner == nil || owner.APIVersion != apiGVStr || owner.Kind != "LighthouseJob" {
		return nil
	}
	return []string{owner.Name}
}

// SetupWithManager sets up the reconcilier with it's manager
func (r *LighthouseJobReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &pipelinev1.PipelineRun{}, jobOwnerKey, tektonControllerIndexFunc); err != nil {
		return err
	}

	ctrlr := ctrl.NewControllerManagedBy(mgr).
		For(&lighthousev1alpha1.LighthouseJob{}, builder.WithPredicates(
			lighthouseJobPredicateFactory(r.skipTerminatedReconciles),
		)).
		Owns(&pipelinev1.PipelineRun{}, builder.WithPredicates(
			predicate.ResourceVersionChangedPredicate{},
		))

	if r.maxConcurrentReconciles > 1 {
		ctrlr = ctrlr.WithOptions(
			controller.Options{
				MaxConcurrentReconciles: r.maxConcurrentReconciles,
			},
		)
	}

	return ctrlr.Complete(r)
}

// Reconcile represents an iteration of the reconciliation loop
func (r *LighthouseJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}

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
	var pipelineRunList pipelinev1.PipelineRunList
	if err := r.client.List(ctx, &pipelineRunList, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		r.logger.Errorf("Failed list pipeline runs: %s", err)
		return ctrl.Result{}, err
	}

	// if pipeline run does not exist, create it
	if len(pipelineRunList.Items) == 0 {
		if job.Status.State == lighthousev1alpha1.TriggeredState {
			// construct a pipeline run
			pipelineRun, err := makePipelineRun(ctx, job, r.namespace, r.logger, r.idGenerator, r.apiReader)
			if err != nil {
				r.logger.Errorf("Failed to make pipeline run: %s", err)
				return ctrl.Result{}, err
			}
			// link it to the current lighthouse job
			if err := ctrl.SetControllerReference(&job, pipelineRun, r.scheme); err != nil {
				r.logger.Errorf("Failed to set owner reference: %s", err)
				return ctrl.Result{}, err
			}

			// lets disable the blockOwnerDeletion as it fails on OpenShift
			for i := range pipelineRun.OwnerReferences {
				ref := &pipelineRun.OwnerReferences[i]
				if ref.Kind == "LighthouseJob" && ref.BlockOwnerDeletion != nil {
					ref.BlockOwnerDeletion = nil
				}
			}

			// TODO: changing the status should be a consequence of a pipeline run being created
			// update status
			status := lighthousev1alpha1.LighthouseJobStatus{
				State:     lighthousev1alpha1.PendingState,
				StartTime: metav1.Now(),
			}
			f := func(job *lighthousev1alpha1.LighthouseJob) error {
				job.Status = status
				if err := r.client.Status().Update(ctx, job); err != nil {
					r.logger.Errorf("Failed to update LighthouseJob status: %s", err)
					return err
				}
				return nil
			}
			err = r.retryModifyJob(ctx, req.NamespacedName, &job, f)
			if err != nil {
				return ctrl.Result{}, err
			}

			// create pipeline run
			if err := r.client.Create(ctx, pipelineRun); err != nil {
				r.logger.Errorf("Failed to create pipeline run: %s", err)
				return ctrl.Result{}, err
			}
		}
	} else if len(pipelineRunList.Items) == 1 {
		// if pipeline run exists, create pipelineactivity and update lighthousejob status
		pipelineRun := pipelineRunList.Items[0]
		if !r.disableLogging {
			r.logger.Infof("Reconcile PipelineRun %+v", pipelineRun)
		}
		// update build id
		if job.Labels[util.BuildNumLabel] != pipelineRun.Labels[util.BuildNumLabel] {
			f := func(job *lighthousev1alpha1.LighthouseJob) error {
				job.Labels[util.BuildNumLabel] = pipelineRun.Labels[util.BuildNumLabel]
				if err := r.client.Update(ctx, job); err != nil {
					return errors.Wrapf(err, "failed to add build label Project status")
				}
				return nil
			}
			err := r.retryModifyJob(ctx, req.NamespacedName, &job, f)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		// When terminal shortcut is enabled, skip ConvertPipelineRun + status update if the PipelineRun is
		// done and Activity already matches (covers Owns(PipelineRun) events; same flag as For() predicate).
		if r.skipTerminatedReconciles {
			expectedReportURL := ""
			if r.dashboardURL != "" {
				expectedReportURL = r.getPipelingetPipelineTargetURLeTargetURL(pipelineRun)
			}
			if TerminalActivitySyncedWithPipelineRun(&job, &pipelineRun, expectedReportURL, r.dashboardURL != "") {
				r.logger.Debugf("Skipping terminal in-sync LighthouseJob %s", req.Name)
				return ctrl.Result{}, nil
			}
		}

		f := func(job *lighthousev1alpha1.LighthouseJob) error {
			if r.dashboardURL != "" {
				job.Status.ReportURL = r.getPipelingetPipelineTargetURLeTargetURL(pipelineRun)
			}

			activity, err := ConvertPipelineRun(ctx, r.logger, r.tektonclient, &pipelineRun, req.Namespace)
			if err != nil {
				return errors.Wrapf(err, "failed to convert PipelineRun")
			}
			job.Status.Activity = activity
			if err := r.client.Status().Update(ctx, job); err != nil {
				return errors.Wrapf(err, "failed to update LighthouseJob status")
			}
			return nil
		}
		err := r.retryModifyJob(ctx, req.NamespacedName, &job, f)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		r.logger.Errorf("A lighthouse job should never have more than 1 pipeline run")
	}

	return ctrl.Result{}, nil
}

func (r *LighthouseJobReconciler) getPipelingetPipelineTargetURLeTargetURL(pipelineRun pipelinev1.PipelineRun) string {
	if r.dashboardTemplate == "" {
		return fmt.Sprintf("%s/#/namespaces/%s/pipelineruns/%s", trimDashboardURL(r.dashboardURL), r.namespace, pipelineRun.Name)
	}
	funcMap := map[string]interface{}{}
	tmpl, err := template.New("value.gotmpl").Option("missingkey=error").Funcs(funcMap).Parse(r.dashboardTemplate)
	if err != nil {
		r.logger.WithError(err).Warnf("failed to parse template: %s", r.dashboardTemplate)
		return ""
	}

	labels := pipelineRun.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	namespace := pipelineRun.Namespace
	if namespace == "" {
		namespace = r.namespace
	}
	templateData := map[string]interface{}{
		"Branch":      labels[util.BranchLabel],
		"BuildNum":    labels[util.BuildNumLabel],
		"Context":     labels[util.ContextLabel],
		"Namespace":   namespace,
		"Org":         labels[util.OrgLabel],
		"PipelineRun": pipelineRun.Name,
		"Pull":        labels[util.PullLabel],
		"Repo":        labels[util.RepoLabel],
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, templateData)
	if err != nil {
		r.logger.WithError(err).Warnf("failed to parse template: %s for PipelineRun %s", r.dashboardTemplate, pipelineRun.Name)
		return ""
	}
	return fmt.Sprintf("%s/%s", trimDashboardURL(r.dashboardURL), buf.String())
}

const retryCount = 5

// retryModifyJob tries to modify the Job retrying if it fails
func (r *LighthouseJobReconciler) retryModifyJob(ctx context.Context, ns client.ObjectKey, job *lighthousev1alpha1.LighthouseJob, f func(job *lighthousev1alpha1.LighthouseJob) error) error {
	i := 0
	for {
		i++
		err := f(job)
		if err == nil {
			if i > 1 {
				r.logger.Infof("took %d attempts to update Job %s", i, job.Name)
			}
			return nil
		}
		if i >= retryCount {
			return errors.Wrapf(err, "failed to update Job %s after %d attempts", job.Name, retryCount)
		}

		if err := r.client.Get(ctx, ns, job); err != nil {
			r.logger.Warningf("Unable to get LighthouseJob %s due to: %s", job.Name, err)
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return client.IgnoreNotFound(err)
		}
	}
}

// lighthouseJobPredicateFactory returns the For(LighthouseJob) predicate set. When skipTerminatedReconciles
// is false, only ResourceVersionChangedPredicate is used (legacy behavior). When true, it also skips
// enqueue for LighthouseJobs whose activity is already in a terminal pipeline state (see Reconcile fast path).
func lighthouseJobPredicateFactory(skipTerminatedReconciles bool) predicate.Predicate {
	if skipTerminatedReconciles {
		return predicate.And(
			predicate.ResourceVersionChangedPredicate{},
			predicate.NewPredicateFuncs(func(object client.Object) bool {
				job, ok := object.(*lighthousev1alpha1.LighthouseJob)
				if !ok {
					return true
				}
				if job.Status.Activity != nil {
					return !lighthousev1alpha1.IsTerminalPipelineState(job.Status.Activity.Status)
				}
				return true
			}),
		)
	}
	return predicate.ResourceVersionChangedPredicate{}
}
