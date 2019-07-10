package github

import (
	"github.com/jenkins-x/go-scm/scm"
)

// the default bot name used in tests
var TestBotName = "jenkins-x-bot"

// ToTestGitHubClient converts the scm client to an API that the prow plugins expect
func ToTestGitHubClient(client *scm.Client) *GitHubClient {
	return &GitHubClient{client: client, botName: TestBotName}
}
