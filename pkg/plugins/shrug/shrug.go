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

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

const pluginName = "shrug"

var (
	plugin = plugins.Plugin{
		Description: labels.Shrug,
		Commands: []plugins.Command{{
			Name:        "shrug",
			Description: "Adds the " + labels.Shrug + " label",
			Filter:      func(e scmprovider.GenericCommentEvent) bool { return e.Action == scm.ActionCreate },
			Handler: func(_ plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
				return addLabel(pc.SCMProviderClient, pc.Logger, &e)
			},
		}, {
			Name:        "unshrug",
			Description: "Removes the " + labels.Shrug + " label",
			Filter:      func(e scmprovider.GenericCommentEvent) bool { return e.Action == scm.ActionCreate },
			Handler: func(_ plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
				return removeLabel(pc.SCMProviderClient, pc.Logger, &e)
			},
		}},
	}
)

func init() {
	plugins.RegisterPlugin(pluginName, plugin)
}

type scmProviderClient interface {
	AddLabel(owner, repo string, number int, label string, pr bool) error
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	RemoveLabel(owner, repo string, number int, label string, pr bool) error
	GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error)
	QuoteAuthorForComment(string) string
}

func hasLabel(spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent) (bool, error) {
	issueLabels, err := spc.GetIssueLabels(e.Repo.Namespace, e.Repo.Name, e.Number, e.IsPR)
	if err != nil {
		log.WithError(err).Errorf("Failed to get the labels on %s/%s#%d.", e.Repo.Namespace, e.Repo.Name, e.Number)
		return false, err
	}
	for _, candidate := range issueLabels {
		if candidate.Name == labels.Shrug {
			return true, nil
		}
	}
	return false, nil
}

func addLabel(spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent) error {
	hasLabel, err := hasLabel(spc, log, e)
	if err != nil || hasLabel {
		return err
	}
	log.Info("Adding Shrug label.")
	return spc.AddLabel(e.Repo.Namespace, e.Repo.Name, e.Number, labels.Shrug, e.IsPR)

}

func removeLabel(spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent) error {
	hasLabel, err := hasLabel(spc, log, e)
	if err != nil || !hasLabel {
		return err
	}
	log.Info("Removing Shrug label.")
	resp := "¯\\\\\\_(ツ)\\_/¯"
	log.Infof("Commenting with \"%s\".", resp)
	if err := spc.CreateComment(e.Repo.Namespace, e.Repo.Name, e.Number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(e.Author.Login), resp)); err != nil {
		return fmt.Errorf("failed to comment on %s/%s#%d: %v", e.Repo.Namespace, e.Repo.Name, e.Number, err)
	}
	return spc.RemoveLabel(e.Repo.Namespace, e.Repo.Name, e.Number, labels.Shrug, e.IsPR)
}
