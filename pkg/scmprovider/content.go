package scmprovider

import (
	"context"

	"github.com/jenkins-x/go-scm/scm"
)

// GetFile returns the file from git
func (c *Client) GetFile(owner, repo, filepath, commit string) ([]byte, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	answer, r, err := c.client.Contents.Find(ctx, fullName, filepath, commit)
	// handle files not existing nicely
	if r != nil && r.Status == 404 {
		return nil, nil
	}
	var data []byte
	if answer != nil {
		data = answer.Data
	}
	return data, err
}

// ListFiles returns the files from git
func (c *Client) ListFiles(owner, repo, filepath, commit string) ([]*scm.FileEntry, error) {
	ctx := context.Background()
	fullName := c.repositoryName(owner, repo)
	answer, _, err := c.client.Contents.List(ctx, fullName, filepath, commit, &scm.ListOptions{})
	return answer, err
}
