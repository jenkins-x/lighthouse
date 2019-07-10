package github

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

func (c *GitHubClient) ListTeams(org string) ([]*scm.Team, error) {
	ctx := context.Background()
	teams, _, err := c.client.Organizations.ListTeams(ctx, org, c.createListOptions())
	return teams, err
}

func (c *GitHubClient) ListTeamMembers(id int, role string) ([]*scm.TeamMember, error) {
	ctx := context.Background()
	members, _, err := c.client.Organizations.ListTeamMembers(ctx, id, role, c.createListOptions())
	return members, err
}
