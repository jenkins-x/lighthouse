package webhook

import (
	"os"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
	"github.com/stretchr/testify/assert"
)

func TestProcessWebhook(t *testing.T) {
	options := &Options{}
	var err error
	options.server, err = options.createHookServer()
	assert.NoError(t, err)
	assert.NotNil(t, options.server)

	kubeClient, _, _ := options.GetFactory().CreateKubeClient()
	scmClient, serverURL, token, err := options.createSCMClient()
	assert.NoError(t, err)
	gitClient, err := git.NewClient(serverURL, options.gitKind())
	assert.NoError(t, err)
	user := options.GetBotName()
	gitClient.SetCredentials(user, func() []byte {
		return []byte(token)
	})

	options.server.ClientAgent = &plugins.ClientAgent{
		BotName:          user,
		GitHubClient:     scmClient,
		KubernetesClient: kubeClient,
		GitClient:        gitClient,
	}

	factory := options.GetFactory()

	options = NewWebhook(factory, options.server)

	webhook := &scm.PullRequestCommentHook{
		Action: scm.ActionUpdate,
		Repo: scm.Repository{
			ID:        "1",
			Namespace: "default",
			Name:      "test-repo",
			FullName:  "test-org/test-repo",
			Branch:    "master",
			Private:   false,
		},
	}

	logrusEntry, message, err := options.ProcessWebHook(webhook)
	assert.NoError(t, err)
	assert.Equal(t, "processed PR comment hook", message)
	assert.NotNil(t, logrusEntry)
}

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	os.Setenv("GIT_TOKEN", "abc123")
	os.Exit(m.Run())
}
