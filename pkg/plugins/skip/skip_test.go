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

package skip

import (
	"reflect"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider/fake"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/config"
)

func TestSkipStatus(t *testing.T) {
	tests := []struct {
		name string

		presubmits     []config.Presubmit
		sha            string
		event          *scmprovider.GenericCommentEvent
		prChanges      map[int][]*scm.Change
		existing       []*scm.StatusInput
		combinedStatus scm.State
		expected       []*scm.StatusInput
	}{
		{
			name: "required contexts should not be skipped regardless of their state",

			presubmits: []config.Presubmit{
				{
					Reporter: config.Reporter{
						Context: "passing-tests",
					},
				},
				{
					Reporter: config.Reporter{
						Context: "failed-tests",
					},
				},
				{
					Reporter: config.Reporter{
						Context: "pending-tests",
					},
				},
			},
			sha: "shalala",
			event: &scmprovider.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.StatusInput{
				{
					Label: "passing-tests",
					State: scm.StateSuccess,
				},
				{
					Label: "failed-tests",
					State: scm.StateFailure,
				},
				{
					Label: "pending-tests",
					State: scm.StatePending,
				},
			},

			expected: []*scm.StatusInput{
				{
					Label: "passing-tests",
					State: scm.StateSuccess,
				},
				{
					Label: "failed-tests",
					State: scm.StateFailure,
				},
				{
					Label: "pending-tests",
					State: scm.StatePending,
				},
			},
		},
		{
			name: "optional contexts that have failed or are pending should be skipped",

			presubmits: []config.Presubmit{
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "failed-tests",
					},
				},
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "pending-tests",
					},
				},
			},
			sha: "shalala",
			event: &scmprovider.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.StatusInput{
				{
					State: scm.StateFailure,
					Label: "failed-tests",
				},
				{
					State: scm.StatePending,
					Label: "pending-tests",
				},
			},

			expected: []*scm.StatusInput{
				{
					State: scm.StateSuccess,
					Desc:  "Skipped",
					Label: "failed-tests",
				},
				{
					State: scm.StateSuccess,
					Desc:  "Skipped",
					Label: "pending-tests",
				},
			},
		},
		{
			name: "optional contexts that have failed or are pending should be skipped, with prefix",

			presubmits: []config.Presubmit{
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "failed-tests",
					},
				},
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "pending-tests",
					},
				},
			},
			sha: "shalala",
			event: &scmprovider.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/lh-skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.StatusInput{
				{
					State: scm.StateFailure,
					Label: "failed-tests",
				},
				{
					State: scm.StatePending,
					Label: "pending-tests",
				},
			},

			expected: []*scm.StatusInput{
				{
					State: scm.StateSuccess,
					Desc:  "Skipped",
					Label: "failed-tests",
				},
				{
					State: scm.StateSuccess,
					Desc:  "Skipped",
					Label: "pending-tests",
				},
			},
		},
		{
			name: "optional contexts that have not posted a context should not be skipped",

			presubmits: []config.Presubmit{
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "untriggered-tests",
					},
				},
			},
			sha: "shalala",
			event: &scmprovider.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.StatusInput{},

			expected: []*scm.StatusInput{},
		},
		{
			name: "optional contexts that have succeeded should not be skipped",

			presubmits: []config.Presubmit{
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "succeeded-tests",
					},
				},
			},
			sha: "shalala",
			event: &scmprovider.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.StatusInput{
				{
					State: scm.StateSuccess,
					Label: "succeeded-tests",
				},
			},

			expected: []*scm.StatusInput{
				{
					State: scm.StateSuccess,
					Label: "succeeded-tests",
				},
			},
		},
		{
			name: "optional tests that have failed but will be handled by trigger should not be skipped",

			presubmits: []config.Presubmit{
				{
					Optional:     true,
					Trigger:      `(?m)^/test (?:.*? )?job(?: .*?)?$`,
					RerunCommand: "/test job",
					Reporter: config.Reporter{
						Context: "failed-tests",
					},
				},
			},
			sha: "shalala",
			event: &scmprovider.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body: `/skip
/test job`,
				Number: 1,
				Repo:   scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.StatusInput{
				{
					State: scm.StateFailure,
					Label: "failed-tests",
				},
			},
			expected: []*scm.StatusInput{
				{
					State: scm.StateFailure,
					Label: "failed-tests",
				},
			},
		},
		{
			name: "no contexts should be skipped if the combined status is success",

			presubmits: []config.Presubmit{
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "failed-tests",
					},
				},
				{
					Optional: true,
					Reporter: config.Reporter{
						Context: "pending-tests",
					},
				},
			},
			sha:            "shalala",
			combinedStatus: scm.StateSuccess,
			event: &scmprovider.GenericCommentEvent{
				IsPR:       true,
				IssueState: "open",
				Action:     scm.ActionCreate,
				Body:       "/skip",
				Number:     1,
				Repo:       scm.Repository{Namespace: "org", Name: "repo"},
			},
			existing: []*scm.StatusInput{
				{
					State: scm.StateFailure,
					Label: "failed-tests",
				},
				{
					State: scm.StatePending,
					Label: "pending-tests",
				},
			},
			expected: []*scm.StatusInput{
				{
					State: scm.StateFailure,
					Label: "failed-tests",
				},
				{
					State: scm.StatePending,
					Label: "pending-tests",
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := config.SetPresubmitRegexes(test.presubmits); err != nil {
				t.Fatalf("%s: could not set presubmit regexes: %v", test.name, err)
			}

			fspc := &fake.SCMClient{
				IssueComments: make(map[int][]*scm.Comment),
				PullRequests: map[int]*scm.PullRequest{
					test.event.Number: {
						Head: scm.PullRequestBranch{
							Sha: test.sha,
						},
					},
				},
				PullRequestChanges: test.prChanges,
				CreatedStatuses: map[string][]*scm.StatusInput{
					test.sha: test.existing,
				},
				CombinedStatuses: map[string]*scm.CombinedStatus{
					test.sha: {
						State:    test.combinedStatus,
						Statuses: scm.ConvertStatusInputsToStatuses(test.existing),
					},
				},
			}
			l := logrus.WithField("plugin", pluginName)

			if err := handle(fspc, l, test.event, test.presubmits, true); err != nil {
				t.Fatalf("%s: unexpected error: %v", test.name, err)
			}

			// Check that the correct statuses have been updated.
			created := fspc.CreatedStatuses[test.sha]
			if len(test.expected) != len(created) {
				t.Fatalf("%s: status mismatch: expected:\n%+v\ngot:\n%+v", test.name, test.expected, created)
			}
			for _, got := range created {
				var found bool
				for _, exp := range test.expected {
					if exp.Label == got.Label {
						found = true
						if !reflect.DeepEqual(exp, got) {
							t.Errorf("%s: expected status: %v, got: %v", test.name, exp, got)
						}
					}
				}
				if !found {
					t.Errorf("%s: expected context %q in the results: %v", test.name, got.Label, created)
					break
				}
			}
		})
	}
}
