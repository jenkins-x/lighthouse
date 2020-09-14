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

package approve

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"sigs.k8s.io/yaml"

	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/plugins/approve/approvers"
	"github.com/jenkins-x/lighthouse/pkg/repoowners"
	"k8s.io/apimachinery/pkg/util/sets"
)

const prNumber = 1

// TestPluginConfig validates that there are no duplicate repos in the approve plugin config.
func TestPluginConfig(t *testing.T) {
	// TODO
	t.SkipNow()

	pa := &plugins.ConfigAgent{}

	b, err := ioutil.ReadFile("../../plugins.yaml")
	if err != nil {
		t.Fatalf("Failed to read plugin config: %v.", err)
	}
	np := &plugins.Configuration{}
	if err := yaml.Unmarshal(b, np); err != nil {
		t.Fatalf("Failed to unmarshal plugin config: %v.", err)
	}
	pa.Set(np)

	orgs := map[string]bool{}
	repos := map[string]bool{}
	for _, config := range pa.Config().Approve {
		for _, entry := range config.Repos {
			if strings.Contains(entry, "/") {
				if repos[entry] {
					t.Errorf("The repo %q is duplicated in the 'approve' plugin configuration.", entry)
				}
				repos[entry] = true
			} else {
				if orgs[entry] {
					t.Errorf("The org %q is duplicated in the 'approve' plugin configuration.", entry)
				}
				orgs[entry] = true
			}
		}
	}
}

func newTestComment(user, body string) *scm.Comment {
	return &scm.Comment{Author: scm.User{Login: user}, Body: body}
}

func newTestCommentTime(t time.Time, user, body string) *scm.Comment {
	c := newTestComment(user, body)
	c.Created = t
	return c
}

func newTestReview(user, body string, state string) *scm.Review {
	return &scm.Review{Author: scm.User{Login: user}, Body: body, State: state}
}

func newTestReviewTime(t time.Time, user, body string, state string) *scm.Review {
	r := newTestReview(user, body, state)
	r.Created = t
	return r
}

func newFakeSCMProviderClient(hasLabel, humanApproved, labelComments bool, files []string, comments []*scm.Comment, reviews []*scm.Review, testBotName string) (*scmprovider.TestClient, *fake.Data) {
	labels := []string{"org/repo#1:lgtm"}
	if hasLabel {
		labels = append(labels, fmt.Sprintf("org/repo#%v:approved", prNumber))
	}
	events := []*scm.ListedIssueEvent{
		{
			Event: scmprovider.IssueActionLabeled,
			Label: scm.Label{Name: "approved"},
			Actor: scm.User{Login: "k8s-merge-robot"},
		},
	}
	if humanApproved {
		events = append(
			events,
			&scm.ListedIssueEvent{
				Event:   scmprovider.IssueActionLabeled,
				Label:   scm.Label{Name: "approved"},
				Actor:   scm.User{Login: "human"},
				Created: time.Now(),
			},
		)
	}
	var changes []*scm.Change
	for _, file := range files {
		changes = append(changes, &scm.Change{Path: file})
	}
	var fakeScmClient *scm.Client
	var fc *fake.Data
	var fakeClient *scmprovider.TestClient

	if labelComments {
		fakeClient = scmprovider.NewTestClientForLabelsInComments()
		fc = fakeClient.Data
	} else {
		fakeScmClient, fc = fake.NewDefault()
		fakeClient = scmprovider.ToTestClient(fakeScmClient)
	}
	fc.RepoLabelsExisting = nil
	fc.PullRequestLabelsAdded = labels
	fc.PullRequestChanges[prNumber] = changes
	fc.PullRequestComments[prNumber] = comments
	fc.IssueEvents[prNumber] = events
	fc.Reviews[prNumber] = reviews
	return fakeClient, fc
}

type fakeRepo struct {
	approvers, leafApprovers map[string]sets.String
	approverOwners           map[string]string
}

func (fr fakeRepo) Approvers(path string) sets.String {
	return fr.approvers[path]
}
func (fr fakeRepo) LeafApprovers(path string) sets.String {
	return fr.leafApprovers[path]
}
func (fr fakeRepo) FindApproverOwnersForFile(path string) string {
	return fr.approverOwners[path]
}
func (fr fakeRepo) IsNoParentOwners(path string) bool {
	return false
}

func TestHandle(t *testing.T) {
	// This function does not need to test IsApproved, that is tested in approvers/approvers_test.go.

	testBotName := "k8s-ci-robot"

	// includes tests with mixed case usernames
	// includes tests with stale notifications
	tests := []struct {
		name          string
		branch        string
		prBody        string
		hasLabel      bool
		humanApproved bool
		files         []string
		comments      []*scm.Comment
		reviews       []*scm.Review

		selfApprove         bool
		needsIssue          bool
		lgtmActsAsApprove   bool
		reviewActsAsApprove bool
		githubLinkURL       *url.URL

		expectDelete    bool
		expectComment   bool
		expectedComment string
		expectToggle    bool
		labelComments   bool
	}{

		// breaking cases
		// case: /approve in PR body
		{
			name:                "initial notification (approved)",
			hasLabel:            false,
			files:               []string{"c/c.go"},
			comments:            []*scm.Comment{},
			reviews:             []*scm.Review{},
			selfApprove:         true,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **APPROVED**

This pull-request has been approved by: *[cjwagner](# "Author self-approved")*

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[c/OWNERS](https://github.com/org/repo/blob/master/c/OWNERS)~~ [cjwagner]

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":[]} -->`,
		},
		{
			name:                "initial notification (unapproved)",
			hasLabel:            false,
			files:               []string{"c/c.go"},
			comments:            []*scm.Comment{},
			reviews:             []*scm.Review{},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  false,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by:
To complete the [pull request process](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process), please assign **cjwagner**
You can assign the PR to them by writing ` + "`/assign @cjwagner`" + ` in a comment when ready.

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

<details open>
Needs approval from an approver in each of these files:

- **[c/OWNERS](https://github.com/org/repo/blob/master/c/OWNERS)**

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":["cjwagner"]} -->`,
		},
		{
			name:                "no-issue comment",
			hasLabel:            false,
			files:               []string{"a/a.go"},
			comments:            []*scm.Comment{newTestComment("Alice", "stuff\n/approve no-issue \nmore stuff")},
			reviews:             []*scm.Review{},
			selfApprove:         false,
			needsIssue:          true,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **APPROVED**

This pull-request has been approved by: *[Alice](<> "Approved")*

Associated issue requirement bypassed by: *[Alice](<> "Approved")*

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)~~ [Alice]

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":[]} -->`,
		},
		{
			name:                "issue provided in PR body",
			prBody:              "some changes that fix #42.\n/assign",
			hasLabel:            false,
			files:               []string{"a/a.go"},
			comments:            []*scm.Comment{newTestComment("Alice", "stuff\n/approve")},
			reviews:             []*scm.Review{},
			selfApprove:         false,
			needsIssue:          true,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **APPROVED**

This pull-request has been approved by: *[Alice](<> "Approved")*

Associated issue: *#42*

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)~~ [Alice]

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":[]} -->`,
		},
		{
			name:     "non-implicit self approve no-issue",
			hasLabel: false,
			files:    []string{"a/a.go", "c/c.go"},
			comments: []*scm.Comment{
				newTestComment("ALIcE", "stuff\n/approve"),
				newTestComment("cjwagner", "stuff\n/approve no-issue"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         false,
			needsIssue:          true,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:    false,
			expectToggle:    true,
			expectComment:   true,
			expectedComment: "",
		},
		{
			name:     "implicit self approve, missing issue",
			hasLabel: false,
			files:    []string{"a/a.go", "c/c.go"},
			comments: []*scm.Comment{
				newTestComment("ALIcE", "stuff\n/approve"),
				newTestCommentTime(time.Now(), "k8s-ci-robot", `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by: *[ALIcE](<> "Approved")*, *[cjwagner](# "Author self-approved")*

*No associated issue*. Update pull-request body to add a reference to an issue, or get approval with `+"`/approve no-issue`"+`

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)~~ [ALIcE]
- ~~[c/OWNERS](https://github.com/org/repo/blob/master/c/OWNERS)~~ [cjwagner]

Approvers can indicate their approval by writing `+"`/approve`"+` in a comment
Approvers can cancel approval by writing `+"`/approve cancel`"+` in a comment
</details>
<!-- META={"approvers":[]} -->`),
			},
			reviews:             []*scm.Review{},
			selfApprove:         true,
			needsIssue:          true,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  false,
			expectComment: false,
		},
		{
			name:     "remove approval with /approve cancel",
			hasLabel: true,
			files:    []string{"a/a.go"},
			comments: []*scm.Comment{
				newTestComment("Alice", "/approve no-issue"),
				newTestComment("k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **APPROVED**\n\nblah"),
				newTestComment("Alice", "stuff\n/approve cancel \nmore stuff"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         true, // no-op test
			needsIssue:          true,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by: *[cjwagner](# "Author self-approved")*
To complete the [pull request process](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process), please assign **alice**
You can assign the PR to them by writing ` + "`/assign @alice`" + ` in a comment when ready.

*No associated issue*. Update pull-request body to add a reference to an issue, or get approval with ` + "`/approve no-issue`" + `

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

<details open>
Needs approval from an approver in each of these files:

- **[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)**

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":["alice"]} -->`,
		},
		{
			name:     "remove approval with /approve cancel using comment",
			hasLabel: true,
			files:    []string{"a/a.go"},
			comments: []*scm.Comment{
				newTestComment("Alice", "/approve no-issue"),
				newTestComment("k8s-ci-robot", scmprovider.CreateLabelComment([]string{"approved"})),
				newTestComment("k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **APPROVED**\n\nblah"),
				newTestComment("Alice", "stuff\n/approve cancel \nmore stuff"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         true, // no-op test
			needsIssue:          true,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			labelComments: true,
			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by: *[cjwagner](# "Author self-approved")*
To complete the [pull request process](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process), please assign **alice**
You can assign the PR to them by writing ` + "`/assign @alice`" + ` in a comment when ready.

*No associated issue*. Update pull-request body to add a reference to an issue, or get approval with ` + "`/approve no-issue`" + `

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

<details open>
Needs approval from an approver in each of these files:

- **[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)**

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":["alice"]} -->`,
		},
		{
			name:     "remove approval after sync",
			prBody:   "Changes the thing.\n fixes #42",
			hasLabel: true,
			files:    []string{"a/a.go", "b/b.go"},
			comments: []*scm.Comment{
				newTestComment("bOb", "stuff\n/approve \nblah"),
				newTestComment("k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **APPROVED**\n\nblah"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         true, // no-op test
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
		},
		{
			name:     "cancel implicit self approve",
			prBody:   "Changes the thing.\n fixes #42",
			hasLabel: true,
			files:    []string{"c/c.go"},
			comments: []*scm.Comment{
				newTestComment("k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **APPROVED**\n\nblah"),
				newTestCommentTime(time.Now(), "CJWagner", "stuff\n/approve cancel \nmore stuff"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         true,
			needsIssue:          true,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
		},
		{
			name:     "cancel implicit self approve (with lgtm-after-commit message)",
			prBody:   "Changes the thing.\n fixes #42",
			hasLabel: true,
			files:    []string{"c/c.go"},
			comments: []*scm.Comment{
				newTestComment("k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **APPROVED**\n\nblah"),
				newTestCommentTime(time.Now(), "CJWagner", "/lgtm cancel //PR changed after LGTM, removing LGTM."),
			},
			reviews:             []*scm.Review{},
			selfApprove:         true,
			needsIssue:          true,
			lgtmActsAsApprove:   true,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
		},
		{
			name:     "up to date, poked by pr sync",
			prBody:   "Finally fixes kubernetes/kubernetes#1\n",
			hasLabel: true,
			files:    []string{"a/a.go", "a/aa.go"},
			comments: []*scm.Comment{
				newTestComment("alice", "stuff\n/approve\nblah"),
				newTestCommentTime(time.Now(), "k8s-ci-robot", `[APPROVALNOTIFIER] This PR is **APPROVED**

This pull-request has been approved by: *[alice](<> "Approved")*

Associated issue: *#1*

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)~~ [alice]

Approvers can indicate their approval by writing `+"`/approve`"+` in a comment
Approvers can cancel approval by writing `+"`/approve cancel`"+` in a comment
</details>
<!-- META={"approvers":[]} -->`),
			},
			reviews:             []*scm.Review{},
			selfApprove:         false,
			needsIssue:          true,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  false,
			expectComment: false,
		},
		{
			name:     "out of date, poked by pr sync",
			prBody:   "Finally fixes kubernetes/kubernetes#1\n",
			hasLabel: false,
			files:    []string{"a/a.go", "a/aa.go"}, // previous commits may have been ["b/b.go"]
			comments: []*scm.Comment{
				newTestComment("alice", "stuff\n/approve\nblah"),
				newTestCommentTime(time.Now(), "k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **NOT APPROVED**\n\nblah"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         false,
			needsIssue:          true,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
		},
		{
			name:          "human added approve",
			hasLabel:      true,
			humanApproved: true,
			files:         []string{"a/a.go"},
			comments: []*scm.Comment{
				newTestComment("k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **NOT APPROVED**\n\nblah"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  false,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **APPROVED**

Approval requirements bypassed by manually added approval.

This pull-request has been approved by:

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- **[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)**

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":["alice"]} -->`,
		},
		{
			name:     "lgtm means approve",
			prBody:   "This is a great PR that will fix\nlots of things!",
			hasLabel: false,
			files:    []string{"a/a.go", "a/aa.go"},
			comments: []*scm.Comment{
				newTestComment("k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **NOT APPROVED**\n\nblah"),
				newTestCommentTime(time.Now(), "alice", "stuff\n/lgtm\nblah"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   true,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
		},
		{
			name:     "lgtm means approve using comment",
			prBody:   "This is a great PR that will fix\nlots of things!",
			hasLabel: false,
			files:    []string{"a/a.go", "a/aa.go"},
			comments: []*scm.Comment{
				newTestComment("k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **NOT APPROVED**\n\nblah"),
				newTestCommentTime(time.Now(), "alice", "stuff\n/lgtm\nblah"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   true,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},
			labelComments:       true,

			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
		},
		{
			name:     "lgtm does not mean approve",
			prBody:   "This is a great PR that will fix\nlots of things!",
			hasLabel: false,
			files:    []string{"a/a.go", "a/aa.go"},
			comments: []*scm.Comment{
				newTestComment("k8s-ci-robot", `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by:
To complete the [pull request process](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process), please assign **alice**
You can assign the PR to them by writing `+"`/assign @alice`"+` in a comment when ready.

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

<details open>
Needs approval from an approver in each of these files:

- **[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)**

Approvers can indicate their approval by writing `+"`/approve`"+` in a comment
Approvers can cancel approval by writing `+"`/approve cancel`"+` in a comment
</details>
<!-- META={"approvers":["alice"]} -->`),
				newTestCommentTime(time.Now(), "alice", "stuff\n/lgtm\nblah"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  false,
			expectComment: false,
		},
		{
			name:                "approve in review body with empty state",
			hasLabel:            false,
			files:               []string{"a/a.go"},
			comments:            []*scm.Comment{},
			reviews:             []*scm.Review{newTestReview("Alice", "stuff\n/approve", "")},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **APPROVED**

This pull-request has been approved by: *[Alice](<> "Approved")*

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)~~ [Alice]

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":[]} -->`,
		},
		{
			name:                "approved review but reviewActsAsApprove disabled",
			hasLabel:            false,
			files:               []string{"c/c.go"},
			comments:            []*scm.Comment{},
			reviews:             []*scm.Review{newTestReview("cjwagner", "stuff", scm.ReviewStateApproved)},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  false,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by:
To complete the [pull request process](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process), please assign **cjwagner**
You can assign the PR to them by writing ` + "`/assign @cjwagner`" + ` in a comment when ready.

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

<details open>
Needs approval from an approver in each of these files:

- **[c/OWNERS](https://github.com/org/repo/blob/master/c/OWNERS)**

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":["cjwagner"]} -->`,
		},
		{
			name:                "approved review with reviewActsAsApprove enabled",
			hasLabel:            false,
			files:               []string{"a/a.go"},
			comments:            []*scm.Comment{},
			reviews:             []*scm.Review{newTestReview("Alice", "stuff", scm.ReviewStateApproved)},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: true,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **APPROVED**

This pull-request has been approved by: *[Alice](<> "Approved")*

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)~~ [Alice]

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":[]} -->`,
		},
		{
			name:     "reviews in non-approving state (should not approve)",
			hasLabel: false,
			files:    []string{"c/c.go"},
			comments: []*scm.Comment{},
			reviews: []*scm.Review{
				newTestReview("cjwagner", "stuff", "COMMENTED"),
				newTestReview("cjwagner", "unsubmitted stuff", "PENDING"),
				newTestReview("cjwagner", "dismissed stuff", "DISMISSED"),
			},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: true,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  false,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by:
To complete the [pull request process](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process), please assign **cjwagner**
You can assign the PR to them by writing ` + "`/assign @cjwagner`" + ` in a comment when ready.

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

<details open>
Needs approval from an approver in each of these files:

- **[c/OWNERS](https://github.com/org/repo/blob/master/c/OWNERS)**

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":["cjwagner"]} -->`,
		},
		{
			name:     "review in request changes state means cancel",
			hasLabel: true,
			files:    []string{"c/c.go"},
			comments: []*scm.Comment{
				newTestCommentTime(time.Now().Add(time.Hour), "k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **APPROVED**\n\nblah"), // second
			},
			reviews: []*scm.Review{
				newTestReviewTime(time.Now(), "cjwagner", "yep", scm.ReviewStateApproved),                           // first
				newTestReviewTime(time.Now().Add(time.Hour*2), "cjwagner", "nope", scm.ReviewStateChangesRequested), // third
			},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: true,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by:
To complete the [pull request process](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process), please assign **cjwagner**
You can assign the PR to them by writing ` + "`/assign @cjwagner`" + ` in a comment when ready.

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

<details open>
Needs approval from an approver in each of these files:

- **[c/OWNERS](https://github.com/org/repo/blob/master/c/OWNERS)**

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":["cjwagner"]} -->`,
		},
		{
			name:     "dismissed review doesn't cancel prior approval",
			hasLabel: true,
			files:    []string{"a/a.go"},
			comments: []*scm.Comment{
				newTestCommentTime(time.Now().Add(time.Hour), "k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **APPROVED**\n\nblah"), // second
			},
			reviews: []*scm.Review{
				newTestReviewTime(time.Now(), "Alice", "yep", scm.ReviewStateApproved),                         // first
				newTestReviewTime(time.Now().Add(time.Hour*2), "Alice", "dismissed", scm.ReviewStateDismissed), // third
			},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: true,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  false,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **APPROVED**

This pull-request has been approved by: *[Alice](<> "Approved")*

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)~~ [Alice]

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":[]} -->`,
		},
		{
			name:     "approve cancel command supersedes earlier approved review",
			hasLabel: true,
			files:    []string{"c/c.go"},
			comments: []*scm.Comment{
				newTestCommentTime(time.Now().Add(time.Hour), "k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **APPROVED**\n\nblah"), // second
				newTestCommentTime(time.Now().Add(time.Hour*2), "cjwagner", "stuff\n/approve cancel \nmore stuff"),                  // third
			},
			reviews: []*scm.Review{
				newTestReviewTime(time.Now(), "cjwagner", "yep", scm.ReviewStateApproved), // first
			},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: true,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by:
To complete the [pull request process](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process), please assign **cjwagner**
You can assign the PR to them by writing ` + "`/assign @cjwagner`" + ` in a comment when ready.

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

<details open>
Needs approval from an approver in each of these files:

- **[c/OWNERS](https://github.com/org/repo/blob/master/c/OWNERS)**

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":["cjwagner"]} -->`,
		},
		{
			name:     "approve cancel command supersedes simultaneous approved review",
			hasLabel: false,
			files:    []string{"c/c.go"},
			comments: []*scm.Comment{},
			reviews: []*scm.Review{
				newTestReview("cjwagner", "/approve cancel", scm.ReviewStateApproved),
			},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: true,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  false,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by:
To complete the [pull request process](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process), please assign **cjwagner**
You can assign the PR to them by writing ` + "`/assign @cjwagner`" + ` in a comment when ready.

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

<details open>
Needs approval from an approver in each of these files:

- **[c/OWNERS](https://github.com/org/repo/blob/master/c/OWNERS)**

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":["cjwagner"]} -->`,
		},
		{
			name:                "approve command supersedes simultaneous changes requested review",
			hasLabel:            false,
			files:               []string{"a/a.go"},
			comments:            []*scm.Comment{},
			reviews:             []*scm.Review{newTestReview("Alice", "/approve", scm.ReviewStateChangesRequested)},
			selfApprove:         false,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: true,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **APPROVED**

This pull-request has been approved by: *[Alice](<> "Approved")*

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)~~ [Alice]

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":[]} -->`,
		},
		{
			name:                "different branch, initial notification (approved)",
			branch:              "dev",
			hasLabel:            false,
			files:               []string{"c/c.go"},
			comments:            []*scm.Comment{},
			reviews:             []*scm.Review{},
			selfApprove:         true,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  false,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **APPROVED**

This pull-request has been approved by: *[cjwagner](# "Author self-approved")*

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[c/OWNERS](https://github.com/org/repo/blob/dev/c/OWNERS)~~ [cjwagner]

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":[]} -->`,
		},
		{
			name:                "different GitHub link URL",
			branch:              "dev",
			hasLabel:            false,
			files:               []string{"c/c.go"},
			comments:            []*scm.Comment{},
			reviews:             []*scm.Review{},
			selfApprove:         true,
			needsIssue:          false,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.mycorp.com"},

			expectDelete:  false,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **APPROVED**

This pull-request has been approved by: *[cjwagner](# "Author self-approved")*

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

The pull request process is described [here](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process)

<details >
Needs approval from an approver in each of these files:

- ~~[c/OWNERS](https://github.mycorp.com/org/repo/blob/dev/c/OWNERS)~~ [cjwagner]

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":[]} -->`,
		},
		{
			name:     "remove approval with /lh-approve cancel with prefix",
			hasLabel: true,
			files:    []string{"a/a.go"},
			comments: []*scm.Comment{
				newTestComment("Alice", "/lh-approve no-issue"),
				newTestComment("k8s-ci-robot", "[APPROVALNOTIFIER] This PR is **APPROVED**\n\nblah"),
				newTestComment("Alice", "stuff\n/lh-approve cancel \nmore stuff"),
			},
			reviews:             []*scm.Review{},
			selfApprove:         true, // no-op test
			needsIssue:          true,
			lgtmActsAsApprove:   false,
			reviewActsAsApprove: false,
			githubLinkURL:       &url.URL{Scheme: "https", Host: "github.com"},

			expectDelete:  true,
			expectToggle:  true,
			expectComment: true,
			expectedComment: `[APPROVALNOTIFIER] This PR is **NOT APPROVED**

This pull-request has been approved by: *[cjwagner](# "Author self-approved")*
To complete the [pull request process](https://git.k8s.io/community/contributors/guide/owners.md#the-code-review-process), please assign **alice**
You can assign the PR to them by writing ` + "`/assign @alice`" + ` in a comment when ready.

*No associated issue*. Update pull-request body to add a reference to an issue, or get approval with ` + "`/approve no-issue`" + `

The full list of commands accepted by this bot can be found [here](https://go.k8s.io/bot-commands?repo=org%2Frepo).

<details open>
Needs approval from an approver in each of these files:

- **[a/OWNERS](https://github.com/org/repo/blob/master/a/OWNERS)**

Approvers can indicate their approval by writing ` + "`/approve`" + ` in a comment
Approvers can cancel approval by writing ` + "`/approve cancel`" + ` in a comment
</details>
<!-- META={"approvers":["alice"]} -->`,
		},
	}

	fr := fakeRepo{
		approvers: map[string]sets.String{
			"a":   sets.NewString("alice"),
			"a/b": sets.NewString("alice", "bob"),
			"c":   sets.NewString("cblecker", "cjwagner"),
		},
		leafApprovers: map[string]sets.String{
			"a":   sets.NewString("alice"),
			"a/b": sets.NewString("bob"),
			"c":   sets.NewString("cblecker", "cjwagner"),
		},
		approverOwners: map[string]string{
			"a/a.go":   "a",
			"a/aa.go":  "a",
			"a/b/b.go": "a/b",
			"c/c.go":   "c",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fakeClient, fspc := newFakeSCMProviderClient(test.hasLabel, test.humanApproved, test.labelComments, test.files, test.comments, test.reviews, testBotName)
			branch := "master"
			if test.branch != "" {
				branch = test.branch
			}

			rsa := !test.selfApprove
			irs := !test.reviewActsAsApprove
			if err := handle(
				logrus.WithField("plugin", "approve"),
				fakeClient,
				fr,
				test.githubLinkURL,
				&plugins.Approve{
					Repos:               []string{"org/repo"},
					RequireSelfApproval: &rsa,
					IssueRequired:       test.needsIssue,
					LgtmActsAsApprove:   test.lgtmActsAsApprove,
					IgnoreReviewState:   &irs,
				},
				&state{
					org:       "org",
					repo:      "repo",
					branch:    branch,
					number:    prNumber,
					body:      test.prBody,
					author:    "cjwagner",
					assignees: []scm.User{{Login: "spxtr"}},
				},
			); err != nil {
				t.Errorf("[%s] Unexpected error handling event: %v.", test.name, err)
			}

			fakeLabel := fmt.Sprintf("org/repo#%v:approved", prNumber)

			if err := fakeClient.PopulateFakeLabelsFromComments("org", "repo", prNumber, fakeLabel, test.hasLabel && test.expectToggle); err != nil {
				t.Fatalf("Failure populating labels from comments: %v", err)
			}
			if test.expectDelete {
				deletedComments := 1
				if test.labelComments && test.hasLabel && test.expectToggle {
					deletedComments = 2
				}
				if len(fspc.PullRequestCommentsDeleted) != deletedComments {
					t.Errorf(
						"[%s] Expected %d notification to be deleted but %d notifications were deleted.",
						test.name,
						deletedComments,
						len(fspc.PullRequestCommentsDeleted),
					)
				}
			} else {
				if len(fspc.PullRequestCommentsDeleted) != 0 {
					t.Errorf(
						"[%s] Expected 0 notifications to be deleted but %d notification was deleted.",
						test.name,
						len(fspc.PullRequestCommentsDeleted),
					)
				}
			}
			if test.expectComment {
				addedComments := 1
				if test.labelComments && !test.hasLabel && test.expectToggle {
					addedComments = 2
				}
				if len(fspc.PullRequestCommentsAdded) != addedComments {
					t.Errorf(
						"[%s] Expected %d notification to be added but %d notifications were added.",
						test.name,
						addedComments,
						len(fspc.PullRequestCommentsAdded),
					)
				} else if expect, got := fmt.Sprintf("org/repo#%v:", prNumber)+test.expectedComment, fspc.PullRequestCommentsAdded[0]; test.expectedComment != "" && got != expect {
					t.Errorf(
						"[%s] Expected the created notification to be:\n%s\n\nbut got:\n%s\n\n",
						test.name,
						expect,
						got,
					)
				}
			} else {
				if len(fspc.PullRequestCommentsAdded) != 0 {
					t.Errorf(
						"[%s] Expected 0 notifications to be added but %d notification was added.",
						test.name,
						len(fspc.PullRequestCommentsAdded),
					)
				}
			}

			labelAdded := false
			for _, l := range fspc.PullRequestLabelsAdded {
				if l == fakeLabel {
					if labelAdded {
						t.Errorf("[%s] The approved label was applied to a PR that already had it!", test.name)
					}
					labelAdded = true
				}
			}
			if test.hasLabel {
				labelAdded = false
			}
			toggled := labelAdded
			for _, l := range fspc.PullRequestLabelsRemoved {
				if l == fakeLabel {
					if !test.hasLabel {
						t.Errorf("[%s] The approved label was removed from a PR that doesn't have it!", test.name)
					}
					toggled = true
				}
			}
			if test.expectToggle != toggled {
				t.Errorf(
					"[%s] Expected 'approved' label toggled: %t, but got %t.",
					test.name,
					test.expectToggle,
					toggled,
				)
			}
		})
	}
}

// TODO: cache approvers 'GetFilesApprovers' and 'GetCCs' since these are called repeatedly and are
// expensive.

type fakeOwnersClient struct{}

func (foc fakeOwnersClient) LoadRepoOwners(org, repo, base string) (repoowners.RepoOwner, error) {
	return fakeRepoOwners{}, nil
}

type fakeRepoOwners struct {
	fakeRepo
}

func (fro fakeRepoOwners) FindLabelsForFile(path string) sets.String {
	return sets.NewString()
}

func (fro fakeRepoOwners) FindReviewersOwnersForFile(path string) string {
	return ""
}

func (fro fakeRepoOwners) LeafReviewers(path string) sets.String {
	return sets.NewString()
}

func (fro fakeRepoOwners) Reviewers(path string) sets.String {
	return sets.NewString()
}

func (fro fakeRepoOwners) RequiredReviewers(path string) sets.String {
	return sets.NewString()
}

func TestHandleGenericComment(t *testing.T) {
	tests := []struct {
		name              string
		commentEvent      scmprovider.GenericCommentEvent
		lgtmActsAsApprove bool
		expectHandle      bool
		expectState       *state
		labelComments     bool
	}{
		{
			name: "valid approve command",
			commentEvent: scmprovider.GenericCommentEvent{
				Action: scm.ActionCreate,
				IsPR:   true,
				Body:   "/approve",
				Number: 1,
				Author: scm.User{
					Login: "author",
				},
				IssueBody: "Fix everything",
				IssueAuthor: scm.User{
					Login: "P.R. Author",
				},
			},
			expectHandle: true,
			expectState: &state{
				org:       "org",
				repo:      "repo",
				branch:    "branch",
				number:    1,
				body:      "Fix everything",
				author:    "P.R. Author",
				assignees: nil,
				htmlURL:   "",
			},
		},
		{
			name: "not comment created",
			commentEvent: scmprovider.GenericCommentEvent{
				Action: scm.ActionUpdate,
				IsPR:   true,
				Body:   "/approve",
				Number: 1,
				Author: scm.User{
					Login: "author",
				},
			},
			expectHandle: false,
		},
		{
			name: "not PR",
			commentEvent: scmprovider.GenericCommentEvent{
				Action: scm.ActionUpdate,
				IsPR:   false,
				Body:   "/approve",
				Number: 1,
				Author: scm.User{
					Login: "author",
				},
			},
			expectHandle: false,
		},
		{
			name: "closed PR",
			commentEvent: scmprovider.GenericCommentEvent{
				Action: scm.ActionCreate,
				IsPR:   true,
				Body:   "/approve",
				Number: 1,
				Author: scm.User{
					Login: "author",
				},
				IssueState: "closed",
			},
			expectHandle: false,
		},
		{
			name: "no approve command",
			commentEvent: scmprovider.GenericCommentEvent{
				Action: scm.ActionCreate,
				IsPR:   true,
				Body:   "stuff",
				Number: 1,
				Author: scm.User{
					Login: "author",
				},
			},
			expectHandle: false,
		},
		{
			name: "lgtm without lgtmActsAsApprove",
			commentEvent: scmprovider.GenericCommentEvent{
				Action: scm.ActionCreate,
				IsPR:   true,
				Body:   "/lgtm",
				Number: 1,
				Author: scm.User{
					Login: "author",
				},
			},
			expectHandle: false,
		},
		{
			name: "lgtm with lgtmActsAsApprove",
			commentEvent: scmprovider.GenericCommentEvent{
				Action: scm.ActionCreate,
				IsPR:   true,
				Body:   "/lgtm",
				Number: 1,
				Author: scm.User{
					Login: "author",
				},
			},
			lgtmActsAsApprove: true,
			expectHandle:      true,
		},
		{
			name: "valid approve command with prefix",
			commentEvent: scmprovider.GenericCommentEvent{
				Action: scm.ActionCreate,
				IsPR:   true,
				Body:   "/lh-approve",
				Number: 1,
				Author: scm.User{
					Login: "author",
				},
				IssueBody: "Fix everything",
				IssueAuthor: scm.User{
					Login: "P.R. Author",
				},
			},
			expectHandle: true,
			expectState: &state{
				org:       "org",
				repo:      "repo",
				branch:    "branch",
				number:    1,
				body:      "Fix everything",
				author:    "P.R. Author",
				assignees: nil,
				htmlURL:   "",
			},
		},
	}

	var handled bool
	var gotState *state
	handleFunc = func(log *logrus.Entry, spc scmProviderClient, repo approvers.Repo, serverURL *url.URL, opts *plugins.Approve, pr *state) error {
		gotState = pr
		handled = true
		return nil
	}
	defer func() {
		handleFunc = handle
	}()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := scm.Repository{
				Namespace: "org",
				Name:      "repo",
			}
			pr := scm.PullRequest{
				Base: scm.PullRequestBranch{
					Ref: "branch",
				},
				Number: 1,
			}
			var fakeScmClient *scm.Client
			var fspc *fake.Data
			var fakeClient *scmprovider.TestClient

			if test.labelComments {
				fakeClient = scmprovider.NewTestClientForLabelsInComments()
				fspc = fakeClient.Data
			} else {
				fakeScmClient, fspc = fake.NewDefault()
				fakeClient = scmprovider.ToTestClient(fakeScmClient)
			}
			fspc.PullRequests[1] = &pr

			test.commentEvent.Repo = repo
			config := &plugins.Configuration{}
			config.Approve = append(config.Approve, plugins.Approve{
				Repos:             []string{test.commentEvent.Repo.Namespace},
				LgtmActsAsApprove: test.lgtmActsAsApprove,
			})
			err := plugin.InvokeCommandHandler(&test.commentEvent, func(_ plugins.CommandEventHandler, e *scmprovider.GenericCommentEvent, _ plugins.CommandMatch) error {
				return handleGenericComment(
					logrus.WithField("plugin", "approve"),
					fakeClient,
					fakeOwnersClient{},
					&url.URL{
						Scheme: "https",
						Host:   "github.com",
					},
					config,
					&test.commentEvent,
				)
			})

			if test.expectHandle && !handled {
				t.Errorf("%s: expected call to handleFunc, but it wasn't called", test.name)
			}

			if !test.expectHandle && handled {
				t.Errorf("%s: expected no call to handleFunc, but it was called", test.name)
			}

			if test.expectState != nil && !reflect.DeepEqual(test.expectState, gotState) {
				t.Errorf("%s: expected PR state to equal: %#v, but got: %#v", test.name, test.expectState, gotState)
			}

			if err != nil {
				t.Errorf("%s: error calling handleGenericComment: %v", test.name, err)
			}
			handled = false
		})
	}
}

// GitHub webhooks send state as lowercase, so force it to lowercase here.
func stateToLower(s string) string {
	return strings.ToLower(string(s))
}

func TestHandleReview(t *testing.T) {
	tests := []struct {
		name                string
		reviewEvent         scm.ReviewHook
		lgtmActsAsApprove   bool
		reviewActsAsApprove bool
		expectHandle        bool
		expectState         *state
	}{
		{
			name: "approved state",
			reviewEvent: scm.ReviewHook{
				Action: scm.ActionSubmitted,
				Review: scm.Review{
					Body: "looks good",
					Author: scm.User{
						Login: "author",
					},
					State: stateToLower(scm.ReviewStateApproved),
				},
			},
			reviewActsAsApprove: true,
			expectHandle:        true,
			expectState: &state{
				org:       "org",
				repo:      "repo",
				branch:    "branch",
				number:    1,
				body:      "Fix everything",
				author:    "P.R. Author",
				assignees: nil,
				htmlURL:   "",
			},
		},
		{
			name: "changes requested state",
			reviewEvent: scm.ReviewHook{
				Action: scm.ActionSubmitted,
				Review: scm.Review{
					Body: "looks bad",
					Author: scm.User{
						Login: "author",
					},
					State: stateToLower(scm.ReviewStateChangesRequested),
				},
			},
			reviewActsAsApprove: true,
			expectHandle:        true,
		},
		{
			name: "pending review state",
			reviewEvent: scm.ReviewHook{
				Action: scm.ActionSubmitted,
				Review: scm.Review{
					Body: "looks good",
					Author: scm.User{
						Login: "author",
					},
					State: stateToLower(scm.ReviewStatePending),
				},
			},
			reviewActsAsApprove: true,
			expectHandle:        false,
		},
		{
			name: "edited review",
			reviewEvent: scm.ReviewHook{
				Action: scm.ActionEdited,
				Review: scm.Review{
					Body: "looks good",
					Author: scm.User{
						Login: "author",
					},
					State: stateToLower(scm.ReviewStateApproved),
				},
			},
			reviewActsAsApprove: true,
			expectHandle:        false,
		},
		{
			name: "dismissed review",
			reviewEvent: scm.ReviewHook{
				Action: scm.ActionDismissed,
				Review: scm.Review{
					Body: "looks good",
					Author: scm.User{
						Login: "author",
					},
					State: stateToLower(scm.ReviewStateDismissed),
				},
			},
			reviewActsAsApprove: true,
			expectHandle:        true,
		},
		{
			name: "approve command",
			reviewEvent: scm.ReviewHook{
				Action: scm.ActionSubmitted,
				Review: scm.Review{
					Body: "/approve",
					Author: scm.User{
						Login: "author",
					},
					State: stateToLower(scm.ReviewStateApproved),
				},
			},
			reviewActsAsApprove: true,
			expectHandle:        false,
		},
		{
			name: "lgtm command",
			reviewEvent: scm.ReviewHook{
				Action: scm.ActionSubmitted,
				Review: scm.Review{
					Body: "/lgtm",
					Author: scm.User{
						Login: "author",
					},
					State: stateToLower(scm.ReviewStateApproved),
				},
			},
			lgtmActsAsApprove:   true,
			reviewActsAsApprove: true,
			expectHandle:        false,
		},
		{
			name: "feature disabled",
			reviewEvent: scm.ReviewHook{
				Action: scm.ActionSubmitted,
				Review: scm.Review{
					Body: "looks good",
					Author: scm.User{
						Login: "author",
					},
					State: stateToLower(scm.ReviewStateApproved),
				},
			},
			reviewActsAsApprove: false,
			expectHandle:        false,
		},
	}

	var handled bool
	var gotState *state
	handleFunc = func(log *logrus.Entry, spc scmProviderClient, repo approvers.Repo, serverURL *url.URL, opts *plugins.Approve, pr *state) error {
		gotState = pr
		handled = true
		return nil
	}
	defer func() {
		handleFunc = handle
	}()

	repo := scm.Repository{
		Namespace: "org",
		Name:      "repo",
	}
	pr := scm.PullRequest{
		Author: scm.User{
			Login: "P.R. Author",
		},
		Base: scm.PullRequestBranch{
			Ref: "branch",
		},
		Number: 1,
		Body:   "Fix everything",
	}
	fakeScmClient, fspc := fake.NewDefault()
	fakeClient := scmprovider.ToTestClient(fakeScmClient)
	fspc.PullRequests[1] = &pr

	for _, test := range tests {
		test.reviewEvent.Repo = repo
		test.reviewEvent.PullRequest = pr
		config := &plugins.Configuration{}
		irs := !test.reviewActsAsApprove
		config.Approve = append(config.Approve, plugins.Approve{
			Repos:             []string{test.reviewEvent.Repo.Namespace},
			LgtmActsAsApprove: test.lgtmActsAsApprove,
			IgnoreReviewState: &irs,
		})
		err := handleReview(
			logrus.WithField("plugin", "approve"),
			fakeClient,
			fakeOwnersClient{},
			&url.URL{
				Scheme: "https",
				Host:   "github.com",
			},
			config,
			&test.reviewEvent,
		)

		if test.expectHandle && !handled {
			t.Errorf("%s: expected call to handleFunc, but it wasn't called", test.name)
		}

		if !test.expectHandle && handled {
			t.Errorf("%s: expected no call to handleFunc, but it was called", test.name)
		}

		if test.expectState != nil && !reflect.DeepEqual(test.expectState, gotState) {
			t.Errorf("%s: expected PR state to equal: %#v, but got: %#v", test.name, test.expectState, gotState)
		}

		if err != nil {
			t.Errorf("%s: error calling handleReview: %v", test.name, err)
		}
		handled = false
	}
}

func TestHandlePullRequest(t *testing.T) {
	tests := []struct {
		name         string
		prEvent      scm.PullRequestHook
		expectHandle bool
		expectState  *state
	}{
		{
			name: "pr opened",
			prEvent: scm.PullRequestHook{
				Action: scm.ActionOpen,
				PullRequest: scm.PullRequest{
					Number: 1,
					Author: scm.User{
						Login: "P.R. Author",
					},
					Base: scm.PullRequestBranch{
						Ref: "branch",
					},
					Body: "Fix everything",
				},
			},
			expectHandle: true,
			expectState: &state{
				org:       "org",
				repo:      "repo",
				branch:    "branch",
				number:    1,
				body:      "Fix everything",
				author:    "P.R. Author",
				assignees: nil,
				htmlURL:   "",
			},
		},
		{
			name: "pr reopened",
			prEvent: scm.PullRequestHook{
				Action: scm.ActionReopen,
			},
			expectHandle: true,
		},
		{
			name: "pr sync",
			prEvent: scm.PullRequestHook{
				Action: scm.ActionSync,
			},
			expectHandle: true,
		},
		{
			name: "pr labeled",
			prEvent: scm.PullRequestHook{
				Action: scm.ActionLabel,
				Label: scm.Label{
					Name: labels.Approved,
				},
			},
			expectHandle: true,
		},
		{
			name: "pr another label",
			prEvent: scm.PullRequestHook{
				Action: scm.ActionLabel,
				Label: scm.Label{
					Name: "some-label",
				},
			},
			expectHandle: false,
		},
		{
			name: "pr closed",
			prEvent: scm.PullRequestHook{
				Action: scm.ActionLabel,
				Label: scm.Label{
					Name: labels.Approved,
				},
				PullRequest: scm.PullRequest{
					State: "closed",
				},
			},
			expectHandle: false,
		},
		{
			name: "pr review requested",
			prEvent: scm.PullRequestHook{
				Action: scm.ActionReviewRequested,
			},
			expectHandle: false,
		},
	}

	var handled bool
	var gotState *state
	handleFunc = func(log *logrus.Entry, spc scmProviderClient, repo approvers.Repo, serverURL *url.URL, opts *plugins.Approve, pr *state) error {
		gotState = pr
		handled = true
		return nil
	}
	defer func() {
		handleFunc = handle
	}()

	repo := scm.Repository{
		Namespace: "org",
		Name:      "repo",
	}
	fakeScmClient, _ := fake.NewDefault()
	fakeClient := scmprovider.ToTestClient(fakeScmClient)

	for _, test := range tests {
		test.prEvent.Repo = repo
		err := handlePullRequest(
			logrus.WithField("plugin", "approve"),
			fakeClient,
			fakeOwnersClient{},
			&url.URL{
				Scheme: "https",
				Host:   "github.com",
			},
			&plugins.Configuration{},
			&test.prEvent,
		)

		if test.expectHandle && !handled {
			t.Errorf("%s: expected call to handleFunc, but it wasn't called", test.name)
		}

		if !test.expectHandle && handled {
			t.Errorf("%s: expected no call to handleFunc, but it was called", test.name)
		}

		if test.expectState != nil && !reflect.DeepEqual(test.expectState, gotState) {
			t.Errorf("%s: expected PR state to equal: %#v, but got: %#v", test.name, test.expectState, gotState)
		}

		if err != nil {
			t.Errorf("%s: error calling handlePullRequest: %v", test.name, err)
		}
		handled = false
	}
}

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
				Approve: []plugins.Approve{
					{
						Repos:               []string{"org2"},
						IssueRequired:       true,
						RequireSelfApproval: &[]bool{true}[0],
						LgtmActsAsApprove:   true,
						IgnoreReviewState:   &[]bool{true}[0],
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
