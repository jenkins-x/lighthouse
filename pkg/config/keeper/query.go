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
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/sets"
)

// Query is turned into a GitHub search query. See the docs for details:
// https://help.github.com/articles/searching-issues-and-pull-requests/
type Query struct {
	Orgs                   []string `json:"orgs,omitempty"`
	Repos                  []string `json:"repos,omitempty"`
	ExcludedRepos          []string `json:"excludedRepos,omitempty"`
	ExcludedBranches       []string `json:"excludedBranches,omitempty"`
	IncludedBranches       []string `json:"includedBranches,omitempty"`
	Labels                 []string `json:"labels,omitempty"`
	MissingLabels          []string `json:"missingLabels,omitempty"`
	Milestone              string   `json:"milestone,omitempty"`
	ReviewApprovedRequired bool     `json:"reviewApprovedRequired,omitempty"`
}

// Query returns the corresponding github search string for the keeper query.
func (tq *Query) Query() string {
	toks := []string{"is:pr", "state:open"}
	for _, o := range tq.Orgs {
		toks = append(toks, fmt.Sprintf("org:\"%s\"", o))
	}
	for _, r := range tq.Repos {
		toks = append(toks, fmt.Sprintf("repo:\"%s\"", r))
	}
	for _, r := range tq.ExcludedRepos {
		toks = append(toks, fmt.Sprintf("-repo:\"%s\"", r))
	}
	for _, b := range tq.ExcludedBranches {
		toks = append(toks, fmt.Sprintf("-base:\"%s\"", b))
	}
	for _, b := range tq.IncludedBranches {
		toks = append(toks, fmt.Sprintf("base:\"%s\"", b))
	}
	for _, l := range tq.Labels {
		toks = append(toks, fmt.Sprintf("label:\"%s\"", l))
	}
	for _, l := range tq.MissingLabels {
		toks = append(toks, fmt.Sprintf("-label:\"%s\"", l))
	}
	if tq.Milestone != "" {
		toks = append(toks, fmt.Sprintf("milestone:\"%s\"", tq.Milestone))
	}
	if tq.ReviewApprovedRequired {
		toks = append(toks, "review:approved")
	}
	return strings.Join(toks, " ")
}

// ForRepo indicates if the keeper query applies to the specified repo.
func (tq Query) ForRepo(org, repo string) bool {
	fullName := fmt.Sprintf("%s/%s", org, repo)
	for _, queryOrg := range tq.Orgs {
		if queryOrg != org {
			continue
		}
		// Check for repos excluded from the org.
		for _, excludedRepo := range tq.ExcludedRepos {
			if excludedRepo == fullName {
				return false
			}
		}
		return true
	}
	for _, queryRepo := range tq.Repos {
		if queryRepo == fullName {
			return true
		}
	}
	return false
}

// Validate returns an error if the query has any errors.
//
// Examples include:
// * an org name that is empty or includes a /
// * repos that are not org/repo
// * a label that is in both the labels and missing_labels section
// * a branch that is in both included and excluded branch set.
func (tq *Query) Validate() error {
	duplicates := func(field string, list []string) error {
		dups := sets.NewString()
		seen := sets.NewString()
		for _, elem := range list {
			if seen.Has(elem) {
				dups.Insert(elem)
			} else {
				seen.Insert(elem)
			}
		}
		dupCount := len(list) - seen.Len()
		if dupCount == 0 {
			return nil
		}
		return fmt.Errorf("%q contains %d duplicate entries: %s", field, dupCount, strings.Join(dups.List(), ", "))
	}

	orgs := sets.NewString()
	for o := range tq.Orgs {
		if strings.Contains(tq.Orgs[o], "/") {
			return fmt.Errorf("orgs[%d]: %q contains a '/' which is not valid", o, tq.Orgs[o])
		}
		if len(tq.Orgs[o]) == 0 {
			return fmt.Errorf("orgs[%d]: is an empty string", o)
		}
		orgs.Insert(tq.Orgs[o])
	}
	if err := duplicates("orgs", tq.Orgs); err != nil {
		return err
	}

	for r := range tq.Repos {
		parts := strings.Split(tq.Repos[r], "/")
		// parts can have length of upto 20 for nested repos in gitlab
		if len(parts[0]) == 0 || len(parts[1]) == 0 {
			return fmt.Errorf("repos[%d]: %q is not of the form \"org/repo\"", r, tq.Repos[r])
		}
		if orgs.Has(parts[0]) {
			return fmt.Errorf("repos[%d]: %q is already included via org: %q", r, tq.Repos[r], parts[0])
		}
	}
	if err := duplicates("repos", tq.Repos); err != nil {
		return err
	}

	if len(tq.Orgs) == 0 && len(tq.Repos) == 0 {
		return errors.New("'orgs' and 'repos' cannot both be empty")
	}

	for er := range tq.ExcludedRepos {
		parts := strings.Split(tq.ExcludedRepos[er], "/")
		if len(parts[0]) == 0 || len(parts[1]) == 0 {
			return fmt.Errorf("excludedRepos[%d]: %q is not of the form \"org/repo\"", er, tq.ExcludedRepos[er])
		}
		if !orgs.Has(parts[0]) {
			return fmt.Errorf("excludedRepos[%d]: %q has no effect because org %q is not included", er, tq.ExcludedRepos[er], parts[0])
		}
		// Note: At this point we also know that this excludedRepo is not found in 'repos'.
	}
	if err := duplicates("excludedRepos", tq.ExcludedRepos); err != nil {
		return err
	}

	if invalids := sets.NewString(tq.Labels...).Intersection(sets.NewString(tq.MissingLabels...)); len(invalids) > 0 {
		return fmt.Errorf("the labels: %q are both required and forbidden", invalids.List())
	}
	if err := duplicates("labels", tq.Labels); err != nil {
		return err
	}
	if err := duplicates("missingLabels", tq.MissingLabels); err != nil {
		return err
	}

	if len(tq.ExcludedBranches) > 0 && len(tq.IncludedBranches) > 0 {
		return errors.New("both 'includedBranches' and 'excludedBranches' are specified ('excludedBranches' have no effect)")
	}
	if err := duplicates("includedBranches", tq.IncludedBranches); err != nil {
		return err
	}
	if err := duplicates("excludedBranches", tq.ExcludedBranches); err != nil {
		return err
	}

	return nil
}
