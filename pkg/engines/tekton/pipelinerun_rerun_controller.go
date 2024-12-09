package tekton

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// RerunPipelineRunReconciler reconciles PipelineRun objects with the rerun label
type RerunPipelineRunReconciler struct {
	client client.Client
	logger *logrus.Entry
	scheme *runtime.Scheme
}

// NewRerunPipelineRunReconciler creates a new RerunPipelineRunReconciler
func NewRerunPipelineRunReconciler(client client.Client, scheme *runtime.Scheme) *RerunPipelineRunReconciler {
	return &RerunPipelineRunReconciler{
		client: client,
		logger: logrus.NewEntry(logrus.StandardLogger()).WithField("controller", "RerunPipelineRunController"),
		scheme: scheme,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *RerunPipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pipelinev1.PipelineRun{}).
		WithEventFilter(predicate.NewPredicateFuncs(func(object client.Object) bool {
			labels := object.GetLabels()
			_, exists := labels[util.DashboardTektonRerun]
			return exists
		})).
		Complete(r)
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
	if len(rerunPipelineRun.OwnerReferences) > 0 {
		r.logger.Infof("PipelineRun %s already has an ownerReference set, skipping.", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// Extract Rerun PipelineRun Parent Name
	rerunPipelineRunParentName, ok := rerunPipelineRun.Labels[util.DashboardTektonRerun]
	if !ok {
		return ctrl.Result{}, nil
	}

	// Get Rerun PipelineRun parent PipelineRun
	var rerunPipelineRunParent pipelinev1.PipelineRun
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: rerunPipelineRunParentName}, &rerunPipelineRunParent); err != nil {
		r.logger.Warningf("Unable to get Rerun Parent PipelineRun %s: %v", rerunPipelineRunParentName, err)
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the Rerun Parent PipelineRun doesn't already have an ownerReference set
	if len(rerunPipelineRunParent.OwnerReferences) == 0 {
		r.logger.Infof("Parent PipelineRun %s doesn't already have an ownerReference set, skipping.", rerunPipelineRunParentName)
		return ctrl.Result{}, nil
	}

	// get rerun pipelinerun parent pipelinerun parent lighthousejob
	var parentPipelineRunParentLighthouseJob lighthousev1alpha1.LighthouseJob
	parentPipelineRunRef := rerunPipelineRunParent.OwnerReferences[0]
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: parentPipelineRunRef.Name}, &parentPipelineRunParentLighthouseJob); err != nil {
		r.logger.Warningf("Unable to get Rerun Parent PipelineRun Parent LighthouseJob %s: %v", parentPipelineRunRef.Name, err)
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Clone the LighthouseJob
	rerunLhJob := parentPipelineRunParentLighthouseJob.DeepCopy()
	rerunLhJob.APIVersion = parentPipelineRunParentLighthouseJob.APIVersion
	rerunLhJob.Kind = parentPipelineRunParentLighthouseJob.Kind

	// Trim existing r-xxxxx suffix and append a new one
	re := regexp.MustCompile(`-r-[a-f0-9]{5}$`)
	baseName := re.ReplaceAllString(parentPipelineRunParentLighthouseJob.Name, "")
	rerunLhJob.Name = fmt.Sprintf("%s-%s", baseName, fmt.Sprintf("r-%s", uuid.NewString()[:5]))

	rerunLhJob.ResourceVersion = ""
	rerunLhJob.UID = ""

	// Create the new LighthouseJob
	if err := r.client.Create(ctx, rerunLhJob); err != nil {
		r.logger.Errorf("Failed to create new LighthouseJob: %s", err)
		return ctrl.Result{}, err
	}

	// Prepare the ownerReference
	ownerReference := metav1.OwnerReference{
		APIVersion: parentPipelineRunParentLighthouseJob.APIVersion,
		Kind:       parentPipelineRunParentLighthouseJob.Kind,
		Name:       rerunLhJob.Name,
		UID:        rerunLhJob.UID,
		Controller: ptr.To(true),
	}

	// Set the ownerReference on the PipelineRun
	rerunPipelineRun.OwnerReferences = append(rerunPipelineRun.OwnerReferences, ownerReference)

	// update ownerReference of rerun PipelineRun
	f := func(job *pipelinev1.PipelineRun) error {
		// Patch the PipelineRun with the new ownerReference
		if err := r.client.Update(ctx, &rerunPipelineRun); err != nil {
			return errors.Wrapf(err, "failed to update PipelineRun with ownerReference")
		}
		return nil
	}
	err := r.retryModifyPipelineRun(ctx, req.NamespacedName, &rerunPipelineRun, f)
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
