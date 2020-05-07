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

package lgtm

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/repoowners"
)

type fakeOwnersClient struct {
	approvers map[string]sets.String
	reviewers map[string]sets.String
}

var _ repoowners.Interface = &fakeOwnersClient{}

func (f *fakeOwnersClient) LoadRepoAliases(org, repo, base string) (repoowners.RepoAliases, error) {
	return nil, nil
}

func (f *fakeOwnersClient) LoadRepoOwners(org, repo, base string) (repoowners.RepoOwner, error) {
	return &fakeRepoOwners{approvers: f.approvers, reviewers: f.reviewers}, nil
}

type fakeRepoOwners struct {
	approvers map[string]sets.String
	reviewers map[string]sets.String
}

type fakePruner struct {
	SCMProviderClient   *fake.Data
	PullRequestComments []*scm.Comment
}

func (fp *fakePruner) PruneComments(pr bool, shouldPrune func(*scm.Comment) bool) {
	for _, comment := range fp.PullRequestComments {
		if shouldPrune(comment) {
			fp.SCMProviderClient.PullRequestCommentsDeleted = append(fp.SCMProviderClient.PullRequestCommentsDeleted, comment.Body)
		}
	}
}

var _ repoowners.RepoOwner = &fakeRepoOwners{}

func (f *fakeRepoOwners) FindApproverOwnersForFile(path string) string  { return "" }
func (f *fakeRepoOwners) FindReviewersOwnersForFile(path string) string { return "" }
func (f *fakeRepoOwners) FindLabelsForFile(path string) sets.String     { return nil }
func (f *fakeRepoOwners) IsNoParentOwners(path string) bool             { return false }
func (f *fakeRepoOwners) LeafApprovers(path string) sets.String         { return nil }
func (f *fakeRepoOwners) Approvers(path string) sets.String             { return f.approvers[path] }
func (f *fakeRepoOwners) LeafReviewers(path string) sets.String         { return nil }
func (f *fakeRepoOwners) Reviewers(path string) sets.String             { return f.reviewers[path] }
func (f *fakeRepoOwners) RequiredReviewers(path string) sets.String     { return nil }

var approvers = map[string]sets.String{
	"doc/README.md": {
		"cjwagner": {},
		"jessica":  {},
	},
}

var reviewers = map[string]sets.String{
	"doc/README.md": {
		"alice": {},
		"bob":   {},
		"mark":  {},
		"sam":   {},
	},
}

func TestLGTMComment(t *testing.T) {
	var testcases = []struct {
		name          string
		body          string
		commenter     string
		hasLGTM       bool
		shouldToggle  bool
		shouldComment bool
		shouldAssign  bool
		skipCollab    bool
		storeTreeHash bool
	}{
		{
			name:         "non-lgtm comment",
			body:         "uh oh",
			commenter:    "collab2",
			hasLGTM:      false,
			shouldToggle: false,
		},
		{
			name:          "lgtm comment by reviewer, no lgtm on pr",
			body:          "/lgtm",
			commenter:     "collab1",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: true,
		},
		{
			name:          "LGTM comment by reviewer, no lgtm on pr",
			body:          "/LGTM",
			commenter:     "collab1",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: true,
		},
		{
			name:         "lgtm comment by reviewer, lgtm on pr",
			body:         "/lgtm",
			commenter:    "collab1",
			hasLGTM:      true,
			shouldToggle: false,
		},
		{
			name:          "lgtm comment by author",
			body:          "/lgtm",
			commenter:     "author",
			hasLGTM:       false,
			shouldToggle:  false,
			shouldComment: true,
		},
		{
			name:          "lgtm cancel by author",
			body:          "/lgtm cancel",
			commenter:     "author",
			hasLGTM:       true,
			shouldToggle:  true,
			shouldAssign:  false,
			shouldComment: false,
		},
		{
			name:          "lgtm comment by non-reviewer",
			body:          "/lgtm",
			commenter:     "collab2",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: true,
			shouldAssign:  true,
		},
		{
			name:          "lgtm comment by non-reviewer, with trailing space",
			body:          "/lgtm ",
			commenter:     "collab2",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: true,
			shouldAssign:  true,
		},
		{
			name:          "lgtm comment by non-reviewer, with no-issue",
			body:          "/lgtm no-issue",
			commenter:     "collab2",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: true,
			shouldAssign:  true,
		},
		{
			name:          "lgtm comment by non-reviewer, with no-issue and trailing space",
			body:          "/lgtm no-issue \r",
			commenter:     "collab2",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: true,
			shouldAssign:  true,
		},
		{
			name:          "lgtm comment by rando",
			body:          "/lgtm",
			commenter:     "not-in-the-org",
			hasLGTM:       false,
			shouldToggle:  false,
			shouldComment: true,
			shouldAssign:  false,
		},
		{
			name:          "lgtm cancel by non-reviewer",
			body:          "/lgtm cancel",
			commenter:     "collab2",
			hasLGTM:       true,
			shouldToggle:  true,
			shouldComment: false,
			shouldAssign:  true,
		},
		{
			name:          "lgtm cancel by rando",
			body:          "/lgtm cancel",
			commenter:     "not-in-the-org",
			hasLGTM:       true,
			shouldToggle:  false,
			shouldComment: true,
			shouldAssign:  false,
		},
		{
			name:         "lgtm cancel comment by reviewer",
			body:         "/lgtm cancel",
			commenter:    "collab1",
			hasLGTM:      true,
			shouldToggle: true,
		},
		{
			name:         "lgtm cancel comment by reviewer, with trailing space",
			body:         "/lgtm cancel \r",
			commenter:    "collab1",
			hasLGTM:      true,
			shouldToggle: true,
		},
		{
			name:         "lgtm cancel comment by reviewer, no lgtm",
			body:         "/lgtm cancel",
			commenter:    "collab1",
			hasLGTM:      false,
			shouldToggle: false,
		},
		{
			name:          "lgtm comment, based off OWNERS only",
			body:          "/lgtm",
			commenter:     "sam",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: true,
			skipCollab:    true,
		},
		{
			name:          "lgtm comment by assignee, but not collab",
			body:          "/lgtm",
			commenter:     "assignee1",
			hasLGTM:       false,
			shouldToggle:  false,
			shouldComment: true,
			shouldAssign:  false,
		},
		{
			name:          "lgtm comment by reviewer, no lgtm on pr, with prefix",
			body:          "/lh-lgtm",
			commenter:     "collab1",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: true,
		},
		{
			name:         "lgtm cancel comment by reviewer, with prefix",
			body:         "/lh-lgtm cancel",
			commenter:    "collab1",
			hasLGTM:      true,
			shouldToggle: true,
		},
	}
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeScmClient, fc := fake.NewDefault()
			fakeClient := scmprovider.ToTestClient(fakeScmClient)

			fc.PullRequests[5] = &scm.PullRequest{
				Number: 5,
				Base: scm.PullRequestBranch{
					Ref: "master",
				},
				Head: scm.PullRequestBranch{
					Sha: SHA,
				},
			}
			fc.PullRequestChanges[5] = []*scm.Change{
				{Path: "doc/README.md"},
			}
			fc.Collaborators = []string{"collab1", "collab2"}

			e := &scmprovider.GenericCommentEvent{
				Action:      scm.ActionCreate,
				IssueState:  "open",
				IsPR:        true,
				Body:        tc.body,
				Author:      scm.User{Login: tc.commenter},
				IssueAuthor: scm.User{Login: "author"},
				Number:      5,
				Assignees:   []scm.User{{Login: "collab1"}, {Login: "assignee1"}},
				Repo:        scm.Repository{Namespace: "org", Name: "repo"},
				Link:        "<url>",
			}
			if tc.hasLGTM {
				fc.PullRequestLabelsAdded = []string{"org/repo#5:" + LGTMLabel}
			}
			oc := &fakeOwnersClient{approvers: approvers, reviewers: reviewers}
			pc := &plugins.Configuration{}
			if tc.skipCollab {
				pc.Owners.SkipCollaborators = []string{"org/repo"}
			}
			pc.Lgtm = append(pc.Lgtm, plugins.Lgtm{
				Repos:         []string{"org/repo"},
				StoreTreeHash: true,
			})
			fp := &fakePruner{
				SCMProviderClient:   fc,
				PullRequestComments: fc.PullRequestComments[5],
			}
			if err := handleGenericComment(fakeClient, pc, oc, logrus.WithField("plugin", PluginName), fp, *e); err != nil {
				t.Fatalf("didn't expect error from lgtmComment: %v", err)
			}
			if tc.shouldAssign {
				found := false
				for _, a := range fc.AssigneesAdded {
					if a == fmt.Sprintf("%s/%s#%d:%s", "org", "repo", 5, tc.commenter) {
						found = true
						break
					}
				}
				if !found || len(fc.AssigneesAdded) != 1 {
					t.Errorf("should have assigned %s but added assignees are %s", tc.commenter, fc.AssigneesAdded)
				}
			} else if len(fc.AssigneesAdded) != 0 {
				t.Errorf("should not have assigned anyone but assigned %s", fc.AssigneesAdded)
			}
			if tc.shouldToggle {
				if tc.hasLGTM {
					if len(fc.PullRequestLabelsRemoved) == 0 {
						t.Error("should have removed LGTM.")
					} else if len(fc.PullRequestLabelsAdded) > 1 {
						t.Error("should not have added LGTM.")
					}
				} else {
					if len(fc.PullRequestLabelsAdded) == 0 {
						t.Error("should have added LGTM.")
					} else if len(fc.PullRequestLabelsRemoved) > 0 {
						t.Error("should not have removed LGTM.")
					}
				}
			} else if len(fc.PullRequestLabelsRemoved) > 0 {
				t.Error("should not have removed LGTM.")
			} else if (tc.hasLGTM && len(fc.PullRequestLabelsAdded) > 1) || (!tc.hasLGTM && len(fc.PullRequestLabelsAdded) > 0) {
				t.Error("should not have added LGTM.")
			}
			if tc.shouldComment && len(fc.PullRequestComments[5]) != 1 {
				t.Error("should have commented.")
			} else if !tc.shouldComment && len(fc.PullRequestComments[5]) != 0 {
				t.Error("should not have commented.")
			}
		})
	}
}

func TestLGTMCommentWithLGTMNoti(t *testing.T) {
	var testcases = []struct {
		name         string
		body         string
		commenter    string
		shouldDelete bool
	}{
		{
			name:         "non-lgtm comment",
			body:         "uh oh",
			commenter:    "collab2",
			shouldDelete: false,
		},
		{
			name:         "lgtm comment by reviewer, no lgtm on pr",
			body:         "/lgtm",
			commenter:    "collab1",
			shouldDelete: true,
		},
		{
			name:         "LGTM comment by reviewer, no lgtm on pr",
			body:         "/LGTM",
			commenter:    "collab1",
			shouldDelete: true,
		},
		{
			name:         "lgtm comment by author",
			body:         "/lgtm",
			commenter:    "author",
			shouldDelete: false,
		},
		{
			name:         "lgtm comment by non-reviewer",
			body:         "/lgtm",
			commenter:    "collab2",
			shouldDelete: true,
		},
		{
			name:         "lgtm comment by non-reviewer, with trailing space",
			body:         "/lgtm ",
			commenter:    "collab2",
			shouldDelete: true,
		},
		{
			name:         "lgtm comment by non-reviewer, with no-issue",
			body:         "/lgtm no-issue",
			commenter:    "collab2",
			shouldDelete: true,
		},
		{
			name:         "lgtm comment by non-reviewer, with no-issue and trailing space",
			body:         "/lgtm no-issue \r",
			commenter:    "collab2",
			shouldDelete: true,
		},
		{
			name:         "lgtm comment by rando",
			body:         "/lgtm",
			commenter:    "not-in-the-org",
			shouldDelete: false,
		},
		{
			name:         "lgtm cancel comment by reviewer, no lgtm",
			body:         "/lgtm cancel",
			commenter:    "collab1",
			shouldDelete: false,
		},
	}
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	for _, tc := range testcases {
		fakeScmClient, fc := fake.NewDefault()
		fakeClient := scmprovider.ToTestClient(fakeScmClient)

		fc.PullRequests[5] = &scm.PullRequest{
			Number: 5,
			Head: scm.PullRequestBranch{
				Sha: SHA,
			},
		}
		fc.Collaborators = []string{"collab1", "collab2"}
		e := &scmprovider.GenericCommentEvent{
			Action:      scm.ActionCreate,
			IssueState:  "open",
			IsPR:        true,
			Body:        tc.body,
			Author:      scm.User{Login: tc.commenter},
			IssueAuthor: scm.User{Login: "author"},
			Number:      5,
			Assignees:   []scm.User{{Login: "collab1"}, {Login: "assignee1"}},
			Repo:        scm.Repository{Namespace: "org", Name: "repo"},
			Link:        "<url>",
		}
		botName, err := fakeClient.BotName()
		if err != nil {
			t.Fatalf("For case %s, could not get Bot nam", tc.name)
		}
		ic := &scm.Comment{
			Author: scm.User{
				Login: botName,
			},
			Body: removeLGTMLabelNoti,
		}
		fc.PullRequestComments[5] = append(fc.PullRequestComments[5], ic)
		oc := &fakeOwnersClient{approvers: approvers, reviewers: reviewers}
		pc := &plugins.Configuration{}
		fp := &fakePruner{
			SCMProviderClient:   fc,
			PullRequestComments: fc.PullRequestComments[5],
		}
		if err := handleGenericComment(fakeClient, pc, oc, logrus.WithField("plugin", PluginName), fp, *e); err != nil {
			t.Errorf("For case %s, didn't expect error from lgtmComment: %v", tc.name, err)
			continue
		}
		deleted := false
		for _, body := range fc.PullRequestCommentsDeleted {
			if body == removeLGTMLabelNoti {
				deleted = true
				break
			}
		}
		if tc.shouldDelete {
			if !deleted {
				t.Errorf("For case %s, LGTM removed notification should have been deleted", tc.name)
			}
		} else {
			if deleted {
				t.Errorf("For case %s, LGTM removed notification should not have been deleted", tc.name)
			}
		}
	}
}

func TestLGTMFromApproveReview(t *testing.T) {
	var testcases = []struct {
		name          string
		state         string
		action        scm.Action
		body          string
		reviewer      string
		hasLGTM       bool
		shouldToggle  bool
		shouldComment bool
		shouldAssign  bool
		storeTreeHash bool
	}{
		{
			name:          "Edit approve review by reviewer, no lgtm on pr",
			state:         scm.ReviewStateApproved,
			action:        scm.ActionEdited,
			reviewer:      "collab1",
			hasLGTM:       false,
			shouldToggle:  false,
			storeTreeHash: true,
		},
		{
			name:          "Dismiss approve review by reviewer, no lgtm on pr",
			state:         scm.ReviewStateApproved,
			action:        scm.ActionDismissed,
			reviewer:      "collab1",
			hasLGTM:       false,
			shouldToggle:  false,
			storeTreeHash: true,
		},
		{
			name:          "Request changes review by reviewer, no lgtm on pr",
			state:         scm.ReviewStateChangesRequested,
			action:        scm.ActionSubmitted,
			reviewer:      "collab1",
			hasLGTM:       false,
			shouldToggle:  false,
			shouldAssign:  false,
			shouldComment: false,
		},
		{
			name:         "Request changes review by reviewer, lgtm on pr",
			state:        scm.ReviewStateChangesRequested,
			action:       scm.ActionSubmitted,
			reviewer:     "collab1",
			hasLGTM:      true,
			shouldToggle: true,
			shouldAssign: false,
		},
		{
			name:          "Approve review by reviewer, no lgtm on pr",
			state:         scm.ReviewStateApproved,
			action:        scm.ActionSubmitted,
			reviewer:      "collab1",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: true,
			storeTreeHash: true,
		},
		{
			name:          "Approve review by reviewer, no lgtm on pr, do not store tree_hash",
			state:         scm.ReviewStateApproved,
			action:        scm.ActionSubmitted,
			reviewer:      "collab1",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: false,
		},
		{
			name:         "Approve review by reviewer, lgtm on pr",
			state:        scm.ReviewStateApproved,
			action:       scm.ActionSubmitted,
			reviewer:     "collab1",
			hasLGTM:      true,
			shouldToggle: false,
			shouldAssign: false,
		},
		{
			name:          "Approve review by non-reviewer, no lgtm on pr",
			state:         scm.ReviewStateApproved,
			action:        scm.ActionSubmitted,
			reviewer:      "collab2",
			hasLGTM:       false,
			shouldToggle:  true,
			shouldComment: true,
			shouldAssign:  true,
			storeTreeHash: true,
		},
		{
			name:          "Request changes review by non-reviewer, no lgtm on pr",
			state:         scm.ReviewStateChangesRequested,
			action:        scm.ActionSubmitted,
			reviewer:      "collab2",
			hasLGTM:       false,
			shouldToggle:  false,
			shouldComment: false,
			shouldAssign:  true,
		},
		{
			name:          "Approve review by rando",
			state:         scm.ReviewStateApproved,
			action:        scm.ActionSubmitted,
			reviewer:      "not-in-the-org",
			hasLGTM:       false,
			shouldToggle:  false,
			shouldComment: true,
			shouldAssign:  false,
		},
		{
			name:          "Comment review by issue author, no lgtm on pr",
			state:         scm.ReviewStateCommented,
			action:        scm.ActionSubmitted,
			reviewer:      "author",
			hasLGTM:       false,
			shouldToggle:  false,
			shouldComment: false,
			shouldAssign:  false,
		},
		{
			name:          "Comment body has /lgtm on Comment Review ",
			state:         scm.ReviewStateCommented,
			action:        scm.ActionSubmitted,
			reviewer:      "collab1",
			body:          "/lgtm",
			hasLGTM:       false,
			shouldToggle:  false,
			shouldComment: false,
			shouldAssign:  false,
		},
		{
			name:          "Comment body has /lgtm cancel on Approve Review",
			state:         scm.ReviewStateApproved,
			action:        scm.ActionSubmitted,
			reviewer:      "collab1",
			body:          "/lgtm cancel",
			hasLGTM:       false,
			shouldToggle:  false,
			shouldComment: false,
			shouldAssign:  false,
		},
	}
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	for _, tc := range testcases {
		fakeScmClient, fc := fake.NewDefault()
		fakeClient := scmprovider.ToTestClient(fakeScmClient)

		fc.PullRequests[5] = &scm.PullRequest{
			Number: 5,
			Head: scm.PullRequestBranch{
				Sha: SHA,
			},
		}
		fc.Collaborators = []string{"collab1", "collab2"}

		e := &scm.ReviewHook{
			Action:      tc.action,
			Review:      scm.Review{Body: tc.body, State: tc.state, Link: "<url>", Author: scm.User{Login: tc.reviewer}},
			PullRequest: scm.PullRequest{Author: scm.User{Login: "author"}, Assignees: []scm.User{{Login: "collab1"}, {Login: "assignee1"}}, Number: 5},
			Repo:        scm.Repository{Namespace: "org", Name: "repo"},
		}
		if tc.hasLGTM {
			fc.PullRequestLabelsAdded = append(fc.PullRequestLabelsAdded, "org/repo#5:"+LGTMLabel)
		}
		oc := &fakeOwnersClient{approvers: approvers, reviewers: reviewers}
		pc := &plugins.Configuration{}
		pc.Lgtm = append(pc.Lgtm, plugins.Lgtm{
			Repos:         []string{"org/repo"},
			StoreTreeHash: tc.storeTreeHash,
		})
		fp := &fakePruner{
			SCMProviderClient:   fc,
			PullRequestComments: fc.PullRequestComments[5],
		}
		if err := handlePullRequestReview(fakeClient, pc, oc, logrus.WithField("plugin", PluginName), fp, *e); err != nil {
			t.Errorf("For case %s, didn't expect error from pull request review: %v", tc.name, err)
			continue
		}
		if tc.shouldAssign {
			found := false
			for _, a := range fc.AssigneesAdded {
				if a == fmt.Sprintf("%s/%s#%d:%s", "org", "repo", 5, tc.reviewer) {
					found = true
					break
				}
			}
			if !found || len(fc.AssigneesAdded) != 1 {
				t.Errorf("For case %s, should have assigned %s but added assignees are %s", tc.name, tc.reviewer, fc.AssigneesAdded)
			}
		} else if len(fc.AssigneesAdded) != 0 {
			t.Errorf("For case %s, should not have assigned anyone but assigned %s", tc.name, fc.AssigneesAdded)
		}
		if tc.shouldToggle {
			if tc.hasLGTM {
				if len(fc.PullRequestLabelsRemoved) == 0 {
					t.Errorf("For case %s, should have removed LGTM.", tc.name)
				} else if len(fc.PullRequestLabelsAdded) > 1 {
					t.Errorf("For case %s, should not have added LGTM.", tc.name)
				}
			} else {
				if len(fc.PullRequestLabelsAdded) == 0 {
					t.Errorf("For case %s, should have added LGTM.", tc.name)
				} else if len(fc.PullRequestLabelsRemoved) > 0 {
					t.Errorf("For case %s, should not have removed LGTM.", tc.name)
				}
			}
		} else if len(fc.PullRequestLabelsRemoved) > 0 {
			t.Errorf("For case %s, should not have removed LGTM.", tc.name)
		} else if (tc.hasLGTM && len(fc.PullRequestLabelsAdded) > 1) || (!tc.hasLGTM && len(fc.PullRequestLabelsAdded) > 0) {
			t.Errorf("For case %s, should not have added LGTM.", tc.name)
		}
		if tc.shouldComment && len(fc.PullRequestComments[5]) != 1 {
			t.Errorf("For case %s, should have commented.", tc.name)
		} else if !tc.shouldComment && len(fc.PullRequestComments[5]) != 0 {
			t.Errorf("For case %s, should not have commented.", tc.name)
		}
	}
}

func TestHandlePullRequest(t *testing.T) {
	SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
	treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"
	fakeBotName := "k8s-ci-robot"

	cases := []struct {
		name             string
		event            scm.PullRequestHook
		removeLabelErr   error
		createCommentErr error

		err                      error
		PullRequestLabelsAdded   []string
		PullRequestLabelsRemoved []string
		PullRequestComments      map[int][]*scm.Comment
		trustedTeam              string

		expectNoComments bool
	}{
		{
			name: "pr_synchronize, no RemoveLabel error",
			event: scm.PullRequestHook{
				Action: scm.ActionSync,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
					Head: scm.PullRequestBranch{
						Sha: SHA,
					},
				},
			},
			PullRequestLabelsRemoved: []string{LGTMLabel},
			PullRequestComments: map[int][]*scm.Comment{
				101: {
					{
						Body:   removeLGTMLabelNoti,
						Author: scm.User{Login: fakeBotName},
					},
				},
			},
			expectNoComments: false,
		},
		{
			name: "Sticky LGTM for trusted team members",
			event: scm.PullRequestHook{
				Action: scm.ActionSync,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
					Author: scm.User{
						Login: "sig-lead",
					},
					MergeSha: SHA,
				},
			},
			trustedTeam:      "Leads",
			expectNoComments: true,
		},
		{
			name: "LGTM not sticky for trusted user if disabled",
			event: scm.PullRequestHook{
				Action: scm.ActionSync,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
					Author: scm.User{
						Login: "sig-lead",
					},
					MergeSha: SHA,
				},
			},
			PullRequestLabelsRemoved: []string{LGTMLabel},
			PullRequestComments: map[int][]*scm.Comment{
				101: {
					{
						Body:   removeLGTMLabelNoti,
						Author: scm.User{Login: fakeBotName},
					},
				},
			},
			expectNoComments: false,
		},
		{
			name: "LGTM not sticky for non trusted user",
			event: scm.PullRequestHook{
				Action: scm.ActionSync,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
					Author: scm.User{
						Login: "sig-lead",
					},
					MergeSha: SHA,
				},
			},
			PullRequestLabelsRemoved: []string{LGTMLabel},
			PullRequestComments: map[int][]*scm.Comment{
				101: {
					{
						Body:   removeLGTMLabelNoti,
						Author: scm.User{Login: fakeBotName},
					},
				},
			},
			trustedTeam:      "Committers",
			expectNoComments: false,
		},
		{
			name: "pr_assigned",
			event: scm.PullRequestHook{
				Action: scm.ActionAssigned,
			},
			expectNoComments: true,
		},
		{
			name: "pr_synchronize, same tree-hash, keep label",
			event: scm.PullRequestHook{
				Action: scm.ActionSync,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
					Head: scm.PullRequestBranch{
						Sha: SHA,
					},
				},
			},
			PullRequestComments: map[int][]*scm.Comment{
				101: {
					{
						Body:   fmt.Sprintf(addLGTMLabelNotification, treeSHA),
						Author: scm.User{Login: fakeBotName},
					},
				},
			},
			expectNoComments: true,
		},
		{
			name: "pr_synchronize, same tree-hash, keep label, edited comment",
			event: scm.PullRequestHook{
				Action: scm.ActionSync,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
					Head: scm.PullRequestBranch{
						Sha: SHA,
					},
				},
			},
			PullRequestLabelsRemoved: []string{LGTMLabel},
			PullRequestComments: map[int][]*scm.Comment{
				101: {
					{
						Body:    fmt.Sprintf(addLGTMLabelNotification, treeSHA),
						Author:  scm.User{Login: fakeBotName},
						Created: time.Date(1981, 2, 21, 12, 30, 0, 0, time.UTC),
						Updated: time.Date(1981, 2, 21, 12, 31, 0, 0, time.UTC),
					},
				},
			},
			expectNoComments: false,
		},
		{
			name: "pr_synchronize, 2 tree-hash comments, keep label",
			event: scm.PullRequestHook{
				Action: scm.ActionSync,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
					Head: scm.PullRequestBranch{
						Sha: SHA,
					},
				},
			},
			PullRequestComments: map[int][]*scm.Comment{
				101: {
					{
						Body:   fmt.Sprintf(addLGTMLabelNotification, "older_treeSHA"),
						Author: scm.User{Login: fakeBotName},
					},
					{
						Body:   fmt.Sprintf(addLGTMLabelNotification, treeSHA),
						Author: scm.User{Login: fakeBotName},
					},
				},
			},
			expectNoComments: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fakeScmClient, fakeGitHub := fake.NewDefault()
			fakeClient := scmprovider.ToClient(fakeScmClient, fakeBotName)

			fakeGitHub.PullRequestComments = c.PullRequestComments
			fakeGitHub.PullRequests[101] = &scm.PullRequest{
				Number: 101,
				Base: scm.PullRequestBranch{
					Ref: "master",
				},
				Head: scm.PullRequestBranch{
					Sha: SHA,
				},
			}
			fakeGitHub.Collaborators = []string{"collab"}
			fakeGitHub.PullRequestLabelsAdded = c.PullRequestLabelsAdded
			fakeGitHub.PullRequestLabelsAdded = append(fakeGitHub.PullRequestLabelsAdded, "kubernetes/kubernetes#101:lgtm")

			commit := &scm.Commit{}
			commit.Tree.Sha = treeSHA
			fakeGitHub.Commits[SHA] = commit
			pc := &plugins.Configuration{}
			pc.Lgtm = append(pc.Lgtm, plugins.Lgtm{
				Repos:          []string{"kubernetes/kubernetes"},
				StoreTreeHash:  true,
				StickyLgtmTeam: c.trustedTeam,
			})
			err := handlePullRequest(
				logrus.WithField("plugin", "approve"),
				fakeClient,
				pc,
				&c.event,
			)

			if err != nil && c.err == nil {
				t.Fatalf("handlePullRequest error: %v", err)
			}

			if err == nil && c.err != nil {
				t.Fatalf("handlePullRequest wanted error: %v, got nil", c.err)
			}

			if got, want := err, c.err; !equality.Semantic.DeepEqual(got, want) {
				t.Fatalf("handlePullRequest error mismatch: got %v, want %v", got, want)
			}

			if got, want := len(fakeGitHub.PullRequestLabelsRemoved), len(c.PullRequestLabelsRemoved); got != want {
				t.Logf("PullRequestLabelsRemoved: got %v, want: %v", fakeGitHub.PullRequestLabelsRemoved, c.PullRequestLabelsRemoved)
				t.Fatalf("PullRequestLabelsRemoved length mismatch: got %d, want %d", got, want)
			}

			if got, want := fakeGitHub.PullRequestComments, c.PullRequestComments; !equality.Semantic.DeepEqual(got, want) {
				t.Fatalf("LGTM revmoved notifications mismatch: got %v, want %v", got, want)
			}
			if c.expectNoComments && len(fakeGitHub.PullRequestCommentsAdded) > 0 {
				t.Fatalf("expected no comments but got %v", fakeGitHub.PullRequestCommentsAdded)
			}
			if !c.expectNoComments && len(fakeGitHub.PullRequestCommentsAdded) == 0 {
				t.Fatalf("expected comments but got none")
			}
		})
	}
}

func TestAddTreeHashComment(t *testing.T) {
	cases := []struct {
		name          string
		author        string
		trustedTeam   string
		expectTreeSha bool
	}{
		{
			name:          "Tree SHA added",
			author:        "Bob",
			expectTreeSha: true,
		},
		{
			name:          "Tree SHA if sticky lgtm off",
			author:        "sig-lead",
			expectTreeSha: true,
		},
		{
			name:          "No Tree SHA if sticky lgtm",
			author:        "sig-lead",
			trustedTeam:   "Leads",
			expectTreeSha: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			SHA := "0bd3ed50c88cd53a09316bf7a298f900e9371652"
			treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"
			pc := &plugins.Configuration{}
			pc.Lgtm = append(pc.Lgtm, plugins.Lgtm{
				Repos:          []string{"kubernetes/kubernetes"},
				StoreTreeHash:  true,
				StickyLgtmTeam: c.trustedTeam,
			})
			rc := reviewCtx{
				author:      "collab1",
				issueAuthor: c.author,
				repo: scm.Repository{
					Namespace: "kubernetes",
					Name:      "kubernetes",
				},
				number: 101,
				body:   "/lgtm",
			}
			fakeScmClient, fc := fake.NewDefault()
			fakeClient := scmprovider.ToTestClient(fakeScmClient)

			fc.PullRequests[101] = &scm.PullRequest{
				Number: 101,
				Base: scm.PullRequestBranch{
					Ref: "master",
				},
				Head: scm.PullRequestBranch{
					Sha: SHA,
				},
			}
			fc.Collaborators = []string{"collab1", "collab2"}

			commit := &scm.Commit{}
			commit.Tree.Sha = treeSHA
			fc.Commits[SHA] = commit
			handle(true, pc, &fakeOwnersClient{}, rc, fakeClient, logrus.WithField("plugin", PluginName), &fakePruner{})
			found := false
			for _, body := range fc.PullRequestCommentsAdded {
				if addLGTMLabelNotificationRe.MatchString(body) {
					found = true
					break
				}
			}
			if c.expectTreeSha {
				if !found {
					t.Fatalf("expected tree_hash comment but got none")
				}
			} else {
				if found {
					t.Fatalf("expected no tree_hash comment but got one")
				}
			}
		})
	}
}

func TestRemoveTreeHashComment(t *testing.T) {
	treeSHA := "6dcb09b5b57875f334f61aebed695e2e4193db5e"
	pc := &plugins.Configuration{}
	pc.Lgtm = append(pc.Lgtm, plugins.Lgtm{
		Repos:         []string{"kubernetes/kubernetes"},
		StoreTreeHash: true,
	})
	rc := reviewCtx{
		author:      "collab1",
		issueAuthor: "bob",
		repo: scm.Repository{
			Namespace: "kubernetes",
			Name:      "kubernetes",
		},
		assignees: []scm.User{{Login: "alice"}},
		number:    101,
		body:      "/lgtm cancel",
	}
	fakeBotName := "k8s-ci-robot"
	fakeScmClient, fc := fake.NewDefault()
	fakeClient := scmprovider.ToClient(fakeScmClient, fakeBotName)

	fc.PullRequestComments[101] = []*scm.Comment{&scm.Comment{
		Body:   fmt.Sprintf(addLGTMLabelNotification, treeSHA),
		Author: scm.User{Login: fakeBotName},
	},
	}
	fc.Collaborators = []string{"collab1", "collab2"}

	fc.PullRequestLabelsAdded = []string{"kubernetes/kubernetes#101:" + LGTMLabel}
	fp := &fakePruner{
		SCMProviderClient:   fc,
		PullRequestComments: fc.PullRequestComments[101],
	}
	handle(false, pc, &fakeOwnersClient{}, rc, fakeClient, logrus.WithField("plugin", PluginName), fp)
	found := false
	for _, body := range fc.PullRequestCommentsDeleted {
		if addLGTMLabelNotificationRe.MatchString(body) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected deleted tree_hash comment but got none")
	}
}

func TestHelpProvider(t *testing.T) {
	cases := []struct {
		name               string
		config             *plugins.Configuration
		enabledRepos       []string
		err                bool
		configInfoIncludes []string
		configInfoExcludes []string
	}{
		{
			name:               "Empty config",
			config:             &plugins.Configuration{},
			enabledRepos:       []string{"org1", "org2/repo"},
			configInfoExcludes: []string{configInfoReviewActsAsLgtm, configInfoStoreTreeHash, configInfoStickyLgtmTeam("team1")},
		},
		{
			name:               "Overlapping org and org/repo",
			config:             &plugins.Configuration{},
			enabledRepos:       []string{"org2", "org2/repo"},
			configInfoExcludes: []string{configInfoReviewActsAsLgtm, configInfoStoreTreeHash, configInfoStickyLgtmTeam("team1")},
		},
		{
			name:         "Invalid enabledRepos",
			config:       &plugins.Configuration{},
			enabledRepos: []string{"org1", "org2/repo/extra"},
			err:          true,
		},
		{
			name: "StoreTreeHash enabled",
			config: &plugins.Configuration{
				Lgtm: []plugins.Lgtm{
					{
						Repos:         []string{"org2"},
						StoreTreeHash: true,
					},
				},
			},
			enabledRepos:       []string{"org1", "org2/repo"},
			configInfoExcludes: []string{configInfoReviewActsAsLgtm, configInfoStickyLgtmTeam("team1")},
			configInfoIncludes: []string{configInfoStoreTreeHash},
		},
		{
			name: "All configs enabled",
			config: &plugins.Configuration{
				Lgtm: []plugins.Lgtm{
					{
						Repos:            []string{"org2"},
						ReviewActsAsLgtm: true,
						StoreTreeHash:    true,
						StickyLgtmTeam:   "team1",
					},
				},
			},
			enabledRepos:       []string{"org1", "org2/repo"},
			configInfoIncludes: []string{configInfoReviewActsAsLgtm, configInfoStoreTreeHash, configInfoStickyLgtmTeam("team1")},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pluginHelp, err := helpProvider(c.config, c.enabledRepos)
			if err != nil && !c.err {
				t.Fatalf("helpProvider error: %v", err)
			}
			for _, msg := range c.configInfoExcludes {
				if strings.Contains(pluginHelp.Config["org2/repo"], msg) {
					t.Fatalf("helpProvider.Config error mismatch: got %v, but didn't want it", msg)
				}
			}
			for _, msg := range c.configInfoIncludes {
				if !strings.Contains(pluginHelp.Config["org2/repo"], msg) {
					t.Fatalf("helpProvider.Config error mismatch: didn't get %v, but wanted it", msg)
				}
			}
		})
	}
}
