/*
Copyright 2025 The Kubernetes Authors.

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

package job_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
)

const (
	testBeforeSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testAfterSHA  = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

const emptyGitSHA = "0000000000000000000000000000000000000000"

type pushCompareClient struct {
	changes []*scm.Change
	err     error
	calls   int
}

func (c *pushCompareClient) CompareCommits(_, _, _, _ string) ([]*scm.Change, error) {
	c.calls++
	if c.err != nil {
		return nil, c.err
	}
	return c.changes, nil
}

func postsubmitMatcher(t *testing.T, name, runIfChanged, ignoreChanges string) job.Postsubmit {
	t.Helper()
	p := job.Postsubmit{
		Base: job.Base{Name: name},
		RegexpChangeMatcher: job.RegexpChangeMatcher{
			RunIfChanged:  runIfChanged,
			IgnoreChanges: ignoreChanges,
		},
	}
	if err := p.SetRegexes(); err != nil {
		t.Fatalf("SetRegexes: %v", err)
	}
	return p
}

func TestNewPushCompareChangedFilesProvider(t *testing.T) {

	testCases := []struct {
		name     string
		client   pushCompareClient
		pushHook scm.PushHook
		want     []string
		wantErr  string
	}{
		{
			name: "compare before and after ignores stale commits list",
			client: pushCompareClient{
				changes: []*scm.Change{
					{Path: "pkg/b/foo.go"},
				},
			},
			pushHook: scm.PushHook{
				Before: testBeforeSHA,
				After:  testAfterSHA,
				Repo: scm.Repository{
					Namespace: "org",
					Name:      "repo",
				},
				Commits: []scm.PushCommit{
					{
						Modified: []string{
							"pkg/a/lock.json",
							"pkg/a/config.yaml",
						},
					},
					{
						Modified: []string{"pkg/b/foo.go"},
					},
				},
			},
			want: []string{"pkg/b/foo.go"},
		},
		{
			name: "all_commits fallback without compare SHAs",
			pushHook: scm.PushHook{
				Commits: []scm.PushCommit{
					{Modified: []string{"pkg/a/lock.json"}},
					{Modified: []string{".lighthouse/jenkins-x/triggers.yaml"}},
				},
			},
			want: []string{"pkg/a/lock.json", ".lighthouse/jenkins-x/triggers.yaml"},
		},
		{
			name: "empty compare range",
			pushHook: scm.PushHook{
				Before: testAfterSHA,
				After:  testAfterSHA,
			},
			want: nil,
		},
		{
			name: "new branch falls back to all_commits file lists",
			pushHook: scm.PushHook{
				Before: emptyGitSHA,
				After:  testAfterSHA,
				Commits: []scm.PushCommit{
					{Modified: []string{"pkg/a/lock.json"}},
					{Modified: []string{"pkg/b/foo.go"}},
				},
			},
			want: []string{"pkg/a/lock.json", "pkg/b/foo.go"},
		},
		{
			name: "compare error is returned",
			client: pushCompareClient{
				err: fmt.Errorf("compare failed"),
			},
			pushHook: scm.PushHook{
				Before: testBeforeSHA,
				After:  testAfterSHA,
				Repo: scm.Repository{
					Namespace: "org",
					Name:      "repo",
				},
			},
			wantErr: "compare failed",
		},
		{
			name: "rebase-and-merge uses compare across the whole push",
			client: pushCompareClient{
				changes: []*scm.Change{
					{Path: "pkg/b/foo.go"},
					{Path: "pkg/c/bar.go"},
					{Path: "docs/README.md"},
				},
			},
			pushHook: scm.PushHook{
				Before: testBeforeSHA,
				After:  testAfterSHA,
				Commits: []scm.PushCommit{
					{Added: []string{"pkg/b/foo.go"}},
					{Modified: []string{"pkg/c/bar.go"}},
					{Modified: []string{"docs/README.md"}},
				},
				Repo: scm.Repository{Namespace: "org", Name: "repo"},
			},
			want: []string{"pkg/b/foo.go", "pkg/c/bar.go", "docs/README.md"},
		},
		{
			name: "net revert within push yields empty compare result",
			client: pushCompareClient{
				changes: nil,
			},
			pushHook: scm.PushHook{
				Before: testBeforeSHA,
				After:  testAfterSHA,
				Commits: []scm.PushCommit{
					{Modified: []string{"pkg/a/lock.json"}},
					{Removed: []string{"pkg/a/lock.json"}},
				},
				Repo: scm.Repository{Namespace: "org", Name: "repo"},
			},
			want: nil,
		},
		{
			name: "deleted and renamed paths are returned from compare",
			client: pushCompareClient{
				changes: []*scm.Change{
					{Path: "pkg/legacy/old.go", Deleted: true},
					{Path: "pkg/new/name.go", Renamed: true, PreviousPath: "pkg/old/name.go"},
				},
			},
			pushHook: scm.PushHook{
				Before: testBeforeSHA,
				After:  testAfterSHA,
				Repo:   scm.Repository{Namespace: "org", Name: "repo"},
			},
			want: []string{"pkg/legacy/old.go", "pkg/new/name.go"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			client := testCase.client
			provider := job.NewPushCompareChangedFilesProvider(&client, testCase.pushHook, nil)
			got, err := provider()
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("expected error containing %q, got %v", testCase.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("provider returned error: %v", err)
			}
			if diff := cmp.Diff(testCase.want, got, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
				t.Fatalf("unexpected changed files (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewPushCompareChangedFilesProviderCachesResult(t *testing.T) {
	client := &pushCompareClient{
		changes: []*scm.Change{{Path: "pkg/b/foo.go"}},
	}
	pushHook := scm.PushHook{
		Before: testBeforeSHA,
		After:  testAfterSHA,
		Repo: scm.Repository{
			Namespace: "org",
			Name:      "repo",
		},
	}
	provider := job.NewPushCompareChangedFilesProvider(client, pushHook, nil)

	for i := 0; i < 2; i++ {
		got, err := provider()
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if diff := cmp.Diff([]string{"pkg/b/foo.go"}, got); diff != "" {
			t.Fatalf("call %d: unexpected files (-want +got):\n%s", i, diff)
		}
	}
	if client.calls != 1 {
		t.Fatalf("expected CompareCommits to be called once, got %d", client.calls)
	}
}

func TestPostsubmitShouldRunWithPushCompareProvider(t *testing.T) {
	staleCommits := []scm.PushCommit{
		{Modified: []string{"pkg/a/lock.json", "pkg/a/config.yaml"}},
		{Modified: []string{"pkg/b/foo.go"}},
	}

	testCases := []struct {
		name           string
		runIfChanged   string
		ignoreChanges  string
		branches       []string
		branch         string
		compareChanges []string
		commits        []scm.PushCommit
		wantRun        bool
	}{
		{
			name:           "postsubmit without matcher always runs on empty compare",
			compareChanges: nil,
			wantRun:        true,
		},
		{
			name:           "run_if_changed does not run on empty compare",
			runIfChanged:   `^pkg/b/`,
			compareChanges: nil,
			wantRun:        false,
		},
		{
			name:           "run_if_changed matches net compare diff",
			runIfChanged:   `^pkg/b/`,
			compareChanges: []string{"pkg/b/foo.go"},
			commits:        staleCommits,
			wantRun:        true,
		},
		{
			name:           "run_if_changed ignores stale commits list",
			runIfChanged:   `^pkg/a/`,
			compareChanges: []string{"pkg/b/foo.go"},
			commits:        staleCommits,
			wantRun:        false,
		},
		{
			name:           "run_if_changed with ignore skips when only ignored files changed",
			runIfChanged:   `^pkg/b/`,
			ignoreChanges:  `lock\.json$`,
			compareChanges: []string{"pkg/b/lock.json"},
			wantRun:        false,
		},
		{
			name:           "branch filter blocks job on non-matching ref",
			runIfChanged:   `^pkg/b/`,
			branches:       []string{`^release-.*$`},
			branch:         "main",
			compareChanges: []string{"pkg/b/foo.go"},
			wantRun:        false,
		},
		{
			name:           "branch filter allows job on matching ref",
			runIfChanged:   `^pkg/b/`,
			branches:       []string{`^release-.*$`},
			branch:         "release-1.2",
			compareChanges: []string{"pkg/b/foo.go"},
			wantRun:        true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			changes := make([]*scm.Change, len(testCase.compareChanges))
			for i, path := range testCase.compareChanges {
				changes[i] = &scm.Change{Path: path}
			}
			client := &pushCompareClient{changes: changes}
			pushHook := scm.PushHook{
				Before: testBeforeSHA,
				After:  testAfterSHA,
				Repo: scm.Repository{
					Namespace: "org",
					Name:      "repo",
				},
				Commits: testCase.commits,
			}
			postsubmit := job.Postsubmit{
				Base: job.Base{Name: "release"},
				RegexpChangeMatcher: job.RegexpChangeMatcher{
					RunIfChanged:  testCase.runIfChanged,
					IgnoreChanges: testCase.ignoreChanges,
				},
				Brancher: job.Brancher{Branches: testCase.branches},
			}
			if err := postsubmit.SetRegexes(); err != nil {
				t.Fatalf("SetRegexes: %v", err)
			}
			branch := testCase.branch
			if branch == "" {
				branch = "main"
			}
			provider := job.NewPushCompareChangedFilesProvider(client, pushHook, nil)

			got, err := postsubmit.ShouldRun(branch, provider)
			if err != nil {
				t.Fatalf("ShouldRun: %v", err)
			}
			if got != testCase.wantRun {
				t.Fatalf("ShouldRun = %v, want %v", got, testCase.wantRun)
			}
		})
	}
}

func TestPostsubmitShouldRunWithPushCompareProviderError(t *testing.T) {
	client := &pushCompareClient{err: errors.New("api down")}
	pushHook := scm.PushHook{
		Before: testBeforeSHA,
		After:  testAfterSHA,
		Repo: scm.Repository{
			Namespace: "org",
			Name:      "repo",
		},
	}
	postsubmit := postsubmitMatcher(t, "release", `^pkg/b/`, "")
	_, err := postsubmit.ShouldRun("main", job.NewPushCompareChangedFilesProvider(client, pushHook, nil))
	if err == nil {
		t.Fatal("expected error from compare failure")
	}
}

func TestNewPushAllCommitsChangedFilesProvider(t *testing.T) {
	pushHook := scm.PushHook{
		Commits: []scm.PushCommit{
			{Modified: []string{"pkg/a/lock.json"}},
			{Modified: []string{"pkg/b/foo.go"}},
		},
	}
	got, err := job.NewPushAllCommitsChangedFilesProvider(pushHook)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	want := []string{"pkg/a/lock.json", "pkg/b/foo.go"}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
		t.Fatalf("unexpected changed files (-want +got):\n%s", diff)
	}
}

func TestPushCompareErrNotSupportedFallsBack(t *testing.T) {
	client := &pushCompareClient{err: scm.ErrNotSupported}
	pushHook := scm.PushHook{
		Before: testBeforeSHA,
		After:  testAfterSHA,
		Commits: []scm.PushCommit{
			{Modified: []string{"pkg/a/lock.json"}},
			{Added: []string{"pkg/b/foo.go"}},
		},
		Repo: scm.Repository{
			Namespace: "org",
			Name:      "repo",
		},
	}
	var warnings []string
	warn := func(format string, args ...interface{}) {
		warnings = append(warnings, fmt.Sprintf(format, args...))
	}
	got, err := job.NewPushCompareChangedFilesProvider(client, pushHook, warn)()
	if err != nil {
		t.Fatalf("provider returned error: %v", err)
	}
	want := []string{"pkg/a/lock.json", "pkg/b/foo.go"}
	if diff := cmp.Diff(want, got, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
		t.Fatalf("unexpected changed files (-want +got):\n%s", diff)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected one warning, got %d: %v", len(warnings), warnings)
	}
}

func TestNewPushChangedFilesProviderDispatches(t *testing.T) {
	pushHook := scm.PushHook{
		Before: testBeforeSHA,
		After:  testAfterSHA,
		Commits: []scm.PushCommit{
			{Modified: []string{"pkg/a/lock.json"}},
			{Modified: []string{"pkg/b/foo.go"}},
		},
		Repo: scm.Repository{
			Namespace: "org",
			Name:      "repo",
		},
	}
	client := &pushCompareClient{
		changes: []*scm.Change{{Path: "pkg/b/foo.go"}},
	}

	legacyProvider, err := job.NewPushChangedFilesProvider(job.PushChangedFilesAllCommits, client, pushHook, nil)
	if err != nil {
		t.Fatalf("legacy provider: %v", err)
	}
	legacy, err := legacyProvider()
	if err != nil {
		t.Fatalf("legacy provider: %v", err)
	}
	legacyWant := []string{"pkg/a/lock.json", "pkg/b/foo.go"}
	if diff := cmp.Diff(legacyWant, legacy, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
		t.Fatalf("legacy changed files (-want +got):\n%s", diff)
	}

	compareProvider, err := job.NewPushChangedFilesProvider(job.PushChangedFilesCompare, client, pushHook, nil)
	if err != nil {
		t.Fatalf("compare provider: %v", err)
	}
	compare, err := compareProvider()
	if err != nil {
		t.Fatalf("compare provider: %v", err)
	}
	compareWant := []string{"pkg/b/foo.go"}
	if diff := cmp.Diff(compareWant, compare, cmpopts.SortSlices(func(a, b string) bool { return a < b })); diff != "" {
		t.Fatalf("compare changed files (-want +got):\n%s", diff)
	}
}

func TestParsePushChangedFilesMode(t *testing.T) {
	testCases := []struct {
		value   string
		want    job.PushChangedFilesMode
		wantErr bool
	}{
		{value: "", want: job.PushChangedFilesAllCommits},
		{value: "all_commits", want: job.PushChangedFilesAllCommits},
		{value: "compare", want: job.PushChangedFilesCompare},
		{value: "invalid", wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.value, func(t *testing.T) {
			got, err := job.ParsePushChangedFilesMode(testCase.value)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePushChangedFilesMode: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("mode = %q, want %q", got, testCase.want)
			}
		})
	}
}
