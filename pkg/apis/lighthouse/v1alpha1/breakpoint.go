package v1alpha1

import (
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=lhbp

// LighthouseBreakpoint defines the debug breakpoints for Pipelines
type LighthouseBreakpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec LighthouseBreakpointSpec `json:"spec,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// LighthouseBreakpointList represents a list of breakpoint options
type LighthouseBreakpointList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LighthouseBreakpoint `json:"items"`
}

// LighthouseBreakpointSpec the spec of a breakpoint request
type LighthouseBreakpointSpec struct {
	// Filter filters which kinds of pipeilnes to apply this debug to
	Filter LighthousePipelineFilter `json:"filter,omitempty"`

	// Debug the debug configuration to apply
	Debug pipelinev1.TaskRunDebug `json:"debug,omitempty"`
}

// LighthousePipelineFilter defines the filter to use to apply breakpoints to new breakpoints
type LighthousePipelineFilter struct {
	// Type is the type of job and informs how
	// the jobs is triggered
	Type job.PipelineKind `json:"type,omitempty"`
	// Owner is the git organisation or user that owns the repository
	Owner string `json:"owner,omitempty"`
	// Repository the name of the git repository within the owner
	Repository string `json:"repository,omitempty"`
	// Branch the name of the branch
	Branch string `json:"branch,omitempty"`
	// Context the name of the context
	Context string `json:"context,omitempty"`
	// Task the name of the task
	Task string `json:"task,omitempty"`
}

// Matches returns true if this filter matches the given object data
func (f *LighthousePipelineFilter) Matches(o *LighthousePipelineFilter) bool {
	if string(f.Type) != "" {
		if f.Type != o.Type {
			return false
		}
	}
	if f.Owner != "" && f.Owner != o.Owner {
		return false
	}
	if f.Repository != "" && f.Repository != o.Repository {
		return false
	}
	if f.Branch != "" && f.Branch != o.Branch {
		return false
	}
	if f.Context != "" && f.Context != o.Context {
		return false
	}
	if f.Task != "" && f.Task != o.Task {
		return false
	}
	return true
}

// ResolveDebug resolves the debug breakpoint
func (f *LighthousePipelineFilter) ResolveDebug(breakpoints []*LighthouseBreakpoint) *pipelinev1.TaskRunDebug {
	// lets match the first breakpoint
	for _, bp := range breakpoints {
		if bp.Spec.Filter.Matches(f) {
			return &bp.Spec.Debug
		}
	}
	return nil
}
