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

import "fmt"

// Postsubmit runs on push events.
type Postsubmit struct {
	Base
	RegexpChangeMatcher
	Brancher
	// TODO(krzyzacy): Move existing `Report` into `Skip_Report` once this is deployed
	Reporter
	JenkinsSpec *JenkinsSpec `json:"jenkins_spec,omitempty"`
}

// JenkinsSpec holds optional Jenkins job config
type JenkinsSpec struct {
	// Job is managed by the GH branch source plugin
	// and requires a specific path
	BranchSourceJob bool `json:"branch_source_job,omitempty"`
}

// SetDefaults initializes default values
func (p *Postsubmit) SetDefaults(namespace string) {
	p.Base.SetDefaults(namespace)
	if p.Context == "" {
		p.Context = p.Name
	}
}

// SetRegexes compiles and validates all the regular expressions
func (p *Postsubmit) SetRegexes() error {
	b, err := p.Brancher.SetBrancherRegexes()
	if err != nil {
		return fmt.Errorf("could not set branch regexes for %s: %v", p.Name, err)
	}
	p.Brancher = b
	c, err := p.RegexpChangeMatcher.SetChangeRegexes()
	if err != nil {
		return fmt.Errorf("could not set change regexes for %s: %v", p.Name, err)
	}
	p.RegexpChangeMatcher = c
	return nil
}

// CouldRun determines if the postsubmit could run against a specific
// base ref
func (p Postsubmit) CouldRun(baseRef string) bool {
	return p.Brancher.ShouldRun(baseRef)
}

// ShouldRun determines if the postsubmit should run in response to a
// set of changes. This is evaluated lazily, if necessary.
func (p Postsubmit) ShouldRun(baseRef string, changes ChangedFilesProvider) (bool, error) {
	if !p.CouldRun(baseRef) {
		return false, nil
	}
	if determined, shouldRun, err := p.RegexpChangeMatcher.ShouldRun(changes); err != nil {
		return false, err
	} else if determined {
		return shouldRun, nil
	}
	// Postsubmits default to always run
	return true, nil
}
