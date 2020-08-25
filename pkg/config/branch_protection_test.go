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

package config

import (
	"reflect"
	"sort"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/config/branchprotection"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"k8s.io/apimachinery/pkg/util/diff"
)

var (
	y   = true
	n   = false
	yes = &y
	no  = &n
)

func normalize(policy *branchprotection.Policy) {
	if policy == nil || policy.RequiredStatusChecks == nil {
		return
	}
	sort.Strings(policy.RequiredStatusChecks.Contexts)
	sort.Strings(policy.Exclude)
}

func TestBranchRequirements(t *testing.T) {
	cases := []struct {
		name                            string
		config                          []job.Presubmit
		masterExpected, otherExpected   []string
		masterOptional, otherOptional   []string
		masterIfPresent, otherIfPresent []string
	}{
		{
			name: "basic",
			config: []job.Presubmit{
				{
					AlwaysRun: true,
					Reporter: job.Reporter{
						Context:    "always-run",
						SkipReport: false,
					},
				},
				{
					RegexpChangeMatcher: job.RegexpChangeMatcher{
						RunIfChanged: "foo",
					},
					AlwaysRun: false,
					Reporter: job.Reporter{
						Context:    "run-if-changed",
						SkipReport: false,
					},
				},
				{
					AlwaysRun: false,
					Reporter: job.Reporter{
						Context:    "not-always",
						SkipReport: false,
					},
				},
				{
					AlwaysRun: true,
					Reporter: job.Reporter{
						Context:    "skip-report",
						SkipReport: true,
					},
					Brancher: job.Brancher{
						SkipBranches: []string{"master"},
					},
				},
				{
					AlwaysRun: true,
					Reporter: job.Reporter{
						Context:    "optional",
						SkipReport: false,
					},
					Optional: true,
				},
			},
			masterExpected:  []string{"always-run"},
			masterIfPresent: []string{"run-if-changed", "not-always"},
			masterOptional:  []string{"optional"},
			otherExpected:   []string{"always-run"},
			otherIfPresent:  []string{"run-if-changed", "not-always"},
			otherOptional:   []string{"skip-report", "optional"},
		},
	}

	for _, tc := range cases {
		for i := range tc.config {
			if err := tc.config[i].SetRegexes(); err != nil {
				t.Fatalf("could not set regexes: %v", err)
			}
		}
		presubmits := map[string][]job.Presubmit{
			"o/r": tc.config,
		}
		masterActual, masterActualIfPresent, masterOptional := BranchRequirements("o", "r", "master", presubmits)
		if !reflect.DeepEqual(masterActual, tc.masterExpected) {
			t.Errorf("%s: identified incorrect required contexts on branch master: %s", tc.name, diff.ObjectReflectDiff(masterActual, tc.masterExpected))
		}
		if !reflect.DeepEqual(masterOptional, tc.masterOptional) {
			t.Errorf("%s: identified incorrect optional contexts on branch master: %s", tc.name, diff.ObjectReflectDiff(masterOptional, tc.masterOptional))
		}
		if !reflect.DeepEqual(masterActualIfPresent, tc.masterIfPresent) {
			t.Errorf("%s: identified incorrect if-present contexts on branch master: %s", tc.name, diff.ObjectReflectDiff(masterActualIfPresent, tc.masterIfPresent))
		}
		otherActual, otherActualIfPresent, otherOptional := BranchRequirements("o", "r", "other", presubmits)
		if !reflect.DeepEqual(masterActual, tc.masterExpected) {
			t.Errorf("%s: identified incorrect required contexts on branch other: : %s", tc.name, diff.ObjectReflectDiff(otherActual, tc.otherExpected))
		}
		if !reflect.DeepEqual(otherOptional, tc.otherOptional) {
			t.Errorf("%s: identified incorrect optional contexts on branch other: %s", tc.name, diff.ObjectReflectDiff(otherOptional, tc.otherOptional))
		}
		if !reflect.DeepEqual(otherActualIfPresent, tc.otherIfPresent) {
			t.Errorf("%s: identified incorrect if-present contexts on branch other: %s", tc.name, diff.ObjectReflectDiff(otherActualIfPresent, tc.otherIfPresent))
		}
	}
}

func TestConfig_GetBranchProtection(t *testing.T) {
	testCases := []struct {
		name     string
		config   Config
		err      bool
		expected *branchprotection.Policy
	}{
		{
			name: "unprotected by default",
		},
		{
			name: "undefined org not protected",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Policy: branchprotection.Policy{
							Protect: yes,
						},
						Orgs: map[string]branchprotection.Org{
							"unknown": {},
						},
					},
				},
			},
		},
		{
			name: "protect via config default",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Policy: branchprotection.Policy{
							Protect: yes,
						},
						Orgs: map[string]branchprotection.Org{
							"org": {},
						},
					},
				},
			},
			expected: &branchprotection.Policy{Protect: yes},
		},
		{
			name: "protect via org default",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Orgs: map[string]branchprotection.Org{
							"org": {
								Policy: branchprotection.Policy{
									Protect: yes,
								},
							},
						},
					},
				},
			},
			expected: &branchprotection.Policy{Protect: yes},
		},
		{
			name: "protect via repo default",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Orgs: map[string]branchprotection.Org{
							"org": {
								Repos: map[string]branchprotection.Repo{
									"repo": {
										Policy: branchprotection.Policy{
											Protect: yes,
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &branchprotection.Policy{Protect: yes},
		},
		{
			name: "protect specific branch",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Orgs: map[string]branchprotection.Org{
							"org": {
								Repos: map[string]branchprotection.Repo{
									"repo": {
										Branches: map[string]branchprotection.Branch{
											"branch": {
												Policy: branchprotection.Policy{
													Protect: yes,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: &branchprotection.Policy{Protect: yes},
		},
		{
			name: "ignore other org settings",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Policy: branchprotection.Policy{
							Protect: no,
						},
						Orgs: map[string]branchprotection.Org{
							"other": {
								Policy: branchprotection.Policy{Protect: yes},
							},
							"org": {},
						},
					},
				},
			},
			expected: &branchprotection.Policy{Protect: no},
		},
		{
			name: "defined branches must make a protection decision",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Orgs: map[string]branchprotection.Org{
							"org": {
								Repos: map[string]branchprotection.Repo{
									"repo": {
										Branches: map[string]branchprotection.Branch{
											"branch": {},
										},
									},
								},
							},
						},
					},
				},
			},
			err: true,
		},
		{
			name: "pushers require protection",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Policy: branchprotection.Policy{
							Protect: no,
							Restrictions: &branchprotection.Restrictions{
								Teams: []string{"oncall"},
							},
						},
						Orgs: map[string]branchprotection.Org{
							"org": {},
						},
					},
				},
			},
			err: true,
		},
		{
			name: "required contexts require protection",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Policy: branchprotection.Policy{
							Protect: no,
							RequiredStatusChecks: &branchprotection.ContextPolicy{
								Contexts: []string{"test-foo"},
							},
						},
						Orgs: map[string]branchprotection.Org{
							"org": {},
						},
					},
				},
			},
			err: true,
		},
		{
			name: "child policy with defined parent can disable protection",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						AllowDisabledPolicies: true,
						Policy: branchprotection.Policy{
							Protect: yes,
							Restrictions: &branchprotection.Restrictions{
								Teams: []string{"oncall"},
							},
						},
						Orgs: map[string]branchprotection.Org{
							"org": {
								Policy: branchprotection.Policy{
									Protect: no,
								},
							},
						},
					},
				},
			},
			expected: &branchprotection.Policy{
				Protect: no,
			},
		},
		{
			name: "Make required presubmits required",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Policy: branchprotection.Policy{
							Protect: yes,
							RequiredStatusChecks: &branchprotection.ContextPolicy{
								Contexts: []string{"cla"},
							},
						},
						Orgs: map[string]branchprotection.Org{
							"org": {},
						},
					},
				},
				JobConfig: job.Config{
					Presubmits: map[string][]job.Presubmit{
						"org/repo": {
							{
								Base: job.Base{
									Name: "required presubmit",
								},
								Reporter: job.Reporter{
									Context: "required presubmit",
								},
								AlwaysRun: true,
							},
						},
					},
				},
			},
			expected: &branchprotection.Policy{
				Protect: yes,
				RequiredStatusChecks: &branchprotection.ContextPolicy{
					Contexts: []string{"required presubmit", "cla"},
				},
			},
		},
		{
			name: "ProtectTested opts into protection",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						ProtectTested: true,
						Orgs: map[string]branchprotection.Org{
							"org": {},
						},
					},
				},
				JobConfig: job.Config{
					Presubmits: map[string][]job.Presubmit{
						"org/repo": {
							{
								Base: job.Base{
									Name: "required presubmit",
								},
								Reporter: job.Reporter{
									Context: "required presubmit",
								},
								AlwaysRun: true,
							},
						},
					},
				},
			},
			expected: &branchprotection.Policy{
				Protect: yes,
				RequiredStatusChecks: &branchprotection.ContextPolicy{
					Contexts: []string{"required presubmit"},
				},
			},
		},
		{
			name: "required presubmits require protection",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						Policy: branchprotection.Policy{
							Protect: no,
						},
						Orgs: map[string]branchprotection.Org{
							"org": {},
						},
					},
				},
				JobConfig: job.Config{
					Presubmits: map[string][]job.Presubmit{
						"org/repo": {
							{
								Base: job.Base{
									Name: "required presubmit",
								},
								Reporter: job.Reporter{
									Context: "required presubmit",
								},
								AlwaysRun: true,
							},
						},
					},
				},
			},
			err: true,
		},
		{
			name: "Optional presubmits do not force protection",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						ProtectTested: true,
						Orgs: map[string]branchprotection.Org{
							"org": {},
						},
					},
				},
				JobConfig: job.Config{
					Presubmits: map[string][]job.Presubmit{
						"org/repo": {
							{
								Base: job.Base{
									Name: "optional presubmit",
								},
								Reporter: job.Reporter{
									Context: "optional presubmit",
								},
								AlwaysRun: true,
								Optional:  true,
							},
						},
					},
				},
			},
		},
		{
			name: "Explicit configuration takes precedence over ProtectTested",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						ProtectTested: true,
						Orgs: map[string]branchprotection.Org{
							"org": {
								Policy: branchprotection.Policy{
									Protect: yes,
								},
							},
						},
					},
				},
				JobConfig: job.Config{
					Presubmits: map[string][]job.Presubmit{
						"org/repo": {
							{
								Base: job.Base{
									Name: "optional presubmit",
								},
								Reporter: job.Reporter{
									Context: "optional presubmit",
								},
								AlwaysRun: true,
								Optional:  true,
							},
						},
					},
				},
			},
			expected: &branchprotection.Policy{Protect: yes},
		},
		{
			name: "Explicit non-configuration takes precedence over ProtectTested",
			config: Config{
				ProwConfig: ProwConfig{
					BranchProtection: branchprotection.Config{
						AllowDisabledJobPolicies: true,
						ProtectTested:            true,
						Orgs: map[string]branchprotection.Org{
							"org": {
								Repos: map[string]branchprotection.Repo{
									"repo": {
										Policy: branchprotection.Policy{
											Protect: no,
										},
									},
								},
							},
						},
					},
				},
				JobConfig: job.Config{
					Presubmits: map[string][]job.Presubmit{
						"org/repo": {
							{
								Base: job.Base{
									Name: "required presubmit",
								},
								Reporter: job.Reporter{
									Context: "required presubmit",
								},
								AlwaysRun: true,
							},
						},
					},
				},
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.config.GetBranchProtection("org", "repo", "branch")
			if err != nil {
				if !tc.err {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if tc.err {
					t.Errorf("failed to receive an error")
				}
			}

			normalize(actual)
			normalize(tc.expected)
			if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("actual %+v != expected %+v", actual, tc.expected)
			}
		})
	}
}
