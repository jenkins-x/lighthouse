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
	"fmt"
	"sync"
)

// QueryMap is a struct mapping from "org/repo" -> KeeperQueries that
// apply to that org or repo. It is lazily populated, but threadsafe.
type QueryMap struct {
	queries Queries

	cache map[string]Queries
	sync.Mutex
}

// ForRepo returns the keeper queries that apply to a repo.
func (qm *QueryMap) ForRepo(org, repo string) Queries {
	res := Queries(nil)
	fullName := fmt.Sprintf("%s/%s", org, repo)

	qm.Lock()
	defer qm.Unlock()

	if qs, ok := qm.cache[fullName]; ok {
		return append(res, qs...) // Return a copy.
	}
	// Cache miss. Need to determine relevant queries.

	for _, query := range qm.queries {
		if query.ForRepo(org, repo) {
			res = append(res, query)
		}
	}
	qm.cache[fullName] = res
	return res
}
