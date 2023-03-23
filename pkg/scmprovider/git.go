package scmprovider

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

// GetRef returns the ref from repository
func (c *Client) GetRef(owner, repo, ref string) (string, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	answer, res, err := c.client.Git.FindRef(ctx, fullName, ref)
	if err != nil {
		return "", connectErrorHandle(res, err)
	}
	return answer, nil
}

// DeleteRef deletes the ref from repository
func (c *Client) DeleteRef(owner, repo, ref string) error {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	_, err := c.client.Git.DeleteRef(ctx, fullName, ref)
	return err
}

// GetSingleCommit returns a single commit
func (c *Client) GetSingleCommit(owner, repo, SHA string) (*scm.Commit, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	commit, _, err := c.client.Git.FindCommit(ctx, fullName, SHA)
	return commit, err
}
