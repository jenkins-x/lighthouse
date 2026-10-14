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
	"errors"
	"fmt"

	"github.com/jenkins-x/go-scm/scm"
)

const emptyGitSHA = "0000000000000000000000000000000000000000"

// PushChangedFilesMode selects how postsubmit change matching resolves changed files on push events.
// This applies to run_if_changed and ignore_changes on postsubmits.
type PushChangedFilesMode string

const (
	// PushChangedFilesAllCommits unions every commit file list from the push webhook (legacy default).
	PushChangedFilesAllCommits PushChangedFilesMode = "all_commits"
	// PushChangedFilesCompare resolves the net diff between push before/after SHAs via the SCM compare API.
	PushChangedFilesCompare PushChangedFilesMode = "compare"
)

// ChangedFilesProvider returns a slice of modified files.
type ChangedFilesProvider func() ([]string, error)

// PushChangedFilesWarnFunc logs non-fatal push changed-files resolution issues.
type PushChangedFilesWarnFunc func(format string, args ...interface{})

type scmClient interface {
	GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error)
}

type pushCompareClient interface {
	CompareCommits(org, repo, baseSHA, headSHA string) ([]*scm.Change, error)
}

// ParsePushChangedFilesMode validates a push_changed_files configuration value.
func ParsePushChangedFilesMode(value string) (PushChangedFilesMode, error) {
	if value == "" {
		return PushChangedFilesAllCommits, nil
	}
	mode := PushChangedFilesMode(value)
	switch mode {
	case PushChangedFilesAllCommits, PushChangedFilesCompare:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid push_changed_files %q: want all_commits or compare", value)
	}
}

// NewPushChangedFilesProvider returns a lazy provider for postsubmit change matching on push events.
func NewPushChangedFilesProvider(mode PushChangedFilesMode, client pushCompareClient, pe scm.PushHook, warn PushChangedFilesWarnFunc) (ChangedFilesProvider, error) {
	switch mode {
	case PushChangedFilesAllCommits:
		return NewPushAllCommitsChangedFilesProvider(pe), nil
	case PushChangedFilesCompare:
		return NewPushCompareChangedFilesProvider(client, pe, warn), nil
	default:
		return nil, fmt.Errorf("unsupported push changed files mode %q", mode)
	}
}

func lazyOnce(fn func() ([]string, error)) ChangedFilesProvider {
	var (
		files []string
		err   error
		done  bool
	)
	return func() ([]string, error) {
		if !done {
			files, err = fn()
			done = true
		}
		return files, err
	}
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

// NewPushAllCommitsChangedFilesProvider lazily resolves changed files by unioning every commit file list
// from a push webhook (legacy behaviour before push compare was introduced).
func NewPushAllCommitsChangedFilesProvider(pe scm.PushHook) ChangedFilesProvider {
	return lazyOnce(func() ([]string, error) {
		return pushAllCommitsFiles(pe), nil
	})
}

// NewPushCompareChangedFilesProvider lazily resolves the net file changes introduced by a push event.
// It compares pe.Before and pe.After when possible, falling back to all_commits when compare is
// unavailable on the SCM driver or the webhook lacks before/after SHAs.
func NewPushCompareChangedFilesProvider(client pushCompareClient, pe scm.PushHook, warn PushChangedFilesWarnFunc) ChangedFilesProvider {
	return lazyOnce(func() ([]string, error) {
		if pe.Before != "" && pe.After != "" && pe.Before == pe.After {
			return nil, nil
		}

		if client != nil && pe.Before != "" && pe.After != "" && pe.Before != emptyGitSHA {
			changes, err := client.CompareCommits(pe.Repo.Namespace, pe.Repo.Name, pe.Before, pe.After)
			if err != nil {
				if errors.Is(err, scm.ErrNotSupported) {
					if warn != nil {
						warn("push compare not supported by SCM driver for %s/%s, falling back to all_commits", pe.Repo.Namespace, pe.Repo.Name)
					}
					return pushAllCommitsFiles(pe), nil
				}
				return nil, fmt.Errorf("error comparing push commits %s...%s: %w", pe.Before, pe.After, err)
			}
			var files []string
			for _, change := range changes {
				files = append(files, change.Path)
			}
			return files, nil
		}

		if pe.Before == "" || pe.Before == emptyGitSHA {
			if warn != nil {
				warn("push webhook missing before SHA for %s/%s, falling back to all_commits", pe.Repo.Namespace, pe.Repo.Name)
			}
		} else if client == nil {
			if warn != nil {
				warn("no SCM client available for push compare on %s/%s, falling back to all_commits", pe.Repo.Namespace, pe.Repo.Name)
			}
		}
		return pushAllCommitsFiles(pe), nil
	})
}

func pushAllCommitsFiles(pe scm.PushHook) []string {
	changed := make(map[string]bool)
	for _, commit := range pe.Commits {
		for _, path := range commit.Added {
			changed[path] = true
		}
		for _, path := range commit.Removed {
			changed[path] = true
		}
		for _, path := range commit.Modified {
			changed[path] = true
		}
	}
	changedFiles := make([]string, 0, len(changed))
	for path := range changed {
		changedFiles = append(changedFiles, path)
	}
	return changedFiles
}
