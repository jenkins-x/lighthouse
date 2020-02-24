/*
Copyright 2019 The Kubernetes Authors.

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
	"reflect"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/plumber/fake"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/fakegitprovider"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestHelpProvider(t *testing.T) {
	cases := []struct {
		name         string
		config       *plugins.Configuration
		enabledRepos []string
		err          bool
	}{
		{
			name:         "Empty config",
			config:       &plugins.Configuration{},
			enabledRepos: []string{"org1", "org2/repo"},
		},
		{
			name:         "Overlapping org and org/repo",
			config:       &plugins.Configuration{},
			enabledRepos: []string{"org2", "org2/repo"},
		},
		{
			name:         "Invalid enabledRepos",
			config:       &plugins.Configuration{},
			enabledRepos: []string{"org1", "org2/repo/extra"},
			err:          true,
		},
		{
			name: "All configs enabled",
			config: &plugins.Configuration{
				Triggers: []plugins.Trigger{
					{
						Repos:          []string{"org2"},
						TrustedOrg:     "org2",
						JoinOrgURL:     "https://join.me",
						OnlyOrgMembers: true,
						IgnoreOkToTest: true,
					},
				},
			},
			enabledRepos: []string{"org1", "org2/repo"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := helpProvider(c.config, c.enabledRepos)
			if err != nil && !c.err {
				t.Fatalf("helpProvider error: %v", err)
			}
		})
	}
}

func TestRunAndSkipJobs(t *testing.T) {
	var testCases = []struct {
		name string

		requestedJobs        []config.Presubmit
		skippedJobs          []config.Presubmit
		elideSkippedContexts bool
		jobCreationErrs      sets.String // job names which fail creation

		expectedJobs     sets.String // by name
		expectedStatuses []*scm.StatusInput
		expectedErr      bool
	}{
		{
			name: "nothing requested means nothing done",
		},
		{
			name: "all requested jobs get run",
			requestedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context"},
			}},
			expectedJobs: sets.NewString("first", "second"),
		},
		{
			name: "failure on job creation bubbles up but doesn't stop others from starting",
			requestedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context"},
			}},
			jobCreationErrs: sets.NewString("first"),
			expectedJobs:    sets.NewString("second"),
			expectedErr:     true,
			expectedStatuses: []*scm.StatusInput{{
				State: scm.StateError,
				Label: "first-context",
				Desc:  "Error creating metapipeline: failed to create job",
			}},
		},
		{
			name: "all skipped jobs get skipped",
			skippedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context"},
			}},
			expectedStatuses: []*scm.StatusInput{{
				State: scm.StateSuccess,
				Label: "first-context",
				Desc:  "Skipped.",
			}, {
				State: scm.StateSuccess,
				Label: "second-context",
				Desc:  "Skipped.",
			}},
		},
		{
			name: "all skipped jobs get ignored if skipped statuses are elided",
			skippedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context"},
			}},
			elideSkippedContexts: true,
		},
		{
			name: "skipped jobs with skip report get ignored",
			skippedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context", SkipReport: true},
			}},
			expectedStatuses: []*scm.StatusInput{{
				State: scm.StateSuccess,
				Label: "first-context",
				Desc:  "Skipped.",
			}},
		},
		{
			name: "overlap between jobs callErrors and has no external action",
			requestedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context"},
			}},
			skippedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}},
			expectedErr: true,
		},
		{
			name: "disjoint sets of jobs get triggered and skipped correctly",
			requestedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context"},
			}},
			skippedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "third",
				},
				Reporter: config.Reporter{Context: "third-context"},
			}, {
				JobBase: config.JobBase{
					Name: "fourth",
				},
				Reporter: config.Reporter{Context: "fourth-context"},
			}},
			expectedJobs: sets.NewString("first", "second"),
			expectedStatuses: []*scm.StatusInput{{
				State: scm.StateSuccess,
				Label: "third-context",
				Desc:  "Skipped.",
			}, {
				State: scm.StateSuccess,
				Label: "fourth-context",
				Desc:  "Skipped.",
			}},
		},
		{
			name: "disjoint sets of jobs get triggered and skipped correctly, even if one creation fails",
			requestedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context"},
			}},
			skippedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "third",
				},
				Reporter: config.Reporter{Context: "third-context"},
			}, {
				JobBase: config.JobBase{
					Name: "fourth",
				},
				Reporter: config.Reporter{Context: "fourth-context"},
			}},
			jobCreationErrs: sets.NewString("first"),
			expectedJobs:    sets.NewString("second"),
			expectedStatuses: []*scm.StatusInput{{
				State: scm.StateError,
				Label: "first-context",
				Desc:  "Error creating metapipeline: failed to create job",
			}, {
				State: scm.StateSuccess,
				Label: "third-context",
				Desc:  "Skipped.",
			}, {
				State: scm.StateSuccess,
				Label: "fourth-context",
				Desc:  "Skipped.",
			}},
			expectedErr: true,
		},
		{
			name: "jobs that fail to run have status",
			requestedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context"},
			}},
			jobCreationErrs: sets.NewString("first", "second"),
			expectedStatuses: []*scm.StatusInput{{
				State: scm.StateError,
				Label: "first-context",
				Desc:  "Error creating metapipeline: failed to create job",
			}, {
				State: scm.StateError,
				Label: "second-context",
				Desc:  "Error creating metapipeline: failed to create job",
			}},
			expectedErr: true,
		},
	}

	pr := &scm.PullRequest{
		Base: scm.PullRequestBranch{
			Repo: scm.Repository{
				Namespace: "org",
				Name:      "repo",
			},
			Ref: "branch",
		},
		Head: scm.PullRequestBranch{
			Sha: "foobar1",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeGitHubClient := fakegitprovider.FakeClient{}
			fakePlumberClient := fake.NewPlumber()
			fakePlumberClient.FailJobs = testCase.jobCreationErrs

			client := Client{
				GitHubClient:  &fakeGitHubClient,
				PlumberClient: fakePlumberClient,
				Logger:        logrus.WithField("testcase", testCase.name),
			}

			err := RunAndSkipJobs(client, pr, testCase.requestedJobs, testCase.skippedJobs, "event-guid", testCase.elideSkippedContexts)
			if err == nil && testCase.expectedErr {
				t.Errorf("%s: expected an error but got none", testCase.name)
			}
			if err != nil && !testCase.expectedErr {
				t.Errorf("%s: expected no error but got one: %v", testCase.name, err)
			}

			if actual, expected := fakeGitHubClient.CreatedStatuses[pr.Head.Ref], testCase.expectedStatuses; !reflect.DeepEqual(actual, expected) {
				t.Errorf("%s: created incorrect statuses: %s", testCase.name, diff.ObjectReflectDiff(actual, expected))
			}

			observedCreatedPlumberJobs := sets.NewString()
			existingPlumberJobs := fakePlumberClient.Pipelines
			for _, job := range existingPlumberJobs {
				observedCreatedPlumberJobs.Insert(job.Spec.Job)
			}

			if missing := testCase.expectedJobs.Difference(observedCreatedPlumberJobs); missing.Len() > 0 {
				t.Errorf("%s: didn't create all expected PlumberJobs, missing: %s", testCase.name, missing.List())
			}
			if extra := observedCreatedPlumberJobs.Difference(testCase.expectedJobs); extra.Len() > 0 {
				t.Errorf("%s: created unexpected PlumberJobs: %s", testCase.name, extra.List())
			}
		})
	}
}

func TestRunRequested(t *testing.T) {
	var testCases = []struct {
		name string

		requestedJobs   []config.Presubmit
		jobCreationErrs sets.String // job names which fail creation

		expectedJobs sets.String // by name
		expectedErr  bool
	}{
		{
			name: "nothing requested means nothing done",
		},
		{
			name: "all requested jobs get run",
			requestedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context"},
			}},
			expectedJobs: sets.NewString("first", "second"),
		},
		{
			name: "failure on job creation bubbles up but doesn't stop others from starting",
			requestedJobs: []config.Presubmit{{
				JobBase: config.JobBase{
					Name: "first",
				},
				Reporter: config.Reporter{Context: "first-context"},
			}, {
				JobBase: config.JobBase{
					Name: "second",
				},
				Reporter: config.Reporter{Context: "second-context"},
			}},
			jobCreationErrs: sets.NewString("first"),
			expectedJobs:    sets.NewString("second"),
			expectedErr:     true,
		},
	}

	pr := &scm.PullRequest{
		Base: scm.PullRequestBranch{
			Repo: scm.Repository{
				Namespace: "org",
				Name:      "repo",
			},
			Ref: "branch",
		},
		Head: scm.PullRequestBranch{
			Sha: "foobar1",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			fakeGitHubClient := fakegitprovider.FakeClient{}
			fakePlumberClient := fake.NewPlumber()
			fakePlumberClient.FailJobs = testCase.jobCreationErrs

			client := Client{
				GitHubClient:  &fakeGitHubClient,
				PlumberClient: fakePlumberClient,
				Logger:        logrus.WithField("testcase", testCase.name),
			}

			err := runRequested(client, pr, testCase.requestedJobs, "event-guid")
			if err == nil && testCase.expectedErr {
				t.Errorf("%s: expected an error but got none", testCase.name)
			}
			if err != nil && !testCase.expectedErr {
				t.Errorf("%s: expected no error but got one: %v", testCase.name, err)
			}

			observedCreatedPlumberJobs := sets.NewString()
			existingPlumberJobs := fakePlumberClient.Pipelines
			for _, job := range existingPlumberJobs {
				observedCreatedPlumberJobs.Insert(job.Spec.Job)
			}

			if missing := testCase.expectedJobs.Difference(observedCreatedPlumberJobs); missing.Len() > 0 {
				t.Errorf("%s: didn't create all expected PlumberJobs, missing: %s", testCase.name, missing.List())
			}
			if extra := observedCreatedPlumberJobs.Difference(testCase.expectedJobs); extra.Len() > 0 {
				t.Errorf("%s: created unexpected PlumberJobs: %s", testCase.name, extra.List())
			}
		})
	}
}

func TestValidateContextOverlap(t *testing.T) {
	var testCases = []struct {
		name          string
		toRun, toSkip []config.Presubmit
		expectedErr   bool
	}{
		{
			name:   "empty inputs mean no error",
			toRun:  []config.Presubmit{},
			toSkip: []config.Presubmit{},
		},
		{
			name:   "disjoint sets mean no error",
			toRun:  []config.Presubmit{{Reporter: config.Reporter{Context: "foo"}}},
			toSkip: []config.Presubmit{{Reporter: config.Reporter{Context: "bar"}}},
		},
		{
			name:   "complex disjoint sets mean no error",
			toRun:  []config.Presubmit{{Reporter: config.Reporter{Context: "foo"}}, {Reporter: config.Reporter{Context: "otherfoo"}}},
			toSkip: []config.Presubmit{{Reporter: config.Reporter{Context: "bar"}}, {Reporter: config.Reporter{Context: "otherbar"}}},
		},
		{
			name:        "overlapping sets error",
			toRun:       []config.Presubmit{{Reporter: config.Reporter{Context: "foo"}}, {Reporter: config.Reporter{Context: "otherfoo"}}},
			toSkip:      []config.Presubmit{{Reporter: config.Reporter{Context: "bar"}}, {Reporter: config.Reporter{Context: "otherfoo"}}},
			expectedErr: true,
		},
		{
			name:        "identical sets error",
			toRun:       []config.Presubmit{{Reporter: config.Reporter{Context: "foo"}}, {Reporter: config.Reporter{Context: "otherfoo"}}},
			toSkip:      []config.Presubmit{{Reporter: config.Reporter{Context: "foo"}}, {Reporter: config.Reporter{Context: "otherfoo"}}},
			expectedErr: true,
		},
		{
			name:        "superset callErrors",
			toRun:       []config.Presubmit{{Reporter: config.Reporter{Context: "foo"}}, {Reporter: config.Reporter{Context: "otherfoo"}}},
			toSkip:      []config.Presubmit{{Reporter: config.Reporter{Context: "foo"}}, {Reporter: config.Reporter{Context: "otherfoo"}}, {Reporter: config.Reporter{Context: "thirdfoo"}}},
			expectedErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			validateErr := validateContextOverlap(testCase.toRun, testCase.toSkip)
			if validateErr == nil && testCase.expectedErr {
				t.Errorf("%s: expected an error but got none", testCase.name)
			}
			if validateErr != nil && !testCase.expectedErr {
				t.Errorf("%s: expected no error but got one: %v", testCase.name, validateErr)
			}
		})
	}
}
