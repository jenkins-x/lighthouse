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

// Package stage defines a Prow plugin that defines the stage of
// the issue in the features process. Eg: alpha, beta, stable.
package stage

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

var (
	stageAlpha  = "stage/alpha"
	stageBeta   = "stage/beta"
	stageStable = "stage/stable"
	stageLabels = []string{stageAlpha, stageBeta, stageStable}
)

const pluginName = "stage"

var (
	plugin = plugins.Plugin{
		Description: "Label the stage of an issue as alpha/beta/stable",
		Commands: []plugins.Command{{
			Name: "stage",
			Arg: &plugins.CommandArg{
				Pattern: "alpha|beta|stable",
			},
			Description: "Labels the stage of an issue as alpha/beta/stable",
			Filter:      func(e scmprovider.GenericCommentEvent) bool { return e.Action == scm.ActionCreate },
			Handler: func(match plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
				return stage(pc.SCMProviderClient, pc.Logger, &e, match.Arg)
			},
		}, {
			Name: "remove-stage",
			Arg: &plugins.CommandArg{
				Pattern: "alpha|beta|stable",
			},
			Description: "Removes the stage label of an issue as alpha/beta/stable",
			Filter:      func(e scmprovider.GenericCommentEvent) bool { return e.Action == scm.ActionCreate },
			Handler: func(match plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
				return unstage(pc.SCMProviderClient, pc.Logger, &e, match.Arg)
			},
		}},
	}
)

func init() {
	plugins.RegisterPlugin(pluginName, plugin)
}

type scmProviderClient interface {
	AddLabel(owner, repo string, number int, label string, pr bool) error
	RemoveLabel(owner, repo string, number int, label string, pr bool) error
	GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error)
}

func unstage(gc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, stage string) error {
	lbl := "stage/" + stage

	// Let's start simple and allow anyone to add/remove alpha, beta and stable labels.
	// Adjust if we find evidence of the community abusing these labels.
	labels, err := gc.GetIssueLabels(e.Repo.Namespace, e.Repo.Name, e.Number, e.IsPR)
	if err != nil {
		log.WithError(err).Errorf("Failed to get labels.")
		return err
	}

	// If the label exists and we asked for it to be removed, remove it.
	if scmprovider.HasLabel(lbl, labels) {
		return gc.RemoveLabel(e.Repo.Namespace, e.Repo.Name, e.Number, lbl, e.IsPR)
	}
	return nil
}

func stage(gc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, stage string) error {
	lbl := "stage/" + stage

	// Let's start simple and allow anyone to add/remove alpha, beta and stable labels.
	// Adjust if we find evidence of the community abusing these labels.
	labels, err := gc.GetIssueLabels(e.Repo.Namespace, e.Repo.Name, e.Number, e.IsPR)
	if err != nil {
		log.WithError(err).Errorf("Failed to get labels.")
	}

	// If the label does not exist and we asked for it to be added,
	// remove other existing stage labels and add it.
	if !scmprovider.HasLabel(lbl, labels) {
		for _, label := range stageLabels {
			if lbl != label && scmprovider.HasLabel(label, labels) {
				if err := gc.RemoveLabel(e.Repo.Namespace, e.Repo.Name, e.Number, label, e.IsPR); err != nil {
					log.WithError(err).Errorf("GitHub failed to remove the following label: %s", label)
				}
			}
		}

		if err := gc.AddLabel(e.Repo.Namespace, e.Repo.Name, e.Number, lbl, e.IsPR); err != nil {
			log.WithError(err).Errorf("GitHub failed to add the following label: %s", lbl)
		}
	}

	return nil
}
