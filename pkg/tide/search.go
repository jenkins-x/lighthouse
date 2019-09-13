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

package tide

import (
	"context"
	"fmt"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	github "github.com/jenkins-x/lighthouse/pkg/prow/gitprovider"

	"github.com/sirupsen/logrus"
)

type querier func(ctx context.Context, result interface{}, vars map[string]interface{}) error

func datedQuery(q string, start, end time.Time) string {
	return fmt.Sprintf("%s %s", q, dateToken(start, end))
}

func floor(t time.Time) time.Time {
	if t.Before(github.FoundingYear) {
		return github.FoundingYear
	}
	return t
}

func search(ghc githubClient, log *logrus.Entry, q string, start, end time.Time) ([]PullRequest, error) {
	start = floor(start)
	end = floor(end)
	log = log.WithFields(logrus.Fields{
		"query": q,
		"start": start.String(),
		"end":   end.String(),
	})
	requestStart := time.Now()
	var ret []PullRequest
	log.Debug("Sending query")
	opts := scm.SearchOptions{Query: q}
	results, rates, err := ghc.Query(opts)
	if err != nil {
		return ret, err
	}
	log.WithField("duration", time.Since(requestStart).String()).Debugf("Query returned %d PRs and %d remaining.", len(ret), rates.Remaining)

	for _, n := range results {
		ret = append(ret, toPullRequest(n))
	}
	return ret, nil
}

func toPullRequest(from *scm.SearchIssue) PullRequest {
	return PullRequest{
		Number: from.Number,

		Labels: from.Labels,
		// TODO Milestone.Title
		Body:      from.Body,
		Title:     from.Title,
		UpdatedAt: from.Updated,
	}

}

// dateToken generates a GitHub search query token for the specified date range.
// See: https://help.github.com/articles/understanding-the-search-syntax/#query-for-dates
func dateToken(start, end time.Time) string {
	// GitHub's GraphQL API silently fails if you provide it with an invalid time
	// string.
	// Dates before 1970 (unix epoch) are considered invalid.
	startString, endString := "*", "*"
	if start.Year() >= 1970 {
		startString = start.Format(github.SearchTimeFormat)
	}
	if end.Year() >= 1970 {
		endString = end.Format(github.SearchTimeFormat)
	}
	return fmt.Sprintf("updated:%s..%s", startString, endString)
}
