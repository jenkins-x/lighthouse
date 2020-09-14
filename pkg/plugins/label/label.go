/*
Copyright 2016 The Kubernetes Authors.

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

package label

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

const pluginName = "label"

var (
	defaultLabels           = []string{"kind", "priority", "area"}
	nonExistentLabelOnIssue = "Those labels are not set on the issue: `%v`"
)

var (
	plugin = plugins.Plugin{
		Description:        "The label plugin provides commands that add or remove certain types of labels. Labels of the following types can be manipulated: 'area/*', 'committee/*', 'kind/*', 'language/*', 'priority/*', 'sig/*', 'triage/*', and 'wg/*'. More labels can be configured to be used via the /label command.",
		ConfigHelpProvider: configHelp,
		Commands: []plugins.Command{{
			Prefix: "remove-",
			Name:   "area|committee|kind|language|priority|sig|triage|wg|label",
			Arg: &plugins.CommandArg{
				Pattern: ".*",
			},
			Description: "Applies or removes a label from one of the recognized types of labels.",
			Filter:      func(e scmprovider.GenericCommentEvent) bool { return e.Action == scm.ActionCreate },
			Handler: func(match plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
				return handle(match.Prefix != "", match.Name, match.Arg, pc.SCMProviderClient, pc.Logger, pc.PluginConfig.Label.AdditionalLabels, &e)
			},
		}},
	}
)

func init() {
	plugins.RegisterPlugin(pluginName, plugin)
}

func configString(labels []string) string {
	var formattedLabels []string
	for _, label := range labels {
		formattedLabels = append(formattedLabels, fmt.Sprintf(`"%s/*"`, label))
	}
	return fmt.Sprintf("The label plugin will work on %s and %s labels.", strings.Join(formattedLabels[:len(formattedLabels)-1], ", "), formattedLabels[len(formattedLabels)-1])
}

func configHelp(config *plugins.Configuration, enabledRepos []string) (map[string]string, error) {
	labels := []string{}
	labels = append(labels, defaultLabels...)
	labels = append(labels, config.Label.AdditionalLabels...)
	return map[string]string{
			"": configString(labels),
		},
		nil
}

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	AddLabel(owner, repo string, number int, label string, pr bool) error
	RemoveLabel(owner, repo string, number int, label string, pr bool) error
	GetRepoLabels(owner, repo string) ([]*scm.Label, error)
	GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error)
	QuoteAuthorForComment(string) string
}

// Get Labels from Regexp matches
func getLabelsFromREMatches(kind string, target string) []string {
	var labels []string
	for _, label := range strings.Split(target, " ") {
		label = strings.ToLower(kind + "/" + strings.TrimSpace(label))
		labels = append(labels, label)
	}
	return labels
}

// getLabelsFromGenericMatches returns label matches with extra labels if those
// have been configured in the plugin config.
func getLabelsFromGenericMatches(label string, additionalLabels []string) []string {
	var labels []string
	for _, l := range additionalLabels {
		if l == label {
			labels = append(labels, label)
		}
	}
	return labels
}

func handle(remove bool, kind string, target string, spc scmProviderClient, log *logrus.Entry, additionalLabels []string, e *scmprovider.GenericCommentEvent) error {
	org := e.Repo.Namespace
	repo := e.Repo.Name

	repoLabels, err := spc.GetRepoLabels(org, repo)
	if err != nil {
		return err
	}
	labels, err := spc.GetIssueLabels(org, repo, e.Number, e.IsPR)
	if err != nil {
		return err
	}

	RepoLabelsExisting := map[string]string{}
	for _, l := range repoLabels {
		RepoLabelsExisting[strings.ToLower(l.Name)] = l.Name
	}
	var (
		nonexistent         []string
		noSuchLabelsOnIssue []string
	)

	// Get labels to add and labels to remove from regexp matches
	var lbls []string
	if kind == "label" {
		lbls = append(lbls, getLabelsFromGenericMatches(target, additionalLabels)...)
	} else {
		lbls = append(lbls, getLabelsFromREMatches(kind, target)...)
	}

	for _, lbl := range lbls {
		if remove {
			if !scmprovider.HasLabel(lbl, labels) {
				noSuchLabelsOnIssue = append(noSuchLabelsOnIssue, lbl)
			} else {
				if _, ok := RepoLabelsExisting[lbl]; !ok {
					nonexistent = append(nonexistent, lbl)
				} else {
					if err := spc.RemoveLabel(org, repo, e.Number, lbl, e.IsPR); err != nil {
						log.WithError(err).Errorf("Failed to remove the following label: %s", lbl)
					}
				}
			}
		} else {
			if !scmprovider.HasLabel(lbl, labels) {
				if _, ok := RepoLabelsExisting[lbl]; !ok {
					nonexistent = append(nonexistent, lbl)
				} else {
					if err := spc.AddLabel(org, repo, e.Number, RepoLabelsExisting[lbl], e.IsPR); err != nil {
						log.WithError(err).Errorf("GitHub failed to add the following label: %s", lbl)
					}
				}
			}
		}
	}

	//TODO(grodrigues3): Once labels are standardized, make this reply with a comment.
	if len(nonexistent) > 0 {
		log.Infof("Nonexistent labels: %v", nonexistent)
	}

	// Tried to remove Labels that were not present on the Issue
	if len(noSuchLabelsOnIssue) > 0 {
		msg := fmt.Sprintf(nonExistentLabelOnIssue, strings.Join(noSuchLabelsOnIssue, ", "))
		log.Info(msg)
		return spc.CreateComment(org, repo, e.Number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(e.Author.Login), msg))
	}

	return nil
}
