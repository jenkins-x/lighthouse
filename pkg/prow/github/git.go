package github

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

func (c *GitHubClient) GetRef(owner, repo, ref string) (string, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	answer, _, err := c.client.Git.FindRef(ctx, fullName, ref)
	return answer, err
}

func (c *GitHubClient) DeleteRef(owner, repo, ref string) error {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	_, err := c.client.Git.DeleteRef(ctx, fullName, ref)
	return err
}

func (c *GitHubClient) GetSingleCommit(owner, repo, SHA string) (*scm.Commit, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	commit, _, err := c.client.Git.FindCommit(ctx, fullName, SHA)
	return commit, err
}
