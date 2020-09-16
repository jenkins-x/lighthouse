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

package welcome

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"

	"sigs.k8s.io/yaml"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	testWelcomeTemplate = "Welcome human! 🤖 {{.AuthorName}} {{.AuthorLogin}} {{.Repo}} {{.Org}}}"
)

type fakeClient struct {
	commentsAdded map[int][]string
	prs           map[string]sets.Int
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		commentsAdded: make(map[int][]string),
		prs:           make(map[string]sets.Int),
	}
}

// CreateComment adds and tracks a comment in the client
func (fc *fakeClient) CreateComment(owner, repo string, number int, pr bool, comment string) error {
	fc.commentsAdded[number] = append(fc.commentsAdded[number], comment)
	return nil
}

// ClearComments removes all comments in the client
func (fc *fakeClient) ClearComments() {
	fc.commentsAdded = map[int][]string{}
}

// NumComments counts the number of tracked comments
func (fc *fakeClient) NumComments() int {
	n := 0
	for _, comments := range fc.commentsAdded {
		n += len(comments)
	}
	return n
}

// AddPR records an PR in the client
func (fc *fakeClient) AddPR(owner, repo, author string, number int) {
	key := fmt.Sprintf("%s,%s,%s", owner, repo, author)
	if _, ok := fc.prs[key]; !ok {
		fc.prs[key] = sets.Int{}
	}
	fc.prs[key].Insert(number)
}

// ClearPRs removes all PRs from the client
func (fc *fakeClient) ClearPRs() {
	fc.prs = make(map[string]sets.Int)
}

// FindIssues fails if the query does not match the expected query regex and
// looks up issues based on parsing the expected query format
func (fc *fakeClient) FindPullRequestsByAuthor(owner, repo string, author string) ([]*scm.PullRequest, error) {
	// "find" results
	key := fmt.Sprintf("%s,%s,%s", owner, repo, author)

	issues := []*scm.PullRequest{}
	for _, number := range fc.prs[key].List() {
		issues = append(issues, &scm.PullRequest{
			Number: number,
		})
	}
	return issues, nil
}

func makeFakePullRequestEvent(owner, repo, author string, number int, action scm.Action) scm.PullRequestHook {
	return scm.PullRequestHook{
		Action: action,
		PullRequest: scm.PullRequest{
			Number: number,
			Base: scm.PullRequestBranch{
				Repo: scm.Repository{
					Namespace: owner,
					Name:      repo,
				},
			},
			Author: scm.User{
				Login: author,
				Name:  author + "fullname",
			},
		},
	}
}

func TestHandlePR(t *testing.T) {
	fc := newFakeClient()
	// old PRs
	fc.AddPR("kubernetes", "test-infra", "contributorA", 1)
	fc.AddPR("kubernetes", "test-infra", "contributorB", 2)
	fc.AddPR("kubernetes", "test-infra", "contributorB", 3)

	testCases := []struct {
		name          string
		repoOwner     string
		repoName      string
		author        string
		prNumber      int
		prAction      scm.Action
		addPR         bool
		expectComment bool
	}{
		{
			name:          "existing contributorA",
			repoOwner:     "kubernetes",
			repoName:      "test-infra",
			author:        "contributorA",
			prNumber:      20,
			prAction:      scm.ActionOpen,
			expectComment: false,
		},
		{
			name:          "existing contributorB",
			repoOwner:     "kubernetes",
			repoName:      "test-infra",
			author:        "contributorB",
			prNumber:      40,
			prAction:      scm.ActionOpen,
			expectComment: false,
		},
		{
			name:          "new contributor",
			repoOwner:     "kubernetes",
			repoName:      "test-infra",
			author:        "newContributor",
			prAction:      scm.ActionOpen,
			prNumber:      50,
			expectComment: true,
		},
		{
			name:          "new contributor and API recorded PR already",
			repoOwner:     "kubernetes",
			repoName:      "test-infra",
			author:        "newContributor",
			prAction:      scm.ActionOpen,
			prNumber:      50,
			expectComment: true,
			addPR:         true,
		},
		{
			name:          "new contributor, not PR open event",
			repoOwner:     "kubernetes",
			repoName:      "test-infra",
			author:        "newContributor",
			prAction:      scm.ActionEdited,
			prNumber:      50,
			expectComment: false,
		},
	}

	c := client{
		SCMProviderClient: fc,
		Logger:            &logrus.Entry{},
	}
	for _, tc := range testCases {
		// clear out comments from the last test case
		fc.ClearComments()

		event := makeFakePullRequestEvent(tc.repoOwner, tc.repoName, tc.author, tc.prNumber, tc.prAction)
		if tc.addPR {
			// make sure the PR in the event is recorded
			fc.AddPR(tc.repoOwner, tc.repoName, tc.author, tc.prNumber)
		}

		// try handling it
		if err := handlePR(c, event, testWelcomeTemplate); err != nil {
			t.Fatalf("did not expect error handling PR for case '%s': %v", tc.name, err)
		}

		// verify that comments were made
		numComments := fc.NumComments()
		if numComments > 1 {
			t.Fatalf("did not expect multiple comments for any test case and got %d comments", numComments)
		}
		if numComments == 0 && tc.expectComment {
			t.Fatalf("expected a comment for case '%s' and got none", tc.name)
		} else if numComments > 0 && !tc.expectComment {
			t.Fatalf("did not expect comments for case '%s' and got %d comments", tc.name, numComments)
		}
	}
}

func TestWelcomeConfig(t *testing.T) {
	var (
		orgMessage  = "defined message for an org"
		repoMessage = "defined message for a repo"
	)

	config := &plugins.Configuration{
		Welcome: []plugins.Welcome{
			{
				Repos:           []string{"kubernetes/test-infra"},
				MessageTemplate: repoMessage,
			},
			{
				Repos:           []string{"kubernetes"},
				MessageTemplate: orgMessage,
			},
			{
				Repos:           []string{"kubernetes/repo-infra"},
				MessageTemplate: repoMessage,
			},
		},
	}

	testCases := []struct {
		name            string
		repo            string
		org             string
		expectedMessage string
	}{
		{
			name:            "default message",
			org:             "kubernetes-sigs",
			repo:            "kind",
			expectedMessage: defaultWelcomeMessage,
		},
		{
			name:            "org defined message",
			org:             "kubernetes",
			repo:            "community",
			expectedMessage: orgMessage,
		},
		{
			name:            "repo defined message, before an org",
			org:             "kubernetes",
			repo:            "test-infra",
			expectedMessage: repoMessage,
		},
		{
			name:            "repo defined message, after an org",
			org:             "kubernetes",
			repo:            "repo-infra",
			expectedMessage: repoMessage,
		},
	}

	for _, tc := range testCases {
		receivedMessage := welcomeMessageForRepo(config, tc.org, tc.repo)
		if receivedMessage != tc.expectedMessage {
			t.Fatalf("%s: expected to get '%s' and received '%s'", tc.name, tc.expectedMessage, receivedMessage)
		}
	}
}

// TestPluginConfig validates that there are no duplicate repos in the welcome plugin config.
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
	for _, config := range pa.Config().Welcome {
		for _, entry := range config.Repos {
			if strings.Contains(entry, "/") {
				if repos[entry] {
					t.Errorf("The repo %q is duplicated in the 'welcome' plugin configuration.", entry)
				}
				repos[entry] = true
			} else {
				if orgs[entry] {
					t.Errorf("The org %q is duplicated in the 'welcome' plugin configuration.", entry)
				}
				orgs[entry] = true
			}
		}
	}
	for repo := range repos {
		org := strings.Split(repo, "/")[0]
		if orgs[org] {
			t.Errorf("The repo %q is duplicated with %q in the 'welcome' plugin configuration.", repo, org)
		}
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
				Welcome: []plugins.Welcome{
					{
						Repos:           []string{"org2"},
						MessageTemplate: "Hello, welcome!",
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
