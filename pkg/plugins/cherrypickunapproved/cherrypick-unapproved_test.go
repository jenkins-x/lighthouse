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

package cherrypickunapproved

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/labels"
)

type fakeClient struct {
	// current labels
	labels []string
	// labels that are added
	added []string
	// labels that are removed
	removed []string
	// commentsAdded tracks the comments in the client
	commentsAdded map[int][]string
}

// AddLabel adds a label to the specified PR or issue
func (fc *fakeClient) AddLabel(owner, repo string, number int, label string, pr bool) error {
	fc.added = append(fc.added, label)
	fc.labels = append(fc.labels, label)
	return nil
}

// RemoveLabel removes the label from the specified PR or issue
func (fc *fakeClient) RemoveLabel(owner, repo string, number int, label string, pr bool) error {
	fc.removed = append(fc.removed, label)

	// remove from existing labels
	for k, v := range fc.labels {
		if label == v {
			fc.labels = append(fc.labels[:k], fc.labels[k+1:]...)
			break
		}
	}

	return nil
}

// GetIssueLabels gets the current labels on the specified PR or issue
func (fc *fakeClient) GetIssueLabels(owner, repo string, number int, pr bool) ([]*scm.Label, error) {
	la := []*scm.Label{}
	for _, l := range fc.labels {
		la = append(la, &scm.Label{Name: l})
	}
	return la, nil
}

// CreateComment adds and tracks a comment in the client
func (fc *fakeClient) CreateComment(owner, repo string, number int, pr bool, comment string) error {
	fc.commentsAdded[number] = append(fc.commentsAdded[number], comment)
	return nil
}

// NumComments counts the number of tracked comments
func (fc *fakeClient) NumComments() int {
	n := 0
	for _, comments := range fc.commentsAdded {
		n += len(comments)
	}
	return n
}

func (fc *fakeClient) QuoteAuthorForComment(author string) string {
	return author
}

type fakePruner struct{}

func (fp *fakePruner) PruneComments(pr bool, shouldPrune func(*scm.Comment) bool) {}

func makeFakePullRequestEvent(action scm.Action, branch string) scm.PullRequestHook {
	return scm.PullRequestHook{
		Action: action,
		PullRequest: scm.PullRequest{
			Ref: branch,
		},
	}
}

func TestCherryPickUnapprovedLabel(t *testing.T) {
	testcases := []struct {
		name          string
		branch        string
		action        scm.Action
		labels        []string
		added         []string
		removed       []string
		expectComment bool
	}{
		{
			name:          "unsupported PR action -> no-op",
			branch:        "release-1.10",
			action:        scm.ActionEdited,
			labels:        []string{},
			added:         []string{},
			removed:       []string{},
			expectComment: false,
		},
		{
			name:          "branch that does match regexp -> no-op",
			branch:        "master",
			action:        scm.ActionOpen,
			labels:        []string{},
			added:         []string{},
			removed:       []string{},
			expectComment: false,
		},
		{
			name:          "has cpUnapproved -> no-op",
			branch:        "release-1.10",
			action:        scm.ActionOpen,
			labels:        []string{labels.CpUnapproved},
			added:         []string{},
			removed:       []string{},
			expectComment: false,
		},
		{
			name:          "has both cpApproved and cpUnapproved -> remove cpUnapproved",
			branch:        "release-1.10",
			action:        scm.ActionOpen,
			labels:        []string{labels.CpApproved, labels.CpUnapproved},
			added:         []string{},
			removed:       []string{labels.CpUnapproved},
			expectComment: false,
		},
		{
			name:          "does not have any labels, PR opened against a release branch -> add cpUnapproved and comment",
			branch:        "release-1.10",
			action:        scm.ActionOpen,
			labels:        []string{},
			added:         []string{labels.CpUnapproved},
			removed:       []string{},
			expectComment: true,
		},
		{
			name:          "does not have any labels, PR reopened against a release branch -> add cpUnapproved and comment",
			branch:        "release-1.10",
			action:        scm.ActionReopen,
			labels:        []string{},
			added:         []string{labels.CpUnapproved},
			removed:       []string{},
			expectComment: true,
		},
	}

	for _, tc := range testcases {
		fc := &fakeClient{
			labels:        tc.labels,
			added:         []string{},
			removed:       []string{},
			commentsAdded: make(map[int][]string),
		}

		event := makeFakePullRequestEvent(tc.action, tc.branch)
		branchRe := regexp.MustCompile(`^release-.*$`)
		comment := "dummy cumment"
		err := handlePR(fc, logrus.WithField("plugin", "fake-cherrypick-unapproved"), &event, &fakePruner{}, branchRe, comment)
		switch {
		case err != nil:
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		case !reflect.DeepEqual(tc.added, fc.added):
			t.Errorf("%s: added %v != actual %v", tc.name, tc.added, fc.added)
		case !reflect.DeepEqual(tc.removed, fc.removed):
			t.Errorf("%s: removed %v != actual %v", tc.name, tc.removed, fc.removed)
		}

		// if we expected a comment, verify that a comment was made
		numComments := fc.NumComments()
		if tc.expectComment && numComments != 1 {
			t.Errorf("%s: expected 1 comment but received %d comments", tc.name, numComments)
		}
		if !tc.expectComment && numComments != 0 {
			t.Errorf("%s: expected no comments but received %d comments", tc.name, numComments)
		}
	}
}
