package v1alpha1

import (
	"fmt"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"regexp"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=lhpsp

// LighthousePipelineSecurityPolicy represents optionally defined restrictions that are required to be applied on a Pipeline before spawning it
type LighthousePipelineSecurityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SecurityPolicySpec `json:"spec"`
}

func (in *LighthousePipelineSecurityPolicy) IsEnforcingMaximumPipelineDuration() bool {
	if in.Spec.Enforce.MaximumPipelineDuration == nil {
		return false
	}
	return in.Spec.Enforce.MaximumPipelineDuration.Duration.String() != "0s"
}

func (in *LighthousePipelineSecurityPolicy) IsEnforcingServiceAccount() bool {
	return in.Spec.Enforce.ServiceAccountName != ""
}

func (in *LighthousePipelineSecurityPolicy) IsEnforcingNamespace() bool {
	return in.Spec.Enforce.Namespace != ""
}

func (in *LighthousePipelineSecurityPolicy) GetMaximumDurationForPipeline(pipelineDuration *metav1.Duration) *metav1.Duration {
	// when pipeline duration is longer than allowed, then set a maximum allowed
	if in.IsEnforcingMaximumPipelineDuration() && pipelineDuration.Duration > in.Spec.Enforce.MaximumPipelineDuration.Duration {
		return in.Spec.Enforce.MaximumPipelineDuration
	}
	return pipelineDuration
}

// SecurityPolicySpec defines a policy specification
type SecurityPolicySpec struct {
	RepositoryPattern string                        `json:"repositoryPattern"`
	Enforce           SecurityPolicyEnforcementSpec `json:"enforce"`
}

func (in *SecurityPolicySpec) IsRepositoryMatchingPattern(repository string) (bool, error) {
	pattern, err := regexp.Compile(in.RepositoryPattern)
	if err != nil {
		return false, errors.Wrapf(err, fmt.Sprintf("cannot compile regexp of `kind: LighthousePipelineSecurityPolicy`, pattern: %v", in.RepositoryPattern))
	}
	return pattern.MatchString(repository), nil
}

// SecurityPolicyEnforcementSpec specifies actual restrictions that will be applied, for example: enforced namespace or enforced usage of specific ServiceAccount
type SecurityPolicyEnforcementSpec struct {
	Namespace               string           `json:"namespace,omitempty" protobuf:"bytes,1,opt,name=namespace"`
	ServiceAccountName      string           `json:"serviceAccountName,omitempty" protobuf:"bytes,2,opt,name=serviceAccountName"`
	MaximumPipelineDuration *metav1.Duration `json:"maximumPipelineDuration,omitempty" protobuf:"bytes,3,opt,name=maximumPipelineDuration"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true

// LighthousePipelineSecurityPolicyList represents a list of Pipeline Security Policies
type LighthousePipelineSecurityPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LighthousePipelineSecurityPolicy `json:"items"`
}
