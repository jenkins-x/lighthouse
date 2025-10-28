//go:build unit
// +build unit

package keeper_test

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/config/keeper"
	"github.com/stretchr/testify/assert"
)

var testCases = []struct {
	description string
	orgs        []string
	repos       []string
	valid       bool
}{
	{
		description: "Valid org and repo names",
		orgs:        []string{"gitlab-org", "github", "jenkins-x"},
		repos:       []string{"jx-pipeline", "jx-secret"},
		valid:       true,
	},
	{
		description: "Invalid org names",
		orgs:        []string{"gitlab-org/foo", "github", "jenkins-x"},
		repos:       []string{"jx-pipeline", "jx-secret"},
		valid:       false,
	},
	{
		description: "Invalid org - No orgs",
		orgs:        []string{},
		repos:       []string{"jx-pipeline", "jx-secret"},
		valid:       false,
	},
	{
		description: "Invalid org - Duplicate orgs",
		orgs:        []string{"gitlab-org", "gitlab-org"},
		repos:       []string{"jx-pipeline", "jx-secret"},
		valid:       false,
	},
	{
		description: "Valid repos - nested",
		orgs:        []string{"gitlab-org", "github"},
		repos:       []string{"foo/jx-pipeline", "foo/bar/jx-secret"},
		valid:       true,
	},
}

func TestValidate(t *testing.T) {
	for tt := range testCases {
		t.Log(testCases[tt].description)
		tq := new(keeper.Query)
		tq.Orgs = testCases[tt].orgs
		err := tq.Validate()
		if testCases[tt].valid {
			assert.NoError(t, err)
		} else {
			assert.Error(t, err)
		}
	}
}

func TestQuery_BucketedQueries(t *testing.T) {
	testCases := []struct {
		name            string
		query           keeper.Query
		bucketSize      int
		expectedQueries []string
	}{
		{
			name: "Single Query - Full Bucket",
			query: keeper.Query{
				Orgs:                   []string{"org"},
				Repos:                  []string{"org/repo1", "org/repo2"},
				ExcludedRepos:          []string{"org/repo0"},
				ExcludedBranches:       []string{"develop"},
				IncludedBranches:       []string{"release"},
				Labels:                 []string{"enhancement"},
				MissingLabels:          []string{"wip"},
				Milestone:              "v1.0",
				ReviewApprovedRequired: true,
			},
			bucketSize: 2,
			expectedQueries: []string{
				"is:pr state:open org:\"org\" repo:\"org/repo1\" repo:\"org/repo2\" -repo:\"org/repo0\" -base:\"develop\" base:\"release\" label:\"enhancement\" -label:\"wip\" milestone:\"v1.0\" review:approved",
			},
		},
		{
			name: "Two Queries - Full Bucket",
			query: keeper.Query{
				Orgs:                   []string{"org"},
				Repos:                  []string{"org/repo1", "org/repo2", "org/repo3", "org/repo4"},
				ExcludedRepos:          []string{"org/repo0"},
				ExcludedBranches:       []string{"develop"},
				IncludedBranches:       []string{"release"},
				Labels:                 []string{"enhancement"},
				MissingLabels:          []string{"wip"},
				Milestone:              "v1.0",
				ReviewApprovedRequired: true,
			},
			bucketSize: 2,
			expectedQueries: []string{
				"is:pr state:open org:\"org\" repo:\"org/repo1\" repo:\"org/repo2\" -repo:\"org/repo0\" -base:\"develop\" base:\"release\" label:\"enhancement\" -label:\"wip\" milestone:\"v1.0\" review:approved",
				"is:pr state:open org:\"org\" repo:\"org/repo3\" repo:\"org/repo4\" -repo:\"org/repo0\" -base:\"develop\" base:\"release\" label:\"enhancement\" -label:\"wip\" milestone:\"v1.0\" review:approved",
			},
		},
		{
			name: "Three Queries - Partial Bucket",
			query: keeper.Query{
				Orgs:                   []string{"org"},
				Repos:                  []string{"org/repo1", "org/repo2", "org/repo3", "org/repo4", "org/repo5"},
				ExcludedRepos:          []string{"org/repo0"},
				ExcludedBranches:       []string{"develop"},
				IncludedBranches:       []string{"release"},
				Labels:                 []string{"enhancement"},
				MissingLabels:          []string{"wip"},
				Milestone:              "v1.0",
				ReviewApprovedRequired: true,
			},
			bucketSize: 2,
			expectedQueries: []string{
				"is:pr state:open org:\"org\" repo:\"org/repo1\" repo:\"org/repo2\" -repo:\"org/repo0\" -base:\"develop\" base:\"release\" label:\"enhancement\" -label:\"wip\" milestone:\"v1.0\" review:approved",
				"is:pr state:open org:\"org\" repo:\"org/repo3\" repo:\"org/repo4\" -repo:\"org/repo0\" -base:\"develop\" base:\"release\" label:\"enhancement\" -label:\"wip\" milestone:\"v1.0\" review:approved",
				"is:pr state:open org:\"org\" repo:\"org/repo5\" -repo:\"org/repo0\" -base:\"develop\" base:\"release\" label:\"enhancement\" -label:\"wip\" milestone:\"v1.0\" review:approved",
			},
		},
		{
			name: "No Repos",
			query: keeper.Query{
				Orgs:                   []string{"org"},
				ExcludedRepos:          []string{"org/repo0"},
				ExcludedBranches:       []string{"develop"},
				IncludedBranches:       []string{"release"},
				Labels:                 []string{"enhancement"},
				MissingLabels:          []string{"wip"},
				Milestone:              "v1.0",
				ReviewApprovedRequired: true,
			},
			bucketSize: 2,
			expectedQueries: []string{
				"is:pr state:open org:\"org\" -repo:\"org/repo0\" -base:\"develop\" base:\"release\" label:\"enhancement\" -label:\"wip\" milestone:\"v1.0\" review:approved",
			},
		},
		{
			name: "Bucket Size == 0",
			query: keeper.Query{
				Orgs:                   []string{"org"},
				Repos:                  []string{"org/repo1", "org/repo2"},
				ExcludedRepos:          []string{"org/repo0"},
				ExcludedBranches:       []string{"develop"},
				IncludedBranches:       []string{"release"},
				Labels:                 []string{"enhancement"},
				MissingLabels:          []string{"wip"},
				Milestone:              "v1.0",
				ReviewApprovedRequired: true,
			},
			bucketSize: 0,
			expectedQueries: []string{
				"is:pr state:open org:\"org\" repo:\"org/repo1\" repo:\"org/repo2\" -repo:\"org/repo0\" -base:\"develop\" base:\"release\" label:\"enhancement\" -label:\"wip\" milestone:\"v1.0\" review:approved",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actualQueries := tc.query.BucketedQueries(tc.bucketSize)
			assert.Equal(t, tc.expectedQueries, actualQueries)
		})
	}
}
