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

// Package override supports the /override context command.
package override

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/gitprovider"
	"github.com/jenkins-x/lighthouse/pkg/prow/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
)

const pluginName = "override"

var (
	overrideRe = regexp.MustCompile(`(?mi)^/override( (.+?)\s*)?$`)
)

type githubClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	CreateStatus(org, repo, ref string, s *scm.StatusInput) (*scm.Status, error)
	GetPullRequest(org, repo string, number int) (*scm.PullRequest, error)
	GetRef(org, repo, ref string) (string, error)
	HasPermission(org, repo, user string, role ...string) (bool, error)
	ListStatuses(org, repo, ref string) ([]*scm.Status, error)
}

type overrideClient interface {
	githubClient
}

type client struct {
	gc                 githubClient
	jc                 config.JobConfig
	clientFactory      jxfactory.Factory
	metapipelineClient metapipeline.Client
}

func (c client) CreateComment(owner, repo string, number int, pr bool, comment string) error {
	return c.gc.CreateComment(owner, repo, number, pr, comment)
}
func (c client) CreateStatus(org, repo, ref string, s *scm.StatusInput) (*scm.Status, error) {
	return c.gc.CreateStatus(org, repo, ref, s)
}

func (c client) GetRef(org, repo, ref string) (string, error) {
	return c.gc.GetRef(org, repo, ref)
}

func (c client) GetPullRequest(org, repo string, number int) (*scm.PullRequest, error) {
	return c.gc.GetPullRequest(org, repo, number)
}
func (c client) ListStatuses(org, repo, ref string) ([]*scm.Status, error) {
	return c.gc.ListStatuses(org, repo, ref)
}
func (c client) HasPermission(org, repo, user string, role ...string) (bool, error) {
	return c.gc.HasPermission(org, repo, user, role...)
}

func init() {
	plugins.RegisterGenericCommentHandler(pluginName, handleGenericComment, helpProvider)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The override plugin allows repo admins to force a github status context to pass",
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/override [context]",
		Description: "Forces a github status context to green (one per line).",
		Featured:    false,
		WhoCanUse:   "Repo administrators",
		Examples:    []string{"/override pull-repo-whatever", "/override ci/circleci", "/override deleted-job"},
	})
	return pluginHelp, nil
}

func handleGenericComment(pc plugins.Agent, e gitprovider.GenericCommentEvent) error {
	c := client{
		gc:                 pc.GitHubClient,
		jc:                 pc.Config.JobConfig,
		clientFactory:      pc.ClientFactory,
		metapipelineClient: pc.MetapipelineClient,
	}
	return handle(pc.ClientFactory, c, pc.Logger, &e)
}

func authorized(gc githubClient, log *logrus.Entry, org, repo, user string) bool {
	ok, err := gc.HasPermission(org, repo, user, gitprovider.RoleAdmin)
	if err != nil {
		log.WithError(err).Warnf("cannot determine whether %s is an admin of %s/%s", user, org, repo)
		return false
	}
	return ok
}

func description(user string) string {
	return fmt.Sprintf("%s %s", util.OverriddenByPrefix, user)
}

func formatList(list []string) string {
	var lines []string
	for _, item := range list {
		lines = append(lines, fmt.Sprintf(" - `%s`", item))
	}
	return strings.Join(lines, "\n")
}

func handle(clientFactory jxfactory.Factory, oc overrideClient, log *logrus.Entry, e *gitprovider.GenericCommentEvent) error {

	if !e.IsPR || e.IssueState != "open" || e.Action != scm.ActionCreate {
		return nil
	}

	mat := overrideRe.FindAllStringSubmatch(e.Body, -1)
	if len(mat) == 0 {
		return nil // no /override commands given in the comment
	}

	org := e.Repo.Namespace
	repo := e.Repo.Name
	number := e.Number
	user := e.Author.Login

	overrides := sets.NewString()
	for _, m := range mat {
		if m[1] == "" {
			resp := "/override requires a failed status context to operate on, but none was given"
			log.Debug(resp)
			return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, user, resp))
		}
		overrides.Insert(m[2])
	}

	if !authorized(oc, log, org, repo, user) {
		resp := fmt.Sprintf("%s unauthorized: /override is restricted to repo administrators", user)
		log.Debug(resp)
		return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, user, resp))
	}

	pr, err := oc.GetPullRequest(org, repo, number)
	if err != nil {
		resp := fmt.Sprintf("Cannot get PR #%d in %s/%s", number, org, repo)
		log.WithError(err).Warn(resp)
		return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, user, resp))
	}

	sha := pr.Head.Sha
	statuses, err := oc.ListStatuses(org, repo, sha)
	if err != nil {
		resp := fmt.Sprintf("Cannot get commit statuses for PR #%d in %s/%s", number, org, repo)
		log.WithError(err).Warn(resp)
		return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, user, resp))
	}

	contexts := sets.NewString()
	for _, status := range statuses {
		if status.State == scm.StateSuccess {
			continue
		}
		contexts.Insert(status.Label)
	}
	if unknown := overrides.Difference(contexts); unknown.Len() > 0 {
		resp := fmt.Sprintf(`/override requires a failed status context to operate on.
The following unknown contexts were given:
%s

Only the following contexts were expected:
%s`, formatList(unknown.List()), formatList(contexts.List()))
		log.Debug(resp)
		return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, user, resp))
	}

	done := sets.String{}

	defer func() {
		if len(done) == 0 {
			return
		}
		msg := fmt.Sprintf("Overrode contexts on behalf of %s: %s", user, strings.Join(done.List(), ", "))
		log.Info(msg)
		err := oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, user, msg))
		if err != nil {
			log.WithError(err).Warn("Failed to create the comment")
		}
	}()

	for _, status := range statuses {
		if status.State == scm.StateSuccess || !overrides.Has(status.Label) {
			continue
		}
		statusInput := &scm.StatusInput{
			State:  scm.StateSuccess,
			Label:  status.Label,
			Target: status.Target,
			Desc:   description(user),
		}
		if _, err := oc.CreateStatus(org, repo, sha, statusInput); err != nil {
			resp := fmt.Sprintf("Cannot update PR status for context %s", statusInput.Label)
			log.WithError(err).Warn(resp)
			return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, user, resp))
		}
		done.Insert(status.Label)
	}
	return nil
}
