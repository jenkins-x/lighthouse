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
