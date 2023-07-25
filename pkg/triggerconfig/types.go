package triggerconfig

import (
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config represents local trigger configurations which can be merged into the configuration
type Config struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the Config from the client
	// +optional
	Spec ConfigSpec `json:"spec"`
}

// ConfigSpec specifies the optional presubmit/postsubmit/trigger configurations
type ConfigSpec struct {
	// Presubmit zero or more presubmits
	Presubmits []job.Presubmit `json:"presubmits,omitempty"`

	// Postsubmit zero or more postsubmits
	Postsubmits []job.Postsubmit `json:"postsubmits,omitempty"`

	Periodics []job.Periodic `json:"periodics,omitempty"`

	Deployments []job.Deployment `json:"deployments,omitempty"`
}

// ConfigList contains a list of Config
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}
