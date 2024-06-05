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

package ownerslabel

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
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

type fakeOwnersClient struct {
	labels map[string]sets.String
}

func (foc *fakeOwnersClient) FindLabelsForFile(path string) sets.String {
	return foc.labels[path]
}

// TestHandle tests that the handle function requests reviews from the correct number of unique users.
func TestHandle(t *testing.T) {
	foc := &fakeOwnersClient{
		labels: map[string]sets.String{
			"a.go": sets.NewString(labels.LGTM, labels.Approved, "kind/docs"),
			"b.go": sets.NewString(labels.LGTM),
			"c.go": sets.NewString(labels.LGTM, "dnm/frozen-docs"),
			"d.sh": sets.NewString("dnm/bash"),
			"e.sh": sets.NewString("dnm/bash"),
		},
	}

	type testCase struct {
		name              string
		filesChanged      []string
		expectedNewLabels []string
		repoLabels        []string
		prLabels          []string
	}
	testcases := []testCase{
		{
			name:              "no labels",
			filesChanged:      []string{"other.go", "something.go"},
			expectedNewLabels: []string{},
			repoLabels:        []string{},
			prLabels:          []string{},
		},
		{
			name:              "1 file 1 label",
			filesChanged:      []string{"b.go"},
			expectedNewLabels: formatLabels(labels.LGTM),
			repoLabels:        []string{labels.LGTM},
			prLabels:          []string{},
		},
		{
			name:              "1 file 3 labels",
			filesChanged:      []string{"a.go"},
			expectedNewLabels: formatLabels(labels.LGTM, labels.Approved, "kind/docs"),
			repoLabels:        []string{labels.LGTM, labels.Approved, "kind/docs"},
			prLabels:          []string{},
		},
		{
			name:              "2 files no overlap",
			filesChanged:      []string{"c.go", "d.sh"},
			expectedNewLabels: formatLabels(labels.LGTM, "dnm/frozen-docs", "dnm/bash"),
			repoLabels:        []string{labels.LGTM, "dnm/frozen-docs", "dnm/bash"},
			prLabels:          []string{},
		},
		{
			name:              "2 files partial overlap",
			filesChanged:      []string{"a.go", "b.go"},
			expectedNewLabels: formatLabels(labels.LGTM, labels.Approved, "kind/docs"),
			repoLabels:        []string{labels.LGTM, labels.Approved, "kind/docs"},
			prLabels:          []string{},
		},
		{
			name:              "2 files complete overlap",
			filesChanged:      []string{"d.sh", "e.sh"},
			expectedNewLabels: formatLabels("dnm/bash"),
			repoLabels:        []string{"dnm/bash"},
			prLabels:          []string{},
		},
		{
			name:              "3 files partial overlap",
			filesChanged:      []string{"a.go", "b.go", "c.go"},
			expectedNewLabels: formatLabels(labels.LGTM, labels.Approved, "kind/docs", "dnm/frozen-docs"),
			repoLabels:        []string{labels.LGTM, labels.Approved, "kind/docs", "dnm/frozen-docs"},
			prLabels:          []string{},
		},
		{
			name:              "no labels to add, initial unrelated label",
			filesChanged:      []string{"other.go", "something.go"},
			expectedNewLabels: []string{},
			repoLabels:        []string{labels.LGTM},
			prLabels:          []string{labels.LGTM},
		},
		{
			name:              "1 file 1 label, already present",
			filesChanged:      []string{"b.go"},
			expectedNewLabels: []string{},
			repoLabels:        []string{labels.LGTM},
			prLabels:          []string{labels.LGTM},
		},
		{
			name:              "1 file 1 label, doesn't exist on the repo",
			filesChanged:      []string{"b.go"},
			expectedNewLabels: []string{},
			repoLabels:        []string{labels.Approved},
			prLabels:          []string{},
		},
		{
			name:              "2 files no overlap, 1 label already present",
			filesChanged:      []string{"c.go", "d.sh"},
			expectedNewLabels: formatLabels(labels.LGTM, "dnm/frozen-docs"),
			repoLabels:        []string{"dnm/bash", labels.Approved, labels.LGTM, "dnm/frozen-docs"},
			prLabels:          []string{"dnm/bash", labels.Approved},
		},
		{
			name:              "2 files complete overlap, label already present",
			filesChanged:      []string{"d.sh", "e.sh"},
			expectedNewLabels: []string{},
			repoLabels:        []string{"dnm/bash"},
			prLabels:          []string{"dnm/bash"},
		},
	}

	for _, tc := range testcases {
		basicPR := scm.PullRequest{
			Number: 1,
			Base: scm.PullRequestBranch{
				Repo: scm.Repository{
					Namespace: "org",
					Name:      "repo",
				},
			},
			Author: scm.User{
				Login: "user",
			},
		}

		t.Logf("Running scenario %q", tc.name)
		sort.Strings(tc.expectedNewLabels)
		changes := make([]*scm.Change, 0, len(tc.filesChanged))
		for _, name := range tc.filesChanged {
			changes = append(changes, &scm.Change{Path: name})
		}
		fakeScmClient, fspc := fake.NewDefault()
		fakeClient := scmprovider.ToTestClient(fakeScmClient)

		fspc.PullRequests[basicPR.Number] = &basicPR
		fspc.PullRequestChanges[basicPR.Number] = changes
		fspc.RepoLabelsExisting = tc.repoLabels

		// Add initial labels
		for _, label := range tc.prLabels {
			_ = fakeClient.AddLabel(basicPR.Base.Repo.Namespace, basicPR.Base.Repo.Name, basicPR.Number, label, true)
		}
		pre := &scm.PullRequestHook{
			Action:      scm.ActionOpen,
			PullRequest: basicPR,
			Repo:        basicPR.Base.Repo,
		}

		err := handle(fakeClient, foc, logrus.WithField("plugin", pluginName), pre)
		if err != nil {
			t.Errorf("[%s] unexpected error from handle: %v", tc.name, err)
			continue
		}

		// Check that all the correct labels (and only the correct labels) were added.
		expectLabels := append(formatLabels(tc.prLabels...), tc.expectedNewLabels...)
		if expectLabels == nil {
			expectLabels = []string{}
		}
		sort.Strings(expectLabels)
		sort.Strings(fspc.PullRequestLabelsAdded)
		if !reflect.DeepEqual(expectLabels, fspc.PullRequestLabelsAdded) {
			t.Errorf("expected the labels %q to be added, but %q were added.", expectLabels, fspc.PullRequestLabelsAdded)
		}

	}
}
