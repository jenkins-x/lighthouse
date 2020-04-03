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

package wip

import (
	"fmt"
	"testing"

	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/prow/labels"
)

func TestWipLabel(t *testing.T) {
	const (
		wipTitle     = "[WIP] title"
		regularTitle = "title"
	)

	var testcases = []struct {
		name          string
		title         string
		draft         bool
		hasLabel      bool
		shouldLabel   bool
		shouldUnlabel bool
	}{
		{
			name:          "regular PR, need nothing",
			title:         regularTitle,
			draft:         false,
			hasLabel:      false,
			shouldLabel:   false,
			shouldUnlabel: false,
		},
		{
			name:          "wip title PR, needs label",
			title:         wipTitle,
			draft:         false,
			hasLabel:      false,
			shouldLabel:   true,
			shouldUnlabel: false,
		},
		{
			name:          "draft PR, needs label",
			title:         regularTitle,
			draft:         true,
			hasLabel:      false,
			shouldLabel:   true,
			shouldUnlabel: false,
		},
		{
			name:          "regular PR, remove label",
			title:         regularTitle,
			draft:         false,
			hasLabel:      true,
			shouldLabel:   false,
			shouldUnlabel: true,
		},
		{
			name:          "wip title PR, nothing to do",
			title:         wipTitle,
			draft:         false,
			hasLabel:      true,
			shouldLabel:   false,
			shouldUnlabel: false,
		},
		{
			name:          "draft PR, nothing to do",
			title:         regularTitle,
			draft:         true,
			hasLabel:      true,
			shouldLabel:   false,
			shouldUnlabel: false,
		},
	}
	for _, tc := range testcases {
		fakeScmClient, fc := fake.NewDefault()
		fakeClient := scmprovider.ToTestClient(fakeScmClient)

		org, repo, number := "org", "repo", 5
		e := &event{
			org:      org,
			repo:     repo,
			number:   number,
			title:    tc.title,
			draft:    tc.draft,
			hasLabel: tc.hasLabel,
		}

		if err := handle(fakeClient, logrus.WithField("plugin", PluginName), e); err != nil {
			t.Errorf("For case %s, didn't expect error from wip: %v", tc.name, err)
			continue
		}

		fakeLabel := fmt.Sprintf("%s/%s#%d:%s", org, repo, number, labels.WorkInProgress)
		if tc.shouldLabel {
			if len(fc.PullRequestLabelsAdded) != 1 || fc.PullRequestLabelsAdded[0] != fakeLabel {
				t.Errorf("For case %s: expected to add %q Label but instead added: %v", tc.name, labels.WorkInProgress, fc.PullRequestLabelsAdded)
			}
		} else if len(fc.PullRequestLabelsAdded) > 0 {
			t.Errorf("For case %s, expected to not add %q Label but added: %v", tc.name, labels.WorkInProgress, fc.PullRequestLabelsAdded)
		}
		if tc.shouldUnlabel {
			if len(fc.PullRequestLabelsRemoved) != 1 || fc.PullRequestLabelsRemoved[0] != fakeLabel {
				t.Errorf("For case %s: expected to remove %q Label but instead removed: %v", tc.name, labels.WorkInProgress, fc.PullRequestLabelsRemoved)
			}
		} else if len(fc.PullRequestLabelsRemoved) > 0 {
			t.Errorf("For case %s, expected to not remove %q Label but removed: %v", tc.name, labels.WorkInProgress, fc.PullRequestLabelsRemoved)
		}
	}
}

func TestHasWipPrefix(t *testing.T) {
	var tests = []struct {
		title    string
		expected bool
	}{
		{
			title:    "dummy title",
			expected: false,
		},
		{
			title:    "WIP dummy title",
			expected: true,
		},
		{
			title:    "WIP: dummy title",
			expected: true,
		},
		{
			title:    "[WIP] dummy title",
			expected: true,
		},
		{
			title:    "(WIP) dummy title",
			expected: true,
		},
		{
			title:    "<WIP> dummy title",
			expected: true,
		},
		{
			title:    "wip dummy title",
			expected: true,
		},
		{
			title:    "[wip] dummy title",
			expected: true,
		},
		{
			title:    "(wip) dummy title",
			expected: true,
		},
		{
			title:    "<wip> dummy title",
			expected: true,
		},
		{
			title:    "Wipe out GCP project before reusing",
			expected: false,
		},
	}

	for _, test := range tests {
		if actual, expected := titleRegex.MatchString(test.title), test.expected; actual != expected {
			t.Errorf("for title %q, got WIP prefix match %v but got %v", test.title, actual, expected)
		}
	}
}
