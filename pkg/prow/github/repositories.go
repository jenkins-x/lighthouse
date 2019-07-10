package github

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

func (c *GitHubClient) GetRepoLabels(owner, repo string) ([]*scm.Label, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	labels, _, err := c.client.Repositories.ListLabels(ctx, fullName, c.createListOptions())
	return labels, err
}

func (c *GitHubClient) IsCollaborator(owner, repo, login string) (bool, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	flag, _, err := c.client.Repositories.IsCollaborator(ctx, fullName, login)
	return flag, err
}

func (c *GitHubClient) ListCollaborators(owner, repo string) ([]scm.User, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	resources, _, err := c.client.Repositories.ListCollaborators(ctx, fullName)
	return resources, err
}

func (c *GitHubClient) CreateStatus(owner, repo, ref string, s *scm.StatusInput) (*scm.Status, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	status, _, err := c.client.Repositories.CreateStatus(ctx, fullName, ref, s)
	return status, err
}

func (c *GitHubClient) ListStatuses(owner, repo, ref string) ([]*scm.Status, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	resources, _, err := c.client.Repositories.ListStatus(ctx, fullName, ref, c.createListOptions())
	return resources, err
}

func (c *GitHubClient) GetCombinedStatus(owner, repo, ref string) (*scm.CombinedStatus, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	resources, _, err := c.client.Repositories.FindCombinedStatus(ctx, fullName, ref)
	return resources, err
}

// HasPermission returns true if GetUserPermission() returns any of the roles.
func (c *GitHubClient) HasPermission(org, repo, user string, roles ...string) (bool, error) {
	perm, err := c.GetUserPermission(org, repo, user)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r == perm {
			return true, nil
		}
	}
	return false, nil
}

// GetUserPermission returns the user's permission level for a repo
func (c *GitHubClient) GetUserPermission(org, repo, user string) (string, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	perm, _, err := c.client.Repositories.FindUserPermission(ctx, fullName, user)
	return perm, err
}

func (c *GitHubClient) IsMember(org, user string) (bool, error) {
	panic("implement me")
}
