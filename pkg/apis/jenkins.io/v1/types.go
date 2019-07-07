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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WebhookList is a list of Webhook resources
type WebhookList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Webhook `json:"items"`
}

// WebhookSpec defines a generic webhook emitted by a git provider
type WebhookSpec struct {
	EventType   string       `json:"eventType"`
	Ref         string       `json:"ref,omitempty"`
	Provider    string       `json:"provider"`
	URL         string       `json:"url"`
	Payload     string       `json:"payload"`
	Issue       *Issue       `json:"issue,omitempty"`
	PullRequest *PullRequest `json:"pullRequest,omitempty"`
	Commit      *Commit      `json:"commit,omitempty"`
}

// Issue represents a single issue on a git provider's issue tracker
type Issue struct {
	Repo    string `json:"repo"`
	Org     string `json:"org"`
	Comment string `json:"comment,omitempty"`
	ID      string `json:"id"`
}

// Comment represents a comment on an issue or PR
type Comment struct {
	Repo          string `json:"repo"`
	Org           string `json:"org"`
	ID            string `json:"id"`
	Commenter     string `json:"commenter"`
	Body          string `json:"body"`
	IssueID       string `json:"issueID,omitempty"`
	PullRequestID string `json:"pullRequestID,omitempty"`
}

// PullRequest represents a request to merge changes into a target repo
type PullRequest struct {
	Repo   string `json:"repo"`
	Org    string `json:"org"`
	Base   string `json:"base"`
	Head   string `json:"head"`
	Number string `json:"number"`
}

// Commit is a single git commit
type Commit struct {
	Repo string `json:"repo"`
	Org  string `json:"org"`
	SHA  string `json:"sha"`
}
