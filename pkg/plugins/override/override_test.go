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

package override

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	fakeOrg   = "fake-org"
	fakeRepo  = "fake-repo"
	fakePR    = 33
	adminUser = "admin-user"
)

func TestAuthorized(t *testing.T) {
	cases := []struct {
		name     string
		user     string
		expected bool
	}{
		{
			name: "fail closed",
			user: "fail",
		},
		{
			name: "reject random",
			user: "random",
		},
		{
			name:     "accept admin",
			user:     adminUser,
			expected: true,
		},
	}

	log := logrus.WithField("plugin", pluginName)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fakeScmClient, fc := fake.NewDefault()
			fakeClient := scmprovider.ToTestClient(fakeScmClient)
			fc.UserPermissions[fakeOrg+"/"+fakeRepo] = map[string]string{
				adminUser: "admin",
			}
			if actual := authorized(&fakeClient.Client, log, fakeOrg, fakeRepo, tc.user); actual != tc.expected {
				t.Errorf("actual %t != expected %t", actual, tc.expected)
			}
		})
	}
}

func TestHandle(t *testing.T) {
	cases := []struct {
		name          string
		action        scm.Action
		issue         bool
		state         string
		comment       string
		contexts      map[string]*scm.Status
		presubmits    map[string]job.Presubmit
		user          string
		number        int
		expected      []*scm.Status
		jobs          sets.String
		checkComments []string
		err           bool
	}{
		{
			name:    "successfully override failure",
			comment: "/override broken-test",
			contexts: map[string]*scm.Status{
				"broken-test": {
					Label: "broken-test",
					State: scm.StateFailure,
				},
			},
			expected: []*scm.Status{
				{
					Label: "broken-test",
					Desc:  description(adminUser),
					State: scm.StateSuccess,
				},
			},
			checkComments: []string{"on behalf of " + adminUser},
		},
		{
			name:    "successfully override failure with prefix",
			comment: "/lh-override broken-test",
			contexts: map[string]*scm.Status{
				"broken-test": {
					Label: "broken-test",
					State: scm.StateFailure,
				},
			},
			expected: []*scm.Status{
				{
					Label: "broken-test",
					Desc:  description(adminUser),
					State: scm.StateSuccess,
				},
			},
			checkComments: []string{"on behalf of " + adminUser},
		},
		{
			name:    "successfully override pending",
			comment: "/override hung-test",
			contexts: map[string]*scm.Status{
				"hung-test": {
					Label: "hung-test",
					State: scm.StatePending,
				},
			},
			expected: []*scm.Status{
				{
					Label: "hung-test",
					Desc:  description(adminUser),
					State: scm.StateSuccess,
				},
			},
		},
		{
			name:    "comment for incorrect context",
			comment: "/override whatever-you-want",
			contexts: map[string]*scm.Status{
				"hung-test": {
					Label: "hung-test",
					State: scm.StatePending,
				},
			},
			expected: []*scm.Status{
				{
					Label: "hung-test",
					State: scm.StatePending,
				},
			},
			checkComments: []string{
				"The following unknown contexts were given", "whatever-you-want",
				"Only the following contexts were expected", "hung-context",
			},
		},
		{
			name:    "refuse override from non-admin",
			comment: "/override broken-test",
			contexts: map[string]*scm.Status{
				"broken-test": {
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
			user:          "rando",
			checkComments: []string{"unauthorized"},
			expected: []*scm.Status{
				{
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
		},
		{
			name:    "comment for override with no target",
			comment: "/override",
			contexts: map[string]*scm.Status{
				"broken-test": {
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
			user:          "rando",
			checkComments: []string{"but none was given"},
			expected: []*scm.Status{
				{
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
		},
		{
			name:    "override multiple",
			comment: "/override broken-test\n/override hung-test",
			contexts: map[string]*scm.Status{
				"broken-test": {
					Label: "broken-test",
					State: scm.StateFailure,
				},
				"hung-test": {
					Label: "hung-test",
					State: scm.StatePending,
				},
			},
			expected: []*scm.Status{
				{
					Label: "broken-test",
					Desc:  description(adminUser),
					State: scm.StateSuccess,
				},
				{
					Label: "hung-test",
					Desc:  description(adminUser),
					State: scm.StateSuccess,
				},
			},
			checkComments: []string{fmt.Sprintf("%s: broken-test, hung-test", adminUser)},
		},
		{
			name:    "ignore non-PRs",
			issue:   true,
			comment: "/override broken-test",
			contexts: map[string]*scm.Status{
				"broken-test": {
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
			expected: []*scm.Status{
				{
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
		},
		{
			name:    "ignore closed issues",
			state:   "closed",
			comment: "/override broken-test",
			contexts: map[string]*scm.Status{
				"broken-test": {
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
			expected: []*scm.Status{
				{
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
		},
		{
			name:    "ignore edits",
			action:  scm.ActionUpdate,
			comment: "/override broken-test",
			contexts: map[string]*scm.Status{
				"broken-test": {
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
			expected: []*scm.Status{
				{
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
		},
		{
			name:    "ignore random text",
			comment: "/test broken-test",
			contexts: map[string]*scm.Status{
				"broken-test": {
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
			expected: []*scm.Status{
				{
					Label: "broken-test",
					State: scm.StatePending,
				},
			},
		},
		{
			name:    "comment on get pr failure",
			number:  fakePR * 2,
			comment: "/override broken-test",
			contexts: map[string]*scm.Status{
				"broken-test": {
					Label: "broken-test",
					State: scm.StateFailure,
				},
			},
			expected: []*scm.Status{
				{
					Label: "broken-test",
					Desc:  description(adminUser),
					State: scm.StateSuccess,
				},
			},
			checkComments: []string{"Cannot get PR"},
		},
		{
			name:    "do not override passing contexts",
			comment: "/override passing-test",
			contexts: map[string]*scm.Status{
				"passing-test": {
					Label: "passing-test",
					Desc:  "preserve description",
					State: scm.StateSuccess,
				},
			},
			expected: []*scm.Status{
				{
					Label: "passing-test",
					State: scm.StateSuccess,
					Desc:  "preserve description",
				},
			},
		},
		{
			name:    "create successful prow job",
			comment: "/override prow-job",
			contexts: map[string]*scm.Status{
				"prow-job": {
					Label: "prow-job",
					Desc:  "failed",
					State: scm.StateFailure,
				},
			},
			presubmits: map[string]job.Presubmit{
				"prow-job": {
					Reporter: job.Reporter{
						Context: "prow-job",
					},
				},
			},
			jobs: sets.NewString("prow-job"),
			expected: []*scm.Status{
				{
					Label: "prow-job",
					State: scm.StateSuccess,
					Desc:  description(adminUser),
				},
			},
		},
		{
			name:    "override with explanation works",
			comment: "/override job\r\nobnoxious flake", // github ends lines with \r\n
			contexts: map[string]*scm.Status{
				"job": {
					Label: "job",
					Desc:  "failed",
					State: scm.StateFailure,
				},
			},
			expected: []*scm.Status{
				{
					Label: "job",
					Desc:  description(adminUser),
					State: scm.StateSuccess,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var event scmprovider.GenericCommentEvent
			event.Repo.Namespace = fakeOrg
			event.Repo.Name = fakeRepo
			event.Body = tc.comment
			event.Number = fakePR
			event.IsPR = !tc.issue
			if tc.user == "" {
				tc.user = adminUser
			}
			event.Author.Login = tc.user
			if tc.state == "" {
				tc.state = "open"
			}
			event.IssueState = tc.state
			if int(tc.action) == 0 {
				tc.action = scm.ActionCreate
			}
			event.Action = tc.action
			fakeScmClient, fc := fake.NewDefault()
			fakeClient := scmprovider.ToTestClient(fakeScmClient)
			fc.UserPermissions[fakeOrg+"/"+fakeRepo] = map[string]string{
				adminUser: "admin",
			}
			fc.PullRequests[fakePR] = &scm.PullRequest{
				Head: scm.PullRequestBranch{
					Sha: fakeOrg + "/" + fakeRepo,
				},
			}
			for _, v := range tc.contexts {
				fc.Statuses[fakeOrg+"/"+fakeRepo] = append(fc.Statuses[fakeOrg+"/"+fakeRepo], v)
			}

			agent := plugins.Agent{
				SCMProviderClient: &fakeClient.Client,
				Logger:            logrus.WithField("plugin", pluginName),
				Config: &config.Config{
					JobConfig: job.Config{
						// Presubmits: tc.presubmits,
					},
				},
			}
			err := plugin.InvokeCommandHandler(&event, func(handler plugins.CommandEventHandler, e *scmprovider.GenericCommentEvent, match plugins.CommandMatch) error {
				return handler(match, agent, event)
			})
			if tc.err && err == nil {
				t.Error("failed to receive an error")
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			} else {
				assert.ElementsMatch(t, fc.Statuses[fakeOrg+"/"+fakeRepo], tc.expected)
			}
		})
	}
}
