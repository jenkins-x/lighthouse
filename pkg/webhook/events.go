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
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/v2/pkg/jxfactory"
	"github.com/jenkins-x/jx/v2/pkg/tekton/metapipeline"
	"github.com/jenkins-x/lighthouse-config/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
)

// Server keeps the information required to start a server
type Server struct {
	ClientFactory      jxfactory.Factory
	MetapipelineClient metapipeline.Client
	ClientAgent        *plugins.ClientAgent
	Plugins            *plugins.ConfigAgent
	ConfigAgent        *config.Agent
	ServerURL          *url.URL
	TokenGenerator     func() []byte
	Metrics            *Metrics

	// Tracks running handlers for graceful shutdown
	wg sync.WaitGroup
}

const failedCommentCoerceFmt = "Could not coerce %s event to a GenericCommentEvent. Unknown 'action': %q."

// HandleIssueCommentEvent handle comment events
func (s *Server) HandleIssueCommentEvent(l *logrus.Entry, ic scm.IssueCommentHook) {
	l = l.WithFields(logrus.Fields{
		scmprovider.OrgLogField:  ic.Repo.Namespace,
		scmprovider.RepoLogField: ic.Repo.Name,
		scmprovider.PrLogField:   ic.Issue.Number,
		"author":                 ic.Comment.Author.Login,
		"url":                    ic.Comment.Link,
	})
	l.Infof("Issue comment %s.", ic.Action)
	for p, h := range s.Plugins.IssueCommentHandlers(ic.Repo.Namespace, ic.Repo.Name) {
		s.wg.Add(1)
		go func(p string, h plugins.IssueCommentHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ClientFactory, s.ConfigAgent, s.Plugins, s.ClientAgent, s.MetapipelineClient, s.ServerURL, l.WithField("plugin", p))
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
		&scmprovider.GenericCommentEvent{
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
		},
	)
}

func (s *Server) handleGenericComment(l *logrus.Entry, ce *scmprovider.GenericCommentEvent) {
	for p, h := range s.Plugins.GenericCommentHandlers(ce.Repo.Namespace, ce.Repo.Name) {
		s.wg.Add(1)
		go func(p string, h plugins.GenericCommentHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ClientFactory, s.ConfigAgent, s.Plugins, s.ClientAgent, s.MetapipelineClient, s.ServerURL, l.WithField("plugin", p))
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
		scmprovider.OrgLogField:  repo.Namespace,
		scmprovider.RepoLogField: repo.Name,
		"ref":                    pe.Ref,
		"head":                   pe.After,
	})
	l.Info("Push event.")
	c := 0
	for p, h := range s.Plugins.PushEventHandlers(repo.Namespace, repo.Name) {
		s.wg.Add(1)
		c++
		go func(p string, h plugins.PushEventHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ClientFactory, s.ConfigAgent, s.Plugins, s.ClientAgent, s.MetapipelineClient, s.ServerURL, l.WithField("plugin", p))
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
	for p, h := range s.Plugins.PullRequestHandlers(repo.Namespace, repo.Name) {
		s.wg.Add(1)
		c++
		go func(p string, h plugins.PullRequestHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ClientFactory, s.ConfigAgent, s.Plugins, s.ClientAgent, s.MetapipelineClient, s.ServerURL, l.WithField("plugin", p))
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
		},
	)
}

// HandleBranchEvent handles a branch event
func (s *Server) HandleBranchEvent(entry *logrus.Entry, hook *scm.BranchHook) {
	// TODO
}

// HandleReviewEvent handles a PR review event
func (s *Server) HandleReviewEvent(l *logrus.Entry, re scm.ReviewHook) {
	l = l.WithFields(logrus.Fields{
		scmprovider.OrgLogField:  re.Repo.Namespace,
		scmprovider.RepoLogField: re.Repo.Name,
		scmprovider.PrLogField:   re.PullRequest.Number,
		"review":                 re.Review.ID,
		"reviewer":               re.Review.Author.Login,
		"url":                    re.Review.Link,
	})
	l.Infof("Review %s.", re.Action)
	for p, h := range s.Plugins.ReviewEventHandlers(re.PullRequest.Base.Repo.Namespace, re.PullRequest.Base.Repo.Name) {
		s.wg.Add(1)
		go func(p string, h plugins.ReviewEventHandler) {
			defer s.wg.Done()
			agent := plugins.NewAgent(s.ClientFactory, s.ConfigAgent, s.Plugins, s.ClientAgent, s.MetapipelineClient, s.ServerURL, l.WithField("plugin", p))
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
		},
	)
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

// externalPluginsForEvent returns whether there are any external plugins that need to
// get the present event.
func (s *Server) externalPluginsForEvent(eventKind scm.WebhookKind, srcRepo string) []plugins.ExternalPlugin {
	var matching []plugins.ExternalPlugin
	srcOrg := strings.Split(srcRepo, "/")[0]

	for repo, extPlugins := range s.Plugins.Config().ExternalPlugins {
		// Make sure the repositories match
		if repo != srcRepo && repo != srcOrg {
			continue
		}

		// Make sure the events match
		for _, p := range extPlugins {
			if len(p.Events) == 0 {
				matching = append(matching, p)
			} else {
				for _, et := range p.Events {
					if et == string(eventKind) || et == string(eventKind)+"s" {
						matching = append(matching, p)
						break
					}
				}
			}
		}
	}
	return matching
}

// callExternalPlugins dispatches the provided payload to the external plugins.
func (s *Server) callExternalPlugins(l *logrus.Entry, externalPlugins []plugins.ExternalPlugin, webhook scm.Webhook, hmacToken string) {
	headers := http.Header{}
	headers.Set("User-Agent", util.LighthouseUserAgent)
	headers.Set(util.LighthouseWebhookKindHeader, string(webhook.Kind()))
	payload, err := json.Marshal(webhook)
	if err != nil {
		l.WithError(err).Errorf("Unable to marshal webhook for relaying to external plugins. Webhook is: %v", webhook)
		return
	}
	mac := hmac.New(sha256.New, []byte(hmacToken))
	_, err = mac.Write(payload)
	if err != nil {
		l.WithError(err).Error("Unable to generate signature for relayed webhook")
		return
	}
	sum := mac.Sum(nil)
	signature := "sha256=" + hex.EncodeToString(sum)
	headers.Set(util.LighthouseSignatureHeader, signature)
	for _, p := range externalPlugins {
		s.wg.Add(1)
		go func(p plugins.ExternalPlugin) {
			defer s.wg.Done()
			if err := s.dispatch(p.Endpoint, payload, headers); err != nil {
				l.WithError(err).WithField("external-plugin", p.Name).Error("Error dispatching event to external plugin.")
			} else {
				l.WithField("external-plugin", p.Name).Info("Dispatched event to external plugin")
			}
		}(p)
	}
}

// dispatch creates a new request using the provided payload and headers
// and dispatches the request to the provided endpoint.
func (s *Server) dispatch(endpoint string, payload []byte, h http.Header) error {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header = h
	resp, err := s.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("response has status %q and body %q", resp.Status, string(rb))
	}
	return nil
}

func (s *Server) do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	backoff := 100 * time.Millisecond
	maxRetries := 5

	c := &http.Client{}
	for retries := 0; retries < maxRetries; retries++ {
		resp, err = c.Do(req)
		if err == nil {
			break
		}
		time.Sleep(backoff)
		backoff *= 2
	}
	return resp, err
}
