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

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/launcher/fake"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	fake2 "github.com/jenkins-x/lighthouse/pkg/scmprovider/fake"
	"github.com/sirupsen/logrus"
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
			_, err := configHelp(c.config, c.enabledRepos)
			if err != nil && !c.err {
				t.Fatalf("helpProvider error: %v", err)
			}
		})
	}
}

func TestRunAndSkipJobs(t *testing.T) {
	var testCases = []struct {
		name string

		requestedJobs        []job.Presubmit
		skippedJobs          []job.Presubmit
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
			requestedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context"},
			}},
			expectedJobs: sets.NewString("first", "second"),
		},
		{
			name: "failure on job creation bubbles up but doesn't stop others from starting",
			requestedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context"},
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
			skippedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context"},
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
			skippedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context"},
			}},
			elideSkippedContexts: true,
		},
		{
			name: "skipped jobs with skip report get ignored",
			skippedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context", SkipReport: true},
			}},
			expectedStatuses: []*scm.StatusInput{{
				State: scm.StateSuccess,
				Label: "first-context",
				Desc:  "Skipped.",
			}},
		},
		{
			name: "overlap between jobs callErrors and has no external action",
			requestedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context"},
			}},
			skippedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}},
			expectedErr: true,
		},
		{
			name: "disjoint sets of jobs get triggered and skipped correctly",
			requestedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context"},
			}},
			skippedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "third",
				},
				Reporter: job.Reporter{Context: "third-context"},
			}, {
				Base: job.Base{
					Name: "fourth",
				},
				Reporter: job.Reporter{Context: "fourth-context"},
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
			requestedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context"},
			}},
			skippedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "third",
				},
				Reporter: job.Reporter{Context: "third-context"},
			}, {
				Base: job.Base{
					Name: "fourth",
				},
				Reporter: job.Reporter{Context: "fourth-context"},
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
			requestedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context"},
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
			fakeSCMClient := fake2.SCMClient{}
			fakeLauncher := fake.NewLauncher()
			fakeLauncher.FailJobs = testCase.jobCreationErrs

			client := Client{
				SCMProviderClient: &fakeSCMClient,
				LauncherClient:    fakeLauncher,
				Logger:            logrus.WithField("testcase", testCase.name),
			}

			err := RunAndSkipJobs(client, pr, testCase.requestedJobs, testCase.skippedJobs, "event-guid", testCase.elideSkippedContexts)
			if err == nil && testCase.expectedErr {
				t.Errorf("%s: expected an error but got none", testCase.name)
			}
			if err != nil && !testCase.expectedErr {
				t.Errorf("%s: expected no error but got one: %v", testCase.name, err)
			}

			if actual, expected := fakeSCMClient.CreatedStatuses[pr.Head.Ref], testCase.expectedStatuses; !reflect.DeepEqual(actual, expected) {
				t.Errorf("%s: created incorrect statuses: %s", testCase.name, cmp.Diff(actual, expected))
			}

			observedCreatedLighthouseJobs := sets.NewString()
			existingLighthouseJobs := fakeLauncher.Pipelines
			for _, job := range existingLighthouseJobs {
				observedCreatedLighthouseJobs.Insert(job.Spec.Job)
			}

			if missing := testCase.expectedJobs.Difference(observedCreatedLighthouseJobs); missing.Len() > 0 {
				t.Errorf("%s: didn't create all expected LighthouseJobs, missing: %s", testCase.name, missing.List())
			}
			if extra := observedCreatedLighthouseJobs.Difference(testCase.expectedJobs); extra.Len() > 0 {
				t.Errorf("%s: created unexpected LighthouseJobs: %s", testCase.name, extra.List())
			}
		})
	}
}

func TestRunRequested(t *testing.T) {
	var testCases = []struct {
		name string

		requestedJobs   []job.Presubmit
		jobCreationErrs sets.String // job names which fail creation

		expectedJobs sets.String // by name
		expectedErr  bool
	}{
		{
			name: "nothing requested means nothing done",
		},
		{
			name: "all requested jobs get run",
			requestedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context"},
			}},
			expectedJobs: sets.NewString("first", "second"),
		},
		{
			name: "failure on job creation bubbles up but doesn't stop others from starting",
			requestedJobs: []job.Presubmit{{
				Base: job.Base{
					Name: "first",
				},
				Reporter: job.Reporter{Context: "first-context"},
			}, {
				Base: job.Base{
					Name: "second",
				},
				Reporter: job.Reporter{Context: "second-context"},
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
			fakeSCMClient := fake2.SCMClient{}
			fakeLauncher := fake.NewLauncher()
			fakeLauncher.FailJobs = testCase.jobCreationErrs

			client := Client{
				SCMProviderClient: &fakeSCMClient,
				LauncherClient:    fakeLauncher,
				Logger:            logrus.WithField("testcase", testCase.name),
			}

			err := runRequested(client, pr, testCase.requestedJobs, "event-guid")
			if err == nil && testCase.expectedErr {
				t.Errorf("%s: expected an error but got none", testCase.name)
			}
			if err != nil && !testCase.expectedErr {
				t.Errorf("%s: expected no error but got one: %v", testCase.name, err)
			}

			observedCreatedLighthouseJobs := sets.NewString()
			existingLighthouseJobs := fakeLauncher.Pipelines
			for _, job := range existingLighthouseJobs {
				observedCreatedLighthouseJobs.Insert(job.Spec.Job)
			}

			if missing := testCase.expectedJobs.Difference(observedCreatedLighthouseJobs); missing.Len() > 0 {
				t.Errorf("%s: didn't create all expected LighthouseJobs, missing: %s", testCase.name, missing.List())
			}
			if extra := observedCreatedLighthouseJobs.Difference(testCase.expectedJobs); extra.Len() > 0 {
				t.Errorf("%s: created unexpected LighthouseJobs: %s", testCase.name, extra.List())
			}
		})
	}
}

func TestValidateContextOverlap(t *testing.T) {
	var testCases = []struct {
		name          string
		toRun, toSkip []job.Presubmit
		expectedErr   bool
	}{
		{
			name:   "empty inputs mean no error",
			toRun:  []job.Presubmit{},
			toSkip: []job.Presubmit{},
		},
		{
			name:   "disjoint sets mean no error",
			toRun:  []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}},
			toSkip: []job.Presubmit{{Reporter: job.Reporter{Context: "bar"}}},
		},
		{
			name:   "complex disjoint sets mean no error",
			toRun:  []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "otherfoo"}}},
			toSkip: []job.Presubmit{{Reporter: job.Reporter{Context: "bar"}}, {Reporter: job.Reporter{Context: "otherbar"}}},
		},
		{
			name:        "overlapping sets error",
			toRun:       []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "otherfoo"}}},
			toSkip:      []job.Presubmit{{Reporter: job.Reporter{Context: "bar"}}, {Reporter: job.Reporter{Context: "otherfoo"}}},
			expectedErr: true,
		},
		{
			name:        "identical sets error",
			toRun:       []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "otherfoo"}}},
			toSkip:      []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "otherfoo"}}},
			expectedErr: true,
		},
		{
			name:        "superset callErrors",
			toRun:       []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "otherfoo"}}},
			toSkip:      []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "otherfoo"}}, {Reporter: job.Reporter{Context: "thirdfoo"}}},
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

func TestTrustedUser(t *testing.T) {
	var testcases = []struct {
		name string

		onlyOrgMembers bool
		trustedApps    []string
		trustedOrg     string

		user string
		org  string
		repo string

		expectedTrusted bool
	}{
		{
			name:            "user is member of trusted org",
			onlyOrgMembers:  false,
			user:            "test",
			org:             "kubernetes",
			repo:            "kubernetes",
			expectedTrusted: true,
		},
		{
			name:            "user is member of trusted org (only org members enabled)",
			onlyOrgMembers:  true,
			user:            "test",
			org:             "kubernetes",
			repo:            "kubernetes",
			expectedTrusted: true,
		},
		{
			name:            "user is collaborator",
			onlyOrgMembers:  false,
			user:            "test-collaborator",
			org:             "kubernetes",
			repo:            "kubernetes",
			expectedTrusted: true,
		},
		{
			name:            "user is collaborator (only org members enabled)",
			onlyOrgMembers:  true,
			user:            "test-collaborator",
			org:             "kubernetes",
			repo:            "kubernetes",
			expectedTrusted: false,
		},
		{
			name:            "user is trusted org member",
			onlyOrgMembers:  false,
			trustedOrg:      "kubernetes",
			user:            "test",
			org:             "kubernetes-sigs",
			repo:            "test",
			expectedTrusted: true,
		},
		{
			name:            "user is not org member",
			onlyOrgMembers:  false,
			user:            "test-2",
			org:             "kubernetes",
			repo:            "kubernetes",
			expectedTrusted: false,
		},
		{
			name:            "user is not org member or trusted org member",
			onlyOrgMembers:  false,
			trustedOrg:      "kubernetes-sigs",
			user:            "test-2",
			org:             "kubernetes",
			repo:            "kubernetes",
			expectedTrusted: false,
		},
		{
			name:            "user is not org member or trusted org member, onlyOrgMembers true",
			onlyOrgMembers:  true,
			trustedOrg:      "kubernetes-sigs",
			user:            "test-2",
			org:             "kubernetes",
			repo:            "kubernetes",
			expectedTrusted: false,
		},
		{
			name:            "Self as bot is trusted",
			user:            "k8s-ci-robot",
			expectedTrusted: true,
		},
		{
			name:            "github-app[bot] is in trusted list",
			user:            "github-app[bot]",
			trustedApps:     []string{"github-app"},
			expectedTrusted: true,
		},
		{
			name:            "github-app[bot] is not in trusted list",
			user:            "github-app[bot]",
			trustedApps:     []string{"other-app"},
			expectedTrusted: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeSCMClient := fake2.SCMClient{}
			fakeSCMClient.OrgMembers = map[string][]string{
				"kubernetes": {"test"},
			}
			fakeSCMClient.Collaborators = []string{"test-collaborator"}

			triggerPlugin := plugins.Trigger{
				TrustedOrg:     tc.trustedOrg,
				TrustedApps:    tc.trustedApps,
				OnlyOrgMembers: tc.onlyOrgMembers,
			}

			trustedResponse, err := TrustedUser(&fakeSCMClient, &triggerPlugin, tc.user, tc.org, tc.repo)
			if err != nil {
				t.Errorf("For case %s, didn't expect error from TrustedUser: %v", tc.name, err)
			}
			if trustedResponse != tc.expectedTrusted {
				t.Errorf("For case %s, expect trusted: %v, but got: %v", tc.name, tc.expectedTrusted, trustedResponse)
			}
		})
	}
}
