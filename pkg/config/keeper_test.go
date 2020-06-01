package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigGetKeeperContextPolicy(t *testing.T) {
	cases := []struct {
		name                 string
		bpOrgs               map[string]Org
		presubmits           []Presubmit
		skipUnknownContexts  bool
		fromBranchProtection bool

		expectedRequired  []string
		expectedOptional  []string
		expectedIfPresent []string
	}{
		{
			name: "basic",
			presubmits: []Presubmit{
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "always-run",
						SkipReport: false,
					},
				},
				{
					RegexpChangeMatcher: RegexpChangeMatcher{
						RunIfChanged: "foo",
					},
					AlwaysRun: false,
					Reporter: Reporter{
						Context:    "run-if-changed",
						SkipReport: false,
					},
				},
				{
					AlwaysRun: false,
					Reporter: Reporter{
						Context:    "not-always",
						SkipReport: false,
					},
				},
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "skip-report",
						SkipReport: true,
					},
					Brancher: Brancher{
						SkipBranches: []string{"master"},
					},
				},
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "optional",
						SkipReport: false,
					},
					Optional: true,
				},
			},
			expectedRequired:  []string{"always-run"},
			expectedIfPresent: []string{"run-if-changed", "not-always"},
			expectedOptional:  []string{"optional"},
		},
		{
			name: "from branch protection",
			presubmits: []Presubmit{
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "always-run",
						SkipReport: false,
					},
				},
				{
					RegexpChangeMatcher: RegexpChangeMatcher{
						RunIfChanged: "foo",
					},
					AlwaysRun: false,
					Reporter: Reporter{
						Context:    "run-if-changed",
						SkipReport: false,
					},
				},
				{
					AlwaysRun: false,
					Reporter: Reporter{
						Context:    "not-always",
						SkipReport: false,
					},
				},
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "skip-report",
						SkipReport: true,
					},
					Brancher: Brancher{
						SkipBranches: []string{"master"},
					},
				},
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "optional",
						SkipReport: false,
					},
					Optional: true,
				},
			},
			fromBranchProtection: true,
			bpOrgs: map[string]Org{
				"o": {
					Policy: Policy{},
					Repos: map[string]Repo{
						"r": {
							Policy: Policy{
								RequiredStatusChecks: &ContextPolicy{
									Contexts: []string{
										"always-run",
										"run-if-changed",
									},
								},
							},
						},
					},
				},
			},
			expectedRequired:  []string{"always-run"},
			expectedIfPresent: []string{"run-if-changed", "not-always"},
			expectedOptional:  []string{"optional"},
		},
		{
			name: "from branch protection with unknown context",
			presubmits: []Presubmit{
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "always-run",
						SkipReport: false,
					},
				},
				{
					RegexpChangeMatcher: RegexpChangeMatcher{
						RunIfChanged: "foo",
					},
					AlwaysRun: false,
					Reporter: Reporter{
						Context:    "run-if-changed",
						SkipReport: false,
					},
				},
				{
					AlwaysRun: false,
					Reporter: Reporter{
						Context:    "not-always",
						SkipReport: false,
					},
				},
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "skip-report",
						SkipReport: true,
					},
					Brancher: Brancher{
						SkipBranches: []string{"master"},
					},
				},
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "optional",
						SkipReport: false,
					},
					Optional: true,
				},
			},
			fromBranchProtection: true,
			bpOrgs: map[string]Org{
				"o": {
					Policy: Policy{},
					Repos: map[string]Repo{
						"r": {
							Policy: Policy{
								RequiredStatusChecks: &ContextPolicy{
									Contexts: []string{
										"always-run",
										"run-if-changed",
										"non-lighthouse-job",
									},
								},
							},
						},
					},
				},
			},
			expectedRequired:  []string{"always-run", "non-lighthouse-job"},
			expectedIfPresent: []string{"run-if-changed", "not-always"},
			expectedOptional:  []string{"optional"},
		},
		{
			name: "from branch protection skipping unknown context",
			presubmits: []Presubmit{
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "always-run",
						SkipReport: false,
					},
				},
				{
					RegexpChangeMatcher: RegexpChangeMatcher{
						RunIfChanged: "foo",
					},
					AlwaysRun: false,
					Reporter: Reporter{
						Context:    "run-if-changed",
						SkipReport: false,
					},
				},
				{
					AlwaysRun: false,
					Reporter: Reporter{
						Context:    "not-always",
						SkipReport: false,
					},
				},
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "skip-report",
						SkipReport: true,
					},
					Brancher: Brancher{
						SkipBranches: []string{"master"},
					},
				},
				{
					AlwaysRun: true,
					Reporter: Reporter{
						Context:    "optional",
						SkipReport: false,
					},
					Optional: true,
				},
			},
			fromBranchProtection: true,
			bpOrgs: map[string]Org{
				"o": {
					Policy: Policy{},
					Repos: map[string]Repo{
						"r": {
							Policy: Policy{
								RequiredStatusChecks: &ContextPolicy{
									Contexts: []string{
										"always-run",
										"run-if-changed",
										"non-lighthouse-job",
									},
								},
							},
						},
					},
				},
			},
			skipUnknownContexts: true,
			expectedRequired:    []string{"always-run", "non-lighthouse-job"},
			expectedIfPresent:   []string{"run-if-changed", "not-always"},
			expectedOptional:    []string{"optional"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			if err := SetPresubmitRegexes(tc.presubmits); err != nil {
				t.Fatalf("could not set regexes: %v", err)
			}
			presubmits := map[string][]Presubmit{
				"o/r": tc.presubmits,
			}
			cfg := Config{
				JobConfig: JobConfig{
					Presubmits: presubmits,
				},
				ProwConfig: ProwConfig{
					Keeper: Keeper{
						ContextOptions: KeeperContextPolicyOptions{
							KeeperContextPolicy: KeeperContextPolicy{
								SkipUnknownContexts:  &tc.skipUnknownContexts,
								FromBranchProtection: &tc.fromBranchProtection,
							},
						},
					},
				},
			}
			if tc.bpOrgs != nil {
				cfg.ProwConfig.BranchProtection = BranchProtection{
					ProtectTested: true,
					Orgs:          tc.bpOrgs,
				}
			}
			ctxPolicy, err := cfg.GetKeeperContextPolicy("o", "r", "master")
			assert.NoError(t, err)
			assert.NotNil(t, ctxPolicy)

			assert.ElementsMatch(t, tc.expectedRequired, ctxPolicy.RequiredContexts)
			assert.ElementsMatch(t, tc.expectedIfPresent, ctxPolicy.RequiredIfPresentContexts)
			assert.ElementsMatch(t, tc.expectedOptional, ctxPolicy.OptionalContexts)
		})
	}
}
