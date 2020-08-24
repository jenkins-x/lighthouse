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

package jobutil

import (
	"reflect"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/util/diff"

	"github.com/jenkins-x/lighthouse/pkg/config"
)

func TestPostsubmitSpec(t *testing.T) {
	tests := []struct {
		name     string
		p        config.Postsubmit
		refs     v1alpha1.Refs
		expected v1alpha1.LighthouseJobSpec
	}{
		{
			name: "can override path alias and cloneuri",
			p: config.Postsubmit{
				JobBase: config.JobBase{
					UtilityConfig: config.UtilityConfig{
						PathAlias: "foo",
						CloneURI:  "bar",
					},
				},
			},
			expected: v1alpha1.LighthouseJobSpec{
				Type: config.PostsubmitJob,
				Refs: &v1alpha1.Refs{
					PathAlias: "foo",
					CloneURI:  "bar",
				},
			},
		},
		{
			name: "controller can default path alias and cloneuri",
			refs: v1alpha1.Refs{
				PathAlias: "fancy",
				CloneURI:  "cats",
			},
			expected: v1alpha1.LighthouseJobSpec{
				Type: config.PostsubmitJob,
				Refs: &v1alpha1.Refs{
					PathAlias: "fancy",
					CloneURI:  "cats",
				},
			},
		},
		{
			name: "job overrides take precedence over controller defaults",
			p: config.Postsubmit{
				JobBase: config.JobBase{
					UtilityConfig: config.UtilityConfig{
						PathAlias: "foo",
						CloneURI:  "bar",
					},
				},
			},
			refs: v1alpha1.Refs{
				PathAlias: "fancy",
				CloneURI:  "cats",
			},
			expected: v1alpha1.LighthouseJobSpec{
				Type: config.PostsubmitJob,
				Refs: &v1alpha1.Refs{
					PathAlias: "foo",
					CloneURI:  "bar",
				},
			},
		},
	}

	for _, tc := range tests {
		actual := PostsubmitSpec(tc.p, tc.refs)
		if expected := tc.expected; !reflect.DeepEqual(actual, expected) {
			t.Errorf("%s: actual %#v != expected %#v", tc.name, actual, expected)
		}
	}
}

func TestPresubmitSpec(t *testing.T) {
	tests := []struct {
		name     string
		p        config.Presubmit
		refs     v1alpha1.Refs
		expected v1alpha1.LighthouseJobSpec
	}{
		{
			name: "can override path alias and cloneuri",
			p: config.Presubmit{
				JobBase: config.JobBase{
					UtilityConfig: config.UtilityConfig{
						PathAlias: "foo",
						CloneURI:  "bar",
					},
				},
			},
			expected: v1alpha1.LighthouseJobSpec{
				Type: config.PresubmitJob,
				Refs: &v1alpha1.Refs{
					PathAlias: "foo",
					CloneURI:  "bar",
				},
			},
		},
		{
			name: "controller can default path alias and cloneuri",
			refs: v1alpha1.Refs{
				PathAlias: "fancy",
				CloneURI:  "cats",
			},
			expected: v1alpha1.LighthouseJobSpec{
				Type: config.PresubmitJob,
				Refs: &v1alpha1.Refs{
					PathAlias: "fancy",
					CloneURI:  "cats",
				},
			},
		},
		{
			name: "job overrides take precedence over controller defaults",
			p: config.Presubmit{
				JobBase: config.JobBase{
					UtilityConfig: config.UtilityConfig{
						PathAlias: "foo",
						CloneURI:  "bar",
					},
				},
			},
			refs: v1alpha1.Refs{
				PathAlias: "fancy",
				CloneURI:  "cats",
			},
			expected: v1alpha1.LighthouseJobSpec{
				Type: config.PresubmitJob,
				Refs: &v1alpha1.Refs{
					PathAlias: "foo",
					CloneURI:  "bar",
				},
			},
		},
	}

	for _, tc := range tests {
		actual := PresubmitSpec(tc.p, tc.refs)
		if expected := tc.expected; !reflect.DeepEqual(actual, expected) {
			t.Errorf("%s: actual %#v != expected %#v", tc.name, actual, expected)
		}
	}
}

func TestBatchSpec(t *testing.T) {
	tests := []struct {
		name     string
		p        config.Presubmit
		refs     v1alpha1.Refs
		expected v1alpha1.LighthouseJobSpec
	}{
		{
			name: "can override path alias and cloneuri",
			p: config.Presubmit{
				JobBase: config.JobBase{
					UtilityConfig: config.UtilityConfig{
						PathAlias: "foo",
						CloneURI:  "bar",
					},
				},
			},
			expected: v1alpha1.LighthouseJobSpec{
				Type: config.BatchJob,
				Refs: &v1alpha1.Refs{
					PathAlias: "foo",
					CloneURI:  "bar",
				},
			},
		},
		{
			name: "controller can default path alias and cloneuri",
			refs: v1alpha1.Refs{
				PathAlias: "fancy",
				CloneURI:  "cats",
			},
			expected: v1alpha1.LighthouseJobSpec{
				Type: config.BatchJob,
				Refs: &v1alpha1.Refs{
					PathAlias: "fancy",
					CloneURI:  "cats",
				},
			},
		},
		{
			name: "job overrides take precedence over controller defaults",
			p: config.Presubmit{
				JobBase: config.JobBase{
					UtilityConfig: config.UtilityConfig{
						PathAlias: "foo",
						CloneURI:  "bar",
					},
				},
			},
			refs: v1alpha1.Refs{
				PathAlias: "fancy",
				CloneURI:  "cats",
			},
			expected: v1alpha1.LighthouseJobSpec{
				Type: config.BatchJob,
				Refs: &v1alpha1.Refs{
					PathAlias: "foo",
					CloneURI:  "bar",
				},
			},
		},
	}

	for _, tc := range tests {
		actual := BatchSpec(tc.p, tc.refs)
		if expected := tc.expected; !reflect.DeepEqual(actual, expected) {
			t.Errorf("%s: actual %#v != expected %#v", tc.name, actual, expected)
		}
	}
}

func TestNewLighthouseJob(t *testing.T) {
	var testCases = []struct {
		name                string
		spec                v1alpha1.LighthouseJobSpec
		labels              map[string]string
		expectedLabels      map[string]string
		annotations         map[string]string
		expectedAnnotations map[string]string
	}{
		{
			name: "periodic job, no extra labels",
			spec: v1alpha1.LighthouseJobSpec{
				Job:  "job",
				Type: config.PeriodicJob,
			},
			labels: map[string]string{},
			expectedLabels: map[string]string{
				config.CreatedByLighthouse:    "true",
				util.LighthouseJobAnnotation:  "job",
				config.LighthouseJobTypeLabel: "periodic",
			},
			expectedAnnotations: map[string]string{
				util.LighthouseJobAnnotation: "job",
			},
		},
		{
			name: "periodic job, extra labels",
			spec: v1alpha1.LighthouseJobSpec{
				Job:  "job",
				Type: config.PeriodicJob,
			},
			labels: map[string]string{
				"extra": "stuff",
			},
			expectedLabels: map[string]string{
				config.CreatedByLighthouse:    "true",
				util.LighthouseJobAnnotation:  "job",
				config.LighthouseJobTypeLabel: "periodic",
				"extra":                       "stuff",
			},
			expectedAnnotations: map[string]string{
				util.LighthouseJobAnnotation: "job",
			},
		},
		{
			name: "presubmit job",
			spec: v1alpha1.LighthouseJobSpec{
				Job:  "job",
				Type: config.PresubmitJob,
				Refs: &v1alpha1.Refs{
					Org:     "org",
					Repo:    "repo",
					BaseSHA: "abcd1234",
					Pulls: []v1alpha1.Pull{
						{
							Number: 1,
							SHA:    "1234abcd",
						},
					},
				},
				Context: "pr-build",
			},
			labels: map[string]string{},
			expectedLabels: map[string]string{
				config.CreatedByLighthouse:    "true",
				util.LighthouseJobAnnotation:  "job",
				config.LighthouseJobTypeLabel: "presubmit",
				util.OrgLabel:                 "org",
				util.RepoLabel:                "repo",
				util.PullLabel:                "1",
				util.BranchLabel:              "PR-1",
				util.ContextLabel:             "pr-build",
				util.BaseSHALabel:             "abcd1234",
				util.LastCommitSHALabel:       "1234abcd",
			},
			expectedAnnotations: map[string]string{
				util.LighthouseJobAnnotation: "job",
			},
		},
		{
			name: "non-github presubmit job",
			spec: v1alpha1.LighthouseJobSpec{
				Job:  "job",
				Type: config.PresubmitJob,
				Refs: &v1alpha1.Refs{
					Org:     "https://some-gerrit-instance.foo.com",
					Repo:    "some/invalid/repo",
					BaseSHA: "abcd1234",
					Pulls: []v1alpha1.Pull{
						{
							Number: 1,
							SHA:    "1234abcd",
						},
					},
				},
			},
			labels: map[string]string{},
			expectedLabels: map[string]string{
				config.CreatedByLighthouse:    "true",
				util.LighthouseJobAnnotation:  "job",
				config.LighthouseJobTypeLabel: "presubmit",
				util.OrgLabel:                 "some-gerrit-instance.foo.com",
				util.RepoLabel:                "repo",
				util.PullLabel:                "1",
				util.BranchLabel:              "PR-1",
				util.BaseSHALabel:             "abcd1234",
				util.LastCommitSHALabel:       "1234abcd",
			},
			expectedAnnotations: map[string]string{
				util.LighthouseJobAnnotation: "job",
			},
		}, {
			name: "job with name too long to fit in a label",
			spec: v1alpha1.LighthouseJobSpec{
				Job:  "job-created-by-someone-who-loves-very-very-very-long-names-so-long-that-it-does-not-fit-into-the-Kubernetes-label-so-it-needs-to-be-truncated-to-63-characters",
				Type: config.PresubmitJob,
				Refs: &v1alpha1.Refs{
					Org:      "org",
					Repo:     "repo",
					BaseSHA:  "abcd1234",
					CloneURI: "https://github.com/org/repo.git",
					Pulls: []v1alpha1.Pull{
						{
							Number: 1,
							SHA:    "1234abcd",
						},
					},
				},
			},
			labels: map[string]string{},
			expectedLabels: map[string]string{
				config.CreatedByLighthouse:    "true",
				util.LighthouseJobAnnotation:  "job-created-by-someone-who-loves-very-very-very-long-names-so-l",
				config.LighthouseJobTypeLabel: "presubmit",
				util.OrgLabel:                 "org",
				util.RepoLabel:                "repo",
				util.PullLabel:                "1",
				util.BranchLabel:              "PR-1",
				util.BaseSHALabel:             "abcd1234",
				util.LastCommitSHALabel:       "1234abcd",
			},
			expectedAnnotations: map[string]string{
				util.LighthouseJobAnnotation: "job-created-by-someone-who-loves-very-very-very-long-names-so-long-that-it-does-not-fit-into-the-Kubernetes-label-so-it-needs-to-be-truncated-to-63-characters",
				util.CloneURIAnnotation:      "https://github.com/org/repo.git",
			},
		},
		{
			name: "periodic job, extra labels, extra annotations",
			spec: v1alpha1.LighthouseJobSpec{
				Job:  "job",
				Type: config.PeriodicJob,
			},
			labels: map[string]string{
				"extra": "stuff",
			},
			annotations: map[string]string{
				"extraannotation": "foo",
			},
			expectedLabels: map[string]string{
				config.CreatedByLighthouse:    "true",
				util.LighthouseJobAnnotation:  "job",
				config.LighthouseJobTypeLabel: "periodic",
				"extra":                       "stuff",
			},
			expectedAnnotations: map[string]string{
				util.LighthouseJobAnnotation: "job",
				"extraannotation":            "foo",
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pj := NewLighthouseJob(testCase.spec, testCase.labels, testCase.annotations)
			if actual, expected := pj.Spec, testCase.spec; !equality.Semantic.DeepEqual(actual, expected) {
				t.Errorf("%s: incorrect PipelineOptionsSpec created: %s", testCase.name, diff.ObjectReflectDiff(actual, expected))
			}
			if actual, expected := pj.Labels, testCase.expectedLabels; !reflect.DeepEqual(actual, expected) {
				t.Errorf("%s: incorrect PipelineOptions labels created: %s", testCase.name, diff.ObjectReflectDiff(actual, expected))
			}
			if actual, expected := pj.Annotations, testCase.expectedAnnotations; !reflect.DeepEqual(actual, expected) {
				t.Errorf("%s: incorrect PipelineOptions annotations created: %s", testCase.name, diff.ObjectReflectDiff(actual, expected))
			}
		})
	}
}

func TestNewLighthouseJobWithAnnotations(t *testing.T) {
	var testCases = []struct {
		name                string
		spec                v1alpha1.LighthouseJobSpec
		annotations         map[string]string
		expectedAnnotations map[string]string
	}{
		{
			name: "job without annotation",
			spec: v1alpha1.LighthouseJobSpec{
				Job:  "job",
				Type: config.PeriodicJob,
			},
			annotations: nil,
			expectedAnnotations: map[string]string{
				util.LighthouseJobAnnotation: "job",
			},
		},
		{
			name: "job with annotation",
			spec: v1alpha1.LighthouseJobSpec{
				Job:  "job",
				Type: config.PeriodicJob,
			},
			annotations: map[string]string{
				"annotation": "foo",
			},
			expectedAnnotations: map[string]string{
				"annotation":                 "foo",
				util.LighthouseJobAnnotation: "job",
			},
		},
	}

	for _, testCase := range testCases {
		pj := NewLighthouseJob(testCase.spec, nil, testCase.annotations)
		if actual, expected := pj.Spec, testCase.spec; !equality.Semantic.DeepEqual(actual, expected) {
			t.Errorf("%s: incorrect PipelineOptionsSpec created: %s", testCase.name, diff.ObjectReflectDiff(actual, expected))
		}
		if actual, expected := pj.Annotations, testCase.expectedAnnotations; !reflect.DeepEqual(actual, expected) {
			t.Errorf("%s: incorrect PipelineOptions labels created: %s", testCase.name, diff.ObjectReflectDiff(actual, expected))
		}
	}
}

func TestCreateRefs(t *testing.T) {
	pr := &scm.PullRequest{
		Number: 42,
		Link:   "https://github.example.com/kubernetes/Hello-World/pull/42",
		Head: scm.PullRequestBranch{
			Sha: "123456",
		},
		Base: scm.PullRequestBranch{
			Ref: "master",
			Repo: scm.Repository{
				Name:      "Hello-World",
				Link:      "https://github.example.com/kubernetes/Hello-World",
				Namespace: "kubernetes",
			},
		},
		Author: scm.User{
			Login: "ibzib",
			Link:  "https://github.example.com/ibzib",
		},
	}
	expected := v1alpha1.Refs{
		Org:      "kubernetes",
		Repo:     "Hello-World",
		RepoLink: "https://github.example.com/kubernetes/Hello-World",
		BaseRef:  "master",
		BaseSHA:  "abcdef",
		BaseLink: "https://github.example.com/kubernetes/Hello-World/commit/abcdef",
		Pulls: []v1alpha1.Pull{
			{
				Number:     42,
				Author:     "ibzib",
				SHA:        "123456",
				Link:       "https://github.example.com/kubernetes/Hello-World/pull/42",
				AuthorLink: "https://github.example.com/ibzib",
				CommitLink: "https://github.example.com/kubernetes/Hello-World/pull/42/commits/123456",
				Ref:        "refs/pull/42/head",
			},
		},
	}
	if actual := createRefs(pr, "abcdef", "refs/pull/%d/head"); !reflect.DeepEqual(expected, actual) {
		t.Errorf("diff between expected and actual refs:%s", diff.ObjectReflectDiff(expected, actual))
	}
}

func TestSpecFromJobBase(t *testing.T) {
	testCases := []struct {
		name    string
		jobBase config.JobBase
		verify  func(v1alpha1.LighthouseJobSpec) error
	}{
		{
			name:    "Verify reporter config gets copied",
			jobBase: config.JobBase{
				/*				ReporterConfig: &v1alpha1.ReporterConfig{
									Slack: &v1alpha1.SlackReporterConfig{
										Channel: "my-channel",
									},
								},
				*/
			},
			verify: func(pj v1alpha1.LighthouseJobSpec) error {
				/*				if pj.ReporterConfig == nil {
									return errors.New("Expected ReporterConfig to be non-nil")
								}
								if pj.ReporterConfig.Slack == nil {
									return errors.New("Expected ReporterConfig.Slack to be non-nil")
								}
								if pj.ReporterConfig.Slack.Channel != "my-channel" {
									return fmt.Errorf("Expected pj.ReporterConfig.Slack.Channel to be \"my-channel\", was %q",
										pj.ReporterConfig.Slack.Channel)
								}
				*/
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pj := specFromJobBase(tc.jobBase)
			if err := tc.verify(pj); err != nil {
				t.Fatalf("Verification failed: %v", err)
			}
		})
	}
}
