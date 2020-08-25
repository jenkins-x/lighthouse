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
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	lighthouseclient "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

const pluginName = "override"

var (
	overrideRe = regexp.MustCompile(`(?mi)^/(?:lh-)?override( (.+?)\s*)?$`)
)

type scmProviderClient interface {
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	CreateStatus(org, repo, ref string, s *scm.StatusInput) (*scm.Status, error)
	GetPullRequest(org, repo string, number int) (*scm.PullRequest, error)
	GetRef(org, repo, ref string) (string, error)
	HasPermission(org, repo, user string, role ...string) (bool, error)
	ListStatuses(org, repo, ref string) ([]*scm.Status, error)
	ProviderType() string
	PRRefFmt() string
	IsOrgAdmin(string, string) (bool, error)
	QuoteAuthorForComment(string) string
}

type overrideClient interface {
	scmProviderClient
	presubmitForContext(org, repo, context string) *job.Presubmit
	createOverrideJob(job *v1alpha1.LighthouseJob) (*v1alpha1.LighthouseJob, error)
}

type client struct {
	spc      scmProviderClient
	jc       job.Config
	lhClient lighthouseclient.LighthouseJobInterface
}

func (c client) createOverrideJob(job *v1alpha1.LighthouseJob) (*v1alpha1.LighthouseJob, error) {
	overrideStatus := job.Status
	createdJob, err := c.lhClient.Create(job)
	if err != nil {
		return nil, err
	}
	createdJob.Status = overrideStatus
	return c.lhClient.UpdateStatus(createdJob)
}

func (c client) ProviderType() string {
	return c.spc.ProviderType()
}

func (c client) PRRefFmt() string {
	return c.spc.PRRefFmt()
}

func (c client) IsOrgAdmin(org, user string) (bool, error) {
	return c.spc.IsOrgAdmin(org, user)
}

func (c client) CreateComment(owner, repo string, number int, pr bool, comment string) error {
	return c.spc.CreateComment(owner, repo, number, pr, comment)
}
func (c client) CreateStatus(org, repo, ref string, s *scm.StatusInput) (*scm.Status, error) {
	return c.spc.CreateStatus(org, repo, ref, s)
}

func (c client) GetRef(org, repo, ref string) (string, error) {
	return c.spc.GetRef(org, repo, ref)
}

func (c client) GetPullRequest(org, repo string, number int) (*scm.PullRequest, error) {
	return c.spc.GetPullRequest(org, repo, number)
}

func (c client) ListStatuses(org, repo, ref string) ([]*scm.Status, error) {
	return c.spc.ListStatuses(org, repo, ref)
}

func (c client) HasPermission(org, repo, user string, role ...string) (bool, error) {
	return c.spc.HasPermission(org, repo, user, role...)
}

func (c client) QuoteAuthorForComment(author string) string {
	return c.spc.QuoteAuthorForComment(author)
}

func (c client) presubmitForContext(org, repo, context string) *job.Presubmit {
	for _, p := range c.jc.AllPresubmits([]string{org + "/" + repo}) {
		if p.Context == context {
			return &p
		}
	}
	return nil
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
		Examples:    []string{"/override pull-repo-whatever", "/override ci/circleci", "/override deleted-job", "/lh-override some-job"},
	})
	return pluginHelp, nil
}

func handleGenericComment(pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
	c := client{
		spc:      pc.SCMProviderClient,
		lhClient: pc.LighthouseClient,
	}
	if pc.Config != nil {
		c.jc = pc.Config.JobConfig
	}
	return handle(c, pc.Logger, &e)
}

func authorized(spc scmProviderClient, log *logrus.Entry, org, repo, user string) bool {
	ok, err := spc.HasPermission(org, repo, user, scmprovider.RoleAdmin)
	if err != nil {
		log.WithError(err).Warnf("cannot determine whether %s is an admin of %s/%s", user, org, repo)
		return false
	}
	if !ok && spc.ProviderType() == "stash" {
		ok, err = spc.IsOrgAdmin(org, user)
		if err != nil {
			log.WithError(err).Warnf("cannot determine whether %s is an admin of %s/%s", user, org, repo)
			return false
		}
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

func handle(oc overrideClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent) error {

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
			return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, oc.QuoteAuthorForComment(user), resp))
		}
		overrides.Insert(m[2])
	}

	if !authorized(oc, log, org, repo, user) {
		resp := fmt.Sprintf("%s unauthorized: /override is restricted to repo administrators", user)
		log.Debug(resp)
		return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, oc.QuoteAuthorForComment(user), resp))
	}

	pr, err := oc.GetPullRequest(org, repo, number)
	if err != nil {
		resp := fmt.Sprintf("Cannot get PR #%d in %s/%s", number, org, repo)
		log.WithError(err).Warn(resp)
		return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, oc.QuoteAuthorForComment(user), resp))
	}

	sha := pr.Head.Sha
	statuses, err := oc.ListStatuses(org, repo, sha)
	if err != nil {
		resp := fmt.Sprintf("Cannot get commit statuses for PR #%d in %s/%s", number, org, repo)
		log.WithError(err).Warn(resp)
		return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, oc.QuoteAuthorForComment(user), resp))
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
		return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, oc.QuoteAuthorForComment(user), resp))
	}

	done := sets.String{}

	defer func() {
		if len(done) == 0 {
			return
		}
		msg := fmt.Sprintf("Overrode contexts on behalf of %s: %s", user, strings.Join(done.List(), ", "))
		log.Info(msg)
		err := oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, oc.QuoteAuthorForComment(user), msg))
		if err != nil {
			log.WithError(err).Warn("Failed to create the comment")
		}
	}()

	for _, status := range statuses {
		if status.State == scm.StateSuccess || !overrides.Has(status.Label) {
			continue
		}
		// First create the overridden prow result if necessary
		if pre := oc.presubmitForContext(org, repo, status.Label); pre != nil {
			baseSHA, err := oc.GetRef(org, repo, "heads/"+pr.Base.Ref)
			if err != nil {
				resp := fmt.Sprintf("Cannot get base ref of PR")
				log.WithError(err).Warn(resp)
				return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, oc.QuoteAuthorForComment(user), resp))
			}

			pj := jobutil.NewPresubmit(pr, baseSHA, *pre, e.GUID, oc.PRRefFmt())
			now := metav1.Now()
			pj.Status = v1alpha1.LighthouseJobStatus{
				State:          v1alpha1.SuccessState,
				Description:    description(user),
				StartTime:      now,
				CompletionTime: &now,
			}
			log.WithFields(jobutil.LighthouseJobFields(&pj)).Info("Creating a new override LighthouseJob.")
			if _, err := oc.createOverrideJob(&pj); err != nil {
				resp := fmt.Sprintf("Failed to create override job for %s", status.Label)
				log.WithError(err).Warn(resp)
				return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, oc.QuoteAuthorForComment(user), resp))
			}
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
			return oc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, oc.QuoteAuthorForComment(user), resp))
		}
		done.Insert(status.Label)
	}
	return nil
}
