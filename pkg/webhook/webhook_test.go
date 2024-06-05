package webhook

import (
	"fmt"
	"os"
	"reflect"
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
	WebhookOptions *WebhooksController
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
	t := suite.T()
	configAgent := &config.Agent{}
	pluginAgent := &plugins.ConfigAgent{}

	workDir, err := os.Getwd()
	assert.NoError(t, err)
	configBytes, err := os.ReadFile(fmt.Sprintf("%s/test_data/test_config.yaml", workDir))
	assert.NoError(t, err)
	loadedConfig, err := config.LoadYAMLConfig(configBytes)
	configAgent.Set(loadedConfig)
	assert.NoError(t, err)

	pluginBytes, err := os.ReadFile(fmt.Sprintf("%s/test_data/test_plugins.yaml", workDir))
	assert.NoError(t, err)
	loadedPlugins, err := pluginAgent.LoadYAMLConfig(pluginBytes)
	pluginAgent.Set(loadedPlugins)
	assert.NoError(t, err)

	var objs []runtime.Object
	kubeClient := kubefake.NewSimpleClientset(objs...)
	lhClient := fake.NewSimpleClientset()
	_, scmClient, serverURL, _, err := util.GetSCMClient("", configAgent.Config)
	assert.NoError(t, err)
	gitClient, err := git.NewClient(serverURL, util.GitKind(configAgent.Config))
	assert.NoError(t, err)
	user := util.GetBotName(configAgent.Config)
	token, _ := util.GetSCMToken(util.GitKind(configAgent.Config))
	gitClient.SetCredentials(user, func() []byte {
		return []byte(token)
	})
	util.AddAuthToSCMClient(scmClient, token, false)
	suite.WebhookOptions = &WebhooksController{
		server: &Server{
			ConfigAgent: configAgent,
			Plugins:     pluginAgent,
			ClientAgent: &plugins.ClientAgent{
				BotName:           user,
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

func TestNeedDemux(t *testing.T) {
	tests := []struct {
		name string

		eventType scm.WebhookKind
		srcRepo   string
		plugins   map[string][]plugins.ExternalPlugin

		expected []plugins.ExternalPlugin
	}{
		{
			name: "no external plugins",

			eventType: scm.WebhookKindIssueComment,
			srcRepo:   "kubernetes/test-infra",
			plugins:   nil,

			expected: nil,
		},
		{
			name: "we have variety",

			eventType: scm.WebhookKindIssueComment,
			srcRepo:   "kubernetes/test-infra",
			plugins: map[string][]plugins.ExternalPlugin{
				"kubernetes/test-infra": {
					{
						Name:   "sandwich",
						Events: []string{"pull_request"},
					},
					{
						Name: "coffee",
					},
				},
				"kubernetes/kubernetes": {
					{
						Name:   "gumbo",
						Events: []string{"issue_comment"},
					},
				},
				"kubernetes": {
					{
						Name:   "chicken",
						Events: []string{"push"},
					},
					{
						Name: "water",
					},
					{
						Name:   "chocolate",
						Events: []string{"pull_request", "issue_comment", "issues"},
					},
				},
			},

			expected: []plugins.ExternalPlugin{
				{
					Name: "coffee",
				},
				{
					Name: "water",
				},
				{
					Name:   "chocolate",
					Events: []string{"pull_request", "issue_comment", "issues"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pa := &plugins.ConfigAgent{}
			pa.Set(&plugins.Configuration{
				ExternalPlugins: test.plugins,
			})
			s := &Server{Plugins: pa}

			gotPlugins := util.ExternalPluginsForEvent(s.Plugins, string(test.eventType), test.srcRepo, nil)
			if len(gotPlugins) != len(test.expected) {
				t.Fatalf("expected plugins: %+v, got: %+v", test.expected, gotPlugins)
			}
			for _, expected := range test.expected {
				var found bool
				for _, got := range gotPlugins {
					if got.Name != expected.Name {
						continue
					}
					if !reflect.DeepEqual(expected, got) {
						t.Errorf("expected plugin: %+v, got: %+v", expected, got)
					}
					found = true
				}
				if !found {
					t.Errorf("expected plugins: %+v, got: %+v", test.expected, gotPlugins)
					break
				}
			}
		})
	}
}

func TestWebhookTestSuite(t *testing.T) {
	os.Setenv("GIT_TOKEN", "abc123")
	suite.Run(t, new(WebhookTestSuite))
}
