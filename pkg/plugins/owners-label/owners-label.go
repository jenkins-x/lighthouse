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

package ownerslabel

import (
	"fmt"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	pluginName = "owners-label"
)

func init() {
	plugins.RegisterPlugin(
		pluginName,
		plugins.Plugin{
			Description:        "The owners-label plugin automatically adds labels to PRs based on the files they touch. Specifically, the 'labels' sections of OWNERS files are used to determine which labels apply to the changes.",
			HelpProvider:       helpProvider,
			PullRequestHandler: handlePullRequest,
		},
	)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	return &pluginhelp.PluginHelp{}, nil
}

type ownersClient interface {
	FindLabelsForFile(path string) sets.String
}

type scmProviderClient interface {
	AddLabel(org, repo string, number int, label string, pr bool) error
	GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error)
	GetRepoLabels(owner, repo string) ([]*scm.Label, error)
	GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error)
}

func handlePullRequest(pc plugins.Agent, pre scm.PullRequestHook) error {
	if pre.Action != scm.ActionOpen && pre.Action != scm.ActionReopen && pre.Action != scm.ActionSync {
		return nil
	}

	oc, err := pc.OwnersClient.LoadRepoOwners(pre.Repo.Namespace, pre.Repo.Name, pre.PullRequest.Base.Ref)
	if err != nil {
		return fmt.Errorf("error loading RepoOwners: %v", err)
	}

	return handle(pc.SCMProviderClient, oc, pc.Logger, &pre)
}

func handle(spc scmProviderClient, oc ownersClient, log *logrus.Entry, pre *scm.PullRequestHook) error {
	org := pre.Repo.Namespace
	repo := pre.Repo.Name
	number := pre.PullRequest.Number

	// First see if there are any labels requested based on the files changed.
	changes, err := spc.GetPullRequestChanges(org, repo, number)
	if err != nil {
		return fmt.Errorf("error getting PR changes: %v", err)
	}
	neededLabels := sets.NewString()
	for _, change := range changes {
		neededLabels.Insert(oc.FindLabelsForFile(change.Path).List()...)
	}
	if neededLabels.Len() == 0 {
		// No labels requested for the given files. Return now to save API tokens.
		return nil
	}

	repoLabels, err := spc.GetRepoLabels(org, repo)
	if err != nil {
		return err
	}
	issuelabels, err := spc.GetIssueLabels(org, repo, number, true)
	if err != nil {
		return err
	}

	RepoLabelsExisting := sets.NewString()
	for _, label := range repoLabels {
		RepoLabelsExisting.Insert(label.Name)
	}
	currentLabels := sets.NewString()
	for _, label := range issuelabels {
		currentLabels.Insert(label.Name)
	}

	nonexistent := sets.NewString()
	for _, labelToAdd := range neededLabels.Difference(currentLabels).List() {
		if !RepoLabelsExisting.Has(labelToAdd) {
			nonexistent.Insert(labelToAdd)
			continue
		}
		if err := spc.AddLabel(org, repo, number, labelToAdd, true); err != nil {
			log.WithError(err).Errorf("GitHub failed to add the following label: %s", labelToAdd)
		}
	}

	if nonexistent.Len() > 0 {
		log.Warnf("Unable to add nonexistent labels: %q", nonexistent.List())
	}
	return nil
}
