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

package lifecycle

import (
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
)

type fakeClientReopen struct {
	commented bool
	open      bool
}

func (c *fakeClientReopen) CreateComment(owner, repo string, number int, pr bool, comment string) error {
	c.commented = true
	return nil
}

func (c *fakeClientReopen) ReopenIssue(owner, repo string, number int) error {
	c.open = true
	return nil
}

func (c *fakeClientReopen) ReopenPR(owner, repo string, number int) error {
	c.open = true
	return nil
}

func (c *fakeClientReopen) IsCollaborator(owner, repo, login string) (bool, error) {
	if login == "collaborator" {
		return true, nil
	}
	return false, nil
}

func TestReopenComment(t *testing.T) {
	var testcases = []struct {
		name          string
		action        scm.Action
		state         string
		body          string
		commenter     string
		shouldReopen  bool
		shouldComment bool
	}{
		{
			name:          "non-open comment",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "does not matter",
			commenter:     "random-person",
			shouldReopen:  false,
			shouldComment: false,
		},
		{
			name:          "re-open by author",
			action:        scm.ActionCreate,
			state:         "closed",
			body:          "/reopen",
			commenter:     "author",
			shouldReopen:  true,
			shouldComment: true,
		},
		{
			name:          "re-open by collaborator",
			action:        scm.ActionCreate,
			state:         "closed",
			body:          "/reopen",
			commenter:     "collaborator",
			shouldReopen:  true,
			shouldComment: true,
		},
		{
			name:          "re-open by collaborator, trailing space.",
			action:        scm.ActionCreate,
			state:         "closed",
			body:          "/reopen \r",
			commenter:     "collaborator",
			shouldReopen:  true,
			shouldComment: true,
		},
		{
			name:          "re-open edited by author",
			action:        scm.ActionUpdate,
			state:         "closed",
			body:          "/reopen",
			commenter:     "author",
			shouldReopen:  false,
			shouldComment: false,
		},
		{
			name:          "open by author on already open issue",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "/reopen",
			commenter:     "author",
			shouldReopen:  false,
			shouldComment: false,
		},
		{
			name:          "re-open by non-collaborator, cannot reopen",
			action:        scm.ActionCreate,
			state:         "closed",
			body:          "/reopen",
			commenter:     "non-collaborator",
			shouldReopen:  false,
			shouldComment: true,
		},
		{
			name:          "re-open by author with prefix",
			action:        scm.ActionCreate,
			state:         "closed",
			body:          "/lh-reopen",
			commenter:     "author",
			shouldReopen:  true,
			shouldComment: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fc := &fakeClientReopen{}
			e := &scmprovider.GenericCommentEvent{
				Action:      tc.action,
				IssueState:  tc.state,
				Body:        tc.body,
				Author:      scm.User{Login: tc.commenter},
				Number:      5,
				IssueAuthor: scm.User{Login: "author"},
			}
			if err := handleReopen(fc, logrus.WithField("plugin", "fake-reopen"), e); err != nil {
				t.Fatalf("For case %s, didn't expect error from handle: %v", tc.name, err)
			}
			if tc.shouldReopen && !fc.open {
				t.Errorf("For case %s, should have reopened but didn't.", tc.name)
			} else if !tc.shouldReopen && fc.open {
				t.Errorf("For case %s, should not have reopened but did.", tc.name)
			}
			if tc.shouldComment && !fc.commented {
				t.Errorf("For case %s, should have commented but didn't.", tc.name)
			} else if !tc.shouldComment && fc.commented {
				t.Errorf("For case %s, should not have commented but did.", tc.name)
			}
		})
	}
}
