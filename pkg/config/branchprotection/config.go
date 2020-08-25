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

// Config specifies the global branch protection policy
type Config struct {
	Policy
	// ProtectTested determines if branch protection rules are set for all repos
	// that Prow has registered jobs for, regardless of if those repos are in the
	// branch protection config.
	ProtectTested bool `json:"protect-tested-repos,omitempty"`
	// Orgs holds branch protection options for orgs by name
	Orgs map[string]Org `json:"orgs,omitempty"`
	// AllowDisabledPolicies allows a child to disable all protection even if the
	// branch has inherited protection options from a parent.
	AllowDisabledPolicies bool `json:"allow_disabled_policies,omitempty"`
	// AllowDisabledJobPolicies allows a branch to choose to opt out of branch protection
	// even if Prow has registered required jobs for that branch.
	AllowDisabledJobPolicies bool `json:"allow_disabled_job_policies,omitempty"`
}

// GetOrg returns the org config after merging in any global policies.
func (bp Config) GetOrg(name string) *Org {
	o, ok := bp.Orgs[name]
	if ok {
		o.Policy = bp.Apply(o.Policy)
	} else {
		o.Policy = bp.Policy
	}
	return &o
}
