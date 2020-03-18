package githubapp

import (
	"strings"

	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/sirupsen/logrus"
)

// OwnerQueries separates keeper queries by the owner
type OwnerQueries struct {
	Owner   string
	Queries config.KeeperQueries
}

// SplitKeeperQueries splits the keeper queries into a sequence of owner queries
func SplitKeeperQueries(queries config.KeeperQueries) map[string]config.KeeperQueries {
	answer := map[string]config.KeeperQueries{}
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
