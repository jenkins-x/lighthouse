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

package lifecycle

import (
	"fmt"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

type scmProviderClient interface {
	IsCollaborator(owner, repo, login string) (bool, error)
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	ReopenIssue(owner, repo string, number int) error
	ReopenPR(owner, repo string, number int) error
	QuoteAuthorForComment(string) string
}

func handleReopen(spc scmProviderClient, log *logrus.Entry, e *scmprovider.GenericCommentEvent) error {
	org := e.Repo.Namespace
	repo := e.Repo.Name
	number := e.Number
	commentAuthor := e.Author.Login

	isAuthor := e.IssueAuthor.Login == commentAuthor
	isCollaborator, err := spc.IsCollaborator(org, repo, commentAuthor)
	if err != nil {
		log.WithError(err).Errorf("Failed IsCollaborator(%s, %s, %s)", org, repo, commentAuthor)
	}

	// Only authors and collaborators are allowed to reopen issues or PRs.
	if !isAuthor && !isCollaborator {
		response := "You can't reopen an issue/PR unless you authored it or you are a collaborator."
		log.Infof("Commenting \"%s\".", response)
		return spc.CreateComment(
			org,
			repo,
			number,
			true,
			plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(commentAuthor), response),
		)
	}

	if e.IsPR {
		log.Info("/reopen PR")
		if err := spc.ReopenPR(org, repo, number); err != nil {
			if scbc, ok := err.(scm.StateCannotBeChanged); ok {
				resp := fmt.Sprintf("Failed to re-open PR: %v", scbc)
				return spc.CreateComment(
					org,
					repo,
					number,
					true,
					plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(e.Author.Login), resp),
				)
			}
			return err
		}
		// Add a comment after reopening the PR to leave an audit trail of who
		// asked to reopen it.
		return spc.CreateComment(
			org,
			repo,
			number,
			true,
			plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(commentAuthor), "Reopened this PR."),
		)
	}

	log.Info("/reopen issue")
	if err := spc.ReopenIssue(org, repo, number); err != nil {
		if scbc, ok := err.(scm.StateCannotBeChanged); ok {
			resp := fmt.Sprintf("Failed to re-open Issue: %v", scbc)
			return spc.CreateComment(
				org,
				repo,
				number,
				true,
				plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(e.Author.Login), resp),
			)
		}
		return err
	}
	// Add a comment after reopening the issue to leave an audit trail of who
	// asked to reopen it.
	return spc.CreateComment(
		org,
		repo,
		number,
		true,
		plugins.FormatResponseRaw(e.Body, e.Link, spc.QuoteAuthorForComment(commentAuthor), "Reopened this issue."),
	)
}
