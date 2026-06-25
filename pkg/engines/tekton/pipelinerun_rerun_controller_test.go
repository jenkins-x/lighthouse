package tekton

import (
	"context"
	"testing"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestRerunReconcileSetsOwnerReferenceGVK guards against a regression where the rerun
// controller copied APIVersion/Kind from a LighthouseJob fetched via client.Get. Typed
// clients return objects with an empty TypeMeta, so the resulting OwnerReference would
// have an empty apiVersion/kind (rejected by the API server). The GVK must instead be
// derived from the scheme.
func TestRerunReconcileSetsOwnerReferenceGVK(t *testing.T) {
	ns := "jx"
	jobName := "myorg-myrepo-main-abc12"
	parentPRName := "myorg-myrepo-main-abc12-1"
	rerunPRName := "myorg-myrepo-main-abc12-2-rerun"

	scheme := runtime.NewScheme()
	require.NoError(t, lighthousev1alpha1.AddToScheme(scheme))
	require.NoError(t, pipelinev1.AddToScheme(scheme))

	// the LighthouseJob owning the parent PipelineRun
	job := &lighthousev1alpha1.LighthouseJob{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: ns},
	}

	// parent PipelineRun, owned by the LighthouseJob
	parentPR := &pipelinev1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      parentPRName,
			Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "lighthouse.jenkins.io/v1alpha1",
				Kind:       "LighthouseJob",
				Name:       jobName,
				Controller: ptr.To(true),
			}},
		},
	}

	// the rerun PipelineRun: has the rerun label pointing at the parent, no owner refs yet
	rerunPR := &pipelinev1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rerunPRName,
			Namespace: ns,
			Labels:    map[string]string{util.DashboardTektonRerun: parentPRName},
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(job, parentPR, rerunPR).Build()
	reconciler := NewRerunPipelineRunReconciler(c, scheme, 1)

	_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: ns, Name: rerunPRName},
	})
	require.NoError(t, err)

	// the rerun PipelineRun should now carry a valid OwnerReference to a new LighthouseJob
	var updated pipelinev1.PipelineRun
	require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: rerunPRName}, &updated))
	require.Len(t, updated.OwnerReferences, 1, "expected one owner reference on the rerun PipelineRun")

	ref := updated.OwnerReferences[0]
	assert.Equal(t, "lighthouse.jenkins.io/v1alpha1", ref.APIVersion, "owner reference apiVersion must be derived from the scheme, not the fetched object's empty TypeMeta")
	assert.Equal(t, "LighthouseJob", ref.Kind, "owner reference kind must be derived from the scheme, not the fetched object's empty TypeMeta")
	assert.NotEmpty(t, ref.Name, "owner reference should point at the newly created rerun LighthouseJob")
}
