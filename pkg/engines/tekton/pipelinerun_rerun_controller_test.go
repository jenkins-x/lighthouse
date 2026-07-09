package tekton

import (
	"context"
	"testing"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	configjob "github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&lighthousev1alpha1.LighthouseJob{}).WithObjects(job, parentPR, rerunPR).Build()
	reconciler := NewRerunPipelineRunReconciler(c, c, scheme, 1)

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

func TestReconcileArchivedRerunReconstructsLighthouseJob(t *testing.T) {
	ns := "jx"
	rerunPRName := "myorg-myrepo-main-abc12-2-rerun"
	parentPRName := "myorg-myrepo-main-abc12-1"

	scheme := runtime.NewScheme()
	require.NoError(t, lighthousev1alpha1.AddToScheme(scheme))
	require.NoError(t, pipelinev1.AddToScheme(scheme))

	// The rerun PipelineRun. The parent PR does NOT exist in the fake client.
	rerunPR := &pipelinev1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rerunPRName,
			Namespace: ns,
			Labels: map[string]string{
				util.DashboardTektonRerun:        parentPRName,
				configjob.LighthouseJobTypeLabel: string(configjob.PresubmitJob),
				util.ContextLabel:                "pr-build",
				util.OrgLabel:                    "myorg",
				util.RepoLabel:                   "myrepo",
				util.LastCommitSHALabel:          "sha123",
				util.BaseSHALabel:                "basesha456",
				util.BranchLabel:                 "PR-42",
				util.PullLabel:                   "42",
				"team":                           "platform", // non-lighthouse: must be dropped by best-effort reconstruction
			},
			Annotations: map[string]string{
				util.LighthouseJobAnnotation: "myorg-myrepo-pr-build",
				"custom.company.io/note":     "keep-me", // non-canonical: must be dropped
			},
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&lighthousev1alpha1.LighthouseJob{}).WithObjects(rerunPR).Build()
	reconciler := NewRerunPipelineRunReconciler(c, c, scheme, 1)

	_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: ns, Name: rerunPRName},
	})
	require.NoError(t, err, "Reconcile should succeed for an archived rerun")

	// Verify the LighthouseJob was created
	var jobs lighthousev1alpha1.LighthouseJobList
	err = c.List(context.TODO(), &jobs, client.InNamespace(ns))
	require.NoError(t, err)
	require.Len(t, jobs.Items, 1, "Exactly one LighthouseJob should be reconstructed and created")

	job := jobs.Items[0]

	// Reconstruction is best-effort: it recovers the canonical lighthouse labels/annotations…
	assert.Contains(t, job.Annotations, util.LighthouseJobAnnotation)
	assert.Contains(t, job.Labels, configjob.LighthouseJobTypeLabel)
	// …but discards any non-canonical metadata carried on the rerun PipelineRun
	// (accepted, documented divergence with the fast/clone path).
	assert.NotContains(t, job.Labels, "team")
	assert.NotContains(t, job.Annotations, "custom.company.io/note")

	// Verify it has empty status
	assert.Empty(t, job.Status.State, "Reconstructed job should have an empty state")
	assert.Nil(t, job.Status.CompletionTime, "Reconstructed job should have no completion time")

	// Verify the rerun PipelineRun now has an OwnerReference pointing to this new job
	var updatedPR pipelinev1.PipelineRun
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: rerunPRName}, &updatedPR)
	require.NoError(t, err)
	require.Len(t, updatedPR.OwnerReferences, 1)

	ownerRef := updatedPR.OwnerReferences[0]
	assert.Equal(t, job.Name, ownerRef.Name)
	assert.Equal(t, "LighthouseJob", ownerRef.Kind)
}

// TestResolveParentLighthouseJob exercises resolveParentLighthouseJob in isolation,
// covering the three "not resolvable" branches and the nominal resolution.
func TestResolveParentLighthouseJob(t *testing.T) {
	ns := "jx"
	parentPRName := "myorg-myrepo-main-abc12-1"
	jobName := "myorg-myrepo-main-abc12"

	newReconciler := func(objs ...client.Object) *RerunPipelineRunReconciler {
		scheme := runtime.NewScheme()
		require.NoError(t, lighthousev1alpha1.AddToScheme(scheme))
		require.NoError(t, pipelinev1.AddToScheme(scheme))
		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&lighthousev1alpha1.LighthouseJob{}).WithObjects(objs...).Build()
		return NewRerunPipelineRunReconciler(c, c, scheme, 1)
	}
	rerunPR := func() *pipelinev1.PipelineRun {
		return &pipelinev1.PipelineRun{ObjectMeta: metav1.ObjectMeta{
			Name: "rerun", Namespace: ns,
			Labels: map[string]string{util.DashboardTektonRerun: parentPRName},
		}}
	}

	t.Run("no rerun label returns nil", func(t *testing.T) {
		r := newReconciler()
		pr := &pipelinev1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Name: "x", Namespace: ns}}
		lj, err := r.resolveParentLighthouseJob(context.TODO(), pr)
		require.NoError(t, err)
		assert.Nil(t, lj)
	})

	t.Run("parent PipelineRun missing returns nil", func(t *testing.T) {
		r := newReconciler()
		lj, err := r.resolveParentLighthouseJob(context.TODO(), rerunPR())
		require.NoError(t, err)
		assert.Nil(t, lj)
	})

	t.Run("parent without ownerReference returns nil", func(t *testing.T) {
		parentPR := &pipelinev1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Name: parentPRName, Namespace: ns}}
		r := newReconciler(parentPR)
		lj, err := r.resolveParentLighthouseJob(context.TODO(), rerunPR())
		require.NoError(t, err)
		assert.Nil(t, lj)
	})

	t.Run("parent with non-LighthouseJob controller owner returns nil", func(t *testing.T) {
		parentPR := &pipelinev1.PipelineRun{ObjectMeta: metav1.ObjectMeta{
			Name: parentPRName, Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "tekton.dev/v1",
				Kind:       "Pipeline",
				Name:       "some-pipeline",
				Controller: ptr.To(true),
			}},
		}}
		r := newReconciler(parentPR)
		lj, err := r.resolveParentLighthouseJob(context.TODO(), rerunPR())
		require.NoError(t, err)
		assert.Nil(t, lj, "a non-LighthouseJob controller owner must trigger reconstruction, not a wrong-kind lookup")
	})

	t.Run("resolves the parent LighthouseJob", func(t *testing.T) {
		job := &lighthousev1alpha1.LighthouseJob{ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: ns}}
		parentPR := &pipelinev1.PipelineRun{ObjectMeta: metav1.ObjectMeta{
			Name: parentPRName, Namespace: ns,
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: "lighthouse.jenkins.io/v1alpha1",
				Kind:       "LighthouseJob",
				Name:       jobName,
				Controller: ptr.To(true),
			}},
		}}
		r := newReconciler(job, parentPR)
		lj, err := r.resolveParentLighthouseJob(context.TODO(), rerunPR())
		require.NoError(t, err)
		require.NotNil(t, lj)
		assert.Equal(t, jobName, lj.Name)
	})
}

func TestReconcileMissingRequiredLabelsDrops(t *testing.T) {
	ns := "jx"
	rerunPRName := "myorg-myrepo-main-abc12-2-rerun"
	parentPRName := "myorg-myrepo-main-abc12-1"

	scheme := runtime.NewScheme()
	require.NoError(t, lighthousev1alpha1.AddToScheme(scheme))
	require.NoError(t, pipelinev1.AddToScheme(scheme))

	// The rerun PipelineRun is missing util.ContextLabel and others
	rerunPR := &pipelinev1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rerunPRName,
			Namespace: ns,
			Labels: map[string]string{
				util.DashboardTektonRerun: parentPRName,
			},
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&lighthousev1alpha1.LighthouseJob{}).WithObjects(rerunPR).Build()
	reconciler := NewRerunPipelineRunReconciler(c, c, scheme, 1)

	res, err := reconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: ns, Name: rerunPRName},
	})

	// Must NOT return an error (which would cause a requeue), must just drop it.
	require.NoError(t, err, "Missing required labels is an unrecoverable error and should be dropped without returning an error to the controller")
	assert.Equal(t, ctrl.Result{}, res)

	// Verify NO LighthouseJob was created
	var jobs lighthousev1alpha1.LighthouseJobList
	err = c.List(context.TODO(), &jobs, client.InNamespace(ns))
	require.NoError(t, err)
	assert.Len(t, jobs.Items, 0, "No LighthouseJob should be created when required labels are missing")
}

// TestReconcileRerunIsIdempotent guards against the duplicate-LighthouseJob regression:
// reconciling the same rerun PipelineRun twice must create exactly one job, and StartTime
// must be set so lighthouse-gc-jobs does not GC the still-running job.
func TestReconcileRerunIsIdempotent(t *testing.T) {
	ns := "jx"
	rerunPRName := "myorg-myrepo-main-abc12-2-rerun"

	scheme := runtime.NewScheme()
	require.NoError(t, lighthousev1alpha1.AddToScheme(scheme))
	require.NoError(t, pipelinev1.AddToScheme(scheme))

	rerunPR := &pipelinev1.PipelineRun{ObjectMeta: metav1.ObjectMeta{
		Name: rerunPRName, Namespace: ns,
		Labels: map[string]string{
			util.DashboardTektonRerun:        "myorg-myrepo-main-abc12-1",
			configjob.LighthouseJobTypeLabel: string(configjob.PresubmitJob),
			util.ContextLabel:                "pr-build",
			util.OrgLabel:                    "myorg",
			util.RepoLabel:                   "myrepo",
			util.LastCommitSHALabel:          "sha123",
		},
		Annotations: map[string]string{util.LighthouseJobAnnotation: "myorg-myrepo-pr-build"},
	}}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&lighthousev1alpha1.LighthouseJob{}).
		WithObjects(rerunPR).
		Build()
	reconciler := NewRerunPipelineRunReconciler(c, c, scheme, 1)
	req := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: ns, Name: rerunPRName}}

	_, err := reconciler.Reconcile(context.TODO(), req)
	require.NoError(t, err)

	var jobs lighthousev1alpha1.LighthouseJobList
	require.NoError(t, c.List(context.TODO(), &jobs, client.InNamespace(ns)))
	require.Len(t, jobs.Items, 1)
	assert.False(t, jobs.Items[0].Status.StartTime.IsZero(), "StartTime should be set on creation")

	// A second reconcile (e.g. a re-queued event) must not create a duplicate.
	_, err = reconciler.Reconcile(context.TODO(), req)
	require.NoError(t, err)
	require.NoError(t, c.List(context.TODO(), &jobs, client.InNamespace(ns)))
	assert.Len(t, jobs.Items, 1, "reconciling twice must not create a duplicate LighthouseJob")

	var updated pipelinev1.PipelineRun
	require.NoError(t, c.Get(context.TODO(), req.NamespacedName, &updated))
	assert.Len(t, updated.OwnerReferences, 1)
}

// TestReconcileAdoptsExistingLighthouseJob exercises the AlreadyExists/apiReader adoption
// path: a job with the deterministic name already exists (prior reconcile whose owner
// reference write is not yet visible) and must be adopted, not duplicated.
func TestReconcileAdoptsExistingLighthouseJob(t *testing.T) {
	ns := "jx"
	rerunPRName := "myorg-myrepo-main-abc12-2-rerun"

	scheme := runtime.NewScheme()
	require.NoError(t, lighthousev1alpha1.AddToScheme(scheme))
	require.NoError(t, pipelinev1.AddToScheme(scheme))

	existing := &lighthousev1alpha1.LighthouseJob{ObjectMeta: metav1.ObjectMeta{Name: rerunPRName, Namespace: ns}}
	rerunPR := &pipelinev1.PipelineRun{ObjectMeta: metav1.ObjectMeta{
		Name: rerunPRName, Namespace: ns,
		Labels: map[string]string{
			util.DashboardTektonRerun:        "myorg-myrepo-main-abc12-1",
			configjob.LighthouseJobTypeLabel: string(configjob.PresubmitJob),
			util.ContextLabel:                "pr-build",
			util.OrgLabel:                    "myorg",
			util.RepoLabel:                   "myrepo",
			util.LastCommitSHALabel:          "sha123",
		},
		Annotations: map[string]string{util.LighthouseJobAnnotation: "myorg-myrepo-pr-build"},
	}}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&lighthousev1alpha1.LighthouseJob{}).
		WithObjects(existing, rerunPR).
		Build()
	reconciler := NewRerunPipelineRunReconciler(c, c, scheme, 1)

	_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: ns, Name: rerunPRName},
	})
	require.NoError(t, err)

	var jobs lighthousev1alpha1.LighthouseJobList
	require.NoError(t, c.List(context.TODO(), &jobs, client.InNamespace(ns)))
	assert.Len(t, jobs.Items, 1, "the existing job must be adopted, not duplicated")

	var updated pipelinev1.PipelineRun
	require.NoError(t, c.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: rerunPRName}, &updated))
	require.Len(t, updated.OwnerReferences, 1)
	assert.Equal(t, rerunPRName, updated.OwnerReferences[0].Name)
}

func TestReconcileLiveRerunClearsStaleStatus(t *testing.T) {
	ns := "jx"
	jobName := "myorg-myrepo-main-abc12"
	parentPRName := "myorg-myrepo-main-abc12-1"
	rerunPRName := "myorg-myrepo-main-abc12-2-rerun"

	scheme := runtime.NewScheme()
	require.NoError(t, lighthousev1alpha1.AddToScheme(scheme))
	require.NoError(t, pipelinev1.AddToScheme(scheme))

	// The original LighthouseJob with a terminal status
	now := metav1.Now()
	originalJob := &lighthousev1alpha1.LighthouseJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: ns,
			Labels: map[string]string{
				"team":                             "platform", // custom, non-lighthouse
				"lighthouse.jenkins-x.io/type":     string(configjob.PresubmitJob),
				configjob.CreatedByLighthouseLabel: "true",
			},
			Annotations: map[string]string{
				"custom.company.io/note": "keep-me", // custom annotation
			},
		},
		Status: lighthousev1alpha1.LighthouseJobStatus{
			State:          lighthousev1alpha1.SuccessState,
			CompletionTime: &now,
		},
	}

	// The parent PR, owned by the original job
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

	// The rerun PR
	rerunPR := &pipelinev1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rerunPRName,
			Namespace: ns,
			Labels: map[string]string{
				util.DashboardTektonRerun:          parentPRName,
				"dashboard.tekton.dev/random":      "discard",
				"lighthouse.jenkins-x.io/type":     string(configjob.PresubmitJob),
				configjob.CreatedByLighthouseLabel: "true",
			},
			Annotations: map[string]string{
				util.LighthouseJobAnnotation: "myorg-myrepo-pr-build",
				"some-other-annotation":      "discard",
			},
		},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&lighthousev1alpha1.LighthouseJob{}).WithObjects(originalJob, parentPR, rerunPR).Build()
	reconciler := NewRerunPipelineRunReconciler(c, c, scheme, 1)

	_, err := reconciler.Reconcile(context.TODO(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: ns, Name: rerunPRName},
	})
	require.NoError(t, err, "Reconcile should succeed for a live rerun")

	// Find the newly cloned job
	var updatedPR pipelinev1.PipelineRun
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: rerunPRName}, &updatedPR)
	require.NoError(t, err)
	require.Len(t, updatedPR.OwnerReferences, 1)

	clonedJobName := updatedPR.OwnerReferences[0].Name
	assert.Equal(t, rerunPRName, clonedJobName, "The new job should have the same name as the rerun PipelineRun")

	var clonedJob lighthousev1alpha1.LighthouseJob
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: clonedJobName}, &clonedJob)
	require.NoError(t, err)

	// Assert the cloned job's status was cleared
	assert.Empty(t, clonedJob.Status.State, "Cloned job should have an empty state")
	assert.Nil(t, clonedJob.Status.CompletionTime, "Cloned job should have a nil completion time")

	// Fast path = full-fidelity clone + guaranteed canonical entries (merge, no removal).
	// 1) Parent-owned custom metadata is PRESERVED:
	assert.Equal(t, "platform", clonedJob.Labels["team"], "custom parent label must be preserved on the clone")
	assert.Equal(t, "keep-me", clonedJob.Annotations["custom.company.io/note"], "custom parent annotation must be preserved")
	// 2) Canonical lighthouse metadata is present (either inherited or filled from the PR):
	assert.Contains(t, clonedJob.Labels, "lighthouse.jenkins-x.io/type")
	assert.Contains(t, clonedJob.Labels, configjob.CreatedByLighthouseLabel)
	assert.Contains(t, clonedJob.Annotations, util.LighthouseJobAnnotation)
	// 3) Non-canonical labels/annotations that live only on the rerun PR are NOT injected:
	assert.NotContains(t, clonedJob.Labels, util.DashboardTektonRerun)
	assert.NotContains(t, clonedJob.Labels, "dashboard.tekton.dev/random")
	assert.NotContains(t, clonedJob.Annotations, "some-other-annotation")

	// Verify the original job was NOT modified (deep copy protected it)
	var fetchedOriginalJob lighthousev1alpha1.LighthouseJob
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: ns, Name: jobName}, &fetchedOriginalJob)
	require.NoError(t, err)
	assert.Equal(t, lighthousev1alpha1.SuccessState, fetchedOriginalJob.Status.State, "Original job's status should not be mutated")
}
