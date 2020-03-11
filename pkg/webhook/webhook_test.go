package webhook

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	"github.com/jenkins-x/lighthouse/pkg/prow/hook"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
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
	kubeClient := fake.NewSimpleClientset(objs...)
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
		server: &hook.Server{
			ConfigAgent: configAgent,
			Plugins:     pluginAgent,
			ClientAgent: &plugins.ClientAgent{
				BotName:           options.GetBotName(),
				SCMProviderClient: scmClient,
				KubernetesClient:  kubeClient,
				GitClient:         gitClient,
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
