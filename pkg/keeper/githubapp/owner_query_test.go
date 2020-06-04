package githubapp_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/lighthouse-config/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/keeper/githubapp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitKeeperQueries(t *testing.T) {
	prowConfig := filepath.Join("test_data", "config.yaml")

	cfg, err := config.Load(prowConfig, "")
	require.NoError(t, err, "could not load file %s", prowConfig)

	results := githubapp.SplitKeeperQueries(cfg.Keeper.Queries)
	require.Equal(t, 2, len(results), "wrong number of OwnerQueries for file %s", prowConfig)
	assertOwnerQueries(t, results["jstrachan"], "jstrachan", 2, 3, "for file %s", prowConfig)
	assertOwnerQueries(t, results["rawlingsj"], "rawlingj", 2, 1, " for file %s", prowConfig)
}

func assertOwnerQueries(t *testing.T, ownerQueries config.KeeperQueries, owner string, expectedQueryCount int, expectedRepoCount int, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	require.NotNil(t, ownerQueries, "ownerQueries should not be nil for owner %s %s", owner, message)
	assert.Equal(t, len(ownerQueries), expectedQueryCount, "query count for owner %s %s", owner, message)

	for i, q := range ownerQueries {
		t.Logf("owner %s query %d has repos %s %s", owner, i, strings.Join(q.Repos, ", "), message)
	}

	assert.Equal(t, len(ownerQueries[0].Repos), expectedRepoCount, "repo count for owner %s query 0 %s", owner, message)

}

func TestSplitRepositories(t *testing.T) {
	repos := []string{"jstrachan/a", "rawlingsj/a", "jstrachan/b", "jstrachan/c"}

	m := githubapp.SplitRepositories(repos)
	require.Equal(t, len(m), 2, "should have 2 organisations")

	assert.Equal(t, m["jstrachan"], []string{"jstrachan/a", "jstrachan/b", "jstrachan/c"}, "invalid repos for jstrachan")
	assert.Equal(t, m["rawlingsj"], []string{"rawlingsj/a"}, "invalid repos for rawlingsj")
}
