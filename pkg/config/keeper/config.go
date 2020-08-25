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
	"time"
)

// Config is the config for the keeper pool.
type Config struct {
	// SyncPeriodString compiles into SyncPeriod at load time.
	SyncPeriodString string `json:"sync_period,omitempty"`
	// SyncPeriod specifies how often Keeper will sync jobs with Github. Defaults to 1m.
	SyncPeriod time.Duration `json:"-"`
	// StatusUpdatePeriodString compiles into StatusUpdatePeriod at load time.
	StatusUpdatePeriodString string `json:"status_update_period,omitempty"`
	// StatusUpdatePeriod specifies how often Keeper will update Github status contexts.
	// Defaults to the value of SyncPeriod.
	StatusUpdatePeriod time.Duration `json:"-"`
	// Queries represents a list of GitHub search queries that collectively
	// specify the set of PRs that meet merge requirements.
	Queries Queries `json:"queries,omitempty"`
	// A key/value pair of an org/repo as the key and merge method to override
	// the default method of merge. Valid options are squash, rebase, and merge.
	MergeType map[string]PullRequestMergeType `json:"merge_method,omitempty"`
	// A key/value pair of an org/repo as the key and Go template to override
	// the default merge commit title and/or message. Template is passed the
	// PullRequest struct (prow/github/types.go#PullRequest)
	MergeTemplate map[string]MergeCommitTemplate `json:"merge_commit_template,omitempty"`
	// URL for keeper status contexts.
	// We can consider allowing this to be set separately for separate repos, or
	// allowing it to be a template.
	TargetURL string `json:"target_url,omitempty"`
	// PRStatusBaseURL is the base URL for the PR status page.
	// This is used to link to a merge requirements overview
	// in the keeper status context.
	PRStatusBaseURL string `json:"pr_status_base_url,omitempty"`
	// BlockerLabel is an optional label that is used to identify merge blocking
	// Github issues.
	// Leave this blank to disable this feature and save 1 API token per sync loop.
	BlockerLabel string `json:"blocker_label,omitempty"`
	// SquashLabel is an optional label that is used to identify PRs that should
	// always be squash merged.
	// Leave this blank to disable this feature.
	SquashLabel string `json:"squash_label,omitempty"`
	// RebaseLabel is an optional label that is used to identify PRs that should
	// always be rebased and merged.
	// Leave this blank to disable this feature.
	RebaseLabel string `json:"rebase_label,omitempty"`
	// MergeLabel is an optional label that is used to identify PRs that should
	// always be merged with all individual commits from the PR.
	// Leave this blank to disable this feature.
	MergeLabel string `json:"merge_label,omitempty"`
	// MaxGoroutines is the maximum number of goroutines spawned inside the
	// controller to handle org/repo:branch pools. Defaults to 20. Needs to be a
	// positive number.
	MaxGoroutines int `json:"max_goroutines,omitempty"`
	// KeeperContextPolicyOptions defines merge options for context. If not set it will infer
	// the required and optional contexts from the prow jobs configured and use the github
	// combined status; otherwise it may apply the branch protection setting or let user
	// define their own options in case branch protection is not used.
	ContextOptions ContextPolicyOptions `json:"context_options,omitempty"`
	// BatchSizeLimitMap is a key/value pair of an org or org/repo as the key and
	// integer batch size limit as the value. The empty string key can be used as
	// a global default.
	// Special values:
	//  0 => unlimited batch size
	// -1 => batch merging disabled :(
	BatchSizeLimitMap map[string]int `json:"batch_size_limit,omitempty"`
}

// MergeMethod returns the merge method to use for a repo. The default of merge is
// returned when not overridden.
func (c *Config) MergeMethod(org, repo string) PullRequestMergeType {
	name := org + "/" + repo

	v, ok := c.MergeType[name]
	if !ok {
		if ov, found := c.MergeType[org]; found {
			return ov
		}

		return MergeMerge
	}

	return v
}

// BatchSizeLimit return the batch size limit for the given repo
func (c *Config) BatchSizeLimit(org, repo string) int {
	// TODO: Remove once #564 is fixed and batch builds can work again. (APB)
	return -1
	//if limit, ok := t.BatchSizeLimitMap[fmt.Sprintf("%s/%s", org, repo)]; ok {
	//	return limit
	//}
	//if limit, ok := t.BatchSizeLimitMap[org]; ok {
	//	return limit
	//}
	//return t.BatchSizeLimitMap["*"]
}

// MergeCommitTemplate returns a struct with Go template string(s) or nil
func (c *Config) MergeCommitTemplate(org, repo string) MergeCommitTemplate {
	name := org + "/" + repo

	v, ok := c.MergeTemplate[name]
	if !ok {
		return c.MergeTemplate[org]
	}

	return v
}

// Parse initializes and validates the Config
func (c *Config) Parse() error {
	if c.SyncPeriodString == "" {
		c.SyncPeriod = time.Minute
	} else {
		period, err := time.ParseDuration(c.SyncPeriodString)
		if err != nil {
			return fmt.Errorf("cannot parse duration for tide.sync_period: %v", err)
		}
		c.SyncPeriod = period
	}
	if c.StatusUpdatePeriodString == "" {
		c.StatusUpdatePeriod = c.SyncPeriod
	} else {
		period, err := time.ParseDuration(c.StatusUpdatePeriodString)
		if err != nil {
			return fmt.Errorf("cannot parse duration for tide.status_update_period: %v", err)
		}
		c.StatusUpdatePeriod = period
	}
	if c.MaxGoroutines == 0 {
		c.MaxGoroutines = 20
	}
	if c.MaxGoroutines <= 0 {
		return fmt.Errorf("keeper has invalid max_goroutines (%d), it needs to be a positive number", c.MaxGoroutines)
	}
	for name, method := range c.MergeType {
		if !method.IsValid() {
			return fmt.Errorf("merge type %q for %s is not a valid type", method, name)
		}
	}
	for i, tq := range c.Queries {
		if err := tq.Validate(); err != nil {
			return fmt.Errorf("keeper query (index %d) is invalid: %v", i, err)
		}
	}
	return nil
}
