package repoconfig

// this file contains copies of structs from the plugins package
// using different JSON/YAML marshalling so that they look like
// kubernetes CRDs; using camelCase rather than lower case and underscores

// Approve specifies a configuration for a single approve.
//
// The configuration for the approve plugin is defined as a list of these structures.
type Approve struct {
	// IssueRequired indicates if an associated issue is required for approval in
	// the specified repos.
	IssueRequired bool `json:"issueRequired,omitempty"`

	// RequireSelfApproval requires PR authors to explicitly approve their PRs.
	// Otherwise the plugin assumes the author of the PR approves the changes in the PR.
	RequireSelfApproval *bool `json:"requireSelfApproval,omitempty"`

	// LgtmActsAsApprove indicates that the lgtm command should be used to
	// indicate approval
	LgtmActsAsApprove bool `json:"lgtmActsAsApprove,omitempty"`

	// IgnoreReviewState causes the approve plugin to ignore the GitHub review state. Otherwise:
	// * an APPROVE github review is equivalent to leaving an "/approve" message.
	// * A REQUEST_CHANGES github review is equivalent to leaving an /approve cancel" message.
	IgnoreReviewState *bool `json:"ignoreReviewState,omitempty"`
}
