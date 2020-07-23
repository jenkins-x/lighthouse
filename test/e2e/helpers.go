package e2e

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"text/template"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/lighthouse-config/pkg/config"
	"github.com/jenkins-x/lighthouse-config/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/repoowners"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/onsi/gomega/gexec"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	primarySCMTokenEnvVar  = "E2E_PRIMARY_SCM_TOKEN"
	approverSCMTokenEnvVar = "E2E_APPROVER_SCM_TOKEN"
	baseRepoName           = "lh-e2e-test"
)

// CreateGitClient creates the git client used for cloning and making changes to the test repository
func CreateGitClient(gitServerURL string, userFunc func() string, tokenFunc func() (string, error)) (git.Client, error) {
	gitClient, err := git.NewClient(gitServerURL, gitKind())
	if err != nil {
		return nil, err
	}
	token, err := tokenFunc()
	if err != nil {
		return nil, err
	}
	gitClient.SetCredentials(userFunc(), func() []byte {
		return []byte(token)
	})

	return gitClient, nil
}

// CreateSCMClient takes functions that return the username and token to use, and creates the scm.Client and Lighthouse SCM client
func CreateSCMClient(userFunc func() string, tokenFunc func() (string, error)) (*scm.Client, scmprovider.SCMClient, string, error) {
	kind := gitKind()
	serverURL := os.Getenv("E2E_GIT_SERVER")

	client, err := factory.NewClient(kind, serverURL, "")

	token, err := tokenFunc()
	if err != nil {
		return nil, nil, "", err
	}
	util.AddAuthToSCMClient(client, token, false)

	spc := scmprovider.ToClient(client, userFunc())
	return client, spc, serverURL, err
}

func gitKind() string {
	kind := os.Getenv("E2E_GIT_KIND")
	if kind == "" {
		kind = "github"
	}
	return kind
}

// CreateHMACToken creates an HMAC token for use in webhooks
func CreateHMACToken() (string, error) {
	src := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 21) // can be simplified to n/2 if n is always even

	if _, err := src.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b)[:41], nil
}

// GetBotName gets the bot user name
func GetBotName() string {
	botName := os.Getenv("E2E_GIT_USER")
	if botName == "" {
		botName = "jenkins-x-bot"
	}
	return botName
}

// GetApproverName gets the approver user's username
func GetApproverName() string {
	botName := os.Getenv("E2E_APPROVER_USER")
	if botName == "" {
		botName = "jenkins-x-bot"
	}
	return botName
}

// GetPrimarySCMToken gets the token used by the bot/primary user
func GetPrimarySCMToken() (string, error) {
	return getSCMToken(primarySCMTokenEnvVar, gitKind())
}

// GetApproverSCMToken gets the token used by the approver
func GetApproverSCMToken() (string, error) {
	return getSCMToken(approverSCMTokenEnvVar, gitKind())
}

func getSCMToken(envName, gitKind string) (string, error) {
	value := os.Getenv(envName)
	if value == "" {
		return value, fmt.Errorf("No token available for git kind %s at environment variable $%s", gitKind, envName)
	}
	return value, nil
}

// CreateBaseRepository creates the repository that will be used for tests
func CreateBaseRepository(botUser, approver string, botClient *scm.Client, gitClient git.Client) (*scm.Repository, string, error) {
	repoName := baseRepoName + "-" + strconv.FormatInt(GinkgoRandomSeed(), 10)

	input := &scm.RepositoryInput{
		Namespace: botUser,
		Name:      repoName,
		Private:   true,
	}

	repo, _, err := botClient.Repositories.Create(context.Background(), input)
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to create repository")
	}

	r, err := gitClient.Clone(repo.Namespace + "/" + repo.Name)
	if err != nil {
		return nil, "", errors.Wrapf(err, "could not clone %s", repo.FullName)
	}
	err = r.CheckoutNewBranch("master")
	if err != nil {
		return nil, "", err
	}

	baseScriptFile := filepath.Join("test_data", "baseRepoScript.sh")
	baseScript, err := ioutil.ReadFile(baseScriptFile)

	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to read %s", baseScriptFile)
	}

	scriptOutputFile := filepath.Join(r.Dir, "script.sh")
	err = ioutil.WriteFile(scriptOutputFile, baseScript, 0755)
	if err != nil {
		return nil, "", errors.Wrapf(err, "couldn't write to %s", scriptOutputFile)
	}

	ExpectCommandExecution(r.Dir, 1, 0, "git", "add", scriptOutputFile)

	owners := repoowners.SimpleConfig{
		Config: repoowners.Config{
			Approvers: []string{botUser, approver},
			Reviewers: []string{botUser, approver},
		},
	}

	ownersFile := filepath.Join(r.Dir, "OWNERS")
	ownersYaml, err := yaml.Marshal(owners)
	if err != nil {
		return nil, "", errors.Wrapf(err, "couldn't marshal OWNERS yaml")
	}

	err = ioutil.WriteFile(ownersFile, ownersYaml, 0644)
	if err != nil {
		return nil, "", errors.Wrapf(err, "couldn't write to %s", ownersFile)
	}
	ExpectCommandExecution(r.Dir, 1, 0, "git", "add", ownersFile)

	ExpectCommandExecution(r.Dir, 1, 0, "git", "-a", "-m", "Initial commit of functioning script and OWNERS")

	err = r.Push(repo.Namespace+"/"+repo.Name, "master")
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to push to %s", repo.Clone)
	}

	return repo, r.Dir, nil
}

// ExpectCommandExecution performs the given command in the current work directory and asserts that it completes successfully
func ExpectCommandExecution(dir string, commandTimeout time.Duration, exitCode int, c string, args ...string) {
	f := func() error {
		command := exec.Command(c, args...)
		command.Dir = dir
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		session.Wait(commandTimeout)
		Eventually(session).Should(gexec.Exit(exitCode))
		return err
	}
	err := retryExponentialBackoff(1, f)
	Expect(err).ShouldNot(HaveOccurred())
}

// retryExponentialBackoff retries the given function up to the maximum duration
func retryExponentialBackoff(maxDuration time.Duration, f func() error) error {
	exponentialBackOff := backoff.NewExponentialBackOff()
	exponentialBackOff.MaxElapsedTime = maxDuration
	exponentialBackOff.Reset()
	err := backoff.Retry(f, exponentialBackOff)
	return err
}

type configReplacement struct {
	Owner     string
	Repo      string
	Namespace string
}

// ProcessConfigAndPlugins reads the templates for the config and plugins config maps and replaces the owner, repo, and namespace in them
func ProcessConfigAndPlugins(owner, repo, namespace string) (*config.Config, *plugins.Configuration, error) {
	cfgFile := filepath.Join("test_data", "example-config.tmpl.yml")
	pluginFile := filepath.Join("test_data", "example-plugins.tmpl.yml")

	rawConfig, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "reading config template %s", cfgFile)
	}
	rawPlugins, err := ioutil.ReadFile(pluginFile)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "reading plugins template %s", pluginFile)
	}

	cfgTmpl, err := template.New("cfg").Parse(string(rawConfig))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parsing config template from %s", cfgFile)
	}
	pluginTmpl, err := template.New("plugins").Parse(string(rawPlugins))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "parsing plugins template from %s", pluginFile)
	}

	input := configReplacement{
		Owner:     owner,
		Repo:      repo,
		Namespace: namespace,
	}

	var cfgBuf bytes.Buffer
	var pluginBuf bytes.Buffer

	err = cfgTmpl.Execute(&cfgBuf, &input)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "applying config template from %s", cfgFile)
	}
	err = pluginTmpl.Execute(&pluginBuf, &input)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "applying plugins template from %s", pluginFile)
	}

	var generatedCfg *config.Config
	var generatedPlugin *plugins.Configuration

	err = yaml.Unmarshal(cfgBuf.Bytes(), generatedCfg)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unmarshalling config from %s", cfgFile)
	}

	err = yaml.Unmarshal(pluginBuf.Bytes(), generatedPlugin)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unmarshalling plugins from %s", pluginFile)
	}

	return generatedCfg, generatedPlugin, nil
}

// CreateWebHook creates a webhook on the SCM provider for the repository
func CreateWebHook(scmClient *scm.Client, repo *scm.Repository, hmacToken string) error {
	output, err := exec.Command("kubectl", "get", "ingress", "hook", "-o", "jsonpath='{.spec.rules[0].host}").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to get hook ingress")
	}
	targetURL := string(output)
	input := &scm.HookInput{
		Name:   "lh-test-hook",
		Target: targetURL,
		Secret: hmacToken,
		Events: scm.HookEvents{},
	}
	_, _, err = scmClient.Repositories.CreateHook(context.Background(), repo.Namespace+"/"+repo.Name, input)

	return err
}

// ApplyConfigAndPluginsConfigMaps takes the config and plugins and creates/applies the config maps in the cluster using kubectl
func ApplyConfigAndPluginsConfigMaps(cfg *config.Config, pluginsCfg *plugins.Configuration) error {
	cfgMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config",
			Namespace: cfg.LighthouseJobNamespace,
		},
	}
	cfgData, err := yaml.Marshal(cfg)
	if err != nil {
		return errors.Wrapf(err, "writing config to YAML")
	}
	cfgMap.Data["config.yaml"] = string(cfgData)

	pluginMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "plugins",
			Namespace: cfg.LighthouseJobNamespace,
		},
	}
	pluginData, err := yaml.Marshal(pluginsCfg)
	if err != nil {
		return errors.Wrapf(err, "writing plugins to YAML")
	}
	cfgMap.Data["plugins.yaml"] = string(pluginData)

	tmpDir, err := ioutil.TempDir("", "kubectl")
	if err != nil {
		return errors.Wrapf(err, "creating temp directory")
	}
	defer os.RemoveAll(tmpDir)

	cfgYaml, err := yaml.Marshal(cfgMap)
	if err != nil {
		return errors.Wrapf(err, "marshalling config")
	}
	pluginYaml, err := yaml.Marshal(pluginMap)
	if err != nil {
		return errors.Wrapf(err, "marshalling plugins")
	}
	cfgFile := filepath.Join(tmpDir, "config-map.yaml")
	pluginFile := filepath.Join(tmpDir, "plugins-map.yaml")

	err = ioutil.WriteFile(cfgFile, cfgYaml, 0644)
	if err != nil {
		return errors.Wrapf(err, "writing config map to %s", cfgFile)
	}
	err = ioutil.WriteFile(pluginFile, pluginYaml, 0644)
	if err != nil {
		return errors.Wrapf(err, "writing plugins map to %s", pluginFile)
	}

	ExpectCommandExecution(tmpDir, 1, 0, "kubectl", "apply", "-f", "config-map.yaml")
	ExpectCommandExecution(tmpDir, 1, 0, "kubectl", "apply", "-f", "plugins-map.yaml")

	return nil
}
