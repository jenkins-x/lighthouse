/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package branchprotection

// Policy for the config/org/repo/branch.
// When merging policies, a nil value results in inheriting the parent policy.
type Policy struct {
	// Protect overrides whether branch protection is enabled if set.
	Protect *bool `json:"protect,omitempty"`
	// RequiredStatusChecks configures github contexts
	RequiredStatusChecks *ContextPolicy `json:"required_status_checks,omitempty"`
	// Admins overrides whether protections apply to admins if set.
	Admins *bool `json:"enforce_admins,omitempty"`
	// Restrictions limits who can merge
	Restrictions *Restrictions `json:"restrictions,omitempty"`
	// RequiredPullRequestReviews specifies github approval/review criteria.
	RequiredPullRequestReviews *ReviewPolicy `json:"required_pull_request_reviews,omitempty"`
	// Exclude specifies a set of regular expressions which identify branches
	// that should be excluded from the protection policy
	Exclude []string `json:"exclude,omitempty"`
}

// IsDefined returns true if at least one of its fields is defined (not nil)
func (p Policy) IsDefined() bool {
	return p.Protect != nil || p.RequiredStatusChecks != nil || p.Admins != nil || p.Restrictions != nil || p.RequiredPullRequestReviews != nil
}

// Apply returns a policy that merges the child into the parent
func (p Policy) Apply(child Policy) Policy {
	return Policy{
		Protect:                    selectBool(p.Protect, child.Protect),
		RequiredStatusChecks:       mergeContextPolicy(p.RequiredStatusChecks, child.RequiredStatusChecks),
		Admins:                     selectBool(p.Admins, child.Admins),
		Restrictions:               mergeRestrictions(p.Restrictions, child.Restrictions),
		RequiredPullRequestReviews: mergeReviewPolicy(p.RequiredPullRequestReviews, child.RequiredPullRequestReviews),
		Exclude:                    unionStrings(p.Exclude, child.Exclude),
	}
}
