package github

import (
	"context"
	"fmt"
	"os"

	"github.com/jenkins-x/go-scm/scm"
)

// ToGitHubClient converts the scm client to an API that the prow plugins expect
func ToGitHubClient(client *scm.Client, botName string) *GitHubClient {
	return &GitHubClient{client: client, botName: botName}
}

// GitHubClient represents an interface that prow plugins expect on top of go-scm
type GitHubClient struct {
	client  *scm.Client
	botName string
}

func (c *GitHubClient) ClearMilestone(org, repo string, num int) error {
	panic("implement me")
}

func (c *GitHubClient) SetMilestone(org, repo string, issueNum, milestoneNum int) error {
	panic("implement me")
}

func (c *GitHubClient) ListMilestones(org, repo string) ([]Milestone, error) {
	panic("implement me")
}

func (c *GitHubClient) GetFile(org, repo, filepath, commit string) ([]byte, error) {
	panic("implement me")
}

func (c *GitHubClient) Query(context.Context, interface{}, map[string]interface{}) error {
	panic("implement me")
}

func (c *GitHubClient) BotName() (string, error) {
	botName := c.botName
	if botName == "" {
		botName = os.Getenv("BOT_NAME")
		if botName == "" {
			botName = "jenkins-x-bot"
		}
		c.botName = botName
	}
	return botName, nil
}

func (c *GitHubClient) SetBotName(botName string) {
	c.botName = botName
}

func (c *GitHubClient) repositoryName(owner string, repo string) string {
	return fmt.Sprintf("%s/%s", owner, repo)
}

func (c *GitHubClient) createListOptions() scm.ListOptions {
	return scm.ListOptions{}
}

// FileNotFound happens when github cannot find the file requested by GetFile().
type FileNotFound struct {
	org, repo, path, commit string
}

func (e *FileNotFound) Error() string {
	return fmt.Sprintf("%s/%s/%s @ %s not found", e.org, e.repo, e.path, e.commit)
}
