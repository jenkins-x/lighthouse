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

package keeper

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/config/keeper"
	"github.com/jenkins-x/lighthouse/pkg/keeper/blockers"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/config"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestExpectedStatus(t *testing.T) {
	neededLabels := []string{"need-1", "need-2", "need-a-very-super-duper-extra-not-short-at-all-label-name"}
	titleCasedNeededLabels := []string{"Need-1", "Need-2", "Need-a-very-super-duper-extra-not-short-at-all-label-name"}
	forbiddenLabels := []string{"forbidden-1", "forbidden-2"}
	testcases := []struct {
		name string

		baseref           string
		branchIncludeList []string
		branchExcludeList []string
		sameBranchReqs    bool
		labels            []string
		milestone         string
		contexts          []Context
		inPool            bool
		blocks            []int
		pending           []int
		batchPending      []int

		state string
		desc  string
	}{
		{
			name:   "in pool",
			inPool: true,

			state: scmprovider.StatusSuccess,
			desc:  statusInPool,
		},
		{
			name:      "check truncation of label list",
			milestone: "v1.0",
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Needs need-1, need-2 labels."),
		},
		{
			name:      "check label requirements are case insensitive because Github treats them that way",
			labels:    append([]string{}, titleCasedNeededLabels[:2]...),
			milestone: "v1.0",
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Needs need-a-very-super-duper-extra-not-short-at-all-label-name label."),
		},
		{
			name:      "check truncation of label list is not excessive",
			labels:    append([]string{}, neededLabels[:2]...),
			milestone: "v1.0",
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Needs need-a-very-super-duper-extra-not-short-at-all-label-name label."),
		},
		{
			name:      "has forbidden labels",
			labels:    append(append([]string{}, neededLabels...), forbiddenLabels...),
			milestone: "v1.0",
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Should not have forbidden-1, forbidden-2 labels."),
		},
		{
			name:      "has one forbidden label",
			labels:    append(append([]string{}, neededLabels...), forbiddenLabels[0]),
			milestone: "v1.0",
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Should not have forbidden-1 label."),
		},
		{
			name:      "only mention one requirement class",
			labels:    append(append([]string{}, neededLabels[1:]...), forbiddenLabels[0]),
			milestone: "v1.0",
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Needs need-1 label."),
		},
		{
			name:              "against excluded branch",
			baseref:           "bad",
			branchExcludeList: []string{"bad"},
			sameBranchReqs:    true,
			labels:            neededLabels,
			inPool:            false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Merging to branch bad is forbidden."),
		},
		{
			name:              "not against included branch",
			baseref:           "bad",
			branchIncludeList: []string{"good"},
			sameBranchReqs:    true,
			labels:            neededLabels,
			inPool:            false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Merging to branch bad is forbidden."),
		},
		{
			name:              "choose query for correct branch",
			baseref:           "bad",
			branchIncludeList: []string{"good"},
			milestone:         "v1.0",
			labels:            neededLabels,
			inPool:            false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Needs 1, 2, 3, 4, 5, 6, 7 labels."),
		},
		{
			name:      "only failed keeper context",
			labels:    neededLabels,
			milestone: "v1.0",
			contexts:  []Context{{Context: githubql.String(GetStatusContextLabel()), State: githubql.StatusStateError}},
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, ""),
		},
		{
			name:      "single bad context",
			labels:    neededLabels,
			contexts:  []Context{{Context: githubql.String("job-name"), State: githubql.StatusStateError}},
			milestone: "v1.0",
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Job job-name has not succeeded."),
		},
		{
			name:      "multiple bad contexts",
			labels:    neededLabels,
			milestone: "v1.0",
			contexts: []Context{
				{Context: githubql.String("job-name"), State: githubql.StatusStateError},
				{Context: githubql.String("other-job-name"), State: githubql.StatusStateError},
			},
			inPool: false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Jobs job-name, other-job-name have not succeeded."),
		},
		{
			name:      "wrong milestone",
			labels:    neededLabels,
			milestone: "v1.1",
			contexts:  []Context{{Context: githubql.String("job-name"), State: githubql.StatusStateSuccess}},
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Must be in milestone v1.0."),
		},
		{
			name:      "unknown requirement",
			labels:    neededLabels,
			milestone: "v1.0",
			contexts:  []Context{{Context: githubql.String("job-name"), State: githubql.StatusStateSuccess}},
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, ""),
		},
		{
			name:      "check that min diff query is used",
			labels:    []string{"3", "4", "5", "6", "7"},
			milestone: "v1.0",
			inPool:    false,

			state: scmprovider.StatusPending,
			desc:  fmt.Sprintf(statusNotInPool, " Needs 1, 2 labels."),
		},
		{
			name:      "check that blockers take precedence over other queries",
			labels:    []string{"3", "4", "5", "6", "7"},
			milestone: "v1.0",
			inPool:    false,
			blocks:    []int{1, 2},

			state: scmprovider.StatusError,
			desc:  fmt.Sprintf(statusNotInPool, " Merging is blocked by issues 1, 2."),
		},
		{
			name:    "in pool behind pending",
			inPool:  true,
			pending: []int{1},

			state: scmprovider.StatusSuccess,
			desc:  fmt.Sprintf("%s, waiting for merge of PR(s) #1.", statusInPool),
		},
		{
			name:         "in pool behind batch",
			inPool:       true,
			batchPending: []int{1, 2, 3},

			state: scmprovider.StatusSuccess,
			desc:  fmt.Sprintf("%s, waiting for batch run and merge of PRs #1, #2, #3.", statusInPool),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			secondQuery := keeper.Query{
				Orgs:      []string{""},
				Labels:    []string{"1", "2", "3", "4", "5", "6", "7"}, // lots of requirements
				Milestone: "v1.0",
			}
			if tc.sameBranchReqs {
				secondQuery.ExcludedBranches = tc.branchExcludeList
				secondQuery.IncludedBranches = tc.branchIncludeList
			}
			queriesByRepo := keeper.Queries{
				keeper.Query{
					Orgs:             []string{""},
					ExcludedBranches: tc.branchExcludeList,
					IncludedBranches: tc.branchIncludeList,
					Labels:           neededLabels,
					MissingLabels:    forbiddenLabels,
					Milestone:        "v1.0",
				},
				secondQuery,
			}.QueryMap()
			var pr PullRequest
			pr.BaseRef = GraphQLBaseRef{
				Name: githubql.String(tc.baseref),
			}
			for _, label := range tc.labels {
				pr.Labels.Nodes = append(
					pr.Labels.Nodes,
					struct{ Name githubql.String }{Name: githubql.String(label)},
				)
			}
			if len(tc.contexts) > 0 {
				pr.HeadRefOID = "head"
				pr.Commits.Nodes = append(
					pr.Commits.Nodes,
					struct{ Commit Commit }{
						Commit: Commit{
							Status: struct{ Contexts []Context }{
								Contexts: tc.contexts,
							},
							OID: "head",
						},
					},
				)
			}
			if tc.milestone != "" {
				pr.Milestone = &struct {
					Title githubql.String
				}{githubql.String(tc.milestone)}
			}
			var pool map[string]prWithStatus
			if tc.inPool {
				withStatus := prWithStatus{
					pr:      pr,
					success: false,
				}
				if len(tc.pending) > 0 {
					withStatus.waitingFor = append(withStatus.waitingFor, tc.pending...)
					withStatus.success = true
				}
				if len(tc.batchPending) > 0 {
					withStatus.waitingForBatch = append(withStatus.waitingForBatch, tc.batchPending...)
					withStatus.success = true
				}
				pool = map[string]prWithStatus{"#0": withStatus}
			}
			blocks := blockers.Blockers{
				Repo: map[blockers.OrgRepo][]blockers.Blocker{},
			}
			var items []blockers.Blocker
			for _, block := range tc.blocks {
				items = append(items, blockers.Blocker{Number: block})
			}
			blocks.Repo[blockers.OrgRepo{Org: "", Repo: ""}] = items

			state, desc := expectedStatus(queriesByRepo, &pr, pool, &keeper.ContextPolicy{}, blocks, "fake", nil)
			if state != tc.state {
				t.Errorf("Expected status state %q, but got %q.", string(tc.state), string(state))
			}
			expectedDesc := tc.desc
			if expectedDesc == statusInPool {
				expectedDesc = statusInPool + "."
			}
			if desc != expectedDesc {
				t.Errorf("Expected status description %q, but got %q.", expectedDesc, desc)
			}
		})
	}
}

func TestSetStatuses(t *testing.T) {
	statusNotInPoolEmpty := fmt.Sprintf(statusNotInPool, "")
	testcases := []struct {
		name string

		inPool     bool
		hasContext bool
		state      githubql.StatusState
		desc       string

		shouldSet bool
	}{
		{
			name: "in pool with proper context",

			inPool:     true,
			hasContext: true,
			state:      githubql.StatusStateSuccess,
			desc:       statusInPool + ".",

			shouldSet: false,
		},
		{
			name: "in pool without context",

			inPool:     true,
			hasContext: false,

			shouldSet: true,
		},
		{
			name: "in pool with improper context",

			inPool:     true,
			hasContext: true,
			state:      githubql.StatusStateSuccess,
			desc:       statusNotInPoolEmpty,

			shouldSet: true,
		},
		{
			name: "in pool with wrong state",

			inPool:     true,
			hasContext: true,
			state:      githubql.StatusStatePending,
			desc:       statusInPool,

			shouldSet: true,
		},
		{
			name: "not in pool with proper context",

			inPool:     false,
			hasContext: true,
			state:      githubql.StatusStatePending,
			desc:       statusNotInPoolEmpty,

			shouldSet: false,
		},
		{
			name: "not in pool with improper context",

			inPool:     false,
			hasContext: true,
			state:      githubql.StatusStatePending,
			desc:       statusInPool,

			shouldSet: true,
		},
		{
			name: "not in pool with no context",

			inPool:     false,
			hasContext: false,

			shouldSet: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var pr PullRequest
			pr.Commits.Nodes = []struct{ Commit Commit }{{}}
			if tc.hasContext {
				pr.Commits.Nodes[0].Commit.Status.Contexts = []Context{
					{
						Context:     githubql.String(GetStatusContextLabel()),
						State:       tc.state,
						Description: githubql.String(tc.desc),
					},
				}
			}
			pool := make(map[string]prWithStatus)
			if tc.inPool {
				pool[pr.prKey()] = prWithStatus{
					pr:      pr,
					success: false,
				}
			}
			fc := &fgc{}
			ca := &config.Agent{}
			ca.Set(&config.Config{})
			// setStatuses logs instead of returning errors.
			// Construct a logger to watch for errors to be printed.
			log := logrus.WithField("component", "keeper")
			initialLog, err := log.String()
			if err != nil {
				t.Fatalf("Failed to get log output before testing: %v", err)
			}

			sc := &statusController{spc: fc, config: ca.Config, logger: log}
			sc.setStatuses([]PullRequest{pr}, pool, blockers.Blockers{})
			if str, err := log.String(); err != nil {
				t.Fatalf("For case %s: failed to get log output: %v", tc.name, err)
			} else if str != initialLog {
				t.Errorf("For case %s: error setting status: %s", tc.name, str)
			}
			if tc.shouldSet && !fc.setStatus {
				t.Errorf("For case %s: should set but didn't", tc.name)
			} else if !tc.shouldSet && fc.setStatus {
				t.Errorf("For case %s: should not set but did", tc.name)
			}
		})
	}
}

func TestTargetUrl(t *testing.T) {
	testcases := []struct {
		name   string
		pr     *PullRequest
		config keeper.Config

		expectedURL string
	}{
		{
			name:        "no config",
			pr:          &PullRequest{},
			config:      keeper.Config{},
			expectedURL: "",
		},
		{
			name:        "keeper overview config",
			pr:          &PullRequest{},
			config:      keeper.Config{TargetURL: "tide.com"},
			expectedURL: "tide.com",
		},
		{
			name:        "PR dashboard config and overview config",
			pr:          &PullRequest{},
			config:      keeper.Config{TargetURL: "tide.com", PRStatusBaseURL: "pr.status.com"},
			expectedURL: "tide.com",
		},
		{
			name: "PR dashboard config",
			pr: &PullRequest{
				Author: struct {
					Login githubql.String
				}{Login: githubql.String("author")},
				Repository: Repository{
					NameWithOwner: githubql.String("org/repo"),
				},
				HeadRefName: "head",
			},
			config:      keeper.Config{PRStatusBaseURL: "pr.status.com"},
			expectedURL: "pr.status.com?query=is%3Apr+repo%3Aorg%2Frepo+author%3Aauthor+head%3Ahead",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ca := &config.Agent{}
			ca.Set(&config.Config{ProwConfig: config.ProwConfig{Keeper: tc.config}})
			log := logrus.WithField("controller", "status-update")
			if actual, expected := targetURL(ca.Config, tc.pr, log), tc.expectedURL; actual != expected {
				t.Errorf("%s: expected target URL %s but got %s", tc.name, expected, actual)
			}
		})
	}
}

func TestOpenPRsQuery(t *testing.T) {
	var q string
	checkTok := func(tok string) {
		if !strings.Contains(q, " "+tok+" ") {
			t.Errorf("Expected query to contain \"%s\", got \"%s\"", tok, q)
		}
	}

	orgs := []string{"org", "kuber"}
	repos := []string{"k8s/k8s", "k8s/t-i"}
	exceptions := map[string]sets.String{
		"org":            sets.NewString("org/repo1", "org/repo2"),
		"irrelevant-org": sets.NewString("irrelevant-org/repo1", "irrelevant-org/repo2"),
	}

	q = " " + openPRsQuery(orgs, repos, exceptions) + " "
	checkTok("is:pr")
	checkTok("state:open")
	checkTok("org:\"org\"")
	checkTok("org:\"kuber\"")
	checkTok("repo:\"k8s/k8s\"")
	checkTok("repo:\"k8s/t-i\"")
	checkTok("-repo:\"org/repo1\"")
	checkTok("-repo:\"org/repo2\"")
}
