package github

import (
	"context"
)

func (c *GitHubClient) GetFile(owner, repo, filepath, commit string) ([]byte, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	answer, _, err := c.client.Contents.Find(ctx, fullName, filepath, commit)
	var data []byte
	if answer != nil {
		data = answer.Data
	}
	return data, err
}
