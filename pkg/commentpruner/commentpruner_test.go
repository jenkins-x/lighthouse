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

package commentpruner

import (
	"reflect"
	"sort"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
)

type fakeSCMProviderClient struct {
	comments        []*scm.Comment
	deletedComments []int
	listCallCount   int
}

func (f *fakeSCMProviderClient) BotName() (string, error) {
	return "k8s-ci-robot", nil
}

func (f *fakeSCMProviderClient) ListIssueComments(_, _ string, _ int) ([]*scm.Comment, error) {
	f.listCallCount++
	return f.comments, nil
}

func (f *fakeSCMProviderClient) ListPullRequestComments(_, _ string, _ int) ([]*scm.Comment, error) {
	f.listCallCount++
	return f.comments, nil
}

func (f *fakeSCMProviderClient) DeleteComment(_, _ string, number, ID int, _ bool) error {
	f.deletedComments = append(f.deletedComments, ID)
	return nil
}

func newFakeSCMProviderClient(commentsToLogins map[int]string) *fakeSCMProviderClient {
	comments := make([]*scm.Comment, 0, len(commentsToLogins))
	for num, login := range commentsToLogins {
		comments = append(comments, &scm.Comment{ID: num, Author: scm.User{Login: login}})
	}
	return &fakeSCMProviderClient{
		comments:        comments,
		deletedComments: []int{},
	}
}

func testPruneFunc(errorComments *[]int, toPrunes, toErrs []int) func(*scm.Comment) bool {
	return func(ic *scm.Comment) bool {
		for _, toErr := range toErrs {
			if ic.ID == toErr {
				*errorComments = append(*errorComments, ic.ID)
				break
			}
		}
		for _, toPrune := range toPrunes {
			if ic.ID == toPrune {
				return true
			}
		}
		return false
	}
}

func TestPruneComments(t *testing.T) {
	botLogin := "k8s-ci-robot"
	humanLogin := "cjwagner"

	var errs *[]int
	tcs := []struct {
		name            string
		comments        map[int]string
		callers         []func(*scm.Comment) bool
		expectedDeleted []int
	}{
		{
			name:            "One caller, multiple deletions.",
			comments:        map[int]string{1: botLogin, 2: botLogin, 3: botLogin},
			callers:         []func(*scm.Comment) bool{testPruneFunc(errs, []int{1, 2}, nil)},
			expectedDeleted: []int{1, 2},
		},
		{
			name:            "One caller, no deletions.",
			comments:        map[int]string{3: botLogin},
			callers:         []func(*scm.Comment) bool{testPruneFunc(errs, []int{1, 2}, nil)},
			expectedDeleted: []int{},
		},
		{
			name:     "Two callers.",
			comments: map[int]string{1: botLogin, 2: botLogin, 3: botLogin, 4: botLogin, 5: botLogin},
			callers: []func(*scm.Comment) bool{
				testPruneFunc(errs, []int{1, 2}, nil),
				testPruneFunc(errs, []int{4}, []int{1, 2}),
			},
			expectedDeleted: []int{1, 2, 4},
		},
		{
			name:     "Three callers. Some Human messages",
			comments: map[int]string{1: humanLogin, 2: botLogin, 3: botLogin, 4: botLogin, 5: botLogin, 6: humanLogin, 7: botLogin},
			callers: []func(*scm.Comment) bool{
				testPruneFunc(errs, []int{2, 3}, []int{1, 6}),
				testPruneFunc(errs, []int{5}, []int{1, 2, 3, 6}),
				testPruneFunc(errs, []int{4}, []int{1, 2, 3, 5, 6}),
			},
			expectedDeleted: []int{2, 3, 4, 5},
		},
	}

	/*
		Ensure the following:
		When multiple callers ask for comment deletion from the same client...
		- They should not see comments deleted by previous caller.
		- Comments should be listed only once.
		- All comments that are stale should be deleted.
	*/
	for _, tc := range tcs {
		errs = &[]int{}
		fsc := newFakeSCMProviderClient(tc.comments)
		client := NewEventClient(fsc, logrus.WithField("client", "commentpruner"), "org", "repo", 1)
		for _, call := range tc.callers {
			client.PruneComments(true, call)
		}

		if fsc.listCallCount != 1 {
			t.Errorf("[%s]: Expected comments to be fetched exactly once, instead got %d.", tc.name, fsc.listCallCount)
		}
		if len(*errs) > 0 {
			t.Errorf("[%s]: The following comments should not have been seen be subsequent callers: %v.", tc.name, *errs)
		}
		sort.Ints(tc.expectedDeleted)
		sort.Ints(fsc.deletedComments)
		if !reflect.DeepEqual(tc.expectedDeleted, fsc.deletedComments) {
			t.Errorf("[%s]: Expected the comments %#v to be deleted, but %#v were deleted instead.", tc.name, tc.expectedDeleted, fsc.deletedComments)
		}
	}
}
