package githubapp

import (
	"strings"

	"github.com/jenkins-x/lighthouse/pkg/config/keeper"
	"github.com/sirupsen/logrus"
)

// OwnerQueries separates keeper queries by the owner
type OwnerQueries struct {
	Owner   string
	Queries keeper.Queries
}

// SplitKeeperQueries splits the keeper queries into a sequence of owner queries
func SplitKeeperQueries(queries keeper.Queries) map[string]keeper.Queries {
	answer := map[string]keeper.Queries{}
	for _, q1 := range queries {
		for org, repos := range SplitRepositories(q1.Repos) {
			q := q1
			q.Repos = repos
			answer[org] = append(answer[org], q)
		}
	}
	return answer
}

// SplitRepositories splits the list of repositories into a map indexed by owner
func SplitRepositories(repos []string) map[string][]string {
	answer := map[string][]string{}

	for _, r := range repos {
		paths := strings.SplitN(r, "/", 2)
		if len(paths) < 2 {
			logrus.Warnf("ignoring invalid repo without an owner: %s", r)
			continue
		}
		owner := paths[0]
		answer[owner] = append(answer[owner], r)
	}
	return answer
}
