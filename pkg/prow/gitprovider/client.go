package gitprovider

import (
	"fmt"
	"os"

	"github.com/jenkins-x/go-scm/scm"
)

// ToClient converts the scm client to an API that the prow plugins expect
func ToClient(client *scm.Client, botName string) *Client {
	return &Client{client: client, botName: botName}
}

// Client represents an interface that prow plugins expect on top of go-scm
type Client struct {
	client  *scm.Client
	botName string
}

// ClearMilestone clears milestone
func (c *Client) ClearMilestone(org, repo string, num int) error {
	panic("implement me")
}

// SetMilestone sets milestone
func (c *Client) SetMilestone(org, repo string, issueNum, milestoneNum int) error {
	panic("implement me")
}

// ListMilestones list milestones
func (c *Client) ListMilestones(org, repo string) ([]Milestone, error) {
	panic("implement me")
}

// BotName returns the bot name
func (c *Client) BotName() (string, error) {
	botName := c.botName
	if botName == "" {
		botName = os.Getenv("GIT_USER")
		if botName == "" {
			botName = "jenkins-x-bot"
		}
		c.botName = botName
	}
	return botName, nil
}

// SetBotName sets the bot name
func (c *Client) SetBotName(botName string) {
	c.botName = botName
}

func (c *Client) repositoryName(owner string, repo string) string {
	return fmt.Sprintf("%s/%s", owner, repo)
}

func (c *Client) createListOptions() scm.ListOptions {
	return scm.ListOptions{}
}

// FileNotFound happens when github cannot find the file requested by GetFile().
type FileNotFound struct {
	org, repo, path, commit string
}

// Error formats a file not found error
func (e *FileNotFound) Error() string {
	return fmt.Sprintf("%s/%s/%s @ %s not found", e.org, e.repo, e.path, e.commit)
}
