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

package job

import (
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

// Brancher is for shared code between jobs that only run against certain
// branches. An empty brancher runs against all branches.
type Brancher struct {
	// Do not run against these branches. Default is no branches.
	SkipBranches []string `json:"skip_branches,omitempty"`
	// Only run against these branches. Default is all branches.
	Branches []string `json:"branches,omitempty"`

	// We'll set these when we load it.
	re     *regexp.Regexp
	reSkip *regexp.Regexp
}

// RunsAgainstAllBranch returns true if there are both branches and skip_branches are unset
func (br Brancher) RunsAgainstAllBranch() bool {
	return len(br.SkipBranches) == 0 && len(br.Branches) == 0
}

// SetBrancherRegexes validates and compiles internal regexes
func (br Brancher) SetBrancherRegexes() (Brancher, error) {
	if len(br.Branches) > 0 {
		if re, err := regexp.Compile(strings.Join(br.Branches, `|`)); err == nil {
			br.re = re
		} else {
			return br, fmt.Errorf("could not compile positive branch regex: %v", err)
		}
	}
	if len(br.SkipBranches) > 0 {
		if re, err := regexp.Compile(strings.Join(br.SkipBranches, `|`)); err == nil {
			br.reSkip = re
		} else {
			return br, fmt.Errorf("could not compile negative branch regex: %v", err)
		}
	}
	return br, nil
}

// GetRESkip return the branch skip regexp
func (br Brancher) GetRESkip() *regexp.Regexp {
	if br.reSkip == nil {
		br2, _ := br.SetBrancherRegexes()
		return br2.reSkip
	}
	return br.reSkip
}

// GetRE returns the branch regexp
func (br Brancher) GetRE() *regexp.Regexp {
	if br.re == nil {
		br2, _ := br.SetBrancherRegexes()
		return br2.re
	}
	return br.re
}

// ShouldRun returns true if the input branch matches, given the includes/excludes.
func (br Brancher) ShouldRun(branch string) bool {
	if br.RunsAgainstAllBranch() {
		return true
	}

	// Favor SkipBranches over Branches
	if len(br.SkipBranches) != 0 && br.GetRESkip().MatchString(branch) {
		return false
	}
	if len(br.Branches) == 0 || br.GetRE().MatchString(branch) {
		return true
	}
	return false
}

// Intersects checks if other Brancher would trigger for the same branch.
func (br Brancher) Intersects(other Brancher) bool {
	if br.RunsAgainstAllBranch() || other.RunsAgainstAllBranch() {
		return true
	}
	if len(br.Branches) > 0 {
		baseBranches := sets.NewString(br.Branches...)
		if len(other.Branches) > 0 {
			otherBranches := sets.NewString(other.Branches...)
			return baseBranches.Intersection(otherBranches).Len() > 0
		}
		if !baseBranches.Intersection(sets.NewString(other.SkipBranches...)).Equal(baseBranches) {
			return true
		}
		return false
	}
	if len(other.Branches) == 0 {
		// There can only be one Brancher with skip_branches.
		return true
	}
	return other.Intersects(br)
}
