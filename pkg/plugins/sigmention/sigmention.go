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

// Package sigmention recognize SIG '@' mentions and adds 'sig/*' and 'kind/*' labels as appropriate.
// SIG mentions are also reitierated by the bot if the user who made the mention is not a member in
// order for the mention to trigger a notification for the github team.
package sigmention

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
)

const pluginName = "sigmention"

var (
	chatBack = "Reiterating the mentions to trigger a notification: \n%v\n"

	kindMap = map[string]string{
		"bugs":             labels.Bug,
		"feature-requests": "kind/feature",
		"api-reviews":      "kind/api-change",
		"proposals":        "kind/design",
	}
)

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	IsMember(org, user string) (bool, error)
	AddLabel(owner, repo string, number int, label string, pr bool) error
	RemoveLabel(owner, repo string, number int, label string, pr bool) error
	GetRepoLabels(owner, repo string) ([]*scm.Label, error)
	BotName() (string, error)
	GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error)
	QuoteAuthorForComment(string) string
}

var (
	sigmentionCommand = plugins.Command{
		// TODO help
		Filter:                func(e scmprovider.GenericCommentEvent) bool { return e.Action == scm.ActionCreate },
		GenericCommentHandler: handleGenericComment,
	}
)

func init() {
	description := `The sigmention plugin responds to SIG (Special Interest Group) GitHub team mentions like '@kubernetes/sig-testing-bugs'. The plugin responds in two ways:
<ol><li> The appropriate 'sig/*' and 'kind/*' labels are applied to the issue or pull request. In this case 'sig/testing' and 'kind/bug'.</li>
<li> If the user who mentioned the GitHub team is not a member of the organization that owns the repository the bot will create a comment that repeats the mention. This is necessary because non-member mentions do not trigger GitHub notifications.</li></ol>`

	plugins.RegisterPlugin(
		pluginName,
		plugins.Plugin{
			Description:        description,
			ConfigHelpProvider: configHelp,
			Commands: []plugins.Command{
				sigmentionCommand,
			},
		},
	)
}

func configHelp(config *plugins.Configuration, enabledRepos []string) (map[string]string, error) {
	return map[string]string{
			"": fmt.Sprintf("Labels added by the plugin are triggered by mentions of GitHub teams matching the following regexp:\n%s", config.SigMention.Regexp),
		},
		nil
}

func handleGenericComment(_ []string, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
	return handle(pc.SCMProviderClient, pc.Logger, &e, pc.PluginConfig.SigMention.Re)
}

func handle(spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, re *regexp.Regexp) error {
	// Ignore bot comments and comments that aren't new.
	botName, err := spc.BotName()
	if err != nil {
		return err
	}
	if e.Author.Login == botName {
		return nil
	}

	sigMatches := re.FindAllStringSubmatch(e.Body, -1)
	if len(sigMatches) == 0 {
		return nil
	}

	org := e.Repo.Namespace
	repo := e.Repo.Name

	labels, err := spc.GetIssueLabels(org, repo, e.Number, e.IsPR)
	if err != nil {
		return err
	}
	repoLabels, err := spc.GetRepoLabels(org, repo)
	if err != nil {
		return err
	}
	RepoLabelsExisting := map[string]string{}
	for _, l := range repoLabels {
		RepoLabelsExisting[strings.ToLower(l.Name)] = l.Name
	}

	var nonexistent, toRepeat []string
	for _, sigMatch := range sigMatches {
		sigLabel := strings.ToLower("sig" + "/" + sigMatch[1])
		sigLabel, ok := RepoLabelsExisting[sigLabel]
		if !ok {
			nonexistent = append(nonexistent, "sig/"+sigMatch[1])
			continue
		}
		if !scmprovider.HasLabel(sigLabel, labels) {
			if err := spc.AddLabel(org, repo, e.Number, sigLabel, e.IsPR); err != nil {
				log.WithError(err).Errorf("GitHub failed to add the following label: %s", sigLabel)
			}
		}

		if len(sigMatch) > 2 {
			if kindLabel, ok := kindMap[sigMatch[2]]; ok && !scmprovider.HasLabel(kindLabel, labels) {
				if err := spc.AddLabel(org, repo, e.Number, kindLabel, e.IsPR); err != nil {
					log.WithError(err).Errorf("GitHub failed to add the following label: %s", kindLabel)
				}
			}
		}

		toRepeat = append(toRepeat, sigMatch[0])
	}
	//TODO(grodrigues3): Once labels are standardized, make this reply with a comment.
	if len(nonexistent) > 0 {
		log.Infof("Nonexistent labels: %v", nonexistent)
	}

	isMember, err := spc.IsMember(org, e.Author.Login)
	if err != nil {
		log.WithError(err).Errorf("Error from IsMember(%q of org %q).", e.Author.Login, org)
	}
	if isMember || len(toRepeat) == 0 {
		return nil
	}

	msg := fmt.Sprintf(chatBack, strings.Join(toRepeat, ", "))
	return spc.CreateComment(org, repo, e.Number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(e.Author.Login), msg))
}
