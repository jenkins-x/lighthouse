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

package trigger

import (
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/launcher/fake"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	fake2 "github.com/jenkins-x/lighthouse/pkg/scmprovider/fake"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/equality"
)

const (
	testBeforeSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	testAfterSHA  = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func pushCompareTrigger() *plugins.Trigger {
	return &plugins.Trigger{PushChangedFiles: string(job.PushChangedFilesCompare)}
}

func TestCreateRefs(t *testing.T) {
	pe := &scm.PushHook{
		Ref: "refs/heads/master",
		Repo: scm.Repository{
			Namespace: "kubernetes",
			Name:      "repo",
			Link:      "https://example.com/kubernetes/repo",
		},
		After:   "abcdef",
		Compare: "https://example.com/kubernetes/repo/compare/abcdee...abcdef",
	}
	expected := v1alpha1.Refs{
		Org:      "kubernetes",
		Repo:     "repo",
		BaseRef:  "master",
		BaseSHA:  "abcdef",
		BaseLink: "https://example.com/kubernetes/repo/compare/abcdee...abcdef",
	}
	if actual := createRefs(pe); !equality.Semantic.DeepEqual(expected, actual) {
		t.Errorf("diff between expected and actual refs:%s", cmp.Diff(expected, actual))
	}
}

func runHandlePE(t *testing.T, pe scm.PushHook, trigger *plugins.Trigger, postsubmits map[string][]job.Postsubmit, compareChanges []*scm.Change) *fake.Launcher {
	t.Helper()
	g := &fake2.SCMClient{}
	if pe.Before != "" && pe.After != "" {
		g.PushCompareChanges = map[string][]*scm.Change{
			pe.Before + ":" + pe.After: compareChanges,
		}
	}
	launcher := fake.NewLauncher()
	c := Client{
		SCMProviderClient: g,
		LauncherClient:    launcher,
		Config:            &config.Config{ProwConfig: config.ProwConfig{LighthouseJobNamespace: "lighthouseJobs"}},
		Logger:            logrus.WithField("plugin", pluginName),
	}
	for repo, jobs := range postsubmits {
		for i := range jobs {
			if err := jobs[i].SetRegexes(); err != nil {
				t.Fatalf("SetRegexes(%s): %v", repo, err)
			}
		}
	}
	if err := c.Config.SetPostsubmits(postsubmits); err != nil {
		t.Fatalf("SetPostsubmits: %v", err)
	}
	if err := handlePE(c, pe, trigger); err != nil {
		t.Fatalf("handlePE: %v", err)
	}
	return launcher
}

func TestHandlePE(t *testing.T) {
	testCases := []struct {
		name           string
		pe             scm.PushHook
		compareChanges []*scm.Change
		jobsToRun      int
	}{
		{
			name: "branch deleted",
			pe: scm.PushHook{
				Ref: "refs/heads/master",
				Repo: scm.Repository{FullName: "org/repo"},
				Deleted: true,
			},
		},
		{
			name: "no matching files",
			pe: scm.PushHook{
				Ref:     "refs/heads/master",
				Before:  testBeforeSHA,
				After:   testAfterSHA,
				Commits: []scm.PushCommit{{Added: []string{"example.txt"}}},
				Repo:    scm.Repository{Namespace: "org", Name: "repo", FullName: "org/repo"},
			},
			compareChanges: []*scm.Change{{Path: "example.txt"}},
		},
		{
			name: "one matching file",
			pe: scm.PushHook{
				Ref:    "refs/heads/master",
				Before: testBeforeSHA,
				After:  testAfterSHA,
				Commits: []scm.PushCommit{{
					Added:    []string{"example.txt"},
					Modified: []string{"hack.sh"},
				}},
				Repo: scm.Repository{Namespace: "org", Name: "repo", FullName: "org/repo"},
			},
			compareChanges: []*scm.Change{{Path: "hack.sh"}},
			jobsToRun:      1,
		},
		{
			name: "reverted files only in commits list",
			pe: scm.PushHook{
				Ref:    "refs/heads/main",
				Before: testBeforeSHA,
				After:  testAfterSHA,
				Commits: []scm.PushCommit{
					{Modified: []string{"pkg/a/lock.json", "pkg/a/config.yaml"}},
					{Modified: []string{"pkg/b/foo.go"}},
				},
				Repo: scm.Repository{Namespace: "org", Name: "repo", FullName: "org/repo"},
			},
			compareChanges: []*scm.Change{{Path: "pkg/b/foo.go"}},
		},
		{
			name: "no change matcher",
			pe: scm.PushHook{
				Ref:     "refs/heads/master",
				Before:  testBeforeSHA,
				After:   testAfterSHA,
				Commits: []scm.PushCommit{{Added: []string{"example.txt"}}},
				Repo:    scm.Repository{Namespace: "org2", Name: "repo2", FullName: "org2/repo2"},
			},
			compareChanges: []*scm.Change{{Path: "example.txt"}},
			jobsToRun:      1,
		},
		{
			name: "branch name with a slash",
			pe: scm.PushHook{
				Ref:     "refs/heads/release/v1.14",
				Before:  testBeforeSHA,
				After:   testAfterSHA,
				Commits: []scm.PushCommit{{Added: []string{"hack.sh"}}},
				Repo:    scm.Repository{Namespace: "org3", Name: "repo3", FullName: "org3/repo3"},
			},
			compareChanges: []*scm.Change{{Path: "hack.sh"}},
			jobsToRun:      1,
		},
	}
	postsubmits := map[string][]job.Postsubmit{
		"org/repo": {
			{
				Base:                job.Base{Name: "pass-butter"},
				RegexpChangeMatcher: job.RegexpChangeMatcher{RunIfChanged: "\\.sh$"},
			},
			{
				Base:                job.Base{Name: "pkg-a-release"},
				RegexpChangeMatcher: job.RegexpChangeMatcher{RunIfChanged: "^pkg/a/"},
			},
		},
		"org2/repo2": {{Base: job.Base{Name: "pass-salt"}}},
		"org3/repo3": {{
			Base:     job.Base{Name: "pass-pepper"},
			Brancher: job.Brancher{Branches: []string{"release/v1.14"}},
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			launcher := runHandlePE(t, tc.pe, pushCompareTrigger(), postsubmits, tc.compareChanges)
			if got := len(launcher.Pipelines); got != tc.jobsToRun {
				t.Fatalf("expected %d jobs to run, got %d", tc.jobsToRun, got)
			}
		})
	}
}

func TestHandlePECompareScenarios(t *testing.T) {
	testCases := []struct {
		name           string
		ref            string
		commits        []scm.PushCommit
		compareChanges []*scm.Change
		postsubmits    []job.Postsubmit
		wantJobs       []string
	}{
		{
			name: "merge commit with net empty compare does not run change-matched postsubmits",
			ref:  "refs/heads/main",
			commits: []scm.PushCommit{
				{Modified: []string{"pkg/a/lock.json"}},
				{Removed: []string{"pkg/a/lock.json"}},
			},
			postsubmits: []job.Postsubmit{{
				Base:                job.Base{Name: "pkg-a-release"},
				RegexpChangeMatcher: job.RegexpChangeMatcher{RunIfChanged: `^pkg/a/`},
			}},
		},
		{
			name:        "postsubmit without matcher still runs when compare is empty",
			ref:         "refs/heads/main",
			postsubmits: []job.Postsubmit{{Base: job.Base{Name: "always-run"}}},
			wantJobs:    []string{"always-run"},
		},
		{
			name: "stale commits from merge-from-main do not trigger unrelated pipelines",
			ref:  "refs/heads/main",
			commits: []scm.PushCommit{
				{Modified: []string{"vendor/main-only/deps.go"}},
				{Modified: []string{"pkg/b/foo.go"}},
			},
			compareChanges: []*scm.Change{{Path: "pkg/b/foo.go"}},
			postsubmits: []job.Postsubmit{
				{
					Base:                job.Base{Name: "vendor-release"},
					RegexpChangeMatcher: job.RegexpChangeMatcher{RunIfChanged: `^vendor/`},
				},
				{
					Base:                job.Base{Name: "pkg-b-release"},
					RegexpChangeMatcher: job.RegexpChangeMatcher{RunIfChanged: `^pkg/b/`},
				},
			},
			wantJobs: []string{"pkg-b-release"},
		},
		{
			name:           "change matching with ignore_changes on multi-path compare",
			ref:            "refs/heads/main",
			compareChanges: []*scm.Change{{Path: "pkg/b/foo.go"}, {Path: "pkg/b/lock.json"}, {Path: "pkg/c/bar.go"}},
			postsubmits: []job.Postsubmit{
				{
					Base: job.Base{Name: "pkg-b-release"},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged:  `^pkg/b/`,
						IgnoreChanges: `lock\.json$`,
					},
				},
				{
					Base:                job.Base{Name: "pkg-c-release"},
					RegexpChangeMatcher: job.RegexpChangeMatcher{RunIfChanged: `^pkg/c/`},
				},
			},
			wantJobs: []string{"pkg-b-release", "pkg-c-release"},
		},
		{
			name:           "branch filter prevents postsubmit on wrong branch",
			ref:            "refs/heads/main",
			compareChanges: []*scm.Change{{Path: "pkg/b/foo.go"}},
			postsubmits: []job.Postsubmit{{
				Base:                job.Base{Name: "release-only"},
				RegexpChangeMatcher: job.RegexpChangeMatcher{RunIfChanged: `^pkg/b/`},
				Brancher:            job.Brancher{Branches: []string{"release-.*"}},
			}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pe := scm.PushHook{
				Ref:     tc.ref,
				Before:  testBeforeSHA,
				After:   testAfterSHA,
				Commits: tc.commits,
				Repo:    scm.Repository{Namespace: "org", Name: "repo", FullName: "org/repo"},
			}
			launcher := runHandlePE(t, pe, pushCompareTrigger(), map[string][]job.Postsubmit{
				"org/repo": tc.postsubmits,
			}, tc.compareChanges)
			if diff := cmp.Diff(tc.wantJobs, launchedJobNames(launcher)); diff != "" {
				t.Fatalf("unexpected launched jobs (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHandlePEDefaultsToAllCommits(t *testing.T) {
	pe := scm.PushHook{
		Ref:    "refs/heads/main",
		Before: testBeforeSHA,
		After:  testAfterSHA,
		Commits: []scm.PushCommit{
			{Modified: []string{"pkg/a/lock.json", "pkg/a/config.yaml"}},
			{Modified: []string{"pkg/b/foo.go"}},
		},
		Repo: scm.Repository{Namespace: "org", Name: "repo", FullName: "org/repo"},
	}
	launcher := runHandlePE(t, pe, &plugins.Trigger{}, map[string][]job.Postsubmit{
		"org/repo": {
			{
				Base:                job.Base{Name: "pkg-b-release"},
				RegexpChangeMatcher: job.RegexpChangeMatcher{RunIfChanged: `^pkg/b/`},
			},
			{
				Base:                job.Base{Name: "pkg-a-release"},
				RegexpChangeMatcher: job.RegexpChangeMatcher{RunIfChanged: `^pkg/a/`},
			},
		},
	}, []*scm.Change{{Path: "pkg/b/foo.go"}})

	want := []string{"pkg-a-release", "pkg-b-release"}
	if diff := cmp.Diff(want, launchedJobNames(launcher)); diff != "" {
		t.Fatalf("legacy commits union should match both jobs (-want +got):\n%s", diff)
	}
}

func launchedJobNames(launcher *fake.Launcher) []string {
	if len(launcher.Pipelines) == 0 {
		return nil
	}
	names := make([]string, 0, len(launcher.Pipelines))
	for _, pj := range launcher.Pipelines {
		names = append(names, pj.Spec.Job)
	}
	slices.Sort(names)
	return names
}
