package github

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

// ListReviews list the reviews
func (c *Client) ListReviews(owner, repo string, number int) ([]*scm.Review, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	reviews, _, err := c.client.Reviews.List(ctx, fullName, number, c.createListOptions())
	return reviews, err
}

// RequestReview requests a review
func (c *Client) RequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}

// UnrequestReview unrequest a review
func (c *Client) UnrequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}
