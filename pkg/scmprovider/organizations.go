package scmprovider

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

// ListTeams list teams in the organisation
func (c *Client) ListTeams(org string) ([]*scm.Team, error) {
	ctx := context.Background()
	var allTeams []*scm.Team
	var resp *scm.Response
	var teams []*scm.Team
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		teams, resp, err = c.client.Organizations.ListTeams(ctx, org, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allTeams = append(allTeams, teams...)
		opts.Page++
	}
	return allTeams, nil
}

// ListTeamMembers list the team members
func (c *Client) ListTeamMembers(id int, role string) ([]*scm.TeamMember, error) {
	ctx := context.Background()
	var allMembers []*scm.TeamMember
	var resp *scm.Response
	var members []*scm.TeamMember
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		members, resp, err = c.client.Organizations.ListTeamMembers(ctx, id, role, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allMembers = append(allMembers, members...)
		opts.Page++
	}
	return allMembers, nil
}

// ListOrgMembers list the org members
func (c *Client) ListOrgMembers(org string) ([]*scm.TeamMember, error) {
	ctx := context.Background()
	var allMembers []*scm.TeamMember
	var resp *scm.Response
	var members []*scm.TeamMember
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		members, resp, err = c.client.Organizations.ListOrgMembers(ctx, org, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allMembers = append(allMembers, members...)
		opts.Page++
	}
	return allMembers, nil
}

// IsOrgAdmin returns whether this user is an admin of the org
func (c *Client) IsOrgAdmin(org, user string) (bool, error) {
	ctx := context.Background()
	ok, _, err := c.client.Organizations.IsAdmin(ctx, org, user)
	return ok, err
}
