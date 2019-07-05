package github

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/drone/go-scm/scm"
	"github.com/pkg/errors"
)

// ToGitHubClient converts the scm client to an API that the prow plgins expect
func ToGitHubClient(client *scm.Client) *GitHubClient {
	return &GitHubClient{client}
}

// GitHubClient represents an interface that prow plugins expect on top of go-scm
type GitHubClient struct {
	client *scm.Client
}

func (c *GitHubClient) DeleteRef(owner, repo, ref string) error {
	panic("implement me")
}

func (c *GitHubClient) Query(context.Context, interface{}, map[string]interface{}) error {
	panic("implement me")
}

func (c *GitHubClient) AssignIssue(owner, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) UnassignIssue(owner, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) RequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) UnrequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) GetPullRequest(org, repo string, number int) (*scm.PullRequest, error) {
	panic("implement me")
}

func (c *GitHubClient) ListIssueComments(org, repo string, number int) ([]scm.Comment, error) {
	panic("implement me")
}

func (c *GitHubClient) ListReviews(org, repo string, number int) ([]scm.Review, error) {
	panic("implement me")
}

func (c *GitHubClient) ListPullRequestComments(org, repo string, number int) ([]scm.Review, error) {
	panic("implement me")
}

func (c *GitHubClient) DeleteComment(org, repo string, ID int) error {
	panic("implement me")
}

func (c *GitHubClient) BotName() (string, error) {
	panic("implement me")
}

func (c *GitHubClient) ListIssueEvents(org, repo string, num int) ([]ListedIssueEvent, error) {
	panic("implement me")
}

func (c *GitHubClient) AddLabel(owner, repo string, number int, label string) error {
	panic("implement me")
}

func (c *GitHubClient) RemoveLabel(owner, repo string, number int, label string) error {
	panic("implement me")
}

func (c *GitHubClient) GetIssueLabels(org, repo string, number int) ([]scm.Label, error) {
	panic("implement me")
}

func (c *GitHubClient) GetPullRequestChanges(org, repo string, number int) ([]PullRequestChange, error) {
	panic("implement me")
}

func (c *GitHubClient) CreateComment(owner, repo string, number int, comment string) error {
	fullName := fmt.Sprintf("%s/%s", owner, repo)
	commentInput := scm.CommentInput{
		Body: comment,
	}
	_, response, err := c.client.Issues.CreateComment(context.Background(), fullName, number, &commentInput)
	if err != nil {
		var b bytes.Buffer
		io.Copy(&b, response.Body)
		return errors.Wrapf(err, "response: %s", b.String())
	}
	return nil
}
