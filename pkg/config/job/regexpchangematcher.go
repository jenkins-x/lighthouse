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
)

// RegexpChangeMatcher is for code shared between jobs that run only when certain files are changed.
type RegexpChangeMatcher struct {
	// RunIfChanged defines a regex used to select which subset of file changes should trigger this job.
	// If any file in the changeset matches this regex, the job will be triggered
	RunIfChanged string         `json:"run_if_changed,omitempty"`
	reChanges    *regexp.Regexp // from RunIfChanged
}

// CouldRun determines if its possible for a set of changes to trigger this condition
func (cm RegexpChangeMatcher) CouldRun() bool {
	return cm.RunIfChanged != ""
}

// ShouldRun determines if we can know for certain that the job should run. We can either
// know for certain that the job should or should not run based on the matcher, or we can
// not be able to determine that fact at all.
func (cm RegexpChangeMatcher) ShouldRun(changes ChangedFilesProvider) (determined bool, shouldRun bool, err error) {
	if cm.CouldRun() {
		changeList, err := changes()
		if err != nil {
			return true, false, err
		}
		return true, cm.RunsAgainstChanges(changeList), nil
	}
	return false, false, nil
}

// RunsAgainstChanges returns true if any of the changed input paths match the run_if_changed regex.
func (cm RegexpChangeMatcher) RunsAgainstChanges(changes []string) bool {
	for _, change := range changes {
		if cm.GetRE().MatchString(change) {
			return true
		}
	}
	return false
}

// SetChangeRegexes validates and compiles internal regexes
func (cm RegexpChangeMatcher) SetChangeRegexes() (RegexpChangeMatcher, error) {
	if cm.RunIfChanged != "" {
		re, err := regexp.Compile(cm.RunIfChanged)
		if err != nil {
			return cm, fmt.Errorf("could not compile run_if_changed regex: %v", err)
		}
		cm.reChanges = re
	}
	return cm, nil
}

// GetRE lazily creates the regex
func (cm RegexpChangeMatcher) GetRE() *regexp.Regexp {
	if cm.reChanges == nil {
		cm2, _ := cm.SetChangeRegexes()
		return cm2.reChanges
	}
	return cm.reChanges
}
