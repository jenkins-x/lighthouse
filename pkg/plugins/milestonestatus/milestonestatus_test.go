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

package milestonestatus

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

func formatLabels(labels ...string) []string {
	r := []string{}
	for _, l := range labels {
		r = append(r, fmt.Sprintf("%s/%s#%d:%s", "org", "repo", 1, l))
	}
	return r
}

func TestMilestoneStatus(t *testing.T) {
	type testCase struct {
		name              string
		body              string
		commenter         string
		expectedNewLabels []string
		shouldComment     bool
		noRepoMaintainer  bool
	}
	testcases := []testCase{
		{
			name:              "Don't label when non sig-lead user approves",
			body:              "/status approved-for-milestone",
			expectedNewLabels: []string{},
			commenter:         "sig-follow",
			shouldComment:     true,
		},
		{
			name:              "Don't label when non sig-lead user marks in progress",
			body:              "/status in-progress",
			expectedNewLabels: []string{},
			commenter:         "sig-follow",
			shouldComment:     true,
		},
		{
			name:              "Don't label when non sig-lead user marks in review",
			body:              "/status in-review",
			expectedNewLabels: []string{},
			commenter:         "sig-follow",
			shouldComment:     true,
		},
		{
			name:              "Label when sig-lead user approves",
			body:              "/status approved-for-milestone",
			expectedNewLabels: []string{"status/approved-for-milestone"},
			commenter:         "sig-lead",
			shouldComment:     false,
		},
		{
			name:              "Label when sig-lead user marks in progress",
			body:              "/status in-progress",
			expectedNewLabels: []string{"status/in-progress"},
			commenter:         "sig-lead",
			shouldComment:     false,
		},
		{
			name:              "Label when sig-lead user marks in review",
			body:              "/status in-review",
			expectedNewLabels: []string{"status/in-review"},
			commenter:         "sig-lead",
			shouldComment:     false,
		},
		{
			name:              "Label when sig-lead user marks in review with prefix",
			body:              "/lh-status in-review",
			expectedNewLabels: []string{"status/in-review"},
			commenter:         "sig-lead",
			shouldComment:     false,
		},
		{
			name:              "Don't label when sig-lead user marks invalid status",
			body:              "/status in-valid",
			expectedNewLabels: []string{},
			commenter:         "sig-lead",
			shouldComment:     false,
		},
		{
			name:              "Don't label when sig-lead user marks empty status",
			body:              "/status ",
			expectedNewLabels: []string{},
			commenter:         "sig-lead",
			shouldComment:     false,
		},
		{
			name:              "Use default maintainer team when none is specified",
			body:              "/status in-progress",
			expectedNewLabels: []string{"status/in-progress"},
			commenter:         "default-sig-lead",
			shouldComment:     false,
			noRepoMaintainer:  true,
		},
		{
			name:              "Don't use default maintainer team when one is specified",
			body:              "/status in-progress",
			expectedNewLabels: []string{},
			commenter:         "default-sig-lead",
			shouldComment:     true,
			noRepoMaintainer:  false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeScmClient, fc := fake.NewDefault()
			fakeClient := scmprovider.ToTestClient(fakeScmClient)

			e := &scmprovider.GenericCommentEvent{
				Action: scm.ActionCreate,
				Body:   tc.body,
				Number: 1,
				Repo:   scm.Repository{Namespace: "org", Name: "repo"},
				Author: scm.User{Login: tc.commenter},
			}
			repoMilestone := map[string]plugins.Milestone{"": {MaintainersID: 0}}
			if !tc.noRepoMaintainer {
				repoMilestone["org/repo"] = plugins.Milestone{MaintainersID: 42}
			}
			agent := plugins.Agent{
				SCMProviderClient: &fakeClient.Client,
				Logger:            logrus.WithField("plugin", pluginName),
				PluginConfig: &plugins.Configuration{
					RepoMilestone: repoMilestone,
				},
			}
			if err := plugin.InvokeCommandHandler(e, func(handler plugins.CommandEventHandler, e *scmprovider.GenericCommentEvent, match plugins.CommandMatch) error {
				return handler(match, agent, *e)
			}); err != nil {
				t.Fatalf("(%s): Unexpected error from handle: %v.", tc.name, err)
			}

			// Check that the correct labels were added.
			expectLabels := formatLabels(tc.expectedNewLabels...)
			sort.Strings(expectLabels)
			sort.Strings(fc.IssueLabelsAdded)
			if !reflect.DeepEqual(expectLabels, fc.IssueLabelsAdded) {
				t.Errorf("(%s): Expected issue to end with labels %q, but ended with %q.", tc.name, expectLabels, fc.IssueLabelsAdded)
			}

			// Check that a comment was left iff one should have been left.
			comments := len(fc.IssueComments[1])
			if tc.shouldComment && comments != 1 {
				t.Errorf("(%s): 1 comment should have been made, but %d comments were made.", tc.name, comments)
			} else if !tc.shouldComment && comments != 0 {
				t.Errorf("(%s): No comment should have been made, but %d comments were made.", tc.name, comments)
			}
		})
	}
}
