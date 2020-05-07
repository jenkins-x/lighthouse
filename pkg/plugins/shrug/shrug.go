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

package shrug

import (
	"fmt"
	"regexp"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

const pluginName = "shrug"

var (
	shrugRe   = regexp.MustCompile(`(?mi)^/(?:lh-)?shrug\s*$`)
	unshrugRe = regexp.MustCompile(`(?mi)^/(?:lh-)?unshrug\s*$`)
)

type event struct {
	org           string
	repo          string
	number        int
	prAuthor      string
	commentAuthor string
	body          string
	assignees     []scm.User
	hasLabel      func(label string) (bool, error)
	htmlurl       string
}

func init() {
	plugins.RegisterGenericCommentHandler(pluginName, handleGenericComment, helpProvider)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	// The Config field is omitted because this plugin is not configurable.
	pluginHelp := &pluginhelp.PluginHelp{
		Description: labels.Shrug,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/[un]shrug",
		Description: labels.Shrug,
		Featured:    false,
		WhoCanUse:   "Anyone, " + labels.Shrug,
		Examples:    []string{"/shrug", "/unshrug"},
	})
	return pluginHelp, nil
}

type scmProviderClient interface {
	AddLabel(owner, repo string, number int, label string, pr bool) error
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	RemoveLabel(owner, repo string, number int, label string, pr bool) error
	GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error)
}

func handleGenericComment(pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
	return handle(pc.SCMProviderClient, pc.Logger, &e)
}

func handle(spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent) error {
	if e.Action != scm.ActionCreate {
		return nil
	}

	wantShrug := false
	if shrugRe.MatchString(e.Body) {
		wantShrug = true
	} else if unshrugRe.MatchString(e.Body) {
		wantShrug = false
	} else {
		return nil
	}

	org := e.Repo.Namespace
	repo := e.Repo.Name

	// Only add the label if it doesn't have it yet.
	hasShrug := false
	issueLabels, err := spc.GetIssueLabels(org, repo, e.Number, e.IsPR)
	if err != nil {
		log.WithError(err).Errorf("Failed to get the labels on %s/%s#%d.", org, repo, e.Number)
	}
	for _, candidate := range issueLabels {
		if candidate.Name == labels.Shrug {
			hasShrug = true
			break
		}
	}
	if hasShrug && !wantShrug {
		log.Info("Removing Shrug label.")
		resp := "¯\\\\\\_(ツ)\\_/¯"
		log.Infof("Commenting with \"%s\".", resp)
		if err := spc.CreateComment(org, repo, e.Number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, e.Author.Login, resp)); err != nil {
			return fmt.Errorf("failed to comment on %s/%s#%d: %v", org, repo, e.Number, err)
		}
		return spc.RemoveLabel(org, repo, e.Number, labels.Shrug, e.IsPR)
	} else if !hasShrug && wantShrug {
		log.Info("Adding Shrug label.")
		return spc.AddLabel(org, repo, e.Number, labels.Shrug, e.IsPR)
	}
	return nil
}
