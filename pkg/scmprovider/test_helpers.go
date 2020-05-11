package scmprovider

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/pkg/errors"
)

// TestBotName is the default bot name used in tests
var TestBotName = "k8s-ci-robot"

// TestClient uses the go-scm fake client behind the scenes and allows access to its test data
type TestClient struct {
	Client

	Data *fake.Data
}

// ToTestClient converts the scm client to an API that the prow plugins expect
func ToTestClient(client *scm.Client) *TestClient {
	return &TestClient{
		Client: Client{client: client, botName: TestBotName},
		Data:   nil,
	}
}

// NewTestClientForLabelsInComments returns a test client specifically for testing the scenario of labels not being available
func NewTestClientForLabelsInComments() *TestClient {
	fakeScmClient, fc := fake.NewDefault()
	fakeScmClient.Driver = scm.DriverCoding

	return &TestClient{
		Client: Client{
			client:  fakeScmClient,
			botName: TestBotName,
		},
		Data: fc,
	}
}

// PopulateFakeLabelsFromComments checks if there's Data for this test client, and if it doesn't support PR labels. If so,
// it populates the label-related fields in the data from the comments.
func (t *TestClient) PopulateFakeLabelsFromComments(org, repo string, number int, fakeLabel string, shouldUnlabel bool) error {
	if t.Data != nil && !t.SupportsPRLabels() {
		prLabels, err := t.GetIssueLabels(org, repo, number, true)
		if err != nil {
			return errors.Wrapf(err, "Unexpected error getting label comments")
		}
		hasLabel := false
		for _, c := range prLabels {
			if c.Name == labels.WorkInProgress {
				t.Data.PullRequestLabelsAdded = append(t.Data.PullRequestLabelsAdded, fakeLabel)
				hasLabel = true
			} else {
				t.Data.PullRequestLabelsAdded = append(t.Data.PullRequestLabelsAdded, fakeLabel)
			}
		}
		if !hasLabel && shouldUnlabel {
			t.Data.PullRequestLabelsRemoved = append(t.Data.PullRequestLabelsRemoved, fakeLabel)
		}
	}
	return nil
}
