package scmprovider

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

// ClearMilestone clears milestone
func (c *Client) ClearMilestone(org, repo string, num int, isPR bool) error {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	var err error
	if isPR {
		_, err = c.client.PullRequests.ClearMilestone(ctx, fullName, num)
	} else {
		_, err = c.client.Issues.ClearMilestone(ctx, fullName, num)
	}
	return err
}

// SetMilestone sets milestone
func (c *Client) SetMilestone(org, repo string, issueNum, milestoneNum int, isPR bool) error {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	var err error
	if isPR {
		_, err = c.client.PullRequests.SetMilestone(ctx, fullName, issueNum, milestoneNum)
	} else {
		_, err = c.client.Issues.SetMilestone(ctx, fullName, issueNum, milestoneNum)
	}
	return err
}

// ListMilestones list milestones
func (c *Client) ListMilestones(org, repo string) ([]*scm.Milestone, error) {
	ctx := context.Background()
	fullName := c.repositoryName(org, repo)
	var resp *scm.Response
	var milestones []*scm.Milestone
	var m []*scm.Milestone
	var err error
	firstRun := false
	opts := scm.MilestoneListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		m, resp, err = c.client.Milestones.List(ctx, fullName, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		milestones = append(milestones, m...)
		opts.Page++
	}
	return milestones, err
}
