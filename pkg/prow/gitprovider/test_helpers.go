package gitprovider

import (
	"github.com/jenkins-x/go-scm/scm"
)

// TestBotName is the default bot name used in tests
var TestBotName = "jenkins-x-bot"

// ToTestClient converts the scm client to an API that the prow plugins expect
func ToTestClient(client *scm.Client) *Client {
	return &Client{client: client, botName: TestBotName}
}
