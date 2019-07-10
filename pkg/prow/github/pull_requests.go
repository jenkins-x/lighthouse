package github

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

func (c *GitHubClient) GetPullRequest(owner, repo string, number int) (*scm.PullRequest, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	pr, _, err := c.client.PullRequests.Find(ctx, fullName, number)
	return pr, err
}

func (c *GitHubClient) ListPullRequestComments(owner, repo string, number int) ([]*scm.Comment, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	pr, _, err := c.client.PullRequests.ListComments(ctx, fullName, number, c.createListOptions())
	return pr, err
}

func (c *GitHubClient) GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	changes, _, err := c.client.PullRequests.ListChanges(ctx, fullName, number, c.createListOptions())
	return changes, err
}

func (c *GitHubClient) ReopenPR(owner, repo string, number int) error {
	panic("implement me")
}

func (c *GitHubClient) ClosePR(owner, repo string, number int) error {
	panic("implement me")
}
