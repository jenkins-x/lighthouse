package tekton

import (
	"context"
	"fmt"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// RerunPipelineRunReconciler reconciles PipelineRun objects with the rerun label.
//
// It operates along two paths:
//  1. Fast path: if the parent PipelineRun and LighthouseJob exist, it clones the parent LighthouseJob
//     (clearing its Status to prevent carrying over a stale terminal state) and assigns it.
//  2. Reconstruction path: if either parent resource has been archived/deleted, it reconstructs
//     a new LighthouseJob solely from the metadata (labels, annotations) on the rerun PipelineRun itself.
type RerunPipelineRunReconciler struct {
	client                  client.Client
	apiReader               client.Reader
	logger                  *logrus.Entry
	scheme                  *runtime.Scheme
	maxConcurrentReconciles int
}

// NewRerunPipelineRunReconciler creates a new RerunPipelineRunReconciler
func NewRerunPipelineRunReconciler(client client.Client, apiReader client.Reader, scheme *runtime.Scheme, maxConcurrentReconciles int) *RerunPipelineRunReconciler {
	return &RerunPipelineRunReconciler{
		client:                  client,
		apiReader:               apiReader,
		logger:                  logrus.NewEntry(logrus.StandardLogger()).WithField("controller", "RerunPipelineRunController"),
		scheme:                  scheme,
		maxConcurrentReconciles: maxConcurrentReconciles,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RerunPipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctrlr := ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineRun{}, builder.WithPredicates(
			predicate.NewPredicateFuncs(func(object client.Object) bool {
				labels := object.GetLabels()
				_, hasRerunLabel := labels[util.DashboardTektonRerun]
				// Only reconcile rerun PipelineRuns that have not yet been
				// linked to a LighthouseJob via an OwnerReference.
				return hasRerunLabel && len(object.GetOwnerReferences()) == 0
			}),
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

// Reconcile handles the reconciliation logic for rerun PipelineRuns
func (r *RerunPipelineRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger.Infof("Reconciling rerun PipelineRun %s", req.NamespacedName)

	// Fetch the Rerun PipelineRun instance
	var rerunPipelineRun pipelinev1.PipelineRun
	if err := r.client.Get(ctx, req.NamespacedName, &rerunPipelineRun); err != nil {
		r.logger.Errorf("Failed to get rerun PipelineRun: %s", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the Rerun PipelineRun already has an ownerReference set
	// (safety net — the predicate should already filter these out)
	if len(rerunPipelineRun.OwnerReferences) > 0 {
		r.logger.Infof("PipelineRun %s already has an ownerReference set, skipping.", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// Resolve the parent LighthouseJob
	parentPipelineRunParentLighthouseJob, err := r.resolveParentLighthouseJob(ctx, &rerunPipelineRun)
	if err != nil {
		return ctrl.Result{}, err
	}

	var rerunLhJob *lighthousev1alpha1.LighthouseJob

	if parentPipelineRunParentLighthouseJob == nil {
		r.logger.Infof("Parent PipelineRun or LighthouseJob not found. Attempting reconstruction from rerun PipelineRun %s", req.NamespacedName)
		spec, err := rerunSpecFromPipelineRun(&rerunPipelineRun)
		if err != nil {
			// Missing required labels is a permanent condition.
			// Drop it instead of returning an error.
			var missingLabelsErr *ErrMissingRequiredLabels
			if errors.As(err, &missingLabelsErr) {
				r.logger.Warnf("Cannot reconstruct LighthouseJob for rerun PipelineRun %s, dropping: %v", req.NamespacedName, err)
				return ctrl.Result{}, nil
			}
			// Defensive catch-all for any other unexpected reconstruction errors.
			r.logger.Errorf("Failed to reconstruct LighthouseJobSpec: %v", err)
			return ctrl.Result{}, err
		}
		rerunLhJob = newReconstructedLighthouseJob(&rerunPipelineRun, spec)
	} else {
		r.logger.Infof("Parent LighthouseJob found. Cloning LighthouseJob %s", parentPipelineRunParentLighthouseJob.Name)
		// Clone the LighthouseJob
		rerunLhJob = parentPipelineRunParentLighthouseJob.DeepCopy()

		// Clear the status so a completed parent doesn't immediately mark the rerun job as terminal
		rerunLhJob.Status = lighthousev1alpha1.LighthouseJobStatus{}

		rerunLhJob.Name = rerunPipelineRun.Name

		rerunLhJob.ResourceVersion = ""
		rerunLhJob.UID = ""
	}

	// Create the new LighthouseJob
	if err := r.client.Create(ctx, rerunLhJob); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// A previous reconciliation of this same rerun PipelineRun already created
			// the job (the name is deterministic). Adopt it. We read it through the
			// uncached apiReader on purpose: the object was just created and may not be
			// visible in the cache yet, which is exactly the race this branch handles.
			r.logger.Warnf("LighthouseJob %s already exists, adopting it to set ownerReference", rerunLhJob.Name)
			if err := r.apiReader.Get(ctx, types.NamespacedName{Namespace: rerunLhJob.Namespace, Name: rerunLhJob.Name}, rerunLhJob); err != nil {
				r.logger.Errorf("Failed to get existing LighthouseJob %s: %s", rerunLhJob.Name, err)
				return ctrl.Result{}, err
			}
		} else {
			r.logger.Errorf("Failed to create new LighthouseJob: %s", err)
			return ctrl.Result{}, err
		}
	}

	if rerunLhJob.Status.StartTime.IsZero() {
		rerunLhJob.Status.StartTime = metav1.Now()
		if err := r.client.Status().Update(ctx, rerunLhJob); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "failed to set LighthouseJob StartTime")
		}
	}

	// Prepare the ownerReference.
	// client.Get does not return TypeMeta from LighthouseJob, so
	// derive the GVK from the scheme instead.
	gvk, err := apiutil.GVKForObject(rerunLhJob, r.scheme)
	if err != nil {
		r.logger.Errorf("Failed to determine GVK for LighthouseJob: %v", err)
		return ctrl.Result{}, err
	}
	ownerReference := metav1.OwnerReference{
		APIVersion: gvk.GroupVersion().String(),
		Kind:       gvk.Kind,
		Name:       rerunLhJob.Name,
		UID:        rerunLhJob.UID,
		Controller: ptr.To(true),
	}

	// updates ownerReference of the PipelineRun re-run.
	f := func(job *pipelinev1.PipelineRun) error {
		// Check if ownerReference already exists to be idempotent
		exists := false
		for _, ref := range job.OwnerReferences {
			if ref.UID == ownerReference.UID {
				exists = true
				break
			}
		}
		if !exists {
			job.OwnerReferences = append(job.OwnerReferences, ownerReference)
			// Patch the PipelineRun with the new ownerReference
			if err := r.client.Update(ctx, job); err != nil {
				return errors.Wrapf(err, "failed to update PipelineRun with ownerReference")
			}
		}
		return nil
	}
	err = r.retryModifyPipelineRun(ctx, req.NamespacedName, &rerunPipelineRun, f)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.logger.Infof("Successfully patched PipelineRun %s with new ownerReference to LighthouseJob %s", req.NamespacedName, rerunLhJob.Name)

	return ctrl.Result{}, nil
}

// retryModifyPipelineRun tries to modify the PipelineRun, retrying if it fails
func (r *RerunPipelineRunReconciler) retryModifyPipelineRun(ctx context.Context, ns client.ObjectKey, pipelineRun *pipelinev1.PipelineRun, f func(pipelineRun *pipelinev1.PipelineRun) error) error {
	const retryCount = 5

	i := 0
	for {
		i++
		err := f(pipelineRun)
		if err == nil {
			if i > 1 {
				r.logger.Infof("Took %d attempts to update PipelineRun %s", i, pipelineRun.Name)
			}
			return nil
		}
		if i >= retryCount {
			return fmt.Errorf("failed to update PipelineRun %s after %d attempts: %w", pipelineRun.Name, retryCount, err)
		}

		if err := r.client.Get(ctx, ns, pipelineRun); err != nil {
			r.logger.Warningf("Unable to get PipelineRun %s due to: %s", pipelineRun.Name, err)
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return client.IgnoreNotFound(err)
		}
	}
}

func (r *RerunPipelineRunReconciler) resolveParentLighthouseJob(ctx context.Context, rerunPipelineRun *pipelinev1.PipelineRun) (*lighthousev1alpha1.LighthouseJob, error) {
	// Extract Rerun PipelineRun Parent Name
	rerunPipelineRunParentName, ok := rerunPipelineRun.Labels[util.DashboardTektonRerun]
	if !ok {
		return nil, nil
	}

	// Get Rerun PipelineRun parent PipelineRun
	var rerunPipelineRunParent pipelinev1.PipelineRun
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: rerunPipelineRun.Namespace, Name: rerunPipelineRunParentName}, &rerunPipelineRunParent); err != nil {
		r.logger.Warningf("Unable to get Rerun Parent PipelineRun %s: %v", rerunPipelineRunParentName, err)
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return nil, client.IgnoreNotFound(err)
	}

	// Check if the Rerun Parent PipelineRun doesn't already have an ownerReference set
	if len(rerunPipelineRunParent.OwnerReferences) == 0 {
		r.logger.Infof("Parent PipelineRun %s doesn't already have an ownerReference set, skipping.", rerunPipelineRunParentName)
		return nil, nil
	}

	// get rerun pipelinerun parent lighthousejob
	var parentPipelineRunParentLighthouseJob lighthousev1alpha1.LighthouseJob
	parentPipelineRunRef := rerunPipelineRunParent.OwnerReferences[0]
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: rerunPipelineRun.Namespace, Name: parentPipelineRunRef.Name}, &parentPipelineRunParentLighthouseJob); err != nil {
		r.logger.Warningf("Unable to get Rerun Parent PipelineRun Parent LighthouseJob %s: %v", parentPipelineRunRef.Name, err)
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return nil, client.IgnoreNotFound(err)
	}

	return &parentPipelineRunParentLighthouseJob, nil
}
