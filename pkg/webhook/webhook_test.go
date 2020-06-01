package webhook

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	kubefake "k8s.io/client-go/kubernetes/fake"
)

type WebhookTestSuite struct {
	suite.Suite
	KubeClient     kubernetes.Interface
	SCMCLient      scm.Client
	GitClient      git.Client
	WebhookOptions *Options
	TestRepo       scm.Repository
}

func (suite *WebhookTestSuite) TestProcessWebhookPRComment() {
	t := suite.T()
	webhook := &scm.PullRequestCommentHook{
		Action: scm.ActionUpdate,
		Repo:   suite.TestRepo,
	}
	l := logrus.WithField("test", t.Name())
	logrusEntry, message, err := suite.WebhookOptions.ProcessWebHook(l, webhook)
	assert.NoError(t, err)
	assert.Equal(t, "processed PR comment hook", message)
	assert.NotNil(t, logrusEntry)
}

func (suite *WebhookTestSuite) TestProcessWebhookPR() {
	t := suite.T()

	webhook := &scm.PullRequestHook{
		Action: scm.ActionCreate,
		Repo:   suite.TestRepo,
	}
	l := logrus.WithField("test", t.Name())
	logrusEntry, message, err := suite.WebhookOptions.ProcessWebHook(l, webhook)

	assert.NoError(t, err)
	assert.Equal(t, "processed PR hook", message)
	assert.NotNil(t, logrusEntry)
}

func (suite *WebhookTestSuite) TestProcessWebhookPRReview() {
	t := suite.T()

	webhook := &scm.ReviewHook{
		Action: scm.ActionSubmitted,
		Repo:   suite.TestRepo,
		Review: scm.Review{
			State: "APPROVED",
			Author: scm.User{
				Login: "user",
				Name:  "User",
			},
		},
	}
	l := logrus.WithField("test", t.Name())
	logrusEntry, message, err := suite.WebhookOptions.ProcessWebHook(l, webhook)

	assert.NoError(t, err)
	assert.Equal(t, "processed PR review hook", message)
	assert.NotNil(t, logrusEntry)
}

func (suite *WebhookTestSuite) TestProcessWebhookUnknownRepo() {
	t := suite.T()

	unknownRepo := scm.Repository{
		ID:        "1",
		Namespace: "default",
		Name:      "test-repo",
		FullName:  "test-org/unknown-repo",
		Branch:    "master",
		Link:      "https://github.com/test-org/unknown-repo.git",
		Private:   false,
	}

	l := logrus.WithField("test", t.Name())

	// First, try not in GitHub App mode and expect normal processing.
	origEnvVar := os.Getenv(util.GitHubAppSecretDirEnvVar)
	defer os.Setenv(util.GitHubAppSecretDirEnvVar, origEnvVar)

	os.Unsetenv(util.GitHubAppSecretDirEnvVar)

	webhook := &scm.PullRequestHook{
		Action: scm.ActionCreate,
		Repo:   unknownRepo,
	}

	logrusEntry, message, err := suite.WebhookOptions.ProcessWebHook(l, webhook)

	assert.NoError(t, err)
	assert.Equal(t, "processed PR hook", message)
	assert.NotNil(t, logrusEntry)

	// Now try again in GitHub App mode and expect an error.
	os.Setenv(util.GitHubAppSecretDirEnvVar, "/some/dir")
	_, _, err = suite.WebhookOptions.ProcessWebHook(l, webhook)

	assert.EqualError(t, err, fmt.Sprintf("repository not configured: %s", unknownRepo.Link))
}

func (suite *WebhookTestSuite) SetupSuite() {
	options := &Options{}
	t := suite.T()
	configAgent := &config.Agent{}
	pluginAgent := &plugins.ConfigAgent{}

	workDir, err := os.Getwd()
	assert.NoError(t, err)
	configBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/test_data/test_config.yaml", workDir))
	assert.NoError(t, err)
	loadedConfig, err := config.LoadYAMLConfig(configBytes)
	configAgent.Set(loadedConfig)
	assert.NoError(t, err)

	pluginBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/test_data/test_plugins.yaml", workDir))
	assert.NoError(t, err)
	loadedPlugins, err := pluginAgent.LoadYAMLConfig(pluginBytes)
	pluginAgent.Set(loadedPlugins)
	assert.NoError(t, err)

	var objs []runtime.Object
	kubeClient := kubefake.NewSimpleClientset(objs...)
	lhClient := fake.NewSimpleClientset()
	scmClient, serverURL, err := options.createSCMClient()
	assert.NoError(t, err)
	gitClient, err := git.NewClient(serverURL, options.gitKind())
	assert.NoError(t, err)
	user := options.GetBotName()
	token, err := options.createSCMToken(options.gitKind())
	gitClient.SetCredentials(user, func() []byte {
		return []byte(token)
	})
	util.AddAuthToSCMClient(scmClient, token, false)
	suite.WebhookOptions = &Options{
		server: &Server{
			ConfigAgent: configAgent,
			Plugins:     pluginAgent,
			ClientAgent: &plugins.ClientAgent{
				BotName:           options.GetBotName(),
				SCMProviderClient: scmClient,
				KubernetesClient:  kubeClient,
				GitClient:         gitClient,
				LighthouseClient:  lhClient.LighthouseV1alpha1().LighthouseJobs(""),
			},
		},
	}

	suite.TestRepo = scm.Repository{
		ID:        "1",
		Namespace: "default",
		Name:      "test-repo",
		FullName:  "test-org/test-repo",
		Branch:    "master",
		Private:   false,
	}
}

func TestWebhookTestSuite(t *testing.T) {
	os.Setenv("GIT_TOKEN", "abc123")
	suite.Run(t, new(WebhookTestSuite))
}
