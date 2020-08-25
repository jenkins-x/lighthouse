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

package keeper

import (
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

// Queries is a Query slice.
type Queries []Query

func reposInOrg(org string, repos []string) []string {
	prefix := org + "/"
	var res []string
	for _, repo := range repos {
		if strings.HasPrefix(repo, prefix) {
			res = append(res, repo)
		}
	}
	return res
}

// OrgExceptionsAndRepos determines which orgs and repos a set of queries cover.
// Output is returned as a mapping from 'included org'->'repos excluded in the org'
// and a set of included repos.
func (tqs Queries) OrgExceptionsAndRepos() (map[string]sets.String, sets.String) {
	orgs := make(map[string]sets.String)
	for i := range tqs {
		for _, org := range tqs[i].Orgs {
			applicableRepos := sets.NewString(reposInOrg(org, tqs[i].ExcludedRepos)...)
			if excepts, ok := orgs[org]; !ok {
				// We have not seen this org so the exceptions are just applicable
				// members of 'excludedRepos'.
				orgs[org] = applicableRepos
			} else {
				// We have seen this org so the exceptions are the applicable
				// members of 'excludedRepos' intersected with existing exceptions.
				orgs[org] = excepts.Intersection(applicableRepos)
			}
		}
	}
	repos := sets.NewString()
	for i := range tqs {
		repos.Insert(tqs[i].Repos...)
	}
	// Remove any org exceptions that are explicitly included in a different query.
	reposList := repos.UnsortedList()
	for _, excepts := range orgs {
		excepts.Delete(reposList...)
	}
	return orgs, repos
}

// QueryMap creates a QueryMap from KeeperQueries
func (tqs Queries) QueryMap() *QueryMap {
	return &QueryMap{
		queries: tqs,
		cache:   make(map[string]Queries),
	}
}
