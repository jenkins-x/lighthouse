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
	"regexp"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

const (
	pluginName = "hold"
)

var (
	labelRe       = regexp.MustCompile(`(?mi)^/(?:lh-)?hold\s*$`)
	labelCancelRe = regexp.MustCompile(`(?mi)^/(?:lh-)?hold cancel\s*$`)
)

type hasLabelFunc func(label string, issueLabels []*scm.Label) bool

var (
	plugin = plugins.Plugin{
		Description:  "The hold plugin allows anyone to add or remove the '" + labels.Hold + "' Label from a pull request in order to temporarily prevent the PR from merging without withholding approval.",
		HelpProvider: helpProvider,
		Commands: []plugins.Command{{
			GenericCommentHandler: handleGenericComment,
			Filter:                func(e scmprovider.GenericCommentEvent) bool { return e.Action == scm.ActionCreate },
			Help: []pluginhelp.Command{{
				Usage:       "/hold [cancel]",
				Description: "Adds or removes the `" + labels.Hold + "` Label which is used to indicate that the PR should not be automatically merged.",
				Featured:    false,
				WhoCanUse:   "Anyone can use the /hold command to add or remove the '" + labels.Hold + "' Label.",
				Examples:    []string{"/hold", "/hold cancel"},
			}},
		}},
	}
)

func init() {
	plugins.RegisterPlugin(pluginName, plugin)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	// The Config field is omitted because this plugin is not configurable.
	return &pluginhelp.PluginHelp{}, nil
}

type scmProviderClient interface {
	AddLabel(owner, repo string, number int, label string, pr bool) error
	RemoveLabel(owner, repo string, number int, label string, pr bool) error
	GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error)
}

func handleGenericComment(match []string, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
	hasLabel := func(label string, labels []*scm.Label) bool {
		return scmprovider.HasLabel(label, labels)
	}
	return handle(match, pc.SCMProviderClient, pc.Logger, &e, hasLabel)
}

// handle drives the pull request to the desired state. If any user adds
// a /hold directive, we want to add a label if one does not already exist.
// If they add /hold cancel, we want to remove the label if it exists.
func handle(_ []string, spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent, f hasLabelFunc) error {
	needsLabel := false
	if labelRe.MatchString(e.Body) {
		needsLabel = true
	} else if labelCancelRe.MatchString(e.Body) {
		needsLabel = false
	} else {
		return nil
	}

	org := e.Repo.Namespace
	repo := e.Repo.Name
	issueLabels, err := spc.GetIssueLabels(org, repo, e.Number, e.IsPR)
	if err != nil {
		return fmt.Errorf("failed to get the labels on %s/%s#%d: %v", org, repo, e.Number, err)
	}

	hasLabel := f(labels.Hold, issueLabels)
	if hasLabel && !needsLabel {
		log.Infof("Removing %q Label for %s/%s#%d", labels.Hold, org, repo, e.Number)
		return spc.RemoveLabel(org, repo, e.Number, labels.Hold, e.IsPR)
	} else if !hasLabel && needsLabel {
		log.Infof("Adding %q Label for %s/%s#%d", labels.Hold, org, repo, e.Number)
		return spc.AddLabel(org, repo, e.Number, labels.Hold, e.IsPR)
	}
	return nil
}
