/*
Copyright 2018 The Kubernetes Authors.

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

// Package cherrypickunapproved adds the `do-not-merge/cherry-pick-not-approved`
// label to PRs against a release branch which do not have the
// `cherry-pick-approved` label.
package cherrypickunapproved

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/prow/github"
	"github.com/jenkins-x/lighthouse/pkg/prow/labels"
	"github.com/jenkins-x/lighthouse/pkg/prow/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
)

const (
	// PluginName defines this plugin's registered name.
	PluginName = "cherry-pick-unapproved"
)

func init() {
	plugins.RegisterPullRequestHandler(PluginName, handlePullRequest, helpProvider)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	// Only the 'Config' and Description' fields are necessary because this
	// plugin does not react to any commands.
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "Label PRs against a release branch which do not have the `cherry-pick-approved` label with the `do-not-merge/cherry-pick-not-approved` label.",
		Config: map[string]string{
			"": fmt.Sprintf(
				"The cherry-pick-unapproved plugin treats PRs against branch names satisfying the regular expression `%s` as cherry-pick PRs and adds the following comment:\n%s",
				config.CherryPickUnapproved.BranchRegexp,
				config.CherryPickUnapproved.Comment,
			),
		},
	}
	return pluginHelp, nil
}

type githubClient interface {
	CreateComment(owner, repo string, number int, comment string) error
	AddLabel(owner, repo string, number int, label string) error
	RemoveLabel(owner, repo string, number int, label string) error
	GetIssueLabels(org, repo string, number int) ([]*scm.Label, error)
}

type commentPruner interface {
	PruneComments(shouldPrune func(*scm.Comment) bool)
}

func handlePullRequest(pc plugins.Agent, pr scm.PullRequestHook) error {
	cp, err := pc.CommentPruner()
	if err != nil {
		return err
	}
	return handlePR(
		pc.GitHubClient, pc.Logger, &pr, cp,
		pc.PluginConfig.CherryPickUnapproved.BranchRe, pc.PluginConfig.CherryPickUnapproved.Comment,
	)
}

func handlePR(gc githubClient, log *logrus.Entry, pr *scm.PullRequestHook, cp commentPruner, branchRe *regexp.Regexp, commentBody string) error {
	// Only consider the events that indicate opening of the PR and
	// when the cpApproved and cpUnapproved labels are added or removed
	cpLabelUpdated := (pr.Action == scm.ActionLabel || pr.Action == scm.ActionUnlabel) &&
		(pr.Label.Name == labels.CpApproved || pr.Label.Name == labels.CpUnapproved)
	if pr.Action != scm.ActionOpen && pr.Action != scm.ActionReopen && !cpLabelUpdated {
		return nil
	}

	var (
		org      = pr.Repo.Namespace
		repo     = pr.Repo.Name
		branch   = pr.PullRequest.Ref
		prNumber = pr.PullRequest.Number
	)

	// if the branch doesn't match against the branch names allowed for cherry-picks,
	// don't do anything
	if !branchRe.MatchString(branch) {
		return nil
	}

	issueLabels, err := gc.GetIssueLabels(org, repo, prNumber)
	if err != nil {
		return err
	}
	hasCherryPickApprovedLabel := github.HasLabel(labels.CpApproved, issueLabels)
	hasCherryPickUnapprovedLabel := github.HasLabel(labels.CpUnapproved, issueLabels)

	// if it has the approved label,
	// remove the unapproved label (if it exists) and
	// remove any comments left by this plugin
	if hasCherryPickApprovedLabel {
		if hasCherryPickUnapprovedLabel {
			if err := gc.RemoveLabel(org, repo, prNumber, labels.CpUnapproved); err != nil {
				log.WithError(err).Errorf("GitHub failed to remove the following label: %s", labels.CpUnapproved)
			}
		}
		cp.PruneComments(func(comment *scm.Comment) bool {
			return strings.Contains(comment.Body, commentBody)
		})
		return nil
	}

	// if it already has the unapproved label, we are done here
	if hasCherryPickUnapprovedLabel {
		return nil
	}

	// only add the label and comment if none of the approved and unapproved labels are present
	if err := gc.AddLabel(org, repo, prNumber, labels.CpUnapproved); err != nil {
		log.WithError(err).Errorf("GitHub failed to add the following label: %s", labels.CpUnapproved)
	}

	formattedComment := plugins.FormatSimpleResponse(pr.PullRequest.Author.Login, commentBody)
	if err := gc.CreateComment(org, repo, prNumber, formattedComment); err != nil {
		log.WithError(err).Errorf("Failed to comment %q", formattedComment)
	}

	return nil
}
