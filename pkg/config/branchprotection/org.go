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

// Org holds the default protection policy for an entire org, as well as any repo overrides.
type Org struct {
	Policy
	Repos map[string]Repo `json:"repos,omitempty"`
}

// GetRepo returns the repo config after merging in any org policies.
func (o Org) GetRepo(name string) *Repo {
	r, ok := o.Repos[name]
	if ok {
		r.Policy = o.Apply(r.Policy)
	} else {
		r.Policy = o.Policy
	}
	return &r
}
