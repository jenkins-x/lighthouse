package scmprovider

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

// ListReviews list the reviews
func (c *Client) ListReviews(owner, repo string, number int) ([]*scm.Review, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	var allReviews []*scm.Review
	var resp *scm.Response
	var reviews []*scm.Review
	var err error
	firstRun := false
	opts := scm.ListOptions{
		Page: 1,
	}
	for !firstRun || (resp != nil && opts.Page <= resp.Page.Last) {
		reviews, resp, err = c.client.Reviews.List(ctx, fullName, number, opts)
		if err != nil {
			return nil, err
		}
		firstRun = true
		allReviews = append(allReviews, reviews...)
		opts.Page++
	}
	return allReviews, nil
}

// RequestReview requests a review
func (c *Client) RequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}

// UnrequestReview unrequest a review
func (c *Client) UnrequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}
