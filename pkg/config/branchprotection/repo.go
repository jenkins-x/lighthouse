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

import (
	"errors"
)

// Repo holds protection policy overrides for all branches in a repo, as well as specific branch overrides.
type Repo struct {
	Policy
	Branches map[string]Branch `json:"branches,omitempty"`
}

// GetBranch returns the branch config after merging in any repo policies.
func (r Repo) GetBranch(name string) (*Branch, error) {
	b, ok := r.Branches[name]
	if ok {
		b.Policy = r.Apply(b.Policy)
		if b.Protect == nil {
			return nil, errors.New("defined branch policies must set protect")
		}
	} else {
		b.Policy = r.Policy
	}
	return &b, nil
}
