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

package reporter

import (
	"strings"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
)

func TestParsePRComment(t *testing.T) {
	var testcases = []struct {
		name             string
		context          string
		state            v1alpha1.PipelineState
		prcs             []*scm.Comment
		expectedDeletes  []int
		expectedContexts []string
		expectedUpdate   int
	}{
		{
			name:    "should delete old style comments",
			context: "Jenkins foo test",
			state:   v1alpha1.SuccessState,
			prcs: []*scm.Comment{
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "Jenkins foo test **failed** for such-and-such.",
					ID:     12345,
				},
				{
					Author: scm.User{Login: "someone-else"},
					Body:   "Jenkins foo test **failed**!? Why?",
					ID:     12356,
				},
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "Jenkins foo test **failed** for so-and-so.",
					ID:     12367,
				},
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "Jenkins bar test **failed** for something-or-other.",
					ID:     12378,
				},
			},
			expectedDeletes: []int{12345, 12367},
		},
		{
			name:             "should create a new comment",
			context:          "bla test",
			state:            v1alpha1.FailureState,
			expectedContexts: []string{"bla test"},
		},
		{
			name:    "should not delete an up-to-date comment",
			context: "bla test",
			state:   v1alpha1.SuccessState,
			prcs: []*scm.Comment{
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "--- | --- | ---\nfoo test | something | or other\n\n",
				},
			},
		},
		{
			name:    "should delete when all tests pass",
			context: "bla test",
			state:   v1alpha1.SuccessState,
			prcs: []*scm.Comment{
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "--- | --- | ---\nbla test | something | or other\n\n" + commentTag,
					ID:     123,
				},
			},
			expectedDeletes:  []int{123},
			expectedContexts: []string{},
		},
		{
			name:    "should delete a passing test with \\r",
			context: "bla test",
			state:   v1alpha1.SuccessState,
			prcs: []*scm.Comment{
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "--- | --- | ---\r\nbla test | something | or other\r\n\r\n" + commentTag,
					ID:     123,
				},
			},
			expectedDeletes:  []int{123},
			expectedContexts: []string{},
		},

		{
			name:    "should update a failed test",
			context: "bla test",
			state:   v1alpha1.FailureState,
			prcs: []*scm.Comment{
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "--- | --- | ---\nbla test | something | or other\n\n" + commentTag,
					ID:     123,
				},
			},
			expectedDeletes:  []int{123},
			expectedContexts: []string{"bla test"},
		},
		{
			name:    "should preserve old results when updating",
			context: "bla test",
			state:   v1alpha1.FailureState,
			prcs: []*scm.Comment{
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "--- | --- | ---\nbla test | something | or other\nfoo test | wow | aye\n\n" + commentTag,
					ID:     123,
				},
			},
			expectedDeletes:  []int{123},
			expectedContexts: []string{"bla test", "foo test"},
		},
		{
			name:    "should merge duplicates",
			context: "bla test",
			state:   v1alpha1.FailureState,
			prcs: []*scm.Comment{
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "--- | --- | ---\nbla test | something | or other\nfoo test | wow such\n\n" + commentTag,
					ID:     123,
				},
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "--- | --- | ---\nfoo test | beep | boop\n\n" + commentTag,
					ID:     124,
				},
			},
			expectedDeletes:  []int{123, 124},
			expectedContexts: []string{"bla test", "foo test"},
		},
		{
			name:    "should update an old comment when a test passes",
			context: "bla test",
			state:   v1alpha1.SuccessState,
			prcs: []*scm.Comment{
				{
					Author: scm.User{Login: "k8s-ci-robot"},
					Body:   "--- | --- | ---\nbla test | something | or other\nfoo test | wow | aye\n\n" + commentTag,
					ID:     123,
				},
			},
			expectedDeletes:  []int{},
			expectedContexts: []string{"foo test"},
			expectedUpdate:   123,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			lhj := &v1alpha1.LighthouseJob{
				Spec: v1alpha1.LighthouseJobSpec{
					Context: tc.context,
					Refs:    &v1alpha1.Refs{Pulls: []v1alpha1.Pull{{}}},
				},
				Status: v1alpha1.LighthouseJobStatus{
					State: tc.state,
				},
			}
			deletes, entries, update := parsePRComments(lhj, "k8s-ci-robot", tc.prcs)
			if len(deletes) != len(tc.expectedDeletes) {
				t.Errorf("It %s: wrong number of deletes. Got %v, expected %v", tc.name, deletes, tc.expectedDeletes)
			} else {
				for _, edel := range tc.expectedDeletes {
					found := false
					for _, del := range deletes {
						if del == edel {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("It %s: expected to find %d in %v", tc.name, edel, deletes)
					}
				}
			}
			if len(entries) != len(tc.expectedContexts) {
				t.Errorf("It %s: wrong number of entries. Got %v, expected %v", tc.name, entries, tc.expectedContexts)
			} else {
				for _, econt := range tc.expectedContexts {
					found := false
					for _, ent := range entries {
						if strings.Contains(ent, econt) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("It %s: expected to find %s in %v", tc.name, econt, entries)
					}
				}
			}
			if tc.expectedUpdate != update {
				t.Errorf("It %s: expected update %d, got %d", tc.name, tc.expectedUpdate, update)
			}
		})
	}
}

func TestShouldReport(t *testing.T) {
	var testcases = []struct {
		name       string
		lhj        *v1alpha1.LighthouseJob
		validTypes []job.PipelineKind
		report     bool
	}{
		{
			name: "should report presubmit job",
			lhj: &v1alpha1.LighthouseJob{
				Spec: v1alpha1.LighthouseJobSpec{
					Type: job.PresubmitJob,
				},
			},
			validTypes: []job.PipelineKind{job.PresubmitJob},
			report:     true,
		},
		{
			name: "should not report postsubmit job",
			lhj: &v1alpha1.LighthouseJob{
				Spec: v1alpha1.LighthouseJobSpec{
					Type: job.PostsubmitJob,
				},
			},
			validTypes: []job.PipelineKind{job.PresubmitJob},
		},
		{
			name: "should report postsubmit job if told to",
			lhj: &v1alpha1.LighthouseJob{
				Spec: v1alpha1.LighthouseJobSpec{
					Type: job.PostsubmitJob,
				},
			},
			validTypes: []job.PipelineKind{job.PresubmitJob, job.PostsubmitJob},
			report:     true,
		},
	}

	for _, tc := range testcases {
		r := ShouldReport(tc.lhj, tc.validTypes)

		if r != tc.report {
			t.Errorf("Unexpected result from test: %s.\nExpected: %v\nGot: %v",
				tc.name, tc.report, r)
		}
	}
}
