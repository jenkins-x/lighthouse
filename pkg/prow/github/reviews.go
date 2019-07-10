package github

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

func (c *GitHubClient) ListReviews(owner, repo string, number int) ([]*scm.Review, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	reviews, _, err := c.client.Reviews.List(ctx, fullName, number, c.createListOptions())
	return reviews, err
}

func (c *GitHubClient) RequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}

func (c *GitHubClient) UnrequestReview(org, repo string, number int, logins []string) error {
	panic("implement me")
}
