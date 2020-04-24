/*
Copyright 2016 The Kubernetes Authors.

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
	"errors"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
)

type fakeClientClose struct {
	commented      bool
	closed         bool
	AssigneesAdded []string
	labels         []string
}

func (c *fakeClientClose) CreateComment(owner, repo string, number int, pr bool, comment string) error {
	c.commented = true
	return nil
}

func (c *fakeClientClose) CloseIssue(owner, repo string, number int) error {
	c.closed = true
	return nil
}

func (c *fakeClientClose) ClosePR(owner, repo string, number int) error {
	c.closed = true
	return nil
}

func (c *fakeClientClose) IsCollaborator(owner, repo, login string) (bool, error) {
	if login == "collaborator" {
		return true, nil
	}
	return false, nil
}

func (c *fakeClientClose) GetIssueLabels(owner, repo string, number int, pr bool) ([]*scm.Label, error) {
	var labels []*scm.Label
	for _, l := range c.labels {
		if l == "error" {
			return nil, errors.New("issue label 500")
		}
		labels = append(labels, &scm.Label{Name: l})
	}
	return labels, nil
}

func TestCloseComment(t *testing.T) {
	var testcases = []struct {
		name          string
		action        scm.Action
		state         string
		body          string
		commenter     string
		labels        []string
		shouldClose   bool
		shouldComment bool
	}{
		{
			name:          "non-close comment",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "uh oh",
			commenter:     "random-person",
			shouldClose:   false,
			shouldComment: false,
		},
		{
			name:          "close by author",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "/close",
			commenter:     "author",
			shouldClose:   true,
			shouldComment: true,
		},
		{
			name:          "close by author, trailing space.",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "/close \r",
			commenter:     "author",
			shouldClose:   true,
			shouldComment: true,
		},
		{
			name:          "close by collaborator",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "/close",
			commenter:     "collaborator",
			shouldClose:   true,
			shouldComment: true,
		},
		{
			name:          "close edited by author",
			action:        scm.ActionUpdate,
			state:         "open",
			body:          "/close",
			commenter:     "author",
			shouldClose:   false,
			shouldComment: false,
		},
		{
			name:          "close by author on closed issue",
			action:        scm.ActionCreate,
			state:         "closed",
			body:          "/close",
			commenter:     "author",
			shouldClose:   false,
			shouldComment: false,
		},
		{
			name:          "close by non-collaborator on active issue, cannot close",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "/close",
			commenter:     "non-collaborator",
			shouldClose:   false,
			shouldComment: true,
		},
		{
			name:          "close by non-collaborator on stale issue",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "/close",
			commenter:     "non-collaborator",
			labels:        []string{"lifecycle/stale"},
			shouldClose:   true,
			shouldComment: true,
		},
		{
			name:          "close by non-collaborator on rotten issue",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "/close",
			commenter:     "non-collaborator",
			labels:        []string{"lifecycle/rotten"},
			shouldClose:   true,
			shouldComment: true,
		},
		{
			name:          "cannot close stale issue by non-collaborator when list issue fails",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "/close",
			commenter:     "non-collaborator",
			labels:        []string{"error"},
			shouldClose:   false,
			shouldComment: true,
		},
		{
			name:          "close by author with prefix",
			action:        scm.ActionCreate,
			state:         "open",
			body:          "/lh-close",
			commenter:     "author",
			shouldClose:   true,
			shouldComment: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fc := &fakeClientClose{labels: tc.labels}
			e := &scmprovider.GenericCommentEvent{
				Action:      tc.action,
				IssueState:  tc.state,
				Body:        tc.body,
				Author:      scm.User{Login: tc.commenter},
				Number:      5,
				IssueAuthor: scm.User{Login: "author"},
			}
			if err := handleClose(fc, logrus.WithField("plugin", "fake-close"), e); err != nil {
				t.Fatalf("For case %s, didn't expect error from handle: %v", tc.name, err)
			}
			if tc.shouldClose && !fc.closed {
				t.Errorf("For case %s, should have closed but didn't.", tc.name)
			} else if !tc.shouldClose && fc.closed {
				t.Errorf("For case %s, should not have closed but did.", tc.name)
			}
			if tc.shouldComment && !fc.commented {
				t.Errorf("For case %s, should have commented but didn't.", tc.name)
			} else if !tc.shouldComment && fc.commented {
				t.Errorf("For case %s, should not have commented but did.", tc.name)
			}
		})
	}
}
