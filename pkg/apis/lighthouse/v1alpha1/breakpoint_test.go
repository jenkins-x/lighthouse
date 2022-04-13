package v1alpha1_test

import (
	"testing"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
)

func TestBreakpointResolveDebug(t *testing.T) {
	breakpoints := []*v1alpha1.LighthouseBreakpoint{
		{
			Spec: v1alpha1.LighthouseBreakpointSpec{
				Filter: v1alpha1.LighthousePipelineFilter{
					Type:       "",
					Owner:      "myorg",
					Repository: "myrepo",
					Branch:     "mybranch",
					Context:    "myctx",
					Task:       "sometask",
				},
				Debug: tektonv1beta1.TaskRunDebug{
					Breakpoint: []string{"onFailure"},
				},
			},
		},
		{
			Spec: v1alpha1.LighthouseBreakpointSpec{
				Filter: v1alpha1.LighthousePipelineFilter{
					Task: "special-task",
				},
				Debug: tektonv1beta1.TaskRunDebug{
					Breakpoint: []string{"something"},
				},
			},
		},
	}

	tests := []struct {
		name         string
		filterValues v1alpha1.LighthousePipelineFilter
		expected     *tektonv1beta1.TaskRunDebug
	}{
		{
			name: "matches-all-values",
			filterValues: v1alpha1.LighthousePipelineFilter{
				Type:       "",
				Owner:      "myorg",
				Repository: "myrepo",
				Branch:     "mybranch",
				Context:    "myctx",
				Task:       "sometask",
			},
			expected: &tektonv1beta1.TaskRunDebug{
				Breakpoint: []string{"onFailure"},
			},
		},
		{
			name: "no-debug-org",
			filterValues: v1alpha1.LighthousePipelineFilter{
				Type:       "",
				Owner:      "anotherorg",
				Repository: "myrepo",
				Branch:     "mybranch",
				Context:    "myctx",
				Task:       "sometask",
			},
		},
		{
			name: "no-debug-repo",
			filterValues: v1alpha1.LighthousePipelineFilter{
				Type:       "",
				Owner:      "myorg",
				Repository: "anotherrepo",
				Branch:     "mybranch",
				Context:    "myctx",
				Task:       "sometask",
			},
		},
		{
			name: "just-task",
			filterValues: v1alpha1.LighthousePipelineFilter{
				Type:       "",
				Owner:      "whatever",
				Repository: "whatever",
				Branch:     "whatever",
				Context:    "whatever",
				Task:       "special-task",
			},
			expected: &tektonv1beta1.TaskRunDebug{
				Breakpoint: []string{"something"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := tt.filterValues.ResolveDebug(breakpoints)

			if d := cmp.Diff(tt.expected, got); d != "" {
				t.Errorf("Generated debug for test %s did not match expected: %s", tt.name, d)
			}
		})
	}
}
