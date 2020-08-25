/*
Copyright 2019 The Kubernetes Authors.

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
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"k8s.io/apimachinery/pkg/util/diff"
)

func TestTestAllFilter(t *testing.T) {
	var testCases = []struct {
		name       string
		presubmits []job.Presubmit
		expected   [][]bool
	}{
		{
			name: "test all filter matches jobs which do not require human triggering",
			presubmits: []job.Presubmit{
				{
					Base: job.Base{
						Name: "always-runs",
					},
					AlwaysRun: true,
				},
				{
					Base: job.Base{
						Name: "runs-if-changed",
					},
					AlwaysRun: false,
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "sometimes",
					},
				},
				{
					Base: job.Base{
						Name: "runs-if-triggered",
					},
					Reporter: job.Reporter{
						Context: "runs-if-triggered",
					},
					Trigger:      `(?m)^/test (?:.*? )?trigger(?: .*?)?$`,
					RerunCommand: "/test trigger",
				},
				{
					Base: job.Base{
						Name: "literal-test-all-trigger",
					},
					Reporter: job.Reporter{
						Context: "runs-if-triggered",
					},
					Trigger:      `(?m)^/test (?:.*? )?all(?: .*?)?$`,
					RerunCommand: "/test all",
				},
			},
			expected: [][]bool{{true, false, false}, {true, false, false}, {false, false, false}, {false, false, false}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if len(testCase.presubmits) != len(testCase.expected) {
				t.Fatalf("%s: have %d presubmits but only %d expected filter outputs", testCase.name, len(testCase.presubmits), len(testCase.expected))
			}
			for i := range testCase.presubmits {
				if err := testCase.presubmits[i].SetRegexes(); err != nil {
					t.Fatalf("%s: could not set presubmit regexes: %v", testCase.name, err)
				}
			}
			filter := TestAllFilter()
			for i, presubmit := range testCase.presubmits {
				actualFiltered, actualForced, actualDefault := filter(presubmit)
				expectedFiltered, expectedForced, expectedDefault := testCase.expected[i][0], testCase.expected[i][1], testCase.expected[i][2]
				if actualFiltered != expectedFiltered {
					t.Errorf("%s: filter did not evaluate correctly, expected %v but got %v for %v", testCase.name, expectedFiltered, actualFiltered, presubmit.Name)
				}
				if actualForced != expectedForced {
					t.Errorf("%s: filter did not determine forced correctly, expected %v but got %v for %v", testCase.name, expectedForced, actualForced, presubmit.Name)
				}
				if actualDefault != expectedDefault {
					t.Errorf("%s: filter did not determine default correctly, expected %v but got %v for %v", testCase.name, expectedDefault, actualDefault, presubmit.Name)
				}
			}
		})
	}
}

func TestCommandFilter(t *testing.T) {
	var testCases = []struct {
		name       string
		body       string
		presubmits []job.Presubmit
		expected   [][]bool
	}{
		{
			name: "command filter matches jobs whose triggers match the body",
			body: "/test trigger",
			presubmits: []job.Presubmit{
				{
					Base: job.Base{
						Name: "trigger",
					},
					Trigger:      `(?m)^/test (?:.*? )?trigger(?: .*?)?$`,
					RerunCommand: "/test trigger",
				},
				{
					Base: job.Base{
						Name: "other-trigger",
					},
					Trigger:      `(?m)^/test (?:.*? )?other-trigger(?: .*?)?$`,
					RerunCommand: "/test other-trigger",
				},
			},
			expected: [][]bool{{true, true, true}, {false, false, true}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if len(testCase.presubmits) != len(testCase.expected) {
				t.Fatalf("%s: have %d presubmits but only %d expected filter outputs", testCase.name, len(testCase.presubmits), len(testCase.expected))
			}
			for i := range testCase.presubmits {
				if err := testCase.presubmits[i].SetRegexes(); err != nil {
					t.Fatalf("%s: could not set presubmit regexes: %v", testCase.name, err)
				}
			}
			filter := CommandFilter(testCase.body)
			for i, presubmit := range testCase.presubmits {
				actualFiltered, actualForced, actualDefault := filter(presubmit)
				expectedFiltered, expectedForced, expectedDefault := testCase.expected[i][0], testCase.expected[i][1], testCase.expected[i][2]
				if actualFiltered != expectedFiltered {
					t.Errorf("%s: filter did not evaluate correctly, expected %v but got %v for %v", testCase.name, expectedFiltered, actualFiltered, presubmit.Name)
				}
				if actualForced != expectedForced {
					t.Errorf("%s: filter did not determine forced correctly, expected %v but got %v for %v", testCase.name, expectedForced, actualForced, presubmit.Name)
				}
				if actualDefault != expectedDefault {
					t.Errorf("%s: filter did not determine default correctly, expected %v but got %v for %v", testCase.name, expectedDefault, actualDefault, presubmit.Name)
				}
			}
		})
	}
}

func fakeChangedFilesProvider(shouldError bool) job.ChangedFilesProvider {
	return func() ([]string, error) {
		if shouldError {
			return nil, errors.New("error getting changes")
		}
		return nil, nil
	}
}

func TestFilterPresubmits(t *testing.T) {
	var testCases = []struct {
		name                              string
		filter                            Filter
		presubmits                        []job.Presubmit
		changesError                      bool
		expectedToTrigger, expectedToSkip []job.Presubmit
		expectErr                         bool
	}{
		{
			name: "nothing matches, nothing to run or skip",
			filter: func(p job.Presubmit) (shouldRun bool, forcedToRun bool, defaultBehavior bool) {
				return false, false, false
			},
			presubmits: []job.Presubmit{{
				Base:     job.Base{Name: "ignored"},
				Reporter: job.Reporter{Context: "first"},
			}, {
				Base:     job.Base{Name: "ignored"},
				Reporter: job.Reporter{Context: "second"},
			}},
			changesError:      false,
			expectedToTrigger: nil,
			expectedToSkip:    nil,
			expectErr:         false,
		},
		{
			name: "everything matches and is forced to run, nothing to skip",
			filter: func(p job.Presubmit) (shouldRun bool, forcedToRun bool, defaultBehavior bool) {
				return true, true, true
			},
			presubmits: []job.Presubmit{{
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "first"},
			}, {
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "second"},
			}},
			changesError: false,
			expectedToTrigger: []job.Presubmit{{
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "first"},
			}, {
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "second"},
			}},
			expectedToSkip: nil,
			expectErr:      false,
		},
		{
			name: "error detecting if something should run, nothing to run or skip",
			filter: func(p job.Presubmit) (shouldRun bool, forcedToRun bool, defaultBehavior bool) {
				return true, false, false
			},
			presubmits: []job.Presubmit{{
				Base:                job.Base{Name: "errors"},
				Reporter:            job.Reporter{Context: "first"},
				RegexpChangeMatcher: job.RegexpChangeMatcher{RunIfChanged: "oopsie"},
			}, {
				Base:     job.Base{Name: "ignored"},
				Reporter: job.Reporter{Context: "second"},
			}},
			changesError:      true,
			expectedToTrigger: nil,
			expectedToSkip:    nil,
			expectErr:         true,
		},
		{
			name: "some things match and are forced to run, nothing to skip",
			filter: func(p job.Presubmit) (shouldRun bool, forcedToRun bool, defaultBehavior bool) {
				return p.Name == "should-trigger", true, true
			},
			presubmits: []job.Presubmit{{
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "first"},
			}, {
				Base:     job.Base{Name: "ignored"},
				Reporter: job.Reporter{Context: "second"},
			}},
			changesError: false,
			expectedToTrigger: []job.Presubmit{{
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "first"},
			}},
			expectedToSkip: nil,
			expectErr:      false,
		},
		{
			name: "everything matches and some things are forced to run, others should be skipped",
			filter: func(p job.Presubmit) (shouldRun bool, forcedToRun bool, defaultBehavior bool) {
				return true, p.Name == "should-trigger", p.Name == "should-trigger"
			},
			presubmits: []job.Presubmit{{
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "first"},
			}, {
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "second"},
			}, {
				Base:     job.Base{Name: "should-skip"},
				Reporter: job.Reporter{Context: "third"},
			}, {
				Base:     job.Base{Name: "should-skip2"},
				Reporter: job.Reporter{Context: "fourth"},
			}},
			changesError: false,
			expectedToTrigger: []job.Presubmit{{
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "first"},
			}, {
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "second"},
			}},
			expectedToSkip: []job.Presubmit{{
				Base:     job.Base{Name: "should-skip"},
				Reporter: job.Reporter{Context: "third"},
			}, {
				Base:     job.Base{Name: "should-skip2"},
				Reporter: job.Reporter{Context: "fourth"},
			}},
			expectErr: false,
		},
		{
			name: "everything matches and some that are forces to run supercede some that are skipped due to shared contexts",
			filter: func(p job.Presubmit) (shouldRun bool, forcedToRun bool, defaultBehavior bool) {
				return true, p.Name == "should-trigger", p.Name == "should-trigger"
			},
			presubmits: []job.Presubmit{{
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "first"},
			}, {
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "second"},
			}, {
				Base:     job.Base{Name: "should-skip"},
				Reporter: job.Reporter{Context: "third"},
			}, {
				Base:     job.Base{Name: "should-not-skip"},
				Reporter: job.Reporter{Context: "second"},
			}},
			changesError: false,
			expectedToTrigger: []job.Presubmit{{
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "first"},
			}, {
				Base:     job.Base{Name: "should-trigger"},
				Reporter: job.Reporter{Context: "second"},
			}},
			expectedToSkip: []job.Presubmit{{
				Base:     job.Base{Name: "should-skip"},
				Reporter: job.Reporter{Context: "third"},
			}},
			expectErr: false,
		},
	}

	branch := "foobar"

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actualToTrigger, actualToSkip, err := FilterPresubmits(testCase.filter, fakeChangedFilesProvider(testCase.changesError), branch, testCase.presubmits, logrus.WithField("test-case", testCase.name))
			if testCase.expectErr && err == nil {
				t.Errorf("%s: expected an error filtering presubmits, but got none", testCase.name)
			}
			if !testCase.expectErr && err != nil {
				t.Errorf("%s: expected no error filtering presubmits, but got one: %v", testCase.name, err)
			}
			if !reflect.DeepEqual(actualToTrigger, testCase.expectedToTrigger) {
				t.Errorf("%s: incorrect set of presubmits to skip: %s", testCase.name, diff.ObjectReflectDiff(actualToTrigger, testCase.expectedToTrigger))
			}
			if !reflect.DeepEqual(actualToSkip, testCase.expectedToSkip) {
				t.Errorf("%s: incorrect set of presubmits to skip: %s", testCase.name, diff.ObjectReflectDiff(actualToSkip, testCase.expectedToSkip))
			}
		})
	}
}

func TestDetermineSkippedPresubmits(t *testing.T) {
	var testCases = []struct {
		name                      string
		toTrigger, toSkipSuperset []job.Presubmit
		expectedToSkip            []job.Presubmit
	}{
		{
			name:           "no inputs leads to no output",
			toTrigger:      []job.Presubmit{},
			toSkipSuperset: []job.Presubmit{},
			expectedToSkip: nil,
		},
		{
			name:           "no superset of skips to choose from leads to no output",
			toTrigger:      []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}},
			toSkipSuperset: []job.Presubmit{},
			expectedToSkip: nil,
		},
		{
			name:           "disjoint sets of contexts leads to full skip set",
			toTrigger:      []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "bar"}}},
			toSkipSuperset: []job.Presubmit{{Reporter: job.Reporter{Context: "oof"}}, {Reporter: job.Reporter{Context: "rab"}}},
			expectedToSkip: []job.Presubmit{{Reporter: job.Reporter{Context: "oof"}}, {Reporter: job.Reporter{Context: "rab"}}},
		},
		{
			name:           "overlaps on context removes from skip set",
			toTrigger:      []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "bar"}}},
			toSkipSuperset: []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "rab"}}},
			expectedToSkip: []job.Presubmit{{Reporter: job.Reporter{Context: "rab"}}},
		},
		{
			name:           "full set of overlaps on context removes everything from skip set",
			toTrigger:      []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "bar"}}},
			toSkipSuperset: []job.Presubmit{{Reporter: job.Reporter{Context: "foo"}}, {Reporter: job.Reporter{Context: "bar"}}},
			expectedToSkip: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if actual, expected := determineSkippedPresubmits(testCase.toTrigger, testCase.toSkipSuperset, logrus.WithField("test-case", testCase.name)), testCase.expectedToSkip; !reflect.DeepEqual(actual, expected) {
				t.Errorf("%s: incorrect skipped presubmits determined: %v", testCase.name, diff.ObjectReflectDiff(actual, expected))
			}
		})
	}
}

type orgRepoRef struct {
	org, repo, ref string
}

type fakeContextGetter struct {
	status map[orgRepoRef]*scm.CombinedStatus
	errors map[orgRepoRef]error
}

func (f *fakeContextGetter) getContexts(key orgRepoRef) (sets.String, sets.String, error) {
	allContexts := sets.NewString()
	failedContexts := sets.NewString()
	if err, exists := f.errors[key]; exists {
		return failedContexts, allContexts, err
	}
	combinedStatus, exists := f.status[key]
	if !exists {
		return failedContexts, allContexts, fmt.Errorf("failed to find status for %s/%s@%s", key.org, key.repo, key.ref)
	}
	for _, status := range combinedStatus.Statuses {
		allContexts.Insert(status.Label)
		if status.State == scm.StateError || status.State == scm.StateFailure {
			failedContexts.Insert(status.Label)
		}
	}
	return failedContexts, allContexts, nil
}

func TestPresubmitFilter(t *testing.T) {
	statuses := &scm.CombinedStatus{Statuses: []*scm.Status{
		{
			Label: "existing-successful",
			State: scm.StateSuccess,
		},
		{
			Label: "existing-pending",
			State: scm.StatePending,
		},
		{
			Label: "existing-error",
			State: scm.StateError,
		},
		{
			Label: "existing-failure",
			State: scm.StateFailure,
		},
	}}
	var testCases = []struct {
		name                 string
		honorOkToTest        bool
		body, org, repo, ref string
		presubmits           []job.Presubmit
		expected             [][]bool
		statusErr, expectErr bool
	}{
		{
			name: "test all comment selects all tests that don't need an explicit trigger",
			body: "/test all",
			org:  "org",
			repo: "repo",
			ref:  "ref",
			presubmits: []job.Presubmit{
				{
					Base: job.Base{
						Name: "always-runs",
					},
					AlwaysRun: true,
					Reporter: job.Reporter{
						Context: "always-runs",
					},
				},
				{
					Base: job.Base{
						Name: "runs-if-changed",
					},
					Reporter: job.Reporter{
						Context: "runs-if-changed",
					},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "sometimes",
					},
				},
				{
					Base: job.Base{
						Name: "runs-if-triggered",
					},
					Reporter: job.Reporter{
						Context: "runs-if-triggered",
					},
					Trigger:      `(?m)^/test (?:.*? )?trigger(?: .*?)?$`,
					RerunCommand: "/test trigger",
				},
			},
			expected: [][]bool{{true, false, false}, {true, false, false}, {false, false, false}},
		},
		{
			name:          "honored ok-to-test comment selects all tests that don't need an explicit trigger",
			body:          "/ok-to-test",
			honorOkToTest: true,
			org:           "org",
			repo:          "repo",
			ref:           "ref",
			presubmits: []job.Presubmit{
				{
					Base: job.Base{
						Name: "always-runs",
					},
					AlwaysRun: true,
					Reporter: job.Reporter{
						Context: "always-runs",
					},
				},
				{
					Base: job.Base{
						Name: "runs-if-changed",
					},
					Reporter: job.Reporter{
						Context: "runs-if-changed",
					},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "sometimes",
					},
				},
				{
					Base: job.Base{
						Name: "runs-if-triggered",
					},
					Reporter: job.Reporter{
						Context: "runs-if-triggered",
					},
					Trigger:      `(?m)^/test (?:.*? )?trigger(?: .*?)?$`,
					RerunCommand: "/test trigger",
				},
			},
			expected: [][]bool{{true, false, false}, {true, false, false}, {false, false, false}},
		},
		{
			name:          "not honored ok-to-test comment selects no tests",
			body:          "/ok-to-test",
			honorOkToTest: false,
			org:           "org",
			repo:          "repo",
			ref:           "ref",
			presubmits: []job.Presubmit{
				{
					Base: job.Base{
						Name: "always-runs",
					},
					AlwaysRun: true,
					Reporter: job.Reporter{
						Context: "always-runs",
					},
				},
				{
					Base: job.Base{
						Name: "runs-if-changed",
					},
					Reporter: job.Reporter{
						Context: "runs-if-changed",
					},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "sometimes",
					},
				},
				{
					Base: job.Base{
						Name: "runs-if-triggered",
					},
					Reporter: job.Reporter{
						Context: "runs-if-triggered",
					},
					Trigger:      `(?m)^/test (?:.*? )?trigger(?: .*?)?$`,
					RerunCommand: "/test trigger",
				},
			},
			expected: [][]bool{{false, false, false}, {false, false, false}, {false, false, false}},
		},
		{
			name:       "statuses are not gathered unless retest is specified (will error but we should not see it)",
			body:       "not a command",
			org:        "org",
			repo:       "repo",
			ref:        "ref",
			presubmits: []job.Presubmit{},
			expected:   [][]bool{},
			statusErr:  true,
			expectErr:  false,
		},
		{
			name:       "statuses are gathered when retest is specified and gather error is propagated",
			body:       "/retest",
			org:        "org",
			repo:       "repo",
			ref:        "ref",
			presubmits: []job.Presubmit{},
			expected:   [][]bool{},
			statusErr:  true,
			expectErr:  true,
		},
		{
			name: "retest command selects for errored or failed contexts and required but missing contexts",
			body: "/retest",
			org:  "org",
			repo: "repo",
			ref:  "ref",
			presubmits: []job.Presubmit{
				{
					Base: job.Base{
						Name: "successful-job",
					},
					Reporter: job.Reporter{
						Context: "existing-successful",
					},
				},
				{
					Base: job.Base{
						Name: "pending-job",
					},
					Reporter: job.Reporter{
						Context: "existing-pending",
					},
				},
				{
					Base: job.Base{
						Name: "failure-job",
					},
					Reporter: job.Reporter{
						Context: "existing-failure",
					},
				},
				{
					Base: job.Base{
						Name: "error-job",
					},
					Reporter: job.Reporter{
						Context: "existing-error",
					},
				},
				{
					Base: job.Base{
						Name: "missing-always-runs",
					},
					Reporter: job.Reporter{
						Context: "missing-always-runs",
					},
					AlwaysRun: true,
				},
			},
			expected: [][]bool{{false, false, false}, {false, false, false}, {true, false, true}, {true, false, true}, {true, false, true}},
		},
		{
			name: "explicit test command filters for jobs that match",
			body: "/test trigger",
			org:  "org",
			repo: "repo",
			ref:  "ref",
			presubmits: []job.Presubmit{
				{
					Base: job.Base{
						Name: "always-runs",
					},
					AlwaysRun: true,
					Reporter: job.Reporter{
						Context: "always-runs",
					},
					Trigger:      `(?m)^/test (?:.*? )?trigger(?: .*?)?$`,
					RerunCommand: "/test trigger",
				},
				{
					Base: job.Base{
						Name: "runs-if-changed",
					},
					Reporter: job.Reporter{
						Context: "runs-if-changed",
					},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "sometimes",
					},
					Trigger:      `(?m)^/test (?:.*? )?trigger(?: .*?)?$`,
					RerunCommand: "/test trigger",
				},
				{
					Base: job.Base{
						Name: "runs-if-triggered",
					},
					Reporter: job.Reporter{
						Context: "runs-if-triggered",
					},
					Trigger:      `(?m)^/test (?:.*? )?trigger(?: .*?)?$`,
					RerunCommand: "/test trigger",
				},
				{
					Base: job.Base{
						Name: "always-runs",
					},
					AlwaysRun: true,
					Reporter: job.Reporter{
						Context: "always-runs",
					},
					Trigger:      `(?m)^/test (?:.*? )?other-trigger(?: .*?)?$`,
					RerunCommand: "/test other-trigger",
				},
				{
					Base: job.Base{
						Name: "runs-if-changed",
					},
					Reporter: job.Reporter{
						Context: "runs-if-changed",
					},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "sometimes",
					},
					Trigger:      `(?m)^/test (?:.*? )?other-trigger(?: .*?)?$`,
					RerunCommand: "/test other-trigger",
				},
				{
					Base: job.Base{
						Name: "runs-if-triggered",
					},
					Reporter: job.Reporter{
						Context: "runs-if-triggered",
					},
					Trigger:      `(?m)^/test (?:.*? )?other-trigger(?: .*?)?$`,
					RerunCommand: "/test other-trigger",
				},
			},
			expected: [][]bool{{true, true, true}, {true, true, true}, {true, true, true}, {false, false, false}, {false, false, false}, {false, false, false}},
		},
		{
			name: "comments matching more than one case will select the union of presubmits",
			body: `/test trigger
/test all
/retest`,
			org:  "org",
			repo: "repo",
			ref:  "ref",
			presubmits: []job.Presubmit{
				{
					Base: job.Base{
						Name: "always-runs",
					},
					AlwaysRun: true,
					Reporter: job.Reporter{
						Context: "existing-successful",
					},
					Trigger:      `(?m)^/test (?:.*? )?other-trigger(?: .*?)?$`,
					RerunCommand: "/test other-trigger",
				},
				{
					Base: job.Base{
						Name: "runs-if-changed",
					},
					Reporter: job.Reporter{
						Context: "existing-successful",
					},
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "sometimes",
					},
					Trigger:      `(?m)^/test (?:.*? )?other-trigger(?: .*?)?$`,
					RerunCommand: "/test other-trigger",
				},
				{
					Base: job.Base{
						Name: "runs-if-triggered",
					},
					Reporter: job.Reporter{
						Context: "runs-if-triggered",
					},
					Trigger:      `(?m)^/test (?:.*? )?trigger(?: .*?)?$`,
					RerunCommand: "/test trigger",
				},
				{
					Base: job.Base{
						Name: "successful-job",
					},
					Reporter: job.Reporter{
						Context: "existing-successful",
					},
				},
				{
					Base: job.Base{
						Name: "pending-job",
					},
					Reporter: job.Reporter{
						Context: "existing-pending",
					},
				},
				{
					Base: job.Base{
						Name: "failure-job",
					},
					Reporter: job.Reporter{
						Context: "existing-failure",
					},
				},
				{
					Base: job.Base{
						Name: "error-job",
					},
					Reporter: job.Reporter{
						Context: "existing-error",
					},
				},
				{
					Base: job.Base{
						Name: "missing-always-runs",
					},
					AlwaysRun: true,
					Reporter: job.Reporter{
						Context: "missing-always-runs",
					},
				},
			},
			expected: [][]bool{{true, false, false}, {true, false, false}, {true, true, true}, {false, false, false}, {false, false, false}, {true, false, true}, {true, false, true}, {true, false, true}},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if len(testCase.presubmits) != len(testCase.expected) {
				t.Fatalf("%s: have %d presubmits but only %d expected filter outputs", testCase.name, len(testCase.presubmits), len(testCase.expected))
			}
			for i := range testCase.presubmits {
				if err := testCase.presubmits[i].SetRegexes(); err != nil {
					t.Fatalf("%s: could not set presubmit regexes: %v", testCase.name, err)
				}
			}
			fsg := &fakeContextGetter{
				errors: map[orgRepoRef]error{},
				status: map[orgRepoRef]*scm.CombinedStatus{},
			}
			key := orgRepoRef{org: testCase.org, repo: testCase.repo, ref: testCase.ref}
			if testCase.statusErr {
				fsg.errors[key] = errors.New("failure")
			} else {
				fsg.status[key] = statuses
			}

			fakeContextGetter := func() (sets.String, sets.String, error) {

				return fsg.getContexts(key)
			}

			filter, err := PresubmitFilter(testCase.honorOkToTest, fakeContextGetter, testCase.body, logrus.WithField("test-case", testCase.name))

			if testCase.expectErr && err == nil {
				t.Errorf("%s: expected an error creating the filter, but got none", testCase.name)
			}
			if !testCase.expectErr && err != nil {
				t.Errorf("%s: expected no error creating the filter, but got one: %v", testCase.name, err)
			}
			for i, presubmit := range testCase.presubmits {
				actualFiltered, actualForced, actualDefault := filter(presubmit)
				expectedFiltered, expectedForced, expectedDefault := testCase.expected[i][0], testCase.expected[i][1], testCase.expected[i][2]
				if actualFiltered != expectedFiltered {
					t.Errorf("%s: filter did not evaluate correctly, expected %v but got %v for %v", testCase.name, expectedFiltered, actualFiltered, presubmit.Name)
				}
				if actualForced != expectedForced {
					t.Errorf("%s: filter did not determine forced correctly, expected %v but got %v for %v", testCase.name, expectedForced, actualForced, presubmit.Name)
				}
				if actualDefault != expectedDefault {
					t.Errorf("%s: filter did not determine default correctly, expected %v but got %v for %v", testCase.name, expectedDefault, actualDefault, presubmit.Name)
				}
			}
		})
	}
}
