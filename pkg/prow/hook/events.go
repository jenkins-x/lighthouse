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

package hook

import (
	"strconv"
	"sync"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/prow/github"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
)

// Server keeps the information required to start a server
type Server struct {
	ClientAgent    *plugins.ClientAgent
	Plugins        *plugins.ConfigAgent
	ConfigAgent    *config.Agent
	TokenGenerator func() []byte
	Metrics        *Metrics

	// Tracks running handlers for graceful shutdown
	wg sync.WaitGroup
}

const failedCommentCoerceFmt = "Could not coerce %s event to a GenericCommentEvent. Unknown 'action': %q."

// HandleIssueCommentEvent handle comment events
func (s *Server) HandleIssueCommentEvent(l *logrus.Entry, ic scm.IssueCommentHook) {
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  ic.Repo.Namespace,
		github.RepoLogField: ic.Repo.Name,
		github.PrLogField:   ic.Issue.Number,
		"author":            ic.Comment.Author.Login,
		"url":               ic.Comment.Link,
	})
	l.Infof("Issue comment %s.", ic.Action)
	for p, h := range s.Plugins.IssueCommentHandlers(ic.Repo.Namespace, ic.Repo.Name) {
		s.wg.Add(1)
		go func(p string, h plugins.IssueCommentHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, l.WithField("plugin", p))
			agent.InitializeCommentPruner(
				ic.Repo.Namespace,
				ic.Repo.Name,
				ic.Issue.Number,
			)
			if err := h(agent, ic); err != nil {
				agent.Logger.WithError(err).Error("Error handling IssueCommentEvent.")
			}
		}(p, h)
	}

	s.handleGenericComment(
		l,
		&github.GenericCommentEvent{
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
		},
	)
}

// HandlePullRequestCommentEvent handles pull request comments events
func (s *Server) HandlePullRequestCommentEvent(l *logrus.Entry, pc scm.PullRequestCommentHook) {
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  pc.Repo.Namespace,
		github.RepoLogField: pc.Repo.Name,
		github.PrLogField:   pc.PullRequest.Number,
		"author":            pc.Comment.Author.Login,
		"url":               pc.Comment.Link,
	})
	l.Infof("PR comment %s.", pc.Action)

	/*	for p, h := range s.Plugins.IssueCommentHandlers(pc.Repo.Namespace, pc.Repo.Name) {
			s.wg.Add(1)
			go func(p string, h plugins.IssueCommentHandler) {
				defer s.wg.Done()
				agent := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, l.WithField("plugin", p))
				agent.InitializeCommentPruner(
					pc.Repo.Namespace,
					pc.Repo.Name,
					pc.PullRequest.Number,
				)
				if err := h(agent, pc); err != nil {
					agent.Logger.WithError(err).Error("Error handling IssueCommentEvent.")
				}
			}(p, h)
		}
	*/

	s.handleGenericComment(
		l,
		&github.GenericCommentEvent{
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
		},
	)
}

func (s *Server) handleGenericComment(l *logrus.Entry, ce *github.GenericCommentEvent) {
	for p, h := range s.Plugins.GenericCommentHandlers(ce.Repo.Namespace, ce.Repo.Name) {
		s.wg.Add(1)
		go func(p string, h plugins.GenericCommentHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, l.WithField("plugin", p))
			agent.InitializeCommentPruner(
				ce.Repo.Namespace,
				ce.Repo.Name,
				ce.Number,
			)
			if err := h(agent, *ce); err != nil {
				agent.Logger.WithError(err).Error("Error handling GenericCommentEvent.")
			}
		}(p, h)
	}
}

// HandlePushEvent handles a push event
func (s *Server) HandlePushEvent(l *logrus.Entry, pe *scm.PushHook) {
	repo := pe.Repository()
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  repo.Namespace,
		github.RepoLogField: repo.Name,
		"ref":               pe.Ref,
		"head":              pe.After,
	})
	l.Info("Push event.")
	c := 0
	for p, h := range s.Plugins.PushEventHandlers(repo.Namespace, repo.Name) {
		s.wg.Add(1)
		c++
		go func(p string, h plugins.PushEventHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, l.WithField("plugin", p))
			if err := h(agent, *pe); err != nil {
				agent.Logger.WithError(err).Error("Error handling PushEvent.")
			}
		}(p, h)
	}
	l.WithField("count", strconv.Itoa(c)).Info("number of push handlers")
}

// HandlePullRequestEvent handles a pull request event
func (s *Server) HandlePullRequestEvent(l *logrus.Entry, pr *scm.PullRequestHook) {
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  pr.Repo.Namespace,
		github.RepoLogField: pr.Repo.Name,
		github.PrLogField:   pr.PullRequest.Number,
		"author":            pr.PullRequest.Author.Login,
		"url":               pr.PullRequest.Link,
	})
	action := pr.Action
	l.Infof("Pull request %s.", action)
	c := 0
	repo := pr.PullRequest.Base.Repo
	if repo.Name == "" {
		repo = pr.Repo
	}
	for p, h := range s.Plugins.PullRequestHandlers(repo.Namespace, repo.Name) {
		s.wg.Add(1)
		c++
		go func(p string, h plugins.PullRequestHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, l.WithField("plugin", p))
			agent.InitializeCommentPruner(
				pr.Repo.Namespace,
				pr.Repo.Name,
				pr.PullRequest.Number,
			)
			if err := h(agent, *pr); err != nil {
				agent.Logger.WithError(err).Error("Error handling PullRequestEvent.")
			}
		}(p, h)
	}
	l.WithField("count", strconv.Itoa(c)).Info("number of PR handlers")

	if !actionRelatesToPullRequestComment(action, l) {
		return
	}
	s.handleGenericComment(
		l,
		&github.GenericCommentEvent{
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
		},
	)
}

// HandleBranchEvent handles a branch event
func (s *Server) HandleBranchEvent(entry *logrus.Entry, hook *scm.BranchHook) {
	// TODO
}

func actionRelatesToPullRequestComment(action scm.Action, l *logrus.Entry) bool {
	switch action {

	case scm.ActionCreate, scm.ActionOpen, scm.ActionSubmitted, scm.ActionEdited, scm.ActionDelete, scm.ActionDismissed:
		return true

	/*
		nonCommentPullRequestActions = map[*scm.PullRequestHookAction]bool{
			github.PullRequestActionAssigned:             true,
			github.PullRequestActionUnassigned:           true,
			github.PullRequestActionReviewRequested:      true,
			github.PullRequestActionReviewRequestRemoved: true,
			github.PullRequestActionLabeled:              true,
			github.PullRequestActionUnlabeled:            true,
			github.PullRequestActionClosed:               true,
			github.PullRequestActionReopened:             true,
			github.PullRequestActionSynchronize:          true,
		}
	*/
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
		l.Errorf(failedCommentCoerceFmt, "pull_request", string(action))
		return false
	}
}

/*
func (s *Server) handleReviewEvent(l *logrus.Entry, re github.ReviewEvent) {
	defer s.wg.Done()
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  re.Repo.Namespace,
		github.RepoLogField: re.Repo.Name,
		github.PrLogField:   re.PullRequest.Number,
		"review":            re.Review.ID,
		"reviewer":          re.Review.Author.Login,
		"url":               re.Review.Link,
	})
	l.Infof("Review %s.", re.Action)
	for p, h := range s.Plugins.ReviewEventHandlers(re.PullRequest.Base.Repo.Namespace, re.PullRequest.Base.Repo.Name) {
		s.wg.Add(1)
		go func(p string, h plugins.ReviewEventHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, l.WithField("plugin", p))
			agent.InitializeCommentPruner(
				re.Repo.Namespace,
				re.Repo.Name,
				re.PullRequest.Number,
			)
			if err := h(agent, re); err != nil {
				agent.Logger.WithError(err).Error("Error handling ReviewEvent.")
			}
		}(p, h)
	}
	action := genericCommentAction(re.Action))
	if action == "" {
		l.Errorf(failedCommentCoerceFmt, "pull_request_review", string(re.Action))
		return
	}
	s.handleGenericComment(
		l,
		&github.GenericCommentEvent{
			GUID:         re.GUID,
			IsPR:         true,
			Action:       action,
			Body:         re.Review.Body,
			Link:      re.Review.Link,
			Number:       re.PullRequest.Number,
			Repo:         re.Repo,
			Author:         re.Review.Author,
			IssueAuthor:  re.PullRequest.Author,
			Assignees:    re.PullRequest.Assignees,
			IssueState:   re.PullRequest.State,
			IssueBody:    re.PullRequest.Body,
			IssueLink: re.PullRequest.Link,
		},
	)
}

func (s *Server) handleReviewCommentEvent(l *logrus.Entry, rce scm.ReviewEvent) {
	defer s.wg.Done()
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  rce.Repo.Namespace,
		github.RepoLogField: rce.Repo.Name,
		github.PrLogField:   rce.PullRequest.Number,
		"review":            rce.Comment.ReviewID,
		"commenter":         rce.Comment.Author.Login,
		"url":               rce.Comment.Link,
	})
	l.Infof("Review comment %s.", rce.Action)
	for p, h := range s.Plugins.ReviewCommentEventHandlers(rce.PullRequest.Base.Repo.Namespace, rce.PullRequest.Base.Repo.Name) {
		s.wg.Add(1)
		go func(p string, h plugins.ReviewCommentEventHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, l.WithField("plugin", p))
			agent.InitializeCommentPruner(
				rce.Repo.Namespace,
				rce.Repo.Name,
				rce.PullRequest.Number,
			)
			if err := h(agent, rce); err != nil {
				agent.Logger.WithError(err).Error("Error handling ReviewCommentEvent.")
			}
		}(p, h)
	}
	action := genericCommentAction(rce.Action))
	if action == "" {
		l.Errorf(failedCommentCoerceFmt, "pull_request_review_comment", string(rce.Action))
		return
	}
	s.handleGenericComment(
		l,
		&github.GenericCommentEvent{
			GUID:         rce.GUID,
			IsPR:         true,
			Action:       action,
			Body:         rce.Comment.Body,
			Link:      rce.Comment.Link,
			Number:       rce.PullRequest.Number,
			Repo:         rce.Repo,
			Author:         rce.Comment.Author,
			IssueAuthor:  rce.PullRequest.Author,
			Assignees:    rce.PullRequest.Assignees,
			IssueState:   rce.PullRequest.State,
			IssueBody:    rce.PullRequest.Body,
			IssueLink: rce.PullRequest.Link,
		},
	)
}



func (s *Server) handleIssueEvent(l *logrus.Entry, i github.IssueEvent) {
	defer s.wg.Done()
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  i.Repo.Namespace,
		github.RepoLogField: i.Repo.Name,
		github.PrLogField:   i.Issue.Number,
		"author":            i.Issue.Author.Login,
		"url":               i.Issue.Link,
	})
	l.Infof("Issue %s.", i.Action)
	for p, h := range s.Plugins.IssueHandlers(i.Repo.Namespace, i.Repo.Name) {
		s.wg.Add(1)
		go func(p string, h plugins.IssueHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, l.WithField("plugin", p))
			agent.InitializeCommentPruner(
				i.Repo.Namespace,
				i.Repo.Name,
				i.Issue.Number,
			)
			if err := h(agent, i); err != nil {
				agent.Logger.WithError(err).Error("Error handling IssueEvent.")
			}
		}(p, h)
	}
	action := genericCommentAction(i.Action))
	if action == "" {
		if !nonCommentIssueActions[i.Action] {
			l.Errorf(failedCommentCoerceFmt, "pull_request", string(i.Action))
		}
		return
	}
	s.handleGenericComment(
		l,
		&github.GenericCommentEvent{
			GUID:         i.GUID,
			IsPR:         i.Issue.IsPullRequest(),
			Action:       action,
			Body:         i.Issue.Body,
			Link:      i.Issue.Link,
			Number:       i.Issue.Number,
			Repo:         i.Repo,
			Author:         i.Issue.Author,
			IssueAuthor:  i.Issue.Author,
			Assignees:    i.Issue.Assignees,
			IssueState:   i.Issue.State,
			IssueBody:    i.Issue.Body,
			IssueLink: i.Issue.Link,
		},
	)
}


func (s *Server) handleStatusEvent(l *logrus.Entry, se scm.StateEvent) {
	defer s.wg.Done()
	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  se.Repo.Namespace,
		github.RepoLogField: se.Repo.Name,
		"context":           se.Context,
		"sha":               se.SHA,
		"state":             se.State,
		"id":                se.ID,
	})
	l.Infof("Status description %s.", se.Description)
	for p, h := range s.Plugins.StatusEventHandlers(se.Repo.Namespace, se.Repo.Name) {
		s.wg.Add(1)
		go func(p string, h plugins.StatusEventHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, l.WithField("plugin", p))
			if err := h(agent, se); err != nil {
				agent.Logger.WithError(err).Error("Error handling StatusEvent.")
			}
		}(p, h)
	}
}

*/
