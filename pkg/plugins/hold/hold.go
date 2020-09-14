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

// Package hold contains a plugin which will allow users to label their
// own pull requests as not ready or ready for merge. The submit queue
// will honor the label to ensure pull requests do not merge when it is
// applied.
package hold

import (
	"fmt"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

const (
	pluginName = "hold"
)

type hasLabelFunc func(label string, issueLabels []*scm.Label) bool

var (
	plugin = plugins.Plugin{
		Description: "The hold plugin allows anyone to add or remove the '" + labels.Hold + "' Label from a pull request in order to temporarily prevent the PR from merging without withholding approval.",
		Commands: []plugins.Command{{
			Name:        "hold",
			Description: "Adds or removes the `" + labels.Hold + "` Label which is used to indicate that the PR should not be automatically merged.",
			Arg: &plugins.CommandArg{
				Pattern:  "cancel",
				Optional: true,
			},
			Handler: func(match plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
				return handleGenericComment(match.Arg == "cancel", pc, e)
			},
			Filter: func(e scmprovider.GenericCommentEvent) bool { return e.Action == scm.ActionCreate },
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

func handleGenericComment(cancel bool, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
	hasLabel := func(label string, labels []*scm.Label) bool {
		return scmprovider.HasLabel(label, labels)
	}
	return handle(cancel, pc.SCMProviderClient, pc.Logger, &e, hasLabel)
}

// handle drives the pull request to the desired state. If any user adds
// a /hold directive, we want to add a label if one does not already exist.
// If they add /hold cancel, we want to remove the label if it exists.
func handle(cancel bool, spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, f hasLabelFunc) error {
	needsLabel := !cancel

	issueLabels, err := spc.GetIssueLabels(e.Repo.Namespace, e.Repo.Name, e.Number, e.IsPR)
	if err != nil {
		return fmt.Errorf("failed to get the labels on %s/%s#%d: %v", e.Repo.Namespace, e.Repo.Name, e.Number, err)
	}

	hasLabel := f(labels.Hold, issueLabels)
	if hasLabel && !needsLabel {
		log.Infof("Removing %q Label for %s/%s#%d", labels.Hold, e.Repo.Namespace, e.Repo.Name, e.Number)
		return spc.RemoveLabel(e.Repo.Namespace, e.Repo.Name, e.Number, labels.Hold, e.IsPR)
	} else if !hasLabel && needsLabel {
		log.Infof("Adding %q Label for %s/%s#%d", labels.Hold, e.Repo.Namespace, e.Repo.Name, e.Number)
		return spc.AddLabel(e.Repo.Namespace, e.Repo.Name, e.Number, labels.Hold, e.IsPR)
	}
	return nil
}
