package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// Webhook defines a CRD for representing a generic webhook emitted by a git provider
type Webhook struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// +optional
	Status WebhookStatus `json:"status,omitempty"`

	Spec WebhookSpec `json:"spec,omitempty"`
}

// WebhookStatus defines a status for a Webhook CRD
type WebhookStatus struct {
	Name string
}

// WebhookList is a list of Webhook resources
type WebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Webhook `json:"items"`
}

// WebhookSpec defines a generic webhook emitted by a git provider
type WebhookSpec struct {
	EventType   string
	Ref         string
	Provider    string
	URL         string
	Payload     string
	Issue       Issue
	PullRequest PullRequest
	Commit      Commit
}

// Issue represents a single issue on a git provider's issue tracker
type Issue struct {
	Repo    string
	Org     string
	Comment string
}

// PullRequest represents a request to merge changes into a target repo
type PullRequest struct {
	Repo string
	Org  string
	Base string
	Head string
}

// Commit is a single git commit
type Commit struct {
	Repo string
	Org  string
	SHA  string
}
