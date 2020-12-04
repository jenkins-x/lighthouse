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

package webhook

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
)

// Server keeps the information required to start a server
type Server struct {
	ClientAgent    *plugins.ClientAgent
	Plugins        *plugins.ConfigAgent
	ConfigAgent    *config.Agent
	ServerURL      *url.URL
	TokenGenerator func() []byte
	Metrics        *Metrics

	// Tracks running handlers for graceful shutdown
	wg sync.WaitGroup
}

const failedCommentCoerceFmt = "Could not coerce %s event to a GenericCommentEvent. Unknown 'action': %q."

func (s *Server) getPlugins(org, repo string) map[string]plugins.Plugin {
	return s.Plugins.GetPlugins(org, repo, s.ClientAgent.SCMProviderClient.Driver.String())
}

// handleIssueCommentEvent handle comment events
func (s *Server) handleIssueCommentEvent(l *logrus.Entry, ic scm.IssueCommentHook) {
	l = l.WithFields(logrus.Fields{
		scmprovider.OrgLogField:  ic.Repo.Namespace,
		scmprovider.RepoLogField: ic.Repo.Name,
		scmprovider.PrLogField:   ic.Issue.Number,
		"author":                 ic.Comment.Author.Login,
		"url":                    ic.Comment.Link,
	})
	l.Infof("Issue comment %s.", ic.Action)
	event := &scmprovider.GenericCommentEvent{
		GUID:        strconv.Itoa(ic.Comment.ID),
		IsPR:        ic.Issue.PullRequest,
		Action:      ic.Action,
		Body:        ic.Comment.Body,
		Link:        ic.Comment.Link,
		Number:      ic.Issue.Number,
		Repo:        ic.Repo,
		Author:      ic.Comment.Author,
		IssueAuthor: ic.Issue.Author,
		Assignees:   ic.Issue.Assignees,
		IssueState:  ic.Issue.State,
		IssueBody:   ic.Issue.Body,
		IssueLink:   ic.Issue.Link,
	}
	if ic.Issue.PullRequest {
		updatedPR, _, err := s.ClientAgent.SCMProviderClient.PullRequests.Find(context.Background(), fmt.Sprintf("%s/%s",
			ic.Repo.Namespace, ic.Repo.Name), ic.Issue.Number)
		if err != nil {
			l.WithError(err).Error("Error fetching Pull Request details.")
		} else {
			event.HeadSha = updatedPR.Head.Sha
		}
	}
	s.handleGenericComment(
		l,
		event,
	)
}

// handlePullRequestCommentEvent handles pull request comments events
func (s *Server) handlePullRequestCommentEvent(l *logrus.Entry, pc scm.PullRequestCommentHook) {
	l = l.WithFields(logrus.Fields{
		scmprovider.OrgLogField:  pc.Repo.Namespace,
		scmprovider.RepoLogField: pc.Repo.Name,
		scmprovider.PrLogField:   pc.PullRequest.Number,
		"author":                 pc.Comment.Author.Login,
		"url":                    pc.Comment.Link,
	})
	l.Infof("PR comment %s.", pc.Action)

	s.handleGenericComment(
		l,
		&scmprovider.GenericCommentEvent{
			GUID:        strconv.Itoa(pc.Comment.ID),
			IsPR:        true,
			Action:      pc.Action,
			Body:        pc.Comment.Body,
			Link:        pc.Comment.Link,
			Number:      pc.PullRequest.Number,
			Repo:        pc.Repo,
			Author:      pc.Comment.Author,
			IssueAuthor: pc.PullRequest.Author,
			Assignees:   pc.PullRequest.Assignees,
			IssueState:  pc.PullRequest.State,
			IssueBody:   pc.PullRequest.Body,
			IssueLink:   pc.PullRequest.Link,
			HeadSha:     pc.PullRequest.Head.Sha,
		},
	)
}

func (s *Server) handleGenericComment(l *logrus.Entry, ce *scmprovider.GenericCommentEvent) {
	agent, err := s.CreateAgent(l, ce.Repo.Namespace, ce.Repo.Name, ce.HeadSha)
	if err != nil {
		agent.Logger.WithError(err).Error("Error creating agent for GenericCommentEvent.")
		return
	}
	agent.InitializeCommentPruner(
		ce.Repo.Namespace,
		ce.Repo.Name,
		ce.Number,
	)
	for p, h := range s.getPlugins(ce.Repo.Namespace, ce.Repo.Name) {
		if h.GenericCommentHandler != nil {
			s.wg.Add(1)
			go func(p string, h plugins.GenericCommentHandler) {
				defer s.wg.Done()
				if err := h(agent, *ce); err != nil {
					agent.Logger.WithError(err).Error("Error handling GenericCommentEvent.")
				}
			}(p, h.GenericCommentHandler)
		}
		for _, cmd := range h.Commands {
			err := cmd.InvokeCommandHandler(ce, func(handler plugins.CommandEventHandler, e *scmprovider.GenericCommentEvent, match plugins.CommandMatch) error {
				s.wg.Add(1)
				go func(p string, h plugins.CommandEventHandler, m plugins.CommandMatch) {
					defer s.wg.Done()
					if err := h(m, agent, *ce); err != nil {
						agent.Logger.WithError(err).Error("Error handling GenericCommentEvent.")
					}
				}(p, handler, match)
				return nil
			})
			if err != nil {
				l.WithError(err).Error("Error invoking command handler")
			}
		}
	}
}

// handlePushEvent handles a push event
func (s *Server) handlePushEvent(l *logrus.Entry, pe *scm.PushHook) {
	repo := pe.Repository()
	l = l.WithFields(logrus.Fields{
		scmprovider.OrgLogField:  repo.Namespace,
		scmprovider.RepoLogField: repo.Name,
		"ref":                    pe.Ref,
		"head":                   pe.After,
	})
	l.Info("Push event.")
	c := 0
	agent, err := s.CreateAgent(l, repo.Namespace, repo.Name, pe.Ref)
	if err != nil {
		agent.Logger.WithError(err).Error("Error creating agent for PushEvent.")
		return
	}
	for p, h := range s.getPlugins(pe.Repo.Namespace, pe.Repo.Name) {
		if h.PushEventHandler != nil {
			s.wg.Add(1)
			c++
			go func(p string, h plugins.PushEventHandler) {
				defer s.wg.Done()
				if err := h(agent, *pe); err != nil {
					agent.Logger.WithError(err).Error("Error handling PushEvent.")
				}
			}(p, h.PushEventHandler)
		}
	}
	l.WithField("count", strconv.Itoa(c)).Info("number of push handlers")
}

func (s *Server) handlePullRequestEvent(l *logrus.Entry, pr *scm.PullRequestHook) {
	l = l.WithFields(logrus.Fields{
		scmprovider.OrgLogField:  pr.Repo.Namespace,
		scmprovider.RepoLogField: pr.Repo.Name,
		scmprovider.PrLogField:   pr.PullRequest.Number,
		"author":                 pr.PullRequest.Author.Login,
		"url":                    pr.PullRequest.Link,
	})
	action := pr.Action
	l.Infof("Pull request %s.", action)
	c := 0
	repo := pr.PullRequest.Base.Repo
	if repo.Name == "" {
		repo = pr.Repo
	}
	agent, err := s.CreateAgent(l, repo.Namespace, repo.Name, pr.PullRequest.Sha)
	if err != nil {
		agent.Logger.WithError(err).Error("Error creating agent for PullRequestEvent.")

		// the error could be related to a bad local triggers.yaml change so lets comment on the Pull Request
		s.reportErrorToPullRequest(l, agent, repo, pr, err)
		return
	}
	agent.InitializeCommentPruner(
		pr.Repo.Namespace,
		pr.Repo.Name,
		pr.PullRequest.Number,
	)
	for p, h := range s.getPlugins(repo.Namespace, repo.Name) {
		if h.PullRequestHandler != nil {
			s.wg.Add(1)
			c++
			go func(p string, h plugins.PullRequestHandler) {
				defer s.wg.Done()
				if err := h(agent, *pr); err != nil {
					agent.Logger.WithError(err).Error("Error handling PullRequestEvent.")
				}
			}(p, h.PullRequestHandler)
		}
	}
	l.WithField("count", strconv.Itoa(c)).Info("number of PR handlers")

	if !actionRelatesToPullRequestComment(action, l) {
		return
	}
	s.handleGenericComment(
		l,
		&scmprovider.GenericCommentEvent{
			GUID:        pr.GUID,
			IsPR:        true,
			Action:      action,
			Body:        pr.PullRequest.Body,
			Link:        pr.PullRequest.Link,
			Number:      pr.PullRequest.Number,
			Repo:        pr.Repo,
			Author:      pr.PullRequest.Author,
			IssueAuthor: pr.PullRequest.Author,
			Assignees:   pr.PullRequest.Assignees,
			IssueState:  pr.PullRequest.State,
			IssueBody:   pr.PullRequest.Body,
			IssueLink:   pr.PullRequest.Link,
			HeadSha:     pr.PullRequest.Head.Sha,
		},
	)
}

// handleBranchEvent handles a branch event
func (s *Server) handleBranchEvent(entry *logrus.Entry, hook *scm.BranchHook) {
	// TODO
}

// handleReviewEvent handles a PR review event
func (s *Server) handleReviewEvent(l *logrus.Entry, re scm.ReviewHook) {
	l = l.WithFields(logrus.Fields{
		scmprovider.OrgLogField:  re.Repo.Namespace,
		scmprovider.RepoLogField: re.Repo.Name,
		scmprovider.PrLogField:   re.PullRequest.Number,
		"review":                 re.Review.ID,
		"reviewer":               re.Review.Author.Login,
		"url":                    re.Review.Link,
	})
	l.Infof("Review %s.", re.Action)
	repo := re.PullRequest.Base.Repo
	agent, err := s.CreateAgent(l, repo.Namespace, repo.Name, re.PullRequest.Sha)
	if err != nil {
		agent.Logger.WithError(err).Error("Error creating agent for ReviewEvent.")
		return
	}
	agent.InitializeCommentPruner(
		re.Repo.Namespace,
		re.Repo.Name,
		re.PullRequest.Number,
	)
	for p, h := range s.getPlugins(re.PullRequest.Base.Repo.Namespace, re.PullRequest.Base.Repo.Name) {
		if h.ReviewEventHandler != nil {
			s.wg.Add(1)
			go func(p string, h plugins.ReviewEventHandler) {
				defer s.wg.Done()
				if err := h(agent, re); err != nil {
					agent.Logger.WithError(err).Error("Error handling ReviewEvent.")
				}
			}(p, h.ReviewEventHandler)
		}
	}

	action := re.Action
	if !actionRelatesToPullRequestComment(action, l) {
		return
	}
	s.handleGenericComment(
		l,
		&scmprovider.GenericCommentEvent{
			GUID:        re.GUID,
			IsPR:        true,
			Action:      action,
			Body:        re.Review.Body,
			Link:        re.Review.Link,
			Number:      re.PullRequest.Number,
			Repo:        re.Repo,
			Author:      re.Review.Author,
			IssueAuthor: re.PullRequest.Author,
			Assignees:   re.PullRequest.Assignees,
			IssueState:  re.PullRequest.State,
			IssueBody:   re.PullRequest.Body,
			IssueLink:   re.PullRequest.Link,
			HeadSha:     re.PullRequest.Head.Sha,
		},
	)
}

func (s *Server) reportErrorToPullRequest(l *logrus.Entry, agent plugins.Agent, repo scm.Repository, pr *scm.PullRequestHook, err error) {
	fileLink := repo.Link + "/blob/" + pr.PullRequest.Sha + "/"
	message := "failed to trigger Pull Request pipeline\n" + util.ErrorToMarkdown(err, fileLink)

	err = agent.SCMProviderClient.CreateComment(repo.Namespace, repo.Name, pr.PullRequest.Number, true, message)
	if err != nil {
		l.WithError(err).Errorf("failed to comment the failure on Pull Request")
	}
}

func actionRelatesToPullRequestComment(action scm.Action, l *logrus.Entry) bool {
	switch action {

	case scm.ActionCreate, scm.ActionOpen, scm.ActionSubmitted, scm.ActionEdited, scm.ActionDelete, scm.ActionDismissed, scm.ActionUpdate:
		return true

	case scm.ActionAssigned,
		scm.ActionUnassigned,
		scm.ActionReviewRequested,
		scm.ActionReviewRequestRemoved,
		scm.ActionLabel,
		scm.ActionUnlabel,
		scm.ActionClose,
		scm.ActionReopen,
		scm.ActionSync:
		return false

	default:
		l.Errorf(failedCommentCoerceFmt, "pull_request", action.String())
		return false
	}
}
