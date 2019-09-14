package fake

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/prow/gitprovider"
)

const testBotName = "k8s-ci-robot"

// NewClient creates a new fake git client
func NewClient() (*gitprovider.Client, *scm.Client, *fake.Data) {
	fakeScmClient, data := fake.NewDefault()
	data.RepoLabelsExisting = nil
	client := gitprovider.ToClient(fakeScmClient, testBotName)
	return client, fakeScmClient, data
}
