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

// Package skip implements the `/skip` command which allows users
// to clean up commit statuses of non-blocking presubmits on PRs.
package skip

import (
	"fmt"
	"regexp"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/plugins/trigger"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
)

const pluginName = "skip"

var (
	skipRe = regexp.MustCompile(`(?mi)^/(?:lh-)?skip\s*$`)
)

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	CreateStatus(org, repo, ref string, s *scm.StatusInput) (*scm.Status, error)
	GetPullRequest(org, repo string, number int) (*scm.PullRequest, error)
	GetCombinedStatus(org, repo, ref string) (*scm.CombinedStatus, error)
	GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error)
}

func init() {
	plugins.RegisterGenericCommentHandler(pluginName, handleGenericComment, helpProvider)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The skip plugin allows users to clean up GitHub stale commit statuses for non-blocking jobs on a PR.",
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/skip",
		Description: "Cleans up GitHub stale commit statuses for non-blocking jobs on a PR.",
		Featured:    false,
		WhoCanUse:   "Anyone can trigger this command on a PR.",
		Examples:    []string{"/skip", "/lh-skip"},
	})
	return pluginHelp, nil
}

func handleGenericComment(pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
	honorOkToTest := trigger.HonorOkToTest(pc.PluginConfig.TriggerFor(e.Repo.Namespace, e.Repo.Name))
	return handle(pc.SCMProviderClient, pc.Logger, &e, pc.Config.GetPresubmits(e.Repo), honorOkToTest)
}

func handle(spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, presubmits []config.Presubmit, honorOkToTest bool) error {
	if !e.IsPR || e.IssueState != "open" || e.Action != scm.ActionCreate {
		return nil
	}

	if !skipRe.MatchString(e.Body) {
		return nil
	}

	org := e.Repo.Namespace
	repo := e.Repo.Name
	number := e.Number

	pr, err := spc.GetPullRequest(org, repo, number)
	if err != nil {
		resp := fmt.Sprintf("Cannot get PR #%d in %s/%s: %v", number, org, repo, err)
		log.Warn(resp)
		return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, e.Author.Login, resp))
	}

	combinedStatus, err := spc.GetCombinedStatus(org, repo, pr.Head.Sha)
	if err != nil {
		resp := fmt.Sprintf("Cannot get combined commit statuses for PR #%d in %s/%s: %v", number, org, repo, err)
		log.Warn(resp)
		return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, e.Author.Login, resp))
	}
	if combinedStatus.State == scm.StateSuccess {
		return nil
	}
	statuses := combinedStatus.Statuses

	filteredPresubmits, _, err := trigger.FilterPresubmits(honorOkToTest, spc, e.Body, pr, presubmits, log)
	if err != nil {
		resp := fmt.Sprintf("Cannot get combined status for PR #%d in %s/%s: %v", number, org, repo, err)
		log.Warn(resp)
		return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, e.Author.Login, resp))
	}
	triggerWillHandle := func(p config.Presubmit) bool {
		for _, presubmit := range filteredPresubmits {
			if p.Name == presubmit.Name && p.Context == presubmit.Context {
				return true
			}
		}
		return false
	}

	for _, job := range presubmits {
		// Only consider jobs that have already posted a failed status
		if !statusExists(job, statuses) || isSuccess(job, statuses) {
			continue
		}
		// Ignore jobs that will be handled by the trigger plugin
		// for this specific comment, regardless of whether they
		// are required or not. This allows a comment like
		// >/skip
		// >/test foo
		// To end up testing foo instead of skipping it
		if triggerWillHandle(job) {
			continue
		}
		// Only skip jobs that are not required
		if job.ContextRequired() {
			continue
		}
		context := job.Context
		status := &scm.StatusInput{
			State: scm.StateSuccess,
			Desc:  "Skipped",
			Label: context,
		}
		if _, err := spc.CreateStatus(org, repo, pr.Head.Sha, status); err != nil {
			resp := fmt.Sprintf("Cannot update PR status for context %s: %v", context, err)
			log.Warn(resp)
			return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, e.Author.Login, resp))
		}
	}
	return nil
}

func statusExists(job config.Presubmit, statuses []*scm.Status) bool {
	for _, status := range statuses {
		if status.Label == job.Context {
			return true
		}
	}
	return false
}

func isSuccess(job config.Presubmit, statuses []*scm.Status) bool {
	for _, status := range statuses {
		if status.Label == job.Context && status.State == scm.StateSuccess {
			return true
		}
	}
	return false
}
