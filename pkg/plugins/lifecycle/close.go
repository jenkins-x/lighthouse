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

package lifecycle

import (
	"fmt"
	"regexp"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

var closeRe = regexp.MustCompile(`(?mi)^/(?:lh-)?close\s*$`)

type closeClient interface {
	IsCollaborator(owner, repo, login string) (bool, error)
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	CloseIssue(owner, repo string, number int) error
	ClosePR(owner, repo string, number int) error
	GetIssueLabels(owner, repo string, number int, pr bool) ([]*scm.Label, error)
}

func isActive(gc closeClient, org, repo string, number int, pr bool) (bool, error) {
	labels, err := gc.GetIssueLabels(org, repo, number, pr)
	if err != nil {
		return true, fmt.Errorf("list issue labels error: %v", err)
	}
	for _, label := range []string{"lifecycle/stale", "lifecycle/rotten"} {
		if scmprovider.HasLabel(label, labels) {
			return false, nil
		}
	}
	return true, nil
}

func handleClose(gc closeClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent) error {
	// Only consider open issues and new comments.
	if e.IssueState != "open" || e.Action != scm.ActionCreate {
		return nil
	}

	if !closeRe.MatchString(e.Body) {
		return nil
	}

	org := e.Repo.Namespace
	repo := e.Repo.Name
	number := e.Number
	commentAuthor := e.Author.Login

	isAuthor := e.IssueAuthor.Login == commentAuthor

	isCollaborator, err := gc.IsCollaborator(org, repo, commentAuthor)
	if err != nil {
		log.WithError(err).Errorf("Failed IsCollaborator(%s, %s, %s)", org, repo, commentAuthor)
	}

	active, err := isActive(gc, org, repo, number, e.IsPR)
	if err != nil {
		log.Infof("Cannot determine if issue is active: %v", err)
		active = true // Fail active
	}

	// Only authors and collaborators are allowed to close active issues.
	if !isAuthor && !isCollaborator && active {
		response := "You can't close an active issue/PR unless you authored it or you are a collaborator."
		log.Infof("Commenting \"%s\".", response)
		return gc.CreateComment(
			org,
			repo,
			number,
			true,
			plugins.FormatResponseRaw(e.Body, e.Link, commentAuthor, response),
		)
	}

	// Add a comment after closing the PR or issue
	// to leave an audit trail of who asked to close it.
	if e.IsPR {
		log.Info("Closing PR.")
		if err := gc.ClosePR(org, repo, number); err != nil {
			return fmt.Errorf("Error closing PR: %v", err)
		}
		response := plugins.FormatResponseRaw(e.Body, e.Link, commentAuthor, "Closed this PR.")
		return gc.CreateComment(org, repo, number, true, response)
	}

	log.Info("Closing issue.")
	if err := gc.CloseIssue(org, repo, number); err != nil {
		return fmt.Errorf("Error closing issue: %v", err)
	}
	response := plugins.FormatResponseRaw(e.Body, e.Link, commentAuthor, "Closing this issue.")
	return gc.CreateComment(org, repo, number, true, response)
}
