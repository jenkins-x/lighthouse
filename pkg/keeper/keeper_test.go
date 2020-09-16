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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	fakescm "github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/config/keeper"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"k8s.io/apimachinery/pkg/api/equality"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/diff"

	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/git/localgit"
	"github.com/jenkins-x/lighthouse/pkg/keeper/history"
	launcherfake "github.com/jenkins-x/lighthouse/pkg/launcher/fake"
)

func testPullsMatchList(t *testing.T, test string, actual []PullRequest, expected []int) {
	if len(actual) != len(expected) {
		t.Errorf("Wrong size for case %s. Got PRs %+v, wanted numbers %v.", test, actual, expected)
		return
	}
	for _, pr := range actual {
		var found bool
		n1 := int(pr.Number)
		for _, n2 := range expected {
			if n1 == n2 {
				found = true
			}
		}
		if !found {
			t.Errorf("For case %s, found PR %d but shouldn't have.", test, n1)
		}
	}
}

func clearCompiledRegexes(presubmits []job.Presubmit) {
	for i := range presubmits {
		presubmits[i].ClearCompiledRegexes()
	}
}

func TestAccumulateBatch(t *testing.T) {
	jobSet := []job.Presubmit{
		{
			Reporter: job.Reporter{Context: "foo"},
		},
		{
			Reporter: job.Reporter{Context: "bar"},
		},
		{
			Reporter: job.Reporter{Context: "baz"},
		},
	}
	type pull struct {
		number int
		sha    string
	}
	type activity struct {
		prs   []pull
		job   string
		state v1alpha1.PipelineState
	}
	tests := []struct {
		name       string
		presubmits map[int][]job.Presubmit
		pulls      []pull
		activities []activity

		merges  []int
		pending bool
	}{
		{
			name: "no batches running",
		},
		{
			name: "batch pending",
			presubmits: map[int][]job.Presubmit{
				1: {{Reporter: job.Reporter{Context: "foo"}}},
				2: {{Reporter: job.Reporter{Context: "foo"}}},
			},
			pulls:      []pull{{1, "a"}, {2, "b"}},
			activities: []activity{{job: "foo", state: v1alpha1.PendingState, prs: []pull{{1, "a"}}}},
			pending:    true,
		},
		{
			name:       "pending batch missing presubmits is ignored",
			presubmits: map[int][]job.Presubmit{1: jobSet},
			pulls:      []pull{{1, "a"}, {2, "b"}},
			activities: []activity{{job: "foo", state: v1alpha1.PendingState, prs: []pull{{1, "a"}}}},
		},
		{
			name:       "batch pending, successful previous run",
			presubmits: map[int][]job.Presubmit{1: jobSet, 2: jobSet},
			pulls:      []pull{{1, "a"}, {2, "b"}},
			activities: []activity{
				{job: "foo", state: v1alpha1.PendingState, prs: []pull{{1, "a"}}},
				{job: "bar", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}}},
				{job: "baz", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}}},
				{job: "foo", state: v1alpha1.SuccessState, prs: []pull{{2, "b"}}},
				{job: "bar", state: v1alpha1.SuccessState, prs: []pull{{2, "b"}}},
				{job: "baz", state: v1alpha1.SuccessState, prs: []pull{{2, "b"}}},
			},
			pending: true,
			merges:  []int{2},
		},
		{
			name:       "successful run",
			presubmits: map[int][]job.Presubmit{1: jobSet, 2: jobSet},
			pulls:      []pull{{1, "a"}, {2, "b"}},
			activities: []activity{
				{job: "foo", state: v1alpha1.SuccessState, prs: []pull{{2, "b"}}},
				{job: "bar", state: v1alpha1.SuccessState, prs: []pull{{2, "b"}}},
				{job: "baz", state: v1alpha1.SuccessState, prs: []pull{{2, "b"}}},
			},
			merges: []int{2},
		},
		{
			name:       "successful run, multiple PRs",
			presubmits: map[int][]job.Presubmit{1: jobSet, 2: jobSet},
			pulls:      []pull{{1, "a"}, {2, "b"}},
			activities: []activity{
				{job: "foo", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "bar", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "baz", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
			},
			merges: []int{1, 2},
		},
		{
			name:       "successful run, failures in past",
			presubmits: map[int][]job.Presubmit{1: jobSet, 2: jobSet},
			pulls:      []pull{{1, "a"}, {2, "b"}},
			activities: []activity{
				{job: "foo", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "bar", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "baz", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "foo", state: v1alpha1.FailureState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "baz", state: v1alpha1.FailureState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "foo", state: v1alpha1.FailureState, prs: []pull{{1, "c"}, {2, "b"}}},
			},
			merges: []int{1, 2},
		},
		{
			name:       "failures",
			presubmits: map[int][]job.Presubmit{1: jobSet, 2: jobSet},
			pulls:      []pull{{1, "a"}, {2, "b"}},
			activities: []activity{
				{job: "foo", state: v1alpha1.FailureState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "bar", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "baz", state: v1alpha1.FailureState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "foo", state: v1alpha1.FailureState, prs: []pull{{1, "c"}, {2, "b"}}},
			},
		},
		{
			name:       "missing job required by one PR",
			presubmits: map[int][]job.Presubmit{1: jobSet, 2: append(jobSet, job.Presubmit{Reporter: job.Reporter{Context: "boo"}})},
			pulls:      []pull{{1, "a"}, {2, "b"}},
			activities: []activity{
				{job: "foo", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "bar", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "baz", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
			},
		},
		{
			name:       "successful run with PR that requires additional job",
			presubmits: map[int][]job.Presubmit{1: jobSet, 2: append(jobSet, job.Presubmit{Reporter: job.Reporter{Context: "boo"}})},
			pulls:      []pull{{1, "a"}, {2, "b"}},
			activities: []activity{
				{job: "foo", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "bar", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "baz", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
				{job: "boo", state: v1alpha1.SuccessState, prs: []pull{{1, "a"}, {2, "b"}}},
			},
			merges: []int{1, 2},
		},
		{
			name:    "no presubmits",
			pulls:   []pull{{1, "a"}, {2, "b"}},
			pending: false,
		},
		{
			name:       "pending batch with PR that left pool, successful previous run",
			presubmits: map[int][]job.Presubmit{2: jobSet},
			pulls:      []pull{{2, "b"}},
			activities: []activity{
				{job: "foo", state: v1alpha1.PendingState, prs: []pull{{1, "a"}}},
				{job: "foo", state: v1alpha1.SuccessState, prs: []pull{{2, "b"}}},
				{job: "bar", state: v1alpha1.SuccessState, prs: []pull{{2, "b"}}},
				{job: "baz", state: v1alpha1.SuccessState, prs: []pull{{2, "b"}}},
			},
			pending: false,
			merges:  []int{2},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var pulls []PullRequest
			for _, p := range test.pulls {
				pr := PullRequest{
					Number:     githubql.Int(p.number),
					HeadRefOID: githubql.String(p.sha),
				}
				pulls = append(pulls, pr)
			}
			var pjs []v1alpha1.LighthouseJob
			for _, pj := range test.activities {
				npj := v1alpha1.LighthouseJob{
					Spec: v1alpha1.LighthouseJobSpec{
						Job:     pj.job,
						Context: pj.job,
						Type:    job.BatchJob,
						Refs:    new(v1alpha1.Refs),
					},
					Status: v1alpha1.LighthouseJobStatus{State: pj.state},
				}
				for _, pr := range pj.prs {
					npj.Spec.Refs.Pulls = append(npj.Spec.Refs.Pulls, v1alpha1.Pull{
						Number: pr.number,
						SHA:    pr.sha,
					})
				}
				pjs = append(pjs, npj)
			}
			merges, pending := accumulateBatch(test.presubmits, pulls, pjs, logrus.NewEntry(logrus.New()))
			if (len(pending) > 0) != test.pending {
				t.Errorf("For case \"%s\", got wrong pending.", test.name)
			}
			testPullsMatchList(t, test.name, merges, test.merges)
		})
	}
}

func TestAccumulate(t *testing.T) {
	jobSet := []job.Presubmit{
		{
			Reporter: job.Reporter{
				Context: "job1",
			},
		},
		{
			Reporter: job.Reporter{
				Context: "job2",
			},
		},
	}
	type activity struct {
		prNumber int
		job      string
		state    v1alpha1.PipelineState
		sha      string
	}
	tests := []struct {
		name         string
		presubmits   map[int][]job.Presubmit
		pullRequests map[int]string
		activities   []activity

		successes []int
		pendings  []int
		none      []int
	}{
		{
			name:         "seven PRs, two jobs, with an overridden context and an unrelated success context",
			pullRequests: map[int]string{1: "sha1", 2: "sha2", 3: "sha3", 4: "sha4", 5: "sha5", 6: "sha6", 7: "sha7"},
			presubmits: map[int][]job.Presubmit{
				1: jobSet,
				2: jobSet,
				3: jobSet,
				4: jobSet,
				5: jobSet,
				6: jobSet,
				7: jobSet,
			},
			activities: []activity{
				{2, "job1", v1alpha1.PendingState, "sha2"},
				{2, "job2", v1alpha1.FailureState, "sha2"},
				{3, "job1", v1alpha1.PendingState, "sha3"},
				{3, "job2", v1alpha1.TriggeredState, "sha3"},
				{4, "job1", v1alpha1.FailureState, "sha4"},
				{4, "job2", v1alpha1.PendingState, "sha4"},
				{5, "job1", v1alpha1.PendingState, "sha5"},
				{5, "job2", v1alpha1.FailureState, "sha5"},
				{5, "job2", v1alpha1.PendingState, "sha5"},
				{6, "job1", v1alpha1.SuccessState, "sha6"},
				{6, "job2", v1alpha1.PendingState, "sha6"},
				{7, "job1", v1alpha1.SuccessState, "sha7"},
				{7, "job2", v1alpha1.SuccessState, "sha7"},
				{7, "job1", v1alpha1.FailureState, "sha7"},
			},

			successes: []int{7},
			pendings:  []int{3, 5, 6},
			none:      []int{1, 2, 4},
		},
		{
			name:         "one PR, four jobs, mix of failures and success",
			pullRequests: map[int]string{7: ""},
			presubmits: map[int][]job.Presubmit{
				7: {
					{Reporter: job.Reporter{Context: "job1"}},
					{Reporter: job.Reporter{Context: "job2"}},
					{Reporter: job.Reporter{Context: "job3"}},
					{Reporter: job.Reporter{Context: "job4"}},
				},
			},
			activities: []activity{
				{7, "job1", v1alpha1.SuccessState, ""},
				{7, "job2", v1alpha1.FailureState, ""},
				{7, "job3", v1alpha1.FailureState, ""},
				{7, "job4", v1alpha1.FailureState, ""},
				{7, "job3", v1alpha1.FailureState, ""},
				{7, "job4", v1alpha1.FailureState, ""},
				{7, "job2", v1alpha1.SuccessState, ""},
				{7, "job3", v1alpha1.SuccessState, ""},
				{7, "job4", v1alpha1.FailureState, ""},
			},

			successes: []int{},
			pendings:  []int{},
			none:      []int{7},
		},
		{
			name:         "one PR, four jobs, all failure",
			pullRequests: map[int]string{7: ""},
			presubmits: map[int][]job.Presubmit{
				7: {
					{Reporter: job.Reporter{Context: "job1"}},
					{Reporter: job.Reporter{Context: "job2"}},
					{Reporter: job.Reporter{Context: "job3"}},
					{Reporter: job.Reporter{Context: "job4"}},
				},
			},
			activities: []activity{
				{7, "job1", v1alpha1.FailureState, ""},
				{7, "job2", v1alpha1.FailureState, ""},
				{7, "job3", v1alpha1.FailureState, ""},
				{7, "job4", v1alpha1.FailureState, ""},
				{7, "job3", v1alpha1.FailureState, ""},
				{7, "job4", v1alpha1.FailureState, ""},
				{7, "job2", v1alpha1.FailureState, ""},
				{7, "job3", v1alpha1.FailureState, ""},
				{7, "job4", v1alpha1.FailureState, ""},
			},

			successes: []int{},
			pendings:  []int{},
			none:      []int{7},
		},
		{
			name:         "one PR, four jobs, latest all succeed",
			pullRequests: map[int]string{7: ""},
			presubmits: map[int][]job.Presubmit{
				7: {
					{Reporter: job.Reporter{Context: "job1"}},
					{Reporter: job.Reporter{Context: "job2"}},
					{Reporter: job.Reporter{Context: "job3"}},
					{Reporter: job.Reporter{Context: "job4"}},
				},
			},
			activities: []activity{
				{7, "job1", v1alpha1.SuccessState, ""},
				{7, "job2", v1alpha1.FailureState, ""},
				{7, "job3", v1alpha1.FailureState, ""},
				{7, "job4", v1alpha1.FailureState, ""},
				{7, "job3", v1alpha1.FailureState, ""},
				{7, "job4", v1alpha1.FailureState, ""},
				{7, "job2", v1alpha1.SuccessState, ""},
				{7, "job3", v1alpha1.SuccessState, ""},
				{7, "job4", v1alpha1.SuccessState, ""},
				{7, "job1", v1alpha1.FailureState, ""},
			},

			successes: []int{7},
			pendings:  []int{},
			none:      []int{},
		},
		{
			name:         "one PR, four jobs, one is pending",
			pullRequests: map[int]string{7: ""},
			presubmits: map[int][]job.Presubmit{
				7: {
					{Reporter: job.Reporter{Context: "job1"}},
					{Reporter: job.Reporter{Context: "job2"}},
					{Reporter: job.Reporter{Context: "job3"}},
					{Reporter: job.Reporter{Context: "job4"}},
				},
			},
			activities: []activity{
				{7, "job1", v1alpha1.SuccessState, ""},
				{7, "job2", v1alpha1.FailureState, ""},
				{7, "job3", v1alpha1.FailureState, ""},
				{7, "job4", v1alpha1.FailureState, ""},
				{7, "job3", v1alpha1.FailureState, ""},
				{7, "job4", v1alpha1.FailureState, ""},
				{7, "job2", v1alpha1.SuccessState, ""},
				{7, "job3", v1alpha1.SuccessState, ""},
				{7, "job4", v1alpha1.PendingState, ""},
				{7, "job1", v1alpha1.FailureState, ""},
			},

			successes: []int{},
			pendings:  []int{7},
			none:      []int{},
		},
		{
			name: "two PRs, one job, one success and one failure",
			presubmits: map[int][]job.Presubmit{
				7: {
					{Reporter: job.Reporter{Context: "job1"}},
				},
			},
			pullRequests: map[int]string{7: "new", 8: "new"},
			activities: []activity{
				{7, "job1", v1alpha1.SuccessState, "old"},
				{7, "job1", v1alpha1.FailureState, "new"},
				{8, "job1", v1alpha1.FailureState, "old"},
				{8, "job1", v1alpha1.SuccessState, "new"},
			},

			successes: []int{8},
			pendings:  []int{},
			none:      []int{7},
		},
		{
			name:         "two PRs, no jobs, all success",
			pullRequests: map[int]string{7: "new", 8: "new"},
			activities:   []activity{},

			successes: []int{8, 7},
			pendings:  []int{},
			none:      []int{},
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var pulls []PullRequest
			for num, sha := range test.pullRequests {
				pulls = append(
					pulls,
					PullRequest{Number: githubql.Int(num), HeadRefOID: githubql.String(sha)},
				)
			}
			var pjs []v1alpha1.LighthouseJob
			for _, pj := range test.activities {
				pjs = append(pjs, v1alpha1.LighthouseJob{
					Spec: v1alpha1.LighthouseJobSpec{
						Job:     pj.job,
						Context: pj.job,
						Type:    job.PresubmitJob,
						Refs:    &v1alpha1.Refs{Pulls: []v1alpha1.Pull{{Number: pj.prNumber, SHA: pj.sha}}},
					},
					Status: v1alpha1.LighthouseJobStatus{State: pj.state},
				})
			}

			successes, pendings, nones, _ := accumulate(test.presubmits, pulls, pjs, logrus.NewEntry(logrus.New()))

			t.Logf("test run %d", i)
			testPullsMatchList(t, "successes", successes, test.successes)
			testPullsMatchList(t, "pendings", pendings, test.pendings)
			testPullsMatchList(t, "nones", nones, test.none)
		})
	}
}

type fgc struct {
	prs              []PullRequest
	refs             map[string]string
	merged           int
	setStatus        bool
	mergeErrs        map[int]error
	mergeErrComments map[int]string

	expectedSHA    string
	ignoreExpected bool
	combinedStatus map[string]map[string]commitStatus
	fakeClient     *scm.Client
}

type commitStatus struct {
	status      string
	description string
}

func toCommitStatus(s string, d string) commitStatus {
	return commitStatus{
		status:      s,
		description: d,
	}
}

func (f *fgc) SupportsGraphQL() bool {
	return true
}

func (f *fgc) ProviderType() string {
	return "fake"
}

func (f *fgc) PRRefFmt() string {
	return "refs/pull/%d/head"
}

func (f *fgc) GetRepositoryByFullName(string) (*scm.Repository, error) {
	return nil, scm.ErrNotSupported
}

func (f *fgc) ListAllPullRequestsForFullNameRepo(string, scm.PullRequestListOptions) ([]*scm.PullRequest, error) {
	return nil, scm.ErrNotSupported
}

func (f *fgc) GetRef(o, r, ref string) (string, error) {
	return f.refs[o+"/"+r+" "+ref], nil
}

func (f *fgc) Query(ctx context.Context, q interface{}, vars map[string]interface{}) error {
	sq, ok := q.(*searchQuery)
	if !ok {
		return errors.New("unexpected query type")
	}
	for _, pr := range f.prs {
		sq.Search.Nodes = append(
			sq.Search.Nodes,
			struct {
				PullRequest PullRequest `graphql:"... on PullRequest"`
			}{PullRequest: pr},
		)
	}
	return nil
}

func (f *fgc) CreateComment(org, repo string, number int, isPR bool, commentBody string) error {
	if f.mergeErrComments == nil {
		f.mergeErrComments = make(map[int]string)
	}
	f.mergeErrComments[number] = commentBody
	return nil
}

func (f *fgc) Merge(org, repo string, number int, details scmprovider.MergeDetails) error {
	if err, ok := f.mergeErrs[number]; ok {
		return err
	}
	f.merged++
	return nil
}

func (f *fgc) CreateGraphQLStatus(org, repo, ref string, s *scmprovider.Status) (*scm.Status, error) {
	switch s.State {
	case scmprovider.StatusSuccess, scmprovider.StatusError, scmprovider.StatusPending, scmprovider.StatusFailure:
		f.setStatus = true
		return nil, nil
	}
	return nil, fmt.Errorf("invalid 'state' value: %q", s.State)
}

func (f *fgc) GetCombinedStatus(org, repo, ref string) (*scm.CombinedStatus, error) {
	if !f.ignoreExpected && f.expectedSHA != ref {
		return nil, errors.New("bad combined status request: incorrect sha")
	}
	var statuses []*scm.Status
	for c, s := range f.combinedStatus[ref] {
		statuses = append(statuses, &scm.Status{
			Label: c,
			State: scm.ToState(s.status),
			Desc:  s.description,
		})
	}
	return &scm.CombinedStatus{
			Statuses: statuses,
		},
		nil
}

func (f *fgc) CreateStatus(org, repo, ref string, s *scm.StatusInput) (*scm.Status, error) {
	if s.Label == "fail-create" {
		return nil, errors.New("injected CreateStatus failure")
	}
	if f.combinedStatus == nil {
		f.combinedStatus = make(map[string]map[string]commitStatus)
	}
	if f.combinedStatus[ref] == nil {
		f.combinedStatus[ref] = make(map[string]commitStatus)
	}
	f.combinedStatus[ref][s.Label] = toCommitStatus(s.State.String(), "")
	return scm.ConvertStatusInputToStatus(s), nil
}

func (f *fgc) GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error) {
	if number != 100 {
		return nil, nil
	}
	return []*scm.Change{
			{
				Path: "CHANGED",
			},
		},
		nil
}

// GetFile returns the file from git
func (f *fgc) GetFile(owner, repo, filepath, commit string) ([]byte, error) {
	if f.fakeClient == nil {
		f.fakeClient, _ = fakescm.NewDefault()
	}
	ctx := context.Background()
	fullName := scm.Join(owner, repo)
	answer, r, err := f.fakeClient.Contents.Find(ctx, fullName, filepath, commit)
	// handle files not existing nicely
	if r != nil && r.Status == 404 {
		return nil, nil
	}
	var data []byte
	if answer != nil {
		data = answer.Data
	}
	return data, err
}

// ListFiles returns the files from git
func (f *fgc) ListFiles(owner, repo, filepath, commit string) ([]*scm.FileEntry, error) {
	ctx := context.Background()
	fullName := scm.Join(owner, repo)
	answer, _, err := f.fakeClient.Contents.List(ctx, fullName, filepath, commit)
	return answer, err
}

// TestDividePool ensures that subpools returned by dividePool satisfy a few
// important invariants.
func TestDividePool(t *testing.T) {
	testPulls := []struct {
		org    string
		repo   string
		number int
		branch string
	}{
		{
			org:    "k",
			repo:   "t-i",
			number: 5,
			branch: "master",
		},
		{
			org:    "k",
			repo:   "t-i",
			number: 6,
			branch: "master",
		},
		{
			org:    "k",
			repo:   "k",
			number: 123,
			branch: "master",
		},
		{
			org:    "k",
			repo:   "k",
			number: 1000,
			branch: "release-1.6",
		},
	}
	testPJs := []struct {
		jobType job.PipelineKind
		org     string
		repo    string
		baseRef string
		baseSHA string
	}{
		{
			jobType: job.PresubmitJob,
			org:     "k",
			repo:    "t-i",
			baseRef: "master",
			baseSHA: "123",
		},
		{
			jobType: job.BatchJob,
			org:     "k",
			repo:    "t-i",
			baseRef: "master",
			baseSHA: "123",
		},
		{
			jobType: job.PeriodicJob,
		},
		{
			jobType: job.PresubmitJob,
			org:     "k",
			repo:    "t-i",
			baseRef: "patch",
			baseSHA: "123",
		},
		{
			jobType: job.PresubmitJob,
			org:     "k",
			repo:    "t-i",
			baseRef: "master",
			baseSHA: "abc",
		},
		{
			jobType: job.PresubmitJob,
			org:     "o",
			repo:    "t-i",
			baseRef: "master",
			baseSHA: "123",
		},
		{
			jobType: job.PresubmitJob,
			org:     "k",
			repo:    "other",
			baseRef: "master",
			baseSHA: "123",
		},
	}
	fc := &fgc{
		refs: map[string]string{"k/t-i heads/master": "123"},
	}
	c := &DefaultController{
		spc:    fc,
		logger: logrus.WithField("component", "keeper"),
	}
	pulls := make(map[string]PullRequest)
	for _, p := range testPulls {
		npr := PullRequest{Number: githubql.Int(p.number)}
		npr.BaseRef.Name = githubql.String(p.branch)
		npr.BaseRef.Prefix = "refs/heads/"
		npr.Repository.Name = githubql.String(p.repo)
		npr.Repository.Owner.Login = githubql.String(p.org)
		npr.Repository.URL = githubql.String(fmt.Sprintf("https://github.com/%s/%s.git", p.org, p.repo))
		pulls[npr.prKey()] = npr
	}
	var pjs []v1alpha1.LighthouseJob
	for _, pj := range testPJs {
		pjs = append(pjs, v1alpha1.LighthouseJob{
			Spec: v1alpha1.LighthouseJobSpec{
				Type: pj.jobType,
				Refs: &v1alpha1.Refs{
					Org:     pj.org,
					Repo:    pj.repo,
					BaseRef: pj.baseRef,
					BaseSHA: pj.baseSHA,
				},
			},
		})
	}
	sps, err := c.dividePool(pulls, pjs)
	if err != nil {
		t.Fatalf("Error dividing pool: %v", err)
	}
	if len(sps) == 0 {
		t.Error("No subpools.")
	}
	for _, sp := range sps {
		name := fmt.Sprintf("%s/%s %s", sp.org, sp.repo, sp.branch)
		sha := fc.refs[sp.org+"/"+sp.repo+" heads/"+sp.branch]
		if sp.sha != sha {
			t.Errorf("For subpool %s, got sha %s, expected %s.", name, sp.sha, sha)
		}
		if len(sp.prs) == 0 {
			t.Errorf("Subpool %s has no PRs.", name)
		}
		for _, pr := range sp.prs {
			if string(pr.Repository.Owner.Login) != sp.org || string(pr.Repository.Name) != sp.repo || string(pr.BaseRef.Name) != sp.branch {
				t.Errorf("PR in wrong subpool. Got PR %+v in subpool %s.", pr, name)
			}
		}
		for _, pj := range sp.ljs {
			if pj.Spec.Type != job.PresubmitJob && pj.Spec.Type != job.BatchJob {
				t.Errorf("PJ with bad type in subpool %s: %+v", name, pj)
			}
			if pj.Spec.Refs.Org != sp.org || pj.Spec.Refs.Repo != sp.repo || pj.Spec.Refs.BaseRef != sp.branch || pj.Spec.Refs.BaseSHA != sp.sha {
				t.Errorf("PJ in wrong subpool. Got PJ %+v in subpool %s.", pj, name)
			}
		}
	}
}

func TestPickBatch(t *testing.T) {
	// TODO: Remove once #564 is fixed and batch builds can work again. (APB)
	t.Skip("Skipping TestPickBatch until #564 is fixed and batch builds can work again")
	lg, gc, err := localgit.New()
	if err != nil {
		t.Fatalf("Error making local git: %v", err)
	}
	defer gc.Clean()
	defer lg.Clean()
	if err := lg.MakeFakeRepo("o", "r"); err != nil {
		t.Fatalf("Error making fake repo: %v", err)
	}
	if err := lg.AddCommit("o", "r", map[string][]byte{"foo": []byte("foo")}); err != nil {
		t.Fatalf("Adding initial commit: %v", err)
	}
	testprs := []struct {
		files   map[string][]byte
		success bool
		number  int

		included bool
	}{
		{
			files:    map[string][]byte{"bar": []byte("ok")},
			success:  true,
			number:   0,
			included: true,
		},
		{
			files:    map[string][]byte{"foo": []byte("ok")},
			success:  true,
			number:   1,
			included: true,
		},
		{
			files:    map[string][]byte{"bar": []byte("conflicts with 0")},
			success:  true,
			number:   2,
			included: false,
		},
		{
			files:    map[string][]byte{"qux": []byte("ok")},
			success:  false,
			number:   6,
			included: false,
		},
		{
			files:    map[string][]byte{"bazel": []byte("ok")},
			success:  true,
			number:   7,
			included: false, // batch of 5 smallest excludes this
		},
		{
			files:    map[string][]byte{"other": []byte("ok")},
			success:  true,
			number:   5,
			included: true,
		},
		{
			files:    map[string][]byte{"changes": []byte("ok")},
			success:  true,
			number:   4,
			included: true,
		},
		{
			files:    map[string][]byte{"something": []byte("ok")},
			success:  true,
			number:   3,
			included: true,
		},
	}
	sp := subpool{
		log:    logrus.WithField("component", "keeper"),
		org:    "o",
		repo:   "r",
		branch: "master",
		sha:    "master",
	}
	for _, testpr := range testprs {
		if err := lg.CheckoutNewBranch("o", "r", fmt.Sprintf("pr-%d", testpr.number)); err != nil {
			t.Fatalf("Error checking out new branch: %v", err)
		}
		if err := lg.AddCommit("o", "r", testpr.files); err != nil {
			t.Fatalf("Error adding commit: %v", err)
		}
		if err := lg.Checkout("o", "r", "master"); err != nil {
			t.Fatalf("Error checking out master: %v", err)
		}
		oid := githubql.String(fmt.Sprintf("origin/pr-%d", testpr.number))
		var pr PullRequest
		pr.Number = githubql.Int(testpr.number)
		pr.HeadRefOID = oid
		pr.Commits.Nodes = []struct {
			Commit Commit
		}{{Commit: Commit{OID: oid}}}
		pr.Commits.Nodes[0].Commit.Status.Contexts = append(pr.Commits.Nodes[0].Commit.Status.Contexts, Context{State: githubql.StatusStateSuccess})
		if !testpr.success {
			pr.Commits.Nodes[0].Commit.Status.Contexts[0].State = githubql.StatusStateFailure
		}
		sp.prs = append(sp.prs, pr)
	}
	ca := &config.Agent{}
	ca.Set(&config.Config{
		ProwConfig: config.ProwConfig{
			Keeper: keeper.Config{
				BatchSizeLimitMap: map[string]int{"*": 5},
			},
		},
	})
	c := &DefaultController{
		logger: logrus.WithField("component", "keeper"),
		gc:     gc,
		config: ca.Config,
	}
	prs, err := c.pickBatch(sp, &keeper.ContextPolicy{})
	if err != nil {
		t.Fatalf("Error from pickBatch: %v", err)
	}
	for _, testpr := range testprs {
		var found bool
		for _, pr := range prs {
			if int(pr.Number) == testpr.number {
				found = true
				break
			}
		}
		if found && !testpr.included {
			t.Errorf("PR %d should not be picked.", testpr.number)
		} else if !found && testpr.included {
			t.Errorf("PR %d should be picked.", testpr.number)
		}
	}
}

func TestCheckMergeLabels(t *testing.T) {
	squashLabel := "keeper/squash"
	mergeLabel := "keeper/merge"
	rebaseLabel := "keeper/rebase"

	testcases := []struct {
		name string

		pr        PullRequest
		method    keeper.PullRequestMergeType
		expected  keeper.PullRequestMergeType
		expectErr bool
	}{
		{
			name:      "default method without PR label override",
			pr:        PullRequest{},
			method:    keeper.MergeMerge,
			expected:  keeper.MergeMerge,
			expectErr: false,
		},
		{
			name: "irrelevant PR labels ignored",
			pr: PullRequest{
				Labels: struct {
					Nodes []struct{ Name githubql.String }
				}{Nodes: []struct{ Name githubql.String }{{Name: githubql.String("sig/testing")}}},
			},
			method:    keeper.MergeMerge,
			expected:  keeper.MergeMerge,
			expectErr: false,
		},
		{
			name: "default method overridden by a PR label",
			pr: PullRequest{
				Labels: struct {
					Nodes []struct{ Name githubql.String }
				}{Nodes: []struct{ Name githubql.String }{{Name: githubql.String(squashLabel)}}},
			},
			method:    keeper.MergeMerge,
			expected:  keeper.MergeSquash,
			expectErr: false,
		},
		{
			name: "multiple merge method PR labels should not merge",
			pr: PullRequest{
				Labels: struct {
					Nodes []struct{ Name githubql.String }
				}{Nodes: []struct{ Name githubql.String }{
					{Name: githubql.String(squashLabel)},
					{Name: githubql.String(rebaseLabel)}},
				},
			},
			method:    keeper.MergeMerge,
			expected:  keeper.MergeSquash,
			expectErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := checkMergeLabels(tc.pr, squashLabel, rebaseLabel, mergeLabel, tc.method)
			if err != nil && !tc.expectErr {
				t.Errorf("unexpected error: %v", err)
			} else if err == nil && tc.expectErr {
				t.Errorf("missing expected error from checkMargeLabels")
			} else if err == nil && tc.expected != actual {
				t.Errorf("wanted: %q, got: %q", tc.expected, actual)
			}
		})
	}
}

func TestTakeAction(t *testing.T) {
	sleep = func(time.Duration) {}
	defer func() { sleep = time.Sleep }()

	// PRs 0-9 exist. All are mergable, and all are passing tests.
	testcases := []struct {
		name string

		batchPending bool
		successes    []int
		pendings     []int
		nones        []int
		batchMerges  []int
		presubmits   map[int][]job.Presubmit
		mergeErrs    map[int]error

		merged           int
		triggered        int
		triggeredBatches int
		action           Action
		expectErr        bool
	}{
		{
			name: "no prs to test, should do nothing",

			batchPending: true,
			successes:    []int{},
			pendings:     []int{},
			nones:        []int{},
			batchMerges:  []int{},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:    0,
			triggered: 0,
			action:    Wait,
		},
		{
			name: "pending batch, pending serial, nothing to do",

			batchPending: true,
			successes:    []int{},
			pendings:     []int{1},
			nones:        []int{0, 2},
			batchMerges:  []int{},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:    0,
			triggered: 0,
			action:    Wait,
		},
		{
			name: "pending batch, successful serial, nothing to do",

			batchPending: true,
			successes:    []int{1},
			pendings:     []int{},
			nones:        []int{0, 2},
			batchMerges:  []int{},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:    0,
			triggered: 0,
			action:    Wait,
		},
		{
			name: "pending batch, should trigger serial",

			batchPending: true,
			successes:    []int{},
			pendings:     []int{},
			nones:        []int{0, 1, 2},
			batchMerges:  []int{},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:    0,
			triggered: 1,
			action:    Trigger,
		},
		{
			name: "no pending batch, should trigger batch",

			batchPending: false,
			successes:    []int{},
			pendings:     []int{0},
			nones:        []int{1, 2, 3},
			batchMerges:  []int{},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:           0,
			triggered:        1,
			triggeredBatches: 1,
			action:           TriggerBatch,
		},
		{
			name: "one PR, should not trigger batch",

			batchPending: false,
			successes:    []int{},
			pendings:     []int{},
			nones:        []int{0},
			batchMerges:  []int{},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:    0,
			triggered: 1,
			action:    Trigger,
		},
		{
			name: "successful PR, should merge",

			batchPending: false,
			successes:    []int{0},
			pendings:     []int{},
			nones:        []int{1, 2, 3},
			batchMerges:  []int{},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:    1,
			triggered: 0,
			action:    Merge,
		},
		{
			name: "successful batch, should merge",

			batchPending: false,
			successes:    []int{0, 1},
			pendings:     []int{2, 3},
			nones:        []int{4, 5},
			batchMerges:  []int{6, 7, 8},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:    3,
			triggered: 0,
			action:    MergeBatch,
		},
		{
			name: "one PR that triggers RunIfChangedJob",

			batchPending: false,
			successes:    []int{},
			pendings:     []int{},
			nones:        []int{100},
			batchMerges:  []int{},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:    0,
			triggered: 2,
			action:    Trigger,
		},
		{
			name: "no presubmits, merge",

			batchPending: false,
			successes:    []int{5, 4},
			pendings:     []int{},
			nones:        []int{},
			batchMerges:  []int{},

			merged:    1,
			triggered: 0,
			action:    Merge,
		},
		{
			name: "no presubmits, wait",

			batchPending: false,
			successes:    []int{},
			pendings:     []int{},
			nones:        []int{},
			batchMerges:  []int{},

			merged:    0,
			triggered: 0,
			action:    Wait,
		},
		{
			name: "no pending serial or batch, should trigger batch",

			batchPending: false,
			successes:    []int{},
			pendings:     []int{},
			nones:        []int{1, 2, 3},
			batchMerges:  []int{},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:           0,
			triggered:        1,
			triggeredBatches: 1,
			action:           TriggerBatch,
		},
		{
			name: "pending batch, no serial, should trigger serial",

			batchPending: true,
			successes:    []int{},
			pendings:     []int{},
			nones:        []int{1, 2, 3},
			batchMerges:  []int{},
			presubmits: map[int][]job.Presubmit{
				100: {
					{Reporter: job.Reporter{Context: "foo"}},
					{Reporter: job.Reporter{Context: "if-changed"}},
				},
			},
			merged:    0,
			triggered: 1,
			action:    Trigger,
		},
		{
			name: "batch merge errors but continues if a PR is unmergeable",

			batchMerges: []int{1, 2, 3},
			mergeErrs:   map[int]error{2: scmprovider.UnmergablePRError("test error")},
			merged:      2,
			triggered:   0,
			action:      MergeBatch,
			expectErr:   true,
		},
		{
			name: "batch merge errors but continues if a PR has changed",

			batchMerges: []int{1, 2, 3},
			mergeErrs:   map[int]error{2: scmprovider.ModifiedHeadError("test error")},
			merged:      2,
			triggered:   0,
			action:      MergeBatch,
			expectErr:   true,
		},
		{
			name: "batch merge errors but continues on unknown error",

			batchMerges: []int{1, 2, 3},
			mergeErrs:   map[int]error{2: errors.New("test error")},
			merged:      2,
			triggered:   0,
			action:      MergeBatch,
			expectErr:   true,
		},
		{
			name: "batch merge stops on auth error",

			batchMerges: []int{1, 2, 3},
			mergeErrs:   map[int]error{2: scmprovider.UnauthorizedToPushError("test error")},
			merged:      1,
			triggered:   0,
			action:      MergeBatch,
			expectErr:   true,
		},
		{
			name: "batch merge stops on invalid merge method error",

			batchMerges: []int{1, 2, 3},
			mergeErrs:   map[int]error{2: scmprovider.MergeCommitsForbiddenError("test error")},
			merged:      1,
			triggered:   0,
			action:      MergeBatch,
			expectErr:   true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// TODO: Remove once #564 is fixed and batch builds can work again. (APB)
			if strings.Contains(tc.name, "should trigger batch") {
				t.Skipf("Skipping TestTakeAction/%s until #564 is fixed and batch builds can work again", tc.name)
			}
			ca := &config.Agent{}
			cfg := &config.Config{}
			if err := cfg.SetPresubmits(
				map[string][]job.Presubmit{
					"o/r": {
						{
							Reporter:     job.Reporter{Context: "foo"},
							Trigger:      "/test all",
							RerunCommand: "/test all",
							AlwaysRun:    true,
						},
						{
							Reporter:     job.Reporter{Context: "if-changed"},
							Trigger:      "/test if-changed",
							RerunCommand: "/test if-changed",
							RegexpChangeMatcher: job.RegexpChangeMatcher{
								RunIfChanged: "CHANGED",
							},
						},
					},
				},
			); err != nil {
				t.Fatalf("failed to set presubmits: %v", err)
			}
			ca.Set(cfg)
			if len(tc.presubmits) > 0 {
				for i := 0; i <= 8; i++ {
					tc.presubmits[i] = []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}}
				}
			}
			lg, gc, err := localgit.New()
			if err != nil {
				t.Fatalf("Error making local git: %v", err)
			}
			defer gc.Clean()
			defer lg.Clean()
			if err := lg.MakeFakeRepo("o", "r"); err != nil {
				t.Fatalf("Error making fake repo: %v", err)
			}
			if err := lg.AddCommit("o", "r", map[string][]byte{"foo": []byte("foo")}); err != nil {
				t.Fatalf("Adding initial commit: %v", err)
			}

			sp := subpool{
				log:        logrus.WithField("component", "keeper"),
				presubmits: tc.presubmits,
				cc:         &keeper.ContextPolicy{},
				org:        "o",
				repo:       "r",
				branch:     "master",
				sha:        "master",
			}
			genPulls := func(nums []int) []PullRequest {
				var prs []PullRequest
				for _, i := range nums {
					if err := lg.CheckoutNewBranch("o", "r", fmt.Sprintf("pr-%d", i)); err != nil {
						t.Fatalf("Error checking out new branch: %v", err)
					}
					if err := lg.AddCommit("o", "r", map[string][]byte{fmt.Sprintf("%d", i): []byte("WOW")}); err != nil {
						t.Fatalf("Error adding commit: %v", err)
					}
					if err := lg.Checkout("o", "r", "master"); err != nil {
						t.Fatalf("Error checking out master: %v", err)
					}
					oid := githubql.String(fmt.Sprintf("origin/pr-%d", i))
					var pr PullRequest
					pr.Number = githubql.Int(i)
					pr.HeadRefOID = oid
					pr.Commits.Nodes = []struct {
						Commit Commit
					}{{Commit: Commit{OID: oid}}}
					sp.prs = append(sp.prs, pr)
					prs = append(prs, pr)
				}
				return prs
			}
			fgc := fgc{mergeErrs: tc.mergeErrs}
			fakeLauncher := launcherfake.NewLauncher()
			fakeLighthouseClient := fake.NewSimpleClientset()
			c := &DefaultController{
				logger:         logrus.WithField("controller", "keeper"),
				gc:             gc,
				config:         ca.Config,
				spc:            &fgc,
				launcherClient: fakeLauncher,
				lhClient:       fakeLighthouseClient,
			}
			var batchPending []PullRequest
			if tc.batchPending {
				batchPending = []PullRequest{{}}
			}
			successes := genPulls(tc.successes)
			pendings := genPulls(tc.pendings)
			nones := genPulls(tc.nones)
			batchMerges := genPulls(tc.batchMerges)
			t.Logf("Test case: %s", tc.name)
			if act, _, err := c.takeAction(sp, batchPending, successes, pendings, nones, batchMerges, sp.presubmits); err != nil && !tc.expectErr {
				t.Fatalf("Unexpected error in takeAction: %v", err)
			} else if err == nil && tc.expectErr {
				t.Error("Missing expected error from takeAction.")
			} else if err != nil && tc.expectErr {
				var prsToMerge []PullRequest
				var failed []int
				var merged []int
				if len(batchMerges) > 0 {
					prsToMerge = append(prsToMerge, batchMerges...)
					failedAndStopped := false
					for _, pr := range batchMerges {
						prNum := int(pr.Number)
						if mergeErr, ok := tc.mergeErrs[prNum]; ok {
							failed = append(failed, prNum)
							switch mergeErr.(type) {
							case scmprovider.MergeCommitsForbiddenError, scmprovider.UnauthorizedToPushError:
								failedAndStopped = true
							default:
								continue
							}
						} else if !failedAndStopped {
							merged = append(merged, prNum)
						}
					}
				} else {
					if ok, prToMerge := pickSmallestPassingNumber(sp.log, c.spc, successes, sp.cc); ok {
						prsToMerge = append(prsToMerge, prToMerge)
						failed = append(failed, int(prToMerge.Number))
					}
				}
				for prNum, rawErr := range tc.mergeErrs {
					detailedErr := mergeErrorDetail(rawErr)
					expectedErr := rollupMergeErrors(prsToMerge, failed, merged, []error{detailedErr})
					assert.Equal(t, fmt.Sprintf("Failed to merge this PR due to:\n>%s\n", expectedErr.Error()), fgc.mergeErrComments[prNum])
				}
			} else if act != tc.action {
				t.Errorf("Wrong action. Got %v, wanted %v.", act, tc.action)
			}

			numCreated := 0
			var batchJobs []*v1alpha1.LighthouseJob
			for _, activity := range fakeLauncher.Pipelines {
				pjSha := activity.Spec.Refs.Pulls[0].SHA
				if scm.StatePending.String() != fgc.combinedStatus[pjSha][activity.Spec.Context].status {
					t.Errorf("Status not set to %s for context %s, is %s instead", scm.StatePending.String(), activity.Spec.Context,
						fgc.combinedStatus[pjSha][activity.Spec.Context].status)
				}
				numCreated++
				if activity.Spec.Type == job.BatchJob {
					batchJobs = append(batchJobs, activity)
				}
			}
			if tc.triggered != numCreated {
				t.Errorf("Wrong number of jobs triggered. Got %d, expected %d.", numCreated, tc.triggered)
			}
			if tc.merged != fgc.merged {
				t.Errorf("Wrong number of merges. Got %d, expected %d.", fgc.merged, tc.merged)
			}
			// Ensure that the correct number of batch jobs were triggered
			if tc.triggeredBatches != len(batchJobs) {
				t.Errorf("Wrong number of batches triggered. Got %d, expected %d.", len(batchJobs), tc.triggeredBatches)
			}
			for _, job := range batchJobs {
				if len(job.Spec.Refs.Pulls) <= 1 {
					t.Error("Found a batch job that doesn't contain multiple pull refs!")
				}
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	pr1 := PullRequest{}
	pr1.Commits.Nodes = append(pr1.Commits.Nodes, struct{ Commit Commit }{})
	pr1.Commits.Nodes[0].Commit.Status.Contexts = []Context{{
		Context:     githubql.String("coverage/coveralls"),
		Description: githubql.String("Coverage increased (+0.1%) to 27.599%"),
	}}
	hist, err := history.New(100, "")
	if err != nil {
		t.Fatalf("Failed to create history client: %v", err)
	}
	c := &DefaultController{
		pools: []Pool{
			{
				MissingPRs: []PullRequest{pr1},
				Action:     Merge,
			},
		},
		History: hist,
	}
	s := httptest.NewServer(c)
	defer s.Close()
	resp, err := http.Get(s.URL)
	if err != nil {
		t.Errorf("GET error: %v", err)
	}
	defer resp.Body.Close()
	var pools []Pool
	if err := json.NewDecoder(resp.Body).Decode(&pools); err != nil {
		t.Fatalf("JSON decoding error: %v", err)
	}
	if !reflect.DeepEqual(c.pools, pools) {
		t.Errorf("Received pools %v do not match original pools %v.", pools, c.pools)
	}
}

func TestHeadContexts(t *testing.T) {
	type commitContext struct {
		// one context per commit for testing
		context string
		sha     string
	}

	win := "win"
	lose := "lose"
	headSHA := "head"
	testCases := []struct {
		name           string
		commitContexts []commitContext
		expectAPICall  bool
	}{
		{
			name: "first commit is head",
			commitContexts: []commitContext{
				{context: win, sha: headSHA},
				{context: lose, sha: "other"},
				{context: lose, sha: "sha"},
			},
		},
		{
			name: "last commit is head",
			commitContexts: []commitContext{
				{context: lose, sha: "shaaa"},
				{context: lose, sha: "other"},
				{context: win, sha: headSHA},
			},
		},
		{
			name: "no commit is head",
			commitContexts: []commitContext{
				{context: lose, sha: "shaaa"},
				{context: lose, sha: "other"},
				{context: lose, sha: "sha"},
			},
			expectAPICall: true,
		},
	}

	for _, tc := range testCases {
		t.Logf("Running test case %q", tc.name)
		fgc := &fgc{combinedStatus: map[string]map[string]commitStatus{headSHA: {win: toCommitStatus(string(githubql.StatusStateSuccess), "")}}}
		if tc.expectAPICall {
			fgc.expectedSHA = headSHA
		}
		pr := &PullRequest{HeadRefOID: githubql.String(headSHA)}
		for _, ctx := range tc.commitContexts {
			commit := Commit{
				Status: struct{ Contexts []Context }{
					Contexts: []Context{
						{
							Context: githubql.String(ctx.context),
						},
					},
				},
				OID: githubql.String(ctx.sha),
			}
			pr.Commits.Nodes = append(pr.Commits.Nodes, struct{ Commit Commit }{commit})
		}

		contexts, err := headContexts(logrus.WithField("component", "keeper"), fgc, pr)
		if err != nil {
			t.Fatalf("Unexpected error from headContexts: %v", err)
		}
		if len(contexts) != 1 || string(contexts[0].Context) != win {
			t.Errorf("Expected exactly 1 %q context, but got: %#v", win, contexts)
		}
	}
}

func testPR(org, repo, branch string, number int, mergeable githubql.MergeableState) PullRequest {
	pr := PullRequest{
		Number:     githubql.Int(number),
		Mergeable:  mergeable,
		HeadRefOID: githubql.String("SHA"),
	}
	pr.Repository.Owner.Login = githubql.String(org)
	pr.Repository.Name = githubql.String(repo)
	pr.Repository.NameWithOwner = githubql.String(fmt.Sprintf("%s/%s", org, repo))
	pr.BaseRef.Name = githubql.String(branch)
	pr.Repository.URL = githubql.String(fmt.Sprintf("https://github.com/%s/%s.git", org, repo))

	pr.Commits.Nodes = append(pr.Commits.Nodes, struct{ Commit Commit }{
		Commit{
			Status: struct{ Contexts []Context }{
				Contexts: []Context{
					{
						Context: githubql.String("context"),
						State:   githubql.StatusStateSuccess,
					},
				},
			},
			OID: githubql.String("SHA"),
		},
	})
	return pr
}

func TestSync(t *testing.T) {
	sleep = func(time.Duration) {}
	defer func() { sleep = time.Sleep }()

	mergeableA := testPR("org", "repo", "A", 5, githubql.MergeableStateMergeable)
	unmergeableA := testPR("org", "repo", "A", 6, githubql.MergeableStateConflicting)
	unmergeableB := testPR("org", "repo", "B", 7, githubql.MergeableStateConflicting)
	unknownA := testPR("org", "repo", "A", 8, githubql.MergeableStateUnknown)

	testcases := []struct {
		name string
		prs  []PullRequest

		expectedPools []Pool
	}{
		{
			name:          "no PRs",
			prs:           []PullRequest{},
			expectedPools: []Pool{},
		},
		{
			name: "1 mergeable PR",
			prs:  []PullRequest{mergeableA},
			expectedPools: []Pool{{
				Org:        "org",
				Repo:       "repo",
				Branch:     "A",
				SuccessPRs: []PullRequest{mergeableA},
				Action:     Merge,
				Target:     []PullRequest{mergeableA},
			}},
		},
		{
			name:          "1 unmergeable PR",
			prs:           []PullRequest{unmergeableA},
			expectedPools: []Pool{},
		},
		{
			name: "1 unknown PR",
			prs:  []PullRequest{unknownA},
			expectedPools: []Pool{{
				Org:        "org",
				Repo:       "repo",
				Branch:     "A",
				SuccessPRs: []PullRequest{unknownA},
				Action:     Merge,
				Target:     []PullRequest{unknownA},
			}},
		},
		{
			name: "1 mergeable, 1 unmergeable (different pools)",
			prs:  []PullRequest{mergeableA, unmergeableB},
			expectedPools: []Pool{{
				Org:        "org",
				Repo:       "repo",
				Branch:     "A",
				SuccessPRs: []PullRequest{mergeableA},
				Action:     Merge,
				Target:     []PullRequest{mergeableA},
			}},
		},
		{
			name: "1 mergeable, 1 unmergeable (same pool)",
			prs:  []PullRequest{mergeableA, unmergeableA},
			expectedPools: []Pool{{
				Org:        "org",
				Repo:       "repo",
				Branch:     "A",
				SuccessPRs: []PullRequest{mergeableA},
				Action:     Merge,
				Target:     []PullRequest{mergeableA},
			}},
		},
		{
			name: "1 mergeable PR (satisfies multiple queries)",
			prs:  []PullRequest{mergeableA, mergeableA},
			expectedPools: []Pool{{
				Org:        "org",
				Repo:       "repo",
				Branch:     "A",
				SuccessPRs: []PullRequest{mergeableA},
				Action:     Merge,
				Target:     []PullRequest{mergeableA},
			}},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fgc := &fgc{prs: tc.prs}
			fakeLauncher := launcherfake.NewLauncher()
			fakeLighthouseClient := fake.NewSimpleClientset()
			fakeTektonClient := tektonfake.NewSimpleClientset()
			ca := &config.Agent{}
			ca.Set(&config.Config{
				ProwConfig: config.ProwConfig{
					Keeper: keeper.Config{
						Queries:            []keeper.Query{{}},
						MaxGoroutines:      4,
						StatusUpdatePeriod: time.Second * 0,
					},
				},
			})
			hist, err := history.New(100, "")
			if err != nil {
				t.Fatalf("Failed to create history client: %v", err)
			}
			sc := &statusController{
				logger:         logrus.WithField("controller", "status-update"),
				spc:            fgc,
				config:         ca.Config,
				newPoolPending: make(chan bool, 1),
				shutDown:       make(chan bool),
			}
			go sc.run()
			defer sc.shutdown()
			c := &DefaultController{
				config:         ca.Config,
				spc:            fgc,
				launcherClient: fakeLauncher,
				tektonClient:   fakeTektonClient,
				lhClient:       fakeLighthouseClient,
				ns:             "jx",
				logger:         logrus.WithField("controller", "sync"),
				sc:             sc,
				changedFiles: &changedFilesAgent{
					spc:             fgc,
					nextChangeCache: make(map[changeCacheKey][]string),
				},
				History: hist,
			}

			if err := c.Sync(); err != nil {
				t.Fatalf("Unexpected error from 'Sync()': %v.", err)
			}
			if len(tc.expectedPools) != len(c.pools) {
				t.Fatalf("Keeper pools did not match expected. Got %#v, expected %#v.", c.pools, tc.expectedPools)
			}
			for _, expected := range tc.expectedPools {
				var match *Pool
				for i, actual := range c.pools {
					if expected.Org == actual.Org && expected.Repo == actual.Repo && expected.Branch == actual.Branch {
						match = &c.pools[i]
					}
				}
				if match == nil {
					t.Errorf("Failed to find expected pool %s/%s %s.", expected.Org, expected.Repo, expected.Branch)
				} else if !reflect.DeepEqual(*match, expected) {
					t.Errorf("Expected pool %#v does not match actual pool %#v.", expected, *match)
				}
			}
		})
	}
}

func TestFilterSubpool(t *testing.T) {
	presubmits := map[int][]job.Presubmit{
		1: {{Reporter: job.Reporter{Context: "pj-a"}}},
		2: {{Reporter: job.Reporter{Context: "pj-a"}}, {Reporter: job.Reporter{Context: "pj-b"}}},
	}

	trueVar := true
	cc := &keeper.ContextPolicy{
		RequiredContexts:    []string{"pj-a", "pj-b", "other-a"},
		OptionalContexts:    []string{"keeper", "pj-c"},
		SkipUnknownContexts: &trueVar,
	}

	type pr struct {
		number    int
		mergeable bool
		contexts  []Context
	}
	tcs := []struct {
		name string

		prs         []pr
		expectedPRs []int // Empty indicates no subpool should be returned.
	}{
		{
			name: "one mergeable passing PR (omitting optional context)",
			prs: []pr{
				{
					number:    1,
					mergeable: true,
					contexts: []Context{
						{
							Context: githubql.String("pj-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStateSuccess,
						},
					},
				},
			},
			expectedPRs: []int{1},
		},
		{
			name: "one unmergeable passing PR",
			prs: []pr{
				{
					number:    1,
					mergeable: false,
					contexts: []Context{
						{
							Context: githubql.String("pj-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStateSuccess,
						},
					},
				},
			},
			expectedPRs: []int{},
		},
		{
			name: "one mergeable PR pending non-PJ context (consider failing)",
			prs: []pr{
				{
					number:    2,
					mergeable: true,
					contexts: []Context{
						{
							Context: githubql.String("pj-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStatePending,
						},
					},
				},
			},
			expectedPRs: []int{},
		},
		{
			name: "one mergeable PR pending PJ context (consider in pool)",
			prs: []pr{
				{
					number:    2,
					mergeable: true,
					contexts: []Context{
						{
							Context: githubql.String("pj-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStatePending,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStateSuccess,
						},
					},
				},
			},
			expectedPRs: []int{2},
		},
		{
			name: "one mergeable PR failing PJ context (consider failing)",
			prs: []pr{
				{
					number:    2,
					mergeable: true,
					contexts: []Context{
						{
							Context: githubql.String("pj-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStateFailure,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStateSuccess,
						},
					},
				},
			},
			expectedPRs: []int{},
		},
		{
			name: "one mergeable PR missing PJ context (consider failing)",
			prs: []pr{
				{
					number:    2,
					mergeable: true,
					contexts: []Context{
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStateSuccess,
						},
					},
				},
			},
			expectedPRs: []int{},
		},
		{
			name: "one mergeable PR failing unknown context (consider in pool)",
			prs: []pr{
				{
					number:    2,
					mergeable: true,
					contexts: []Context{
						{
							Context: githubql.String("pj-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("unknown"),
							State:   githubql.StatusStateFailure,
						},
					},
				},
			},
			expectedPRs: []int{2},
		},
		{
			name: "one PR failing non-PJ required context; one PR successful (should not prune pool)",
			prs: []pr{
				{
					number:    1,
					mergeable: true,
					contexts: []Context{
						{
							Context: githubql.String("pj-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStateFailure,
						},
					},
				},
				{
					number:    2,
					mergeable: true,
					contexts: []Context{
						{
							Context: githubql.String("pj-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("unknown"),
							State:   githubql.StatusStateSuccess,
						},
					},
				},
			},
			expectedPRs: []int{2},
		},
		{
			name: "two successful PRs",
			prs: []pr{
				{
					number:    1,
					mergeable: true,
					contexts: []Context{
						{
							Context: githubql.String("pj-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStateSuccess,
						},
					},
				},
				{
					number:    2,
					mergeable: true,
					contexts: []Context{
						{
							Context: githubql.String("pj-a"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("pj-b"),
							State:   githubql.StatusStateSuccess,
						},
						{
							Context: githubql.String("other-a"),
							State:   githubql.StatusStateSuccess,
						},
					},
				},
			},
			expectedPRs: []int{1, 2},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sp := &subpool{
				org:        "org",
				repo:       "repo",
				branch:     "branch",
				presubmits: presubmits,
				cc:         cc,
				log:        logrus.WithFields(logrus.Fields{"org": "org", "repo": "repo", "branch": "branch"}),
			}
			for _, pull := range tc.prs {
				pr := PullRequest{
					Number: githubql.Int(pull.number),
				}
				pr.Commits.Nodes = []struct{ Commit Commit }{
					{
						Commit{
							Status: struct{ Contexts []Context }{
								Contexts: pull.contexts,
							},
						},
					},
				}
				if !pull.mergeable {
					pr.Mergeable = githubql.MergeableStateConflicting
				}
				sp.prs = append(sp.prs, pr)
			}

			filtered := filterSubpool(nil, sp)
			if len(tc.expectedPRs) == 0 {
				if filtered != nil {
					t.Fatalf("Expected subpool to be pruned, but got: %v", filtered)
				}
				return
			}
			if filtered == nil {
				t.Fatalf("Expected subpool to have %d prs, but it was pruned.", len(tc.expectedPRs))
			}
			if got := prNumbers(filtered.prs); !reflect.DeepEqual(got, tc.expectedPRs) {
				t.Errorf("Expected filtered pool to have PRs %v, but got %v.", tc.expectedPRs, got)
			}
		})
	}
}

func TestIsPassing(t *testing.T) {
	yes := true
	no := false
	headSHA := "head"
	success := string(githubql.StatusStateSuccess)
	failure := string(githubql.StatusStateFailure)
	testCases := []struct {
		name             string
		passing          bool
		config           keeper.ContextPolicy
		combinedContexts map[string]commitStatus
	}{
		{
			name:             "empty policy - success (trust combined status)",
			passing:          true,
			combinedContexts: map[string]commitStatus{"c1": toCommitStatus(success, ""), "c2": toCommitStatus(success, ""), statusContext: toCommitStatus(failure, "")},
		},
		{
			name:             "empty policy - failure because of failed context c4 (trust combined status)",
			passing:          false,
			combinedContexts: map[string]commitStatus{"c1": toCommitStatus(success, ""), "c2": toCommitStatus(success, ""), "c3": toCommitStatus(failure, ""), statusContext: toCommitStatus(failure, "")},
		},
		{
			name:    "passing (trust combined status)",
			passing: true,
			config: keeper.ContextPolicy{
				RequiredContexts:    []string{"c1", "c2", "c3"},
				SkipUnknownContexts: &no,
			},
			combinedContexts: map[string]commitStatus{"c1": toCommitStatus(success, ""), "c2": toCommitStatus(success, ""), "c3": toCommitStatus(success, ""), statusContext: toCommitStatus(failure, "")},
		},
		{
			name:    "failing because of missing required check c3",
			passing: false,
			config: keeper.ContextPolicy{
				RequiredContexts: []string{"c1", "c2", "c3"},
			},
			combinedContexts: map[string]commitStatus{"c1": toCommitStatus(success, ""), "c2": toCommitStatus(success, ""), statusContext: toCommitStatus(failure, "")},
		},
		{
			name:             "failing because of failed context c2",
			passing:          false,
			combinedContexts: map[string]commitStatus{"c1": toCommitStatus(success, ""), "c2": toCommitStatus(failure, "")},
			config: keeper.ContextPolicy{
				RequiredContexts: []string{"c1", "c2", "c3"},
				OptionalContexts: []string{"c4"},
			},
		},
		{
			name:    "passing because of failed context c4 is optional",
			passing: true,

			combinedContexts: map[string]commitStatus{"c1": toCommitStatus(success, ""), "c2": toCommitStatus(success, ""), "c3": toCommitStatus(success, ""), "c4": toCommitStatus(failure, "")},
			config: keeper.ContextPolicy{
				RequiredContexts: []string{"c1", "c2", "c3"},
				OptionalContexts: []string{"c4"},
			},
		},
		{
			name:    "skipping unknown contexts - failing because of missing required context c3",
			passing: false,
			config: keeper.ContextPolicy{
				RequiredContexts:    []string{"c1", "c2", "c3"},
				SkipUnknownContexts: &yes,
			},
			combinedContexts: map[string]commitStatus{"c1": toCommitStatus(success, ""), "c2": toCommitStatus(success, ""), statusContext: toCommitStatus(failure, "")},
		},
		{
			name:             "skipping unknown contexts - failing because c2 is failing",
			passing:          false,
			combinedContexts: map[string]commitStatus{"c1": toCommitStatus(success, ""), "c2": toCommitStatus(failure, "")},
			config: keeper.ContextPolicy{
				RequiredContexts:    []string{"c1", "c2"},
				OptionalContexts:    []string{"c4"},
				SkipUnknownContexts: &yes,
			},
		},
		{
			name:             "skipping unknown contexts - passing because c4 is optional",
			passing:          true,
			combinedContexts: map[string]commitStatus{"c1": toCommitStatus(success, ""), "c2": toCommitStatus(success, ""), "c3": toCommitStatus(success, ""), "c4": toCommitStatus(failure, "")},
			config: keeper.ContextPolicy{
				RequiredContexts:    []string{"c1", "c3"},
				OptionalContexts:    []string{"c4"},
				SkipUnknownContexts: &yes,
			},
		},
		{
			name:    "skipping unknown contexts - passing because c4 is optional and c5 is unknown",
			passing: true,

			combinedContexts: map[string]commitStatus{"c1": toCommitStatus(success, ""), "c2": toCommitStatus(success, ""), "c3": toCommitStatus(success, ""), "c4": toCommitStatus(failure, ""), "c5": toCommitStatus(failure, "")},
			config: keeper.ContextPolicy{
				RequiredContexts:    []string{"c1", "c3"},
				OptionalContexts:    []string{"c4"},
				SkipUnknownContexts: &yes,
			},
		},
	}

	for _, tc := range testCases {
		ghc := &fgc{
			combinedStatus: map[string]map[string]commitStatus{headSHA: tc.combinedContexts},
			expectedSHA:    headSHA}
		log := logrus.WithField("component", "keeper")
		_, err := log.String()
		if err != nil {
			t.Errorf("Failed to get log output before testing: %v", err)
			t.FailNow()
		}
		pr := PullRequest{HeadRefOID: githubql.String(headSHA)}
		passing := isPassingTests(log, ghc, pr, &tc.config)
		if passing != tc.passing {
			t.Errorf("%s: Expected %t got %t", tc.name, tc.passing, passing)
		}
	}
}

func TestPresubmitsByPull(t *testing.T) {
	samplePR := PullRequest{
		Number:     githubql.Int(100),
		HeadRefOID: githubql.String("sha"),
	}
	testcases := []struct {
		name string

		initialChangeCache map[changeCacheKey][]string
		presubmits         []job.Presubmit

		expectedPresubmits  map[int][]job.Presubmit
		expectedChangeCache map[changeCacheKey][]string
	}{
		{
			name: "no matching presubmits",
			presubmits: []job.Presubmit{
				{
					Reporter: job.Reporter{Context: "always"},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "foo",
					},
				},
				{
					Reporter: job.Reporter{Context: "never"},
				},
			},
			expectedChangeCache: map[changeCacheKey][]string{{number: 100, sha: "sha"}: {"CHANGED"}},
			expectedPresubmits:  map[int][]job.Presubmit{},
		},
		{
			name:               "no presubmits",
			presubmits:         []job.Presubmit{},
			expectedPresubmits: map[int][]job.Presubmit{},
		},
		{
			name: "no matching presubmits (check cache eviction)",
			presubmits: []job.Presubmit{
				{
					Reporter: job.Reporter{Context: "never"},
				},
			},
			initialChangeCache: map[changeCacheKey][]string{{number: 100, sha: "sha"}: {"FILE"}},
			expectedPresubmits: map[int][]job.Presubmit{},
		},
		{
			name: "no matching presubmits (check cache retention)",
			presubmits: []job.Presubmit{
				{
					Reporter: job.Reporter{Context: "always"},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "foo",
					},
				},
				{
					Reporter: job.Reporter{Context: "never"},
				},
			},
			initialChangeCache:  map[changeCacheKey][]string{{number: 100, sha: "sha"}: {"FILE"}},
			expectedChangeCache: map[changeCacheKey][]string{{number: 100, sha: "sha"}: {"FILE"}},
			expectedPresubmits:  map[int][]job.Presubmit{},
		},
		{
			name: "always_run",
			presubmits: []job.Presubmit{
				{
					Reporter:  job.Reporter{Context: "always"},
					AlwaysRun: true,
				},
				{
					Reporter: job.Reporter{Context: "never"},
				},
			},
			expectedPresubmits: map[int][]job.Presubmit{100: {{
				Reporter:  job.Reporter{Context: "always"},
				AlwaysRun: true,
			}}},
		},
		{
			name: "runs against branch",
			presubmits: []job.Presubmit{
				{
					Reporter:  job.Reporter{Context: "presubmit"},
					AlwaysRun: true,
					Brancher: job.Brancher{
						Branches: []string{"master", "dev"},
					},
				},
				{
					Reporter: job.Reporter{Context: "never"},
				},
			},
			expectedPresubmits: map[int][]job.Presubmit{100: {{
				Reporter:  job.Reporter{Context: "presubmit"},
				AlwaysRun: true,
				Brancher: job.Brancher{
					Branches: []string{"master", "dev"},
				},
			}}},
		},
		{
			name: "doesn't run against branch",
			presubmits: []job.Presubmit{
				{
					Reporter:  job.Reporter{Context: "presubmit"},
					AlwaysRun: true,
					Brancher: job.Brancher{
						Branches: []string{"release", "dev"},
					},
				},
				{
					Reporter:  job.Reporter{Context: "always"},
					AlwaysRun: true,
				},
				{
					Reporter: job.Reporter{Context: "never"},
				},
			},
			expectedPresubmits: map[int][]job.Presubmit{100: {{
				Reporter:  job.Reporter{Context: "always"},
				AlwaysRun: true,
			}}},
		},
		{
			name: "run_if_changed (uncached)",
			presubmits: []job.Presubmit{
				{
					Reporter: job.Reporter{Context: "presubmit"},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "^CHANGE.$",
					},
				},
				{
					Reporter:  job.Reporter{Context: "always"},
					AlwaysRun: true,
				},
				{
					Reporter: job.Reporter{Context: "never"},
				},
			},
			expectedPresubmits: map[int][]job.Presubmit{100: {{
				Reporter: job.Reporter{Context: "presubmit"},
				RegexpChangeMatcher: job.RegexpChangeMatcher{
					RunIfChanged: "^CHANGE.$",
				},
			}, {
				Reporter:  job.Reporter{Context: "always"},
				AlwaysRun: true,
			}}},
			expectedChangeCache: map[changeCacheKey][]string{{number: 100, sha: "sha"}: {"CHANGED"}},
		},
		{
			name: "run_if_changed (cached)",
			presubmits: []job.Presubmit{
				{
					Reporter: job.Reporter{Context: "presubmit"},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "^FIL.$",
					},
				},
				{
					Reporter:  job.Reporter{Context: "always"},
					AlwaysRun: true,
				},
				{
					Reporter: job.Reporter{Context: "never"},
				},
			},
			initialChangeCache: map[changeCacheKey][]string{{number: 100, sha: "sha"}: {"FILE"}},
			expectedPresubmits: map[int][]job.Presubmit{100: {{
				Reporter: job.Reporter{Context: "presubmit"},
				RegexpChangeMatcher: job.RegexpChangeMatcher{
					RunIfChanged: "^FIL.$",
				},
			},
				{
					Reporter:  job.Reporter{Context: "always"},
					AlwaysRun: true,
				}}},
			expectedChangeCache: map[changeCacheKey][]string{{number: 100, sha: "sha"}: {"FILE"}},
		},
		{
			name: "run_if_changed (cached) (skippable)",
			presubmits: []job.Presubmit{
				{
					Reporter: job.Reporter{Context: "presubmit"},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "^CHANGE.$",
					},
				},
				{
					Reporter:  job.Reporter{Context: "always"},
					AlwaysRun: true,
				},
				{
					Reporter: job.Reporter{Context: "never"},
				},
			},
			initialChangeCache: map[changeCacheKey][]string{{number: 100, sha: "sha"}: {"FILE"}},
			expectedPresubmits: map[int][]job.Presubmit{100: {{
				Reporter:  job.Reporter{Context: "always"},
				AlwaysRun: true,
			}}},
			expectedChangeCache: map[changeCacheKey][]string{{number: 100, sha: "sha"}: {"FILE"}},
		},
	}

	for _, tc := range testcases {
		t.Logf("Starting test case: %q", tc.name)

		if tc.initialChangeCache == nil {
			tc.initialChangeCache = map[changeCacheKey][]string{}
		}
		if tc.expectedChangeCache == nil {
			tc.expectedChangeCache = map[changeCacheKey][]string{}
		}

		cfg := &config.Config{}
		cfg.SetPresubmits(map[string][]job.Presubmit{
			"/":       tc.presubmits,
			"foo/bar": {{Reporter: job.Reporter{Context: "wrong-repo"}, AlwaysRun: true}},
		})
		cfgAgent := &config.Agent{}
		cfgAgent.Set(cfg)
		sp := &subpool{
			branch: "master",
			prs:    []PullRequest{samplePR},
		}
		c := &DefaultController{
			config: cfgAgent.Config,
			spc:    &fgc{},
			changedFiles: &changedFilesAgent{
				spc:             &fgc{},
				changeCache:     tc.initialChangeCache,
				nextChangeCache: make(map[changeCacheKey][]string),
			},
		}
		presubmits, err := c.presubmitsByPull(sp)
		if err != nil {
			t.Fatalf("unexpected error from presubmitsByPull: %v", err)
		}
		c.changedFiles.prune()
		// for equality we need to clear the compiled regexes
		for _, jobs := range presubmits {
			clearCompiledRegexes(jobs)
		}
		if !equality.Semantic.DeepEqual(presubmits, tc.expectedPresubmits) {
			t.Errorf("got incorrect presubmit mapping: %v\n", diff.ObjectReflectDiff(tc.expectedPresubmits, presubmits))
		}
		if got := c.changedFiles.changeCache; !reflect.DeepEqual(got, tc.expectedChangeCache) {
			t.Errorf("got incorrect file change cache: %v", diff.ObjectReflectDiff(tc.expectedChangeCache, got))
		}
	}
}

func getTemplate(name, tplStr string) *template.Template {
	tpl, _ := template.New(name).Parse(tplStr)
	return tpl
}

func TestPrepareMergeDetails(t *testing.T) {
	pr := PullRequest{
		Number:     githubql.Int(1),
		Mergeable:  githubql.MergeableStateMergeable,
		HeadRefOID: githubql.String("SHA"),
		Title:      "my commit title",
		Body:       "my commit body",
	}

	testCases := []struct {
		name        string
		tpl         keeper.MergeCommitTemplate
		pr          PullRequest
		mergeMethod keeper.PullRequestMergeType
		expected    scmprovider.MergeDetails
	}{{
		name:        "No commit template",
		tpl:         keeper.MergeCommitTemplate{},
		pr:          pr,
		mergeMethod: "merge",
		expected: scmprovider.MergeDetails{
			SHA:         "SHA",
			MergeMethod: "merge",
		},
	}, {
		name: "No commit template fields",
		tpl: keeper.MergeCommitTemplate{
			Title: nil,
			Body:  nil,
		},
		pr:          pr,
		mergeMethod: "merge",
		expected: scmprovider.MergeDetails{
			SHA:         "SHA",
			MergeMethod: "merge",
		},
	}, {
		name: "Static commit template",
		tpl: keeper.MergeCommitTemplate{
			Title: getTemplate("CommitTitle", "static title"),
			Body:  getTemplate("CommitBody", "static body"),
		},
		pr:          pr,
		mergeMethod: "merge",
		expected: scmprovider.MergeDetails{
			SHA:           "SHA",
			MergeMethod:   "merge",
			CommitTitle:   "static title",
			CommitMessage: "static body",
		},
	}, {
		name: "Commit template uses PullRequest fields",
		tpl: keeper.MergeCommitTemplate{
			Title: getTemplate("CommitTitle", "{{ .Number }}: {{ .Title }}"),
			Body:  getTemplate("CommitBody", "{{ .HeadRefOID }} - {{ .Body }}"),
		},
		pr:          pr,
		mergeMethod: "merge",
		expected: scmprovider.MergeDetails{
			SHA:           "SHA",
			MergeMethod:   "merge",
			CommitTitle:   "1: my commit title",
			CommitMessage: "SHA - my commit body",
		},
	}, {
		name: "Commit template uses nonexistent fields",
		tpl: keeper.MergeCommitTemplate{
			Title: getTemplate("CommitTitle", "{{ .Hello }}"),
			Body:  getTemplate("CommitBody", "{{ .World }}"),
		},
		pr:          pr,
		mergeMethod: "merge",
		expected: scmprovider.MergeDetails{
			SHA:         "SHA",
			MergeMethod: "merge",
		},
	}}

	for _, test := range testCases {
		cfg := &config.Config{}
		cfgAgent := &config.Agent{}
		cfgAgent.Set(cfg)
		c := &DefaultController{
			config: cfgAgent.Config,
			spc:    &fgc{},
			logger: logrus.WithField("component", "keeper"),
		}

		actual := c.prepareMergeDetails(test.tpl, test.pr, test.mergeMethod)

		if !reflect.DeepEqual(actual, test.expected) {
			t.Errorf("Case %s failed: expected %+v, got %+v", test.name, test.expected, actual)
		}
	}
}

func TestAccumulateReturnsCorrectMissingTests(t *testing.T) {
	testCases := []struct {
		name               string
		presubmits         map[int][]job.Presubmit
		prs                []PullRequest
		pjs                []v1alpha1.LighthouseJob
		expectedPresubmits map[int][]job.Presubmit
	}{
		{
			name: "All presubmits missing, no changes",
			prs: []PullRequest{{
				Number:     githubql.Int(1),
				HeadRefOID: githubql.String("sha"),
			}},
			presubmits: map[int][]job.Presubmit{1: {{
				Reporter: job.Reporter{
					Context: "my-presubmit",
				},
			}}},
			expectedPresubmits: map[int][]job.Presubmit{
				1: {{Reporter: job.Reporter{Context: "my-presubmit"}}},
			},
		},
		{
			name: "All presubmits successful, no retesting needed",
			prs: []PullRequest{{
				Number:     githubql.Int(1),
				HeadRefOID: githubql.String("sha"),
			}},
			pjs: []v1alpha1.LighthouseJob{{
				Spec: v1alpha1.LighthouseJobSpec{
					Type: job.PresubmitJob,
					Refs: &v1alpha1.Refs{
						Pulls: []v1alpha1.Pull{{
							Number: 1,
							SHA:    "sha",
						}},
					},
					Context: "my-presubmit",
				},
				Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.SuccessState},
			}},
			presubmits: map[int][]job.Presubmit{
				1: {{Reporter: job.Reporter{Context: "my-presubmit"}}},
			},
		},
		{
			name: "All presubmits pending, no retesting needed",
			prs: []PullRequest{{
				Number:     githubql.Int(1),
				HeadRefOID: githubql.String("sha"),
			}},
			pjs: []v1alpha1.LighthouseJob{{
				Spec: v1alpha1.LighthouseJobSpec{
					Type: job.PresubmitJob,
					Refs: &v1alpha1.Refs{
						Pulls: []v1alpha1.Pull{{
							Number: 1,
							SHA:    "sha",
						}},
					},
					Context: "my-presubmit",
				},
				Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.PendingState},
			}},
			presubmits: map[int][]job.Presubmit{
				1: {{Reporter: job.Reporter{Context: "my-presubmit"}}}},
		},
		{
			name: "One successful, one pending, one missing, one failing, only missing and failing remain",
			prs: []PullRequest{{
				Number:     githubql.Int(1),
				HeadRefOID: githubql.String("sha"),
			}},
			pjs: []v1alpha1.LighthouseJob{
				{
					Spec: v1alpha1.LighthouseJobSpec{
						Type: job.PresubmitJob,
						Refs: &v1alpha1.Refs{
							Pulls: []v1alpha1.Pull{{
								Number: 1,
								SHA:    "sha",
							}},
						},
						Context: "my-successful-presubmit",
					},
					Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.SuccessState},
				},
				{
					Spec: v1alpha1.LighthouseJobSpec{
						Type: job.PresubmitJob,
						Refs: &v1alpha1.Refs{
							Pulls: []v1alpha1.Pull{{
								Number: 1,
								SHA:    "sha",
							}},
						},
						Context: "my-pending-presubmit",
					},
					Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.PendingState},
				},
				{
					Spec: v1alpha1.LighthouseJobSpec{
						Type: job.PresubmitJob,
						Refs: &v1alpha1.Refs{
							Pulls: []v1alpha1.Pull{{
								Number: 1,
								SHA:    "sha",
							}},
						},
						Context: "my-failing-presubmit",
					},
					Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.FailureState},
				},
			},
			presubmits: map[int][]job.Presubmit{
				1: {
					{Reporter: job.Reporter{Context: "my-successful-presubmit"}},
					{Reporter: job.Reporter{Context: "my-pending-presubmit"}},
					{Reporter: job.Reporter{Context: "my-failing-presubmit"}},
					{Reporter: job.Reporter{Context: "my-missing-presubmit"}},
				}},
			expectedPresubmits: map[int][]job.Presubmit{
				1: {
					{Reporter: job.Reporter{Context: "my-failing-presubmit"}},
					{Reporter: job.Reporter{Context: "my-missing-presubmit"}},
				}},
		},
		{
			name: "Two prs, each with one successful, one pending, one missing, one failing, only missing and failing remain",
			prs: []PullRequest{
				{
					Number:     githubql.Int(1),
					HeadRefOID: githubql.String("sha"),
				},
				{
					Number:     githubql.Int(2),
					HeadRefOID: githubql.String("sha"),
				},
			},
			pjs: []v1alpha1.LighthouseJob{
				{
					Spec: v1alpha1.LighthouseJobSpec{
						Type: job.PresubmitJob,
						Refs: &v1alpha1.Refs{
							Pulls: []v1alpha1.Pull{{
								Number: 1,
								SHA:    "sha",
							}},
						},
						Context: "my-successful-presubmit",
					},
					Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.SuccessState},
				},
				{
					Spec: v1alpha1.LighthouseJobSpec{
						Type: job.PresubmitJob,
						Refs: &v1alpha1.Refs{
							Pulls: []v1alpha1.Pull{{
								Number: 1,
								SHA:    "sha",
							}},
						},
						Context: "my-pending-presubmit",
					},
					Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.PendingState},
				},
				{
					Spec: v1alpha1.LighthouseJobSpec{
						Type: job.PresubmitJob,
						Refs: &v1alpha1.Refs{
							Pulls: []v1alpha1.Pull{{
								Number: 1,
								SHA:    "sha",
							}},
						},
						Context: "my-failing-presubmit",
					},
					Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.FailureState},
				},
				{
					Spec: v1alpha1.LighthouseJobSpec{
						Type: job.PresubmitJob,
						Refs: &v1alpha1.Refs{
							Pulls: []v1alpha1.Pull{{
								Number: 2,
								SHA:    "sha",
							}},
						},
						Context: "my-successful-presubmit",
					},
					Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.SuccessState},
				},
				{
					Spec: v1alpha1.LighthouseJobSpec{
						Type: job.PresubmitJob,
						Refs: &v1alpha1.Refs{
							Pulls: []v1alpha1.Pull{{
								Number: 2,
								SHA:    "sha",
							}},
						},
						Context: "my-pending-presubmit",
					},
					Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.PendingState},
				},
				{
					Spec: v1alpha1.LighthouseJobSpec{
						Type: job.PresubmitJob,
						Refs: &v1alpha1.Refs{
							Pulls: []v1alpha1.Pull{{
								Number: 2,
								SHA:    "sha",
							}},
						},
						Context: "my-failing-presubmit",
					},
					Status: v1alpha1.LighthouseJobStatus{State: v1alpha1.FailureState},
				},
			},
			presubmits: map[int][]job.Presubmit{
				1: {
					{Reporter: job.Reporter{Context: "my-successful-presubmit"}},
					{Reporter: job.Reporter{Context: "my-pending-presubmit"}},
					{Reporter: job.Reporter{Context: "my-failing-presubmit"}},
					{Reporter: job.Reporter{Context: "my-missing-presubmit"}},
				},
				2: {
					{Reporter: job.Reporter{Context: "my-successful-presubmit"}},
					{Reporter: job.Reporter{Context: "my-pending-presubmit"}},
					{Reporter: job.Reporter{Context: "my-failing-presubmit"}},
					{Reporter: job.Reporter{Context: "my-missing-presubmit"}},
				},
			},
			expectedPresubmits: map[int][]job.Presubmit{
				1: {
					{Reporter: job.Reporter{Context: "my-failing-presubmit"}},
					{Reporter: job.Reporter{Context: "my-missing-presubmit"}},
				},
				2: {
					{Reporter: job.Reporter{Context: "my-failing-presubmit"}},
					{Reporter: job.Reporter{Context: "my-missing-presubmit"}},
				},
			},
		},
	}

	log := logrus.NewEntry(logrus.New())
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, missingSerialTests := accumulate(tc.presubmits, tc.prs, tc.pjs, log)
			// Apiequality treats nil slices/maps equal to a zero length slice/map, keeping us from
			// the burden of having to always initialize them
			if !apiequality.Semantic.DeepEqual(tc.expectedPresubmits, missingSerialTests) {
				t.Errorf("expected \n%v\n to be \n%v\n", missingSerialTests, tc.expectedPresubmits)
			}
		})
	}
}

func TestReposToQueries(t *testing.T) {
	firstQuery := keeper.Query{
		Repos:  []string{"a", "b"},
		Labels: []string{"first-label"},
	}
	secondQuery := keeper.Query{
		Repos:  []string{"b", "c"},
		Labels: []string{"second-label"},
	}

	queries := keeper.Queries{
		firstQuery,
		secondQuery,
	}

	queryMap := reposToQueries(queries)

	assert.Equal(t, 1, len(queryMap["a"]))
	assert.Equal(t, firstQuery, queryMap["a"][0])
	assert.Equal(t, 2, len(queryMap["b"]))
	assert.Contains(t, queryMap["b"], firstQuery)
	assert.Contains(t, queryMap["b"], secondQuery)
	assert.Equal(t, 1, len(queryMap["c"]))
	assert.Equal(t, secondQuery, queryMap["c"][0])
}
