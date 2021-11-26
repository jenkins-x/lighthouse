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
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	lighthouseclient "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
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

func createOverrideJob(lhClient lighthouseclient.LighthouseJobInterface, job *v1alpha1.LighthouseJob) (*v1alpha1.LighthouseJob, error) {
	overrideStatus := job.Status
	createdJob, err := lhClient.Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	createdJob.Status = overrideStatus
	return lhClient.UpdateStatus(context.TODO(), createdJob, metav1.UpdateOptions{})
}

func presubmitForContext(jc config.JobConfig, org, repo, context string) *job.Presubmit {
	for _, p := range jc.AllPresubmits([]string{org + "/" + repo}) {
		if p.Context == context {
			return &p
		}
	}
	return nil
}

var (
	plugin = plugins.Plugin{
		Description: "The override plugin allows repo admins to force a github status context to pass",
		Commands: []plugins.Command{{
			Name: "override",
			Arg: &plugins.CommandArg{
				Pattern: `[^\r\n]+`,
			},
			Description: "Forces a github status context to green (one per line).",
			WhoCanUse:   "Repo administrators",
			Action: plugins.
				Invoke(func(match plugins.CommandMatch, pc plugins.Agent, e scmprovider.GenericCommentEvent) error {
					return handle(match.Arg, pc.SCMProviderClient, pc.LighthouseClient, pc.Config.JobConfig, pc.Logger, e)
				}).
				When(plugins.Action(scm.ActionCreate), plugins.IsPR(), plugins.IssueState("open")),
		}},
	}
)

func init() {
	plugins.RegisterPlugin(pluginName, plugin)
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

func handle(context string, spc scmProviderClient, lhClient lighthouseclient.LighthouseJobInterface, jc config.JobConfig, log *logrus.Entry, e scmprovider.GenericCommentEvent) error {
	org := e.Repo.Namespace
	repo := e.Repo.Name
	number := e.Number
	user := e.Author.Login

	overrides := sets.NewString()
	if context == "" {
		resp := "/override requires a failed status context to operate on, but none was given"
		log.Debug(resp)
		return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(user), resp))
	}
	overrides.Insert(context)

	if !authorized(spc, log, org, repo, user) {
		resp := fmt.Sprintf("%s unauthorized: /override is restricted to repo administrators", user)
		log.Debug(resp)
		return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(user), resp))
	}

	pr, err := spc.GetPullRequest(org, repo, number)
	if err != nil {
		resp := fmt.Sprintf("Cannot get PR #%d in %s/%s", number, org, repo)
		log.WithError(err).Warn(resp)
		return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(user), resp))
	}

	sha := pr.Head.Sha
	statuses, err := spc.ListStatuses(org, repo, sha)
	if err != nil {
		resp := fmt.Sprintf("Cannot get commit statuses for PR #%d in %s/%s", number, org, repo)
		log.WithError(err).Warn(resp)
		return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(user), resp))
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
		return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(user), resp))
	}

	done := sets.String{}

	defer func() {
		if len(done) == 0 {
			return
		}
		msg := fmt.Sprintf("Overrode contexts on behalf of %s: %s", user, strings.Join(done.List(), ", "))
		log.Info(msg)
		err := spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(user), msg))
		if err != nil {
			log.WithError(err).Warn("Failed to create the comment")
		}
	}()

	for _, status := range statuses {
		if status.State == scm.StateSuccess || !overrides.Has(status.Label) {
			continue
		}
		// First create the overridden prow result if necessary
		if pre := presubmitForContext(jc, org, repo, status.Label); pre != nil {
			baseSHA, err := spc.GetRef(org, repo, "heads/"+pr.Base.Ref)
			if err != nil {
				resp := fmt.Sprintf("Cannot get base ref of PR")
				log.WithError(err).Warn(resp)
				return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(user), resp))
			}

			pj, _ := jobutil.NewPresubmit(log, pr, baseSHA, *pre, e.GUID, spc.PRRefFmt())
			now := metav1.Now()
			pj.Status = v1alpha1.LighthouseJobStatus{
				State:          v1alpha1.SuccessState,
				Description:    description(user),
				StartTime:      now,
				CompletionTime: &now,
			}
			log.WithFields(jobutil.LighthouseJobFields(&pj)).Info("Creating a new override LighthouseJob.")
			if _, err := createOverrideJob(lhClient, &pj); err != nil {
				resp := fmt.Sprintf("Failed to create override job for %s", status.Label)
				log.WithError(err).Warn(resp)
				return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(user), resp))
			}
		}
		statusInput := &scm.StatusInput{
			State:  scm.StateSuccess,
			Label:  status.Label,
			Target: status.Target,
			Desc:   description(user),
		}
		if _, err := spc.CreateStatus(org, repo, sha, statusInput); err != nil {
			resp := fmt.Sprintf("Cannot update PR status for context %s", statusInput.Label)
			log.WithError(err).Warn(resp)
			return spc.CreateComment(org, repo, number, e.IsPR, plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(user), resp))
		}
		done.Insert(status.Label)
	}
	return nil
}
