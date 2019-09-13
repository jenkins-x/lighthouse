package gitprovider

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

// MergeDetails optional extra parameters
type MergeDetails struct {
	SHA         string
	MergeMethod string
	CommitTitle string
}

// GetPullRequest returns the pull request
func (c *Client) GetPullRequest(owner, repo string, number int) (*scm.PullRequest, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	pr, _, err := c.client.PullRequests.Find(ctx, fullName, number)
	return pr, err
}

// ListPullRequestComments list pull request comments
func (c *Client) ListPullRequestComments(owner, repo string, number int) ([]*scm.Comment, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	pr, _, err := c.client.PullRequests.ListComments(ctx, fullName, number, c.createListOptions())
	return pr, err
}

// GetPullRequestChanges returns the changes in a pull request
func (c *Client) GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	changes, _, err := c.client.PullRequests.ListChanges(ctx, fullName, number, c.createListOptions())
	return changes, err
}

// Merge reopens a pull request
func (c *Client) Merge(owner, repo string, number int, details MergeDetails) error {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	_, err := c.client.PullRequests.Merge(ctx, fullName, number)
	return err
}

// ReopenPR reopens a pull request
func (c *Client) ReopenPR(owner, repo string, number int) error {
	panic("implement me")
}

// ClosePR closes a pull request
func (c *Client) ClosePR(owner, repo string, number int) error {
	panic("implement me")
}
