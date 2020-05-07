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

package label

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
)

const (
	orgMember    = "Alice"
	nonOrgMember = "Bob"
)

func formatLabels(labels ...string) []string {
	r := []string{}
	for _, l := range labels {
		r = append(r, fmt.Sprintf("%s/%s#%d:%s", "org", "repo", 1, l))
	}
	if len(r) == 0 {
		return nil
	}
	return r
}

func TestLabel(t *testing.T) {
	type testCase struct {
		name                  string
		body                  string
		commenter             string
		extraLabels           []string
		expectedNewLabels     []string
		expectedRemovedLabels []string
		expectedBotComment    bool
		repoLabels            []string
		issueLabels           []string
	}
	testcases := []testCase{
		{
			name:                  "Irrelevant comment",
			body:                  "irrelelvant",
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			repoLabels:            []string{},
			issueLabels:           []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Empty Area",
			body:                  "/area",
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			repoLabels:            []string{"area/infra"},
			issueLabels:           []string{"area/infra"},
			commenter:             orgMember,
		},
		{
			name:                  "Add Single Area Label",
			body:                  "/area infra",
			repoLabels:            []string{"area/infra"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("area/infra"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Single Area Label with prefix",
			body:                  "/lh-area infra",
			repoLabels:            []string{"area/infra"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("area/infra"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Single Area Label when already present on Issue",
			body:                  "/area infra",
			repoLabels:            []string{"area/infra"},
			issueLabels:           []string{"area/infra"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Single Priority Label",
			body:                  "/priority critical",
			repoLabels:            []string{"area/infra", "priority/critical"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("priority/critical"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Single Kind Label",
			body:                  "/kind bug",
			repoLabels:            []string{"area/infra", "priority/critical", labels.Bug},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(labels.Bug),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Single Triage Label",
			body:                  "/triage needs-information",
			repoLabels:            []string{"area/infra", "triage/needs-information"},
			issueLabels:           []string{"area/infra"},
			expectedNewLabels:     formatLabels("triage/needs-information"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Adding Labels is Case Insensitive",
			body:                  "/kind BuG",
			repoLabels:            []string{"area/infra", "priority/critical", labels.Bug},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(labels.Bug),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Adding Labels is Case Insensitive",
			body:                  "/kind bug",
			repoLabels:            []string{"area/infra", "priority/critical", labels.Bug},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(labels.Bug),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Can't Add Non Existent Label",
			body:                  "/priority critical",
			repoLabels:            []string{"area/infra"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Non Org Member Can't Add",
			body:                  "/area infra",
			repoLabels:            []string{"area/infra", "priority/critical", labels.Bug},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("area/infra"),
			expectedRemovedLabels: []string{},
			commenter:             nonOrgMember,
		},
		{
			name:                  "Command must start at the beginning of the line",
			body:                  "  /area infra",
			repoLabels:            []string{"area/infra", "area/api", "priority/critical", "priority/urgent", "priority/important", labels.Bug},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Can't Add Labels Non Existing Labels",
			body:                  "/area lgtm",
			repoLabels:            []string{"area/infra", "area/api", "priority/critical"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Multiple Area Labels",
			body:                  "/area api infra",
			repoLabels:            []string{"area/infra", "area/api", "priority/critical", "priority/urgent"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("area/api", "area/infra"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Multiple Area Labels one already present on Issue",
			body:                  "/area api infra",
			repoLabels:            []string{"area/infra", "area/api", "priority/critical", "priority/urgent"},
			issueLabels:           []string{"area/api"},
			expectedNewLabels:     formatLabels("area/infra"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Multiple Priority Labels",
			body:                  "/priority critical important",
			repoLabels:            []string{"priority/critical", "priority/important"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("priority/critical", "priority/important"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Label Prefix Must Match Command (Area-Priority Mismatch)",
			body:                  "/area urgent",
			repoLabels:            []string{"area/infra", "area/api", "priority/critical", "priority/urgent"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Label Prefix Must Match Command (Priority-Area Mismatch)",
			body:                  "/priority infra",
			repoLabels:            []string{"area/infra", "area/api", "priority/critical", "priority/urgent"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Multiple Area Labels (Some Valid)",
			body:                  "/area lgtm infra",
			repoLabels:            []string{"area/infra", "area/api"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("area/infra"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Multiple Committee Labels (Some Valid)",
			body:                  "/committee steering calamity",
			repoLabels:            []string{"committee/conduct", "committee/steering"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("committee/steering"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add Multiple Types of Labels Different Lines",
			body:                  "/priority urgent\n/area infra",
			repoLabels:            []string{"area/infra", "priority/urgent"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("priority/urgent", "area/infra"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Remove Area Label when no such Label on Repo",
			body:                  "/remove-area infra",
			repoLabels:            []string{},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
			expectedBotComment:    true,
		},
		{
			name:                  "Remove Area Label when no such Label on Issue",
			body:                  "/remove-area infra",
			repoLabels:            []string{"area/infra"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
			expectedBotComment:    true,
		},
		{
			name:                  "Remove Area Label",
			body:                  "/remove-area infra",
			repoLabels:            []string{"area/infra"},
			issueLabels:           []string{"area/infra"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("area/infra"),
			commenter:             orgMember,
		},
		{
			name:                  "Remove Area Label with prefix",
			body:                  "/lh-remove-area infra",
			repoLabels:            []string{"area/infra"},
			issueLabels:           []string{"area/infra"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("area/infra"),
			commenter:             orgMember,
		},
		{
			name:                  "Remove Committee Label",
			body:                  "/remove-committee infinite-monkeys",
			repoLabels:            []string{"area/infra", "sig/testing", "committee/infinite-monkeys"},
			issueLabels:           []string{"area/infra", "sig/testing", "committee/infinite-monkeys"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("committee/infinite-monkeys"),
			commenter:             orgMember,
		},
		{
			name:                  "Remove Kind Label",
			body:                  "/remove-kind api-server",
			repoLabels:            []string{"area/infra", "priority/high", "kind/api-server"},
			issueLabels:           []string{"area/infra", "priority/high", "kind/api-server"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("kind/api-server"),
			commenter:             orgMember,
		},
		{
			name:                  "Remove Priority Label",
			body:                  "/remove-priority high",
			repoLabels:            []string{"area/infra", "priority/high"},
			issueLabels:           []string{"area/infra", "priority/high"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("priority/high"),
			commenter:             orgMember,
		},
		{
			name:                  "Remove SIG Label",
			body:                  "/remove-sig testing",
			repoLabels:            []string{"area/infra", "sig/testing"},
			issueLabels:           []string{"area/infra", "sig/testing"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("sig/testing"),
			commenter:             orgMember,
		},
		{
			name:                  "Remove WG Policy",
			body:                  "/remove-wg policy",
			repoLabels:            []string{"area/infra", "wg/policy"},
			issueLabels:           []string{"area/infra", "wg/policy"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("wg/policy"),
			commenter:             orgMember,
		},
		{
			name:                  "Remove Triage Label",
			body:                  "/remove-triage needs-information",
			repoLabels:            []string{"area/infra", "triage/needs-information"},
			issueLabels:           []string{"area/infra", "triage/needs-information"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("triage/needs-information"),
			commenter:             orgMember,
		},
		{
			name:                  "Remove Multiple Labels",
			body:                  "/remove-priority low high\n/remove-kind api-server\n/remove-area  infra",
			repoLabels:            []string{"area/infra", "priority/high", "priority/low", "kind/api-server"},
			issueLabels:           []string{"area/infra", "priority/high", "priority/low", "kind/api-server"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("priority/low", "priority/high", "kind/api-server", "area/infra"),
			commenter:             orgMember,
			expectedBotComment:    true,
		},
		{
			name:                  "Add and Remove Label at the same time",
			body:                  "/remove-area infra\n/area test",
			repoLabels:            []string{"area/infra", "area/test"},
			issueLabels:           []string{"area/infra"},
			expectedNewLabels:     formatLabels("area/test"),
			expectedRemovedLabels: formatLabels("area/infra"),
			commenter:             orgMember,
		},
		{
			name:                  "Add and Remove the same Label",
			body:                  "/remove-area infra\n/area infra",
			repoLabels:            []string{"area/infra"},
			issueLabels:           []string{"area/infra"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("area/infra"),
			commenter:             orgMember,
		},
		{
			name:                  "Multiple Add and Delete Labels",
			body:                  "/remove-area ruby\n/remove-kind srv\n/remove-priority l m\n/area go\n/kind cli\n/priority h",
			repoLabels:            []string{"area/go", "area/ruby", "kind/cli", "kind/srv", "priority/h", "priority/m", "priority/l"},
			issueLabels:           []string{"area/ruby", "kind/srv", "priority/l", "priority/m"},
			expectedNewLabels:     formatLabels("area/go", "kind/cli", "priority/h"),
			expectedRemovedLabels: formatLabels("area/ruby", "kind/srv", "priority/l", "priority/m"),
			commenter:             orgMember,
		},
		{
			name:                  "Do nothing with empty /label command",
			body:                  "/label",
			extraLabels:           []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Do nothing with empty /remove-label command",
			body:                  "/remove-label",
			extraLabels:           []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add custom label",
			body:                  "/label orchestrator/foo",
			extraLabels:           []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("orchestrator/foo"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Add custom label with prefix",
			body:                  "/lh-label orchestrator/foo",
			extraLabels:           []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("orchestrator/foo"),
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Cannot add missing custom label",
			body:                  "/label orchestrator/foo",
			extraLabels:           []string{"orchestrator/jar", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
		{
			name:                  "Remove custom label",
			body:                  "/remove-label orchestrator/foo",
			extraLabels:           []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{"orchestrator/foo"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("orchestrator/foo"),
			commenter:             orgMember,
		},
		{
			name:                  "Remove custom label with prefix",
			body:                  "/lh-remove-label orchestrator/foo",
			extraLabels:           []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{"orchestrator/foo"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("orchestrator/foo"),
			commenter:             orgMember,
		},
		{
			name:                  "Cannot remove missing custom label",
			body:                  "/remove-label orchestrator/jar",
			extraLabels:           []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{"orchestrator/foo"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			commenter:             orgMember,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			sort.Strings(tc.expectedNewLabels)
			fakeScmClient, fakeData := fake.NewDefault()
			fakeClient := scmprovider.ToTestClient(fakeScmClient)
			fakeData.OrgMembers["org"] = []string{orgMember}
			fakeData.RepoLabelsExisting = tc.repoLabels

			// Add initial labels
			for _, label := range tc.issueLabels {
				fakeClient.AddLabel("org", "repo", 1, label, false)
			}
			e := &scmprovider.GenericCommentEvent{
				Action: scm.ActionCreate,
				Body:   tc.body,
				Number: 1,
				Repo:   scm.Repository{Namespace: "org", Name: "repo"},
				Author: scm.User{Login: tc.commenter},
			}
			err := handle(fakeClient, logrus.WithField("plugin", pluginName), tc.extraLabels, e)
			if err != nil {
				t.Fatalf("didn't expect error from label test: %v", err)
			}

			// Check that all the correct labels (and only the correct labels) were added.
			expectLabels := append(formatLabels(tc.issueLabels...), tc.expectedNewLabels...)
			if expectLabels == nil {
				expectLabels = []string{}
			}
			sort.Strings(expectLabels)
			sort.Strings(fakeData.IssueLabelsAdded)
			if !reflect.DeepEqual(expectLabels, fakeData.IssueLabelsAdded) {
				t.Errorf("expected the labels %q to be added, but %q were added.", expectLabels, fakeData.IssueLabelsAdded)
			}

			sort.Strings(tc.expectedRemovedLabels)
			sort.Strings(fakeData.IssueLabelsRemoved)
			if !reflect.DeepEqual(tc.expectedRemovedLabels, fakeData.IssueLabelsRemoved) {
				t.Errorf("expected the labels %q to be removed, but %q were removed.", tc.expectedRemovedLabels, fakeData.IssueLabelsRemoved)
			}
			if len(fakeData.IssueCommentsAdded) > 0 && !tc.expectedBotComment {
				t.Errorf("unexpected bot comments: %#v", fakeData.IssueCommentsAdded)
			}
			if len(fakeData.IssueCommentsAdded) == 0 && tc.expectedBotComment {
				t.Error("expected a bot comment but got none")
			}
		})
	}
}

func TestHelpProvider(t *testing.T) {
	cases := []struct {
		name               string
		config             *plugins.Configuration
		enabledRepos       []string
		err                bool
		configInfoIncludes []string
	}{
		{
			name:               "Empty config",
			config:             &plugins.Configuration{},
			enabledRepos:       []string{"org1", "org2/repo"},
			configInfoIncludes: []string{configString(defaultLabels)},
		},
		{
			name:               "Overlapping org and org/repo",
			config:             &plugins.Configuration{},
			enabledRepos:       []string{"org2", "org2/repo"},
			configInfoIncludes: []string{configString(defaultLabels)},
		},
		{
			name:               "Invalid enabledRepos",
			config:             &plugins.Configuration{},
			enabledRepos:       []string{"org1", "org2/repo/extra"},
			err:                true,
			configInfoIncludes: []string{configString(defaultLabels)},
		},
		{
			name: "With AdditionalLabels",
			config: &plugins.Configuration{
				Label: plugins.Label{
					AdditionalLabels: []string{"sig", "triage", "wg"},
				},
			},
			enabledRepos:       []string{"org1", "org2/repo"},
			configInfoIncludes: []string{configString(append(defaultLabels, "sig", "triage", "wg"))},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pluginHelp, err := helpProvider(c.config, c.enabledRepos)
			if err != nil && !c.err {
				t.Fatalf("helpProvider error: %v", err)
			}
			for _, msg := range c.configInfoIncludes {
				if !strings.Contains(pluginHelp.Config[""], msg) {
					t.Fatalf("helpProvider.Config error mismatch: didn't get %v, but wanted it", msg)
				}
			}
		})
	}
}
