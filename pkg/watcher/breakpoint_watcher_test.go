package watcher

import (
	"testing"

	lighthousev1alpha1 "github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	fakelh "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/require"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewBreakpointWatcher(t *testing.T) {
	ns := "jx"
	lhClient := fakelh.NewSimpleClientset(
		&lighthousev1alpha1.LighthouseBreakpoint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-bp",
				Namespace: ns,
			},
			Spec: lighthousev1alpha1.LighthouseBreakpointSpec{
				Filter: lighthousev1alpha1.LighthousePipelineFilter{
					Owner:      "jenkins-x",
					Repository: "lighthouse",
					Branch:     "master",
					Context:    "github",
					Task:       "",
				},
				Debug: tektonv1beta1.TaskRunDebug{
					Breakpoint: []string{"onFailure"},
				},
			},
		},
	)

	w, err := NewBreakpointWatcher(lhClient, ns, nil)
	require.NoError(t, err, "failed to create BreakpointWatcher")
	require.NotNilf(t, w, "no BreakpointWatcher")

	defer w.Stop()

	bps := w.GetBreakpoints()
	require.Len(t, bps, 1, "for breakpoints %#v", bps)
}
