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

package size

import (
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

type spc struct {
	*testing.T
	labels    map[scm.Label]bool
	files     map[string][]byte
	prChanges []*scm.Change

	addLabelErr, removeLabelErr, getIssueLabelsErr,
	getFileErr, getPullRequestChangesErr error
}

func (c *spc) AddLabel(_, _ string, _ int, label string, _ bool) error {
	c.T.Logf("AddLabel: %s", label)
	c.labels[scm.Label{Name: label}] = true

	return c.addLabelErr
}

func (c *spc) RemoveLabel(_, _ string, _ int, label string, _ bool) error {
	c.T.Logf("RemoveLabel: %s", label)
	for k := range c.labels {
		if k.Name == label {
			delete(c.labels, k)
		}
	}

	return c.removeLabelErr
}

func (c *spc) GetIssueLabels(_, _ string, _ int, _ bool) (ls []*scm.Label, err error) {
	c.T.Log("GetIssueLabels")
	for k, ok := range c.labels {
		if ok {
			copy := k
			ls = append(ls, &copy)
		}
	}

	err = c.getIssueLabelsErr
	return
}

func (c *spc) GetFile(_, _, path, _ string) ([]byte, error) {
	c.T.Logf("GetFile: %s", path)
	return c.files[path], c.getFileErr
}

func (c *spc) GetPullRequestChanges(_, _ string, _ int) ([]*scm.Change, error) {
	c.T.Log("GetPullRequestChanges")
	return c.prChanges, c.getPullRequestChangesErr
}

func TestSizesOrDefault(t *testing.T) {
	for _, c := range []struct {
		input    plugins.Size
		expected plugins.Size
	}{
		{
			input:    defaultSizes,
			expected: defaultSizes,
		},
		{
			input: plugins.Size{
				S:   12,
				M:   15,
				L:   17,
				Xl:  21,
				Xxl: 51,
			},
			expected: plugins.Size{
				S:   12,
				M:   15,
				L:   17,
				Xl:  21,
				Xxl: 51,
			},
		},
		{
			input:    plugins.Size{},
			expected: defaultSizes,
		},
	} {
		if c.expected != sizesOrDefault(c.input) {
			t.Fatalf("Unexpected sizes from sizesOrDefault - expected %+v but got %+v", c.expected, sizesOrDefault(c.input))
		}
	}
}

func TestHandlePR(t *testing.T) {
	cases := []struct {
		name        string
		client      *spc
		event       scm.PullRequestHook
		err         error
		sizes       plugins.Size
		finalLabels []*scm.Label
	}{
		{
			name: "simple size/S, no .generated_files",
			client: &spc{
				labels:     map[scm.Label]bool{},
				getFileErr: scm.ErrNotFound,
				prChanges: []*scm.Change{
					{
						Sha:       "abcd",
						Path:      "foobar",
						Additions: 10,
						Deletions: 10,
						Changes:   20,
					},
					{
						Sha:       "abcd",
						Path:      "barfoo",
						Additions: 3,
						Deletions: 4,
						Changes:   7,
					},
				},
			},
			event: scm.PullRequestHook{
				Action: scm.ActionOpen,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Sha: "abcd",
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
				},
			},
			finalLabels: []*scm.Label{
				{Name: "size/S"},
			},
			sizes: defaultSizes,
		},
		{
			name: "simple size/M, with .generated_files",
			client: &spc{
				labels: map[scm.Label]bool{},
				files: map[string][]byte{
					".generated_files": []byte(`
						file-name foobar

						path-prefix generated
					`),
				},
				prChanges: []*scm.Change{
					{
						Sha:       "abcd",
						Path:      "foobar",
						Additions: 10,
						Deletions: 10,
						Changes:   20,
					},
					{
						Sha:       "abcd",
						Path:      "barfoo",
						Additions: 50,
						Deletions: 0,
						Changes:   50,
					},
					{
						Sha:       "abcd",
						Path:      "generated/what.txt",
						Additions: 30,
						Deletions: 0,
						Changes:   30,
					},
					{
						Sha:       "abcd",
						Path:      "generated/my/file.txt",
						Additions: 300,
						Deletions: 0,
						Changes:   300,
					},
				},
			},
			event: scm.PullRequestHook{
				Action: scm.ActionOpen,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Sha: "abcd",
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
				},
			},
			finalLabels: []*scm.Label{
				{Name: "size/M"},
			},
			sizes: defaultSizes,
		},
		{
			name: "simple size/M, with .gitattributes",
			client: &spc{
				labels: map[scm.Label]bool{},
				files: map[string][]byte{
					".gitattributes": []byte(`
						# comments
						foobar linguist-generated=true
						generated/**/*.txt linguist-generated=true
					`),
				},
				prChanges: []*scm.Change{
					{
						Sha:       "abcd",
						Path:      "foobar",
						Additions: 10,
						Deletions: 10,
						Changes:   20,
					},
					{
						Sha:       "abcd",
						Path:      "barfoo",
						Additions: 50,
						Deletions: 0,
						Changes:   50,
					},
					{
						Sha:       "abcd",
						Path:      "generated/what.txt",
						Additions: 30,
						Deletions: 0,
						Changes:   30,
					},
					{
						Sha:       "abcd",
						Path:      "generated/my/file.txt",
						Additions: 300,
						Deletions: 0,
						Changes:   300,
					},
				},
			},
			event: scm.PullRequestHook{
				Action: scm.ActionOpen,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Sha: "abcd",
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
				},
			},
			finalLabels: []*scm.Label{
				{Name: "size/M"},
			},
			sizes: defaultSizes,
		},
		{
			name: "simple size/XS, with .generated_files and paths-from-repo",
			client: &spc{
				labels: map[scm.Label]bool{},
				files: map[string][]byte{
					".generated_files": []byte(`
						# Comments
						file-name foobar

						path-prefix generated

						paths-from-repo docs/.generated_docs
					`),
					"docs/.generated_docs": []byte(`
					# Comments work

					# And empty lines don't matter
					foobar
					mypath1
					mypath2
					mydir/mypath3
					`),
				},
				prChanges: []*scm.Change{
					{
						Sha:       "abcd",
						Path:      "foobar",
						Additions: 10,
						Deletions: 10,
						Changes:   20,
					},
					{ // Notice "barfoo" is the only relevant change.
						Sha:       "abcd",
						Path:      "barfoo",
						Additions: 5,
						Deletions: 0,
						Changes:   5,
					},
					{
						Sha:       "abcd",
						Path:      "generated/what.txt",
						Additions: 30,
						Deletions: 0,
						Changes:   30,
					},
					{
						Sha:       "abcd",
						Path:      "generated/my/file.txt",
						Additions: 300,
						Deletions: 0,
						Changes:   300,
					},
					{
						Sha:       "abcd",
						Path:      "mypath1",
						Additions: 300,
						Deletions: 0,
						Changes:   300,
					},
					{
						Sha:       "abcd",
						Path:      "mydir/mypath3",
						Additions: 300,
						Deletions: 0,
						Changes:   300,
					},
				},
			},
			event: scm.PullRequestHook{
				Action: scm.ActionOpen,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Sha: "abcd",
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
				},
			},
			finalLabels: []*scm.Label{
				{Name: "size/XS"},
			},
			sizes: defaultSizes,
		},
		{
			name:   "pr closed event",
			client: &spc{},
			event: scm.PullRequestHook{
				Action: scm.ActionClose,
			},
			finalLabels: []*scm.Label{},
			sizes:       defaultSizes,
		},
		{
			name: "XS -> S transition",
			client: &spc{
				labels: map[scm.Label]bool{
					{Name: "irrelevant"}: true,
					{Name: "size/XS"}:    true,
				},
				files: map[string][]byte{
					".generated_files": []byte(`
						# Comments
						file-name foobar

						path-prefix generated

						paths-from-repo docs/.generated_docs
					`),
					"docs/.generated_docs": []byte(`
					# Comments work

					# And empty lines don't matter
					foobar
					mypath1
					mypath2
					mydir/mypath3
					`),
				},
				prChanges: []*scm.Change{
					{
						Sha:       "abcd",
						Path:      "foobar",
						Additions: 10,
						Deletions: 10,
						Changes:   20,
					},
					{ // Notice "barfoo" is the only relevant change.
						Sha:       "abcd",
						Path:      "barfoo",
						Additions: 5,
						Deletions: 0,
						Changes:   5,
					},
					{
						Sha:       "abcd",
						Path:      "generated/what.txt",
						Additions: 30,
						Deletions: 0,
						Changes:   30,
					},
					{
						Sha:       "abcd",
						Path:      "generated/my/file.txt",
						Additions: 300,
						Deletions: 0,
						Changes:   300,
					},
					{
						Sha:       "abcd",
						Path:      "mypath1",
						Additions: 300,
						Deletions: 0,
						Changes:   300,
					},
					{
						Sha:       "abcd",
						Path:      "mydir/mypath3",
						Additions: 300,
						Deletions: 0,
						Changes:   300,
					},
				},
			},
			event: scm.PullRequestHook{
				Action: scm.ActionOpen,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Sha: "abcd",
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
				},
			},
			finalLabels: []*scm.Label{
				{Name: "irrelevant"},
				{Name: "size/XS"},
			},
			sizes: defaultSizes,
		},
		{
			name: "pull request reopened",
			client: &spc{
				labels:     map[scm.Label]bool{},
				getFileErr: scm.ErrNotFound,
				prChanges: []*scm.Change{
					{
						Sha:       "abcd",
						Path:      "foobar",
						Additions: 10,
						Deletions: 10,
						Changes:   20,
					},
					{
						Sha:       "abcd",
						Path:      "barfoo",
						Additions: 3,
						Deletions: 4,
						Changes:   7,
					},
				},
			},
			event: scm.PullRequestHook{
				Action: scm.ActionReopen,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Sha: "abcd",
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
				},
			},
			finalLabels: []*scm.Label{
				{Name: "size/S"},
			},
			sizes: defaultSizes,
		},
		{
			name: "pull request edited",
			client: &spc{
				labels:     map[scm.Label]bool{},
				getFileErr: scm.ErrNotFound,
				prChanges: []*scm.Change{
					{
						Sha:       "abcd",
						Path:      "foobar",
						Additions: 30,
						Deletions: 40,
						Changes:   70,
					},
				},
			},
			event: scm.PullRequestHook{
				Action: scm.ActionEdited,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Sha: "abcd",
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
				},
			},
			finalLabels: []*scm.Label{
				{Name: "size/M"},
			},
			sizes: defaultSizes,
		},
		{
			name: "different label constraints",
			client: &spc{
				labels:     map[scm.Label]bool{},
				getFileErr: scm.ErrNotFound,
				prChanges: []*scm.Change{
					{
						Sha:       "abcd",
						Path:      "foobar",
						Additions: 10,
						Deletions: 10,
						Changes:   20,
					},
					{
						Sha:       "abcd",
						Path:      "barfoo",
						Additions: 3,
						Deletions: 4,
						Changes:   7,
					},
				},
			},
			event: scm.PullRequestHook{
				Action: scm.ActionOpen,
				PullRequest: scm.PullRequest{
					Number: 101,
					Base: scm.PullRequestBranch{
						Sha: "abcd",
						Repo: scm.Repository{
							Namespace: "kubernetes",
							Name:      "kubernetes",
						},
					},
				},
			},
			finalLabels: []*scm.Label{
				{Name: "size/XXL"},
			},
			sizes: plugins.Size{
				S:   0,
				M:   1,
				L:   2,
				Xl:  3,
				Xxl: 4,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.client == nil {
				t.Fatalf("case can not have nil github client")
			}

			// Set up test logging.
			c.client.T = t

			err := handlePR(c.client, c.sizes, logrus.NewEntry(logrus.New()), c.event)

			if err != nil && c.err == nil {
				t.Fatalf("handlePR error: %v", err)
			}

			if err == nil && c.err != nil {
				t.Fatalf("handlePR wanted error %v, got nil", err)
			}

			if got, want := err, c.err; got != nil && got.Error() != want.Error() {
				t.Fatalf("handlePR errors mismatch: got %v, want %v", got, want)
			}

			if got, want := len(c.client.labels), len(c.finalLabels); got != want {
				t.Logf("github client labels: got %v; want %v", c.client.labels, c.finalLabels)
				t.Fatalf("finalLabels count mismatch: got %d, want %d", got, want)
			}

			for _, l := range c.finalLabels {
				if !c.client.labels[*l] {
					t.Fatalf("github client labels missing %v", l)
				}
			}
		})
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
			name: "Empty sizes",
			config: &plugins.Configuration{
				Size: plugins.Size{},
			},
			enabledRepos: []string{"org1", "org2/repo"},
		},
		{
			name: "Sizes specified",
			config: &plugins.Configuration{
				Size: plugins.Size{
					S:   12,
					M:   15,
					L:   17,
					Xl:  21,
					Xxl: 51,
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
