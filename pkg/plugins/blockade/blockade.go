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

// Package blockade defines a plugin that adds the 'do-not-merge/blocked-paths' label to PRs that
// modify protected file paths.
// Protected file paths are defined with the plugins.Blockade struct. A PR is blocked if any file
// it changes is blocked by any Blockade. The process for determining if a file is blocked by a
// Blockade is as follows:
// By default, allow the file. Block if the file path matches any of block regexps, and does not
// match any of the exception regexps.
package blockade

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/sirupsen/logrus"
)

const (
	pluginName = "blockade"
)

var blockedPathsBody = fmt.Sprintf("Adding label: `%s` because PR changes a protected file.", labels.BlockedPaths)

type scmProviderClient interface {
	GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error)
	GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error)
	AddLabel(owner, repo string, number int, label string, pr bool) error
	RemoveLabel(owner, repo string, number int, label string, pr bool) error
	CreateComment(org, repo string, number int, pr bool, comment string) error
	QuoteAuthorForComment(string) string
}

type pruneClient interface {
	PruneComments(bool, func(ic *scm.Comment) bool)
}

func init() {
	plugins.RegisterPlugin(
		pluginName,
		plugins.Plugin{
			Description:        "The blockade plugin blocks pull requests from merging if they touch specific files. The plugin applies the '" + labels.BlockedPaths + "' label to pull requests that touch files that match a blockade's block regular expression and none of the corresponding exception regular expressions.",
			ConfigHelpProvider: configHelp,
			PullRequestHandler: handlePullRequest,
		},
	)
}

func configHelp(config *plugins.Configuration, enabledRepos []string) (map[string]string, error) {
	// The {WhoCanUse, Usage, Examples} fields are omitted because this plugin cannot be triggered manually.
	blockConfig := map[string]string{}
	for _, repo := range enabledRepos {
		parts := strings.Split(repo, "/")
		if len(parts) > 2 {
			return nil, fmt.Errorf("invalid repo in enabledRepos: %q", repo)
		}
		var buf bytes.Buffer
		fmt.Fprint(&buf, "The following blockades apply in this repository:")
		for _, blockade := range config.Blockades {
			if !stringInSlice(parts[0], blockade.Repos) && !stringInSlice(repo, blockade.Repos) {
				continue
			}
			fmt.Fprintf(&buf, "<br>Block reason: '%s'<br>&nbsp&nbsp&nbsp&nbspBlock regexps: %q<br>&nbsp&nbsp&nbsp&nbspException regexps: %q<br>", blockade.Explanation, blockade.BlockRegexps, blockade.ExceptionRegexps)
		}
		blockConfig[repo] = buf.String()
	}
	return blockConfig, nil
}

type blockCalc func([]*scm.Change, []blockade) summary

func handlePullRequest(pc plugins.Agent, pre scm.PullRequestHook) error {
	cp, err := pc.CommentPruner()
	if err != nil {
		return err
	}
	return handle(pc.SCMProviderClient, pc.Logger, pc.PluginConfig.Blockades, cp, calculateBlocks, &pre)
}

// blockade is a compiled version of a plugins.Blockade config struct.
type blockade struct {
	blockRegexps, exceptionRegexps []*regexp.Regexp
	explanation                    string
}

func (bd *blockade) isBlocked(file string) bool {
	return matchesAny(file, bd.blockRegexps) && !matchesAny(file, bd.exceptionRegexps)
}

type summary map[string][]*scm.Change

func (s summary) String() string {
	if len(s) == 0 {
		return ""
	}
	var buf bytes.Buffer
	fmt.Fprint(&buf, "#### Reasons for blocking this PR:\n")
	for reason, files := range s {
		fmt.Fprintf(&buf, "[%s]\n", reason)
		for _, file := range files {
			fmt.Fprintf(&buf, "- [%s](%s)\n\n", file.Path, file.BlobURL)
		}
	}
	return buf.String()
}

func handle(spc scmProviderClient, log *logrus.Entry, config []plugins.Blockade, cp pruneClient, blockCalc blockCalc, pre *scm.PullRequestHook) error {
	if pre.Action != scm.ActionSync &&
		pre.Action != scm.ActionOpen &&
		pre.Action != scm.ActionReopen {
		return nil
	}

	org := pre.Repo.Namespace
	repo := pre.Repo.Name
	prNumber := pre.PullRequest.Number
	issueLabels, err := spc.GetIssueLabels(org, repo, prNumber, true)
	if err != nil {
		return err
	}

	labelPresent := hasBlockedLabel(issueLabels)
	blockades := compileApplicableBlockades(org, repo, log, config)
	if len(blockades) == 0 && !labelPresent {
		// Since the label is missing, we assume that we removed any associated comments.
		return nil
	}

	var sum summary
	if len(blockades) > 0 {
		changes, err := spc.GetPullRequestChanges(org, repo, prNumber)
		if err != nil {
			return err
		}
		sum = blockCalc(changes, blockades)
	}

	shouldBlock := len(sum) > 0
	if shouldBlock && !labelPresent {
		// Add the label and leave a comment explaining why the label was added.
		if err := spc.AddLabel(org, repo, prNumber, labels.BlockedPaths, true); err != nil {
			return err
		}
		msg := plugins.FormatResponse(spc.QuoteAuthorForComment(pre.PullRequest.Author.Login), blockedPathsBody, sum.String())
		return spc.CreateComment(org, repo, prNumber, true, msg)
	} else if !shouldBlock && labelPresent {
		// Remove the label and delete any comments created by this plugin.
		if err := spc.RemoveLabel(org, repo, prNumber, labels.BlockedPaths, true); err != nil {
			return err
		}
		cp.PruneComments(true, func(ic *scm.Comment) bool {
			return strings.Contains(ic.Body, blockedPathsBody)
		})
	}
	return nil
}

// compileApplicableBlockades filters the specified blockades and compiles those that apply to the repo.
func compileApplicableBlockades(org, repo string, log *logrus.Entry, blockades []plugins.Blockade) []blockade {
	if len(blockades) == 0 {
		return nil
	}

	orgRepo := fmt.Sprintf("%s/%s", org, repo)
	var compiled []blockade
	for _, raw := range blockades {
		// Only consider blockades that apply to this repo.
		if !stringInSlice(org, raw.Repos) && !stringInSlice(orgRepo, raw.Repos) {
			continue
		}
		b := blockade{}
		for _, str := range raw.BlockRegexps {
			if reg, err := regexp.Compile(str); err != nil {
				log.WithError(err).Errorf("Failed to compile the blockade regexp '%s'.", str)
			} else {
				b.blockRegexps = append(b.blockRegexps, reg)
			}
		}
		if len(b.blockRegexps) == 0 {
			continue
		}
		if raw.Explanation == "" {
			b.explanation = "Files are protected"
		} else {
			b.explanation = raw.Explanation
		}
		for _, str := range raw.ExceptionRegexps {
			if reg, err := regexp.Compile(str); err != nil {
				log.WithError(err).Errorf("Failed to compile the blockade regexp '%s'.", str)
			} else {
				b.exceptionRegexps = append(b.exceptionRegexps, reg)
			}
		}
		compiled = append(compiled, b)
	}
	return compiled
}

// calculateBlocks determines if a PR should be blocked and returns the summary describing the block.
func calculateBlocks(changes []*scm.Change, blockades []blockade) summary {
	sum := make(summary)
	for _, change := range changes {
		for _, b := range blockades {
			if b.isBlocked(change.Path) {
				sum[b.explanation] = append(sum[b.explanation], change)
			}
		}
	}
	return sum
}

func hasBlockedLabel(githubLabels []*scm.Label) bool {
	label := strings.ToLower(labels.BlockedPaths)
	for _, elem := range githubLabels {
		if strings.ToLower(elem.Name) == label {
			return true
		}
	}
	return false
}

func matchesAny(str string, regexps []*regexp.Regexp) bool {
	for _, reg := range regexps {
		if reg.MatchString(str) {
			return true
		}
	}
	return false
}

func stringInSlice(str string, slice []string) bool {
	for _, elem := range slice {
		if elem == str {
			return true
		}
	}
	return false
}
