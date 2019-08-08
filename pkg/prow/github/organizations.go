package github

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

// ListTeams list teams in the organisation
func (c *Client) ListTeams(org string) ([]*scm.Team, error) {
	ctx := context.Background()
	teams, _, err := c.client.Organizations.ListTeams(ctx, org, c.createListOptions())
	return teams, err
}

// ListTeamMembers list the team members
func (c *Client) ListTeamMembers(id int, role string) ([]*scm.TeamMember, error) {
	ctx := context.Background()
	members, _, err := c.client.Organizations.ListTeamMembers(ctx, id, role, c.createListOptions())
	return members, err
}
