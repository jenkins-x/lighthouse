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

	"github.com/jenkins-x/go-scm/scm"
)

// ChangedFilesProvider returns a slice of modified files.
type ChangedFilesProvider func() ([]string, error)

type scmClient interface {
	GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error)
}

// NewGitHubDeferredChangedFilesProvider uses a closure to lazily retrieve the file changes only if they are needed.
// We only have to fetch the changes if there is at least one RunIfChanged job that is not being force run (due to
// a `/retest` after a failure or because it is explicitly triggered with `/test foo`).
func NewGitHubDeferredChangedFilesProvider(client scmClient, org, repo string, num int) ChangedFilesProvider {
	var changedFiles []string
	return func() ([]string, error) {
		// Fetch the changed files from github at most once.
		if changedFiles == nil {
			changes, err := client.GetPullRequestChanges(org, repo, num)
			if err != nil {
				return nil, fmt.Errorf("error getting pull request changes: %v", err)
			}
			for _, change := range changes {
				changedFiles = append(changedFiles, change.Path)
			}
		}
		return changedFiles, nil
	}
}
