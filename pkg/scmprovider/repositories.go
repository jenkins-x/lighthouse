package scmprovider

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

// GetRepoLabels returns the repository labels
func (c *Client) GetRepoLabels(owner, repo string) ([]*scm.Label, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	var allLabels []*scm.Label
	var resp *scm.Response
	var labels []*scm.Label
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		labels, resp, err = c.client.Repositories.ListLabels(ctx, fullName, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allLabels = append(allLabels, labels...)
		opts.Page++
	}
	return allLabels, nil
}

// IsCollaborator check if a user is collaborator to a repository
func (c *Client) IsCollaborator(owner, repo, login string) (bool, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	flag, _, err := c.client.Repositories.IsCollaborator(ctx, fullName, login)
	return flag, err
}

// ListCollaborators list the collaborators to a repository
func (c *Client) ListCollaborators(owner, repo string) ([]scm.User, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	var allCollabs []scm.User
	var resp *scm.Response
	var collabs []scm.User
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		collabs, resp, err = c.client.Repositories.ListCollaborators(ctx, fullName, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allCollabs = append(allCollabs, collabs...)
		opts.Page++
	}
	return allCollabs, nil
}

// CreateStatus create a status into a repository
func (c *Client) CreateStatus(owner, repo, ref string, s *scm.StatusInput) (*scm.Status, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	status, _, err := c.client.Repositories.CreateStatus(ctx, fullName, ref, s)
	return status, err
}

// CreateGraphQLStatus create a status into a repository
func (c *Client) CreateGraphQLStatus(owner, repo, ref string, s *Status) (*scm.Status, error) {
	si := &scm.StatusInput{
		State:  scm.ToState(s.State),
		Label:  s.Context,
		Desc:   s.Description,
		Target: s.TargetURL,
	}
	return c.CreateStatus(owner, repo, ref, si)
}

// Status is used to set a commit status line.
type Status struct {
	State       string `json:"state"`
	TargetURL   string `json:"target_url,omitempty"`
	Description string `json:"description,omitempty"`
	Context     string `json:"context,omitempty"`
}

// ListStatuses list the statuses
func (c *Client) ListStatuses(owner, repo, ref string) ([]*scm.Status, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	var allStatuses []*scm.Status
	var resp *scm.Response
	var statuses []*scm.Status
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		statuses, resp, err = c.client.Repositories.ListStatus(ctx, fullName, ref, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allStatuses = append(allStatuses, statuses...)
		opts.Page++
	}
	return allStatuses, nil
}

// GetCombinedStatus returns the combined status
func (c *Client) GetCombinedStatus(owner, repo, ref string) (*scm.CombinedStatus, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	resources, _, err := c.client.Repositories.FindCombinedStatus(ctx, fullName, ref)
	return resources, err
}

// HasPermission returns true if GetUserPermission() returns any of the roles.
func (c *Client) HasPermission(org, repo, user string, roles ...string) (bool, error) {
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
func (c *Client) GetUserPermission(org, repo, user string) (string, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	perm, _, err := c.client.Repositories.FindUserPermission(ctx, fullName, user)
	return perm, err
}

// IsMember checks if a user is a member of the organisation
func (c *Client) IsMember(org, user string) (bool, error) {
	ctx := context.Background()
	member, _, err := c.client.Organizations.IsMember(ctx, org, user)
	return member, err
}
