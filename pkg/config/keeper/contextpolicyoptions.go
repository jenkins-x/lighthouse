/*
Copyright 2017 The Kubernetes Authors.

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

package keeper

// ContextPolicyOptions holds the default policy, and any org overrides.
type ContextPolicyOptions struct {
	ContextPolicy
	// Github Orgs
	Orgs map[string]OrgContextPolicy `json:"orgs,omitempty"`
}

// Parse returns the context policy for an org/repo/branch
func (options ContextPolicyOptions) Parse(org, repo, branch string) ContextPolicy {
	option := options.ContextPolicy
	if o, ok := options.Orgs[org]; ok {
		option = option.Merge(o.ContextPolicy)
		if r, ok := o.Repos[repo]; ok {
			option = option.Merge(r.ContextPolicy)
			if b, ok := r.Branches[branch]; ok {
				option = option.Merge(b)
			}
		}
	}
	return option
}
