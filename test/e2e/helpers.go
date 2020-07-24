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
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/lighthouse-config/pkg/config"
	cfgplugins "github.com/jenkins-x/lighthouse-config/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/repoowners"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/onsi/gomega/gexec"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	gr "github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

const (
	primarySCMTokenEnvVar  = "E2E_PRIMARY_SCM_TOKEN" /* #nosec */
	primarySCMUserEnvVar   = "E2E_PRIMARY_SCM_USER"
	approverSCMTokenEnvVar = "E2E_APPROVER_SCM_TOKEN" /* #nosec */
	approverSCMUserEnvVar  = "E2E_APPROVER_SCM_USER"
	hmacTokenEnvVar        = "E2E_HMAC_TOKEN" /* #nosec */
	gitServerEnvVar        = "E2E_GIT_SERVER"
	gitKindEnvVar          = "E2E_GIT_KIND"
	baseRepoName           = "lh-e2e-test"
)

var (
	lighthouseActionTimeout = 5 * time.Minute
)

// CreateGitClient creates the git client used for cloning and making changes to the test repository
func CreateGitClient(gitServerURL string, userFunc func() string, tokenFunc func() (string, error)) (git.Client, error) {
	gitClient, err := git.NewClient(gitServerURL, GitKind())
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
	kind := GitKind()
	serverURL := os.Getenv(gitServerEnvVar)

	client, err := factory.NewClient(kind, serverURL, "")

	token, err := tokenFunc()
	if err != nil {
		return nil, nil, "", err
	}
	util.AddAuthToSCMClient(client, token, false)

	spc := scmprovider.ToClient(client, userFunc())
	return client, spc, serverURL, err
}

// GitKind returns the git provider flavor being used
func GitKind() string {
	kind := os.Getenv(gitKindEnvVar)
	if kind == "" {
		kind = "github"
	}
	return kind
}

// CreateHMACToken creates an HMAC token for use in webhooks, defaulting to the E2E_HMAC_TOKEN env var if set
func CreateHMACToken() (string, error) {
	fromEnv := os.Getenv(hmacTokenEnvVar)
	if fromEnv != "" {
		return fromEnv, nil
	}
	src := rand.New(rand.NewSource(time.Now().UnixNano())) /* #nosec */
	b := make([]byte, 21)                                  // can be simplified to n/2 if n is always even

	if _, err := src.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b)[:41], nil
}

// GetBotName gets the bot user name
func GetBotName() string {
	botName := os.Getenv(primarySCMUserEnvVar)
	if botName == "" {
		botName = "jenkins-x-bot"
	}
	return botName
}

// GetApproverName gets the approver user's username
func GetApproverName() string {
	botName := os.Getenv(approverSCMUserEnvVar)
	if botName == "" {
		botName = "jenkins-x-bot"
	}
	return botName
}

// GetPrimarySCMToken gets the token used by the bot/primary user
func GetPrimarySCMToken() (string, error) {
	return getSCMToken(primarySCMTokenEnvVar, GitKind())
}

// GetApproverSCMToken gets the token used by the approver
func GetApproverSCMToken() (string, error) {
	return getSCMToken(approverSCMTokenEnvVar, GitKind())
}

func getSCMToken(envName, gitKind string) (string, error) {
	value := os.Getenv(envName)
	if value == "" {
		return value, fmt.Errorf("No token available for git kind %s at environment variable $%s", gitKind, envName)
	}
	return value, nil
}

// CreateBaseRepository creates the repository that will be used for tests
func CreateBaseRepository(botUser, approver string, botClient *scm.Client, gitClient git.Client) (*scm.Repository, *git.Repo, error) {
	repoName := baseRepoName + "-" + strconv.FormatInt(ginkgo.GinkgoRandomSeed(), 10)

	input := &scm.RepositoryInput{
		Name:    repoName,
		Private: true,
	}

	repo, _, err := botClient.Repositories.Create(context.Background(), input)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create repository")
	}

	// Sleep 5 seconds to ensure repository exists enough to be pushed to.
	time.Sleep(5 * time.Second)

	r, err := gitClient.Clone(repo.Namespace + "/" + repo.Name)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "could not clone %s", repo.FullName)
	}
	err = r.CheckoutNewBranch("master")
	if err != nil {
		return nil, nil, err
	}

	baseScriptFile := filepath.Join("test_data", "baseRepoScript.sh")
	baseScript, err := ioutil.ReadFile(baseScriptFile) /* #nosec */

	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to read %s", baseScriptFile)
	}

	scriptOutputFile := filepath.Join(r.Dir, "script.sh")
	err = ioutil.WriteFile(scriptOutputFile, baseScript, 0600)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't write to %s", scriptOutputFile)
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
		return nil, nil, errors.Wrapf(err, "couldn't marshal OWNERS yaml")
	}

	err = ioutil.WriteFile(ownersFile, ownersYaml, 0600)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "couldn't write to %s", ownersFile)
	}
	ExpectCommandExecution(r.Dir, 1, 0, "git", "add", ownersFile)

	ExpectCommandExecution(r.Dir, 1, 0, "git", "commit", "-a", "-m", "Initial commit of functioning script and OWNERS")

	err = r.Push(repo.Name, "master")
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to push to %s", repo.Clone)
	}

	return repo, r, nil
}

// AddCollaborator adds the approver user to the repo
func AddCollaborator(approver string, repo *scm.Repository, botClient *scm.Client, approverClient *scm.Client) error {
	_, alreadyCollaborator, _, err := botClient.Repositories.AddCollaborator(context.Background(), fmt.Sprintf("%s/%s", repo.Namespace, repo.Name), approver, "admin")
	if alreadyCollaborator {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "adding %s as collaborator for repo %s/%s", approver, repo.Namespace, repo.Name)
	}

	// Don't bother checking for invites with BitBucket Server
	if GitKind() == "stash" {
		return nil
	}

	// Sleep for a bit
	time.Sleep(15 * time.Second)

	invites, _, err := approverClient.Users.ListInvitations(context.Background())
	if err == scm.ErrNotSupported {
		// Ignore any cases of not supported
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "listing invitations for user %s", approver)
	}
	for _, i := range invites {
		_, err = approverClient.Users.AcceptInvitation(context.Background(), i.ID)
		if err == scm.ErrNotSupported {
			return nil
		}
		if err != nil {
			return errors.Wrapf(err, "accepting invitation %d for user %s", i.ID, approver)
		}
	}
	return nil
}

// ExpectCommandExecution performs the given command in the current work directory and asserts that it completes successfully
func ExpectCommandExecution(dir string, commandTimeout time.Duration, exitCode int, c string, args ...string) {
	f := func() error {
		command := exec.Command(c, args...) /* #nosec */
		command.Dir = dir
		session, err := gexec.Start(command, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
		session.Wait(10 * time.Second * commandTimeout)
		gomega.Eventually(session).Should(gexec.Exit(exitCode))
		return err
	}
	err := retryExponentialBackoff(1, f)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
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
	Agent     string
}

// ProcessConfigAndPlugins reads the templates for the config and plugins config maps and replaces the owner, repo, and namespace in them
func ProcessConfigAndPlugins(owner, repo, namespace, agent string) (*config.Config, *cfgplugins.Configuration, error) {
	cfgFile := filepath.Join("test_data", "example-config.tmpl.yml")
	pluginFile := filepath.Join("test_data", "example-plugins.tmpl.yml")

	rawConfig, err := ioutil.ReadFile(cfgFile) /* #nosec */
	if err != nil {
		return nil, nil, errors.Wrapf(err, "reading config template %s", cfgFile)
	}
	rawPlugins, err := ioutil.ReadFile(pluginFile) /* #nosec */
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
		Agent:     agent,
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

	generatedCfg, err := config.LoadYAMLConfig(cfgBuf.Bytes())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unmarshalling config from %s", cfgFile)
	}

	pluginAgent := &plugins.ConfigAgent{}

	generatedPlugin, err := pluginAgent.LoadYAMLConfig(pluginBuf.Bytes())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unmarshalling plugins from %s", pluginFile)
	}

	return generatedCfg, generatedPlugin, nil
}

// CreateWebHook creates a webhook on the SCM provider for the repository
func CreateWebHook(scmClient *scm.Client, repo *scm.Repository, hmacToken string) error {
	output, err := exec.Command("kubectl", "get", "ingress", "hook", "-o", "jsonpath={.spec.rules[0].host}").CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "failed to get hook ingress")
	}
	targetURL := string(output)
	input := &scm.HookInput{
		Name:         "lh-test-hook",
		Target:       fmt.Sprintf("http://%s/hook", targetURL),
		Secret:       hmacToken,
		NativeEvents: []string{"*"},
	}
	_, _, err = scmClient.Repositories.CreateHook(context.Background(), repo.Namespace+"/"+repo.Name, input)

	return err
}

// ApplyConfigAndPluginsConfigMaps takes the config and plugins and creates/applies the config maps in the cluster using kubectl
func ApplyConfigAndPluginsConfigMaps(cfg *config.Config, pluginsCfg *cfgplugins.Configuration) error {
	cfgMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "config",
			Namespace: cfg.LighthouseJobNamespace,
		},
		Data: make(map[string]string),
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
		Data: make(map[string]string),
	}
	pluginData, err := yaml.Marshal(pluginsCfg)
	if err != nil {
		return errors.Wrapf(err, "writing plugins to YAML")
	}
	pluginMap.Data["plugins.yaml"] = string(pluginData)

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

	err = ioutil.WriteFile(cfgFile, cfgYaml, 0600)
	if err != nil {
		return errors.Wrapf(err, "writing config map to %s", cfgFile)
	}
	err = ioutil.WriteFile(pluginFile, pluginYaml, 0600)
	if err != nil {
		return errors.Wrapf(err, "writing plugins map to %s", pluginFile)
	}

	ExpectCommandExecution(tmpDir, 1, 0, "kubectl", "apply", "-f", "config-map.yaml")
	ExpectCommandExecution(tmpDir, 1, 0, "kubectl", "apply", "-f", "plugins-map.yaml")

	return nil
}

// ExpectThatPullRequestHasCommentMatching returns an error if the PR does not have a comment matching the provided function
func ExpectThatPullRequestHasCommentMatching(lhClient scmprovider.SCMClient, pr *scm.PullRequest, matchFunc func(comments []*scm.Comment) error) error {
	f := func() error {
		comments, err := lhClient.ListPullRequestComments(pr.Repository().Namespace, pr.Repository().Name, pr.Number)
		if err != nil {
			return err
		}
		return matchFunc(comments)
	}

	return retryExponentialBackoff(lighthouseActionTimeout, f)
}

// WaitForPullRequestCommitStatus checks a pull request until either it reaches a given status in all the contexts supplied
// or a timeout is reached.
func WaitForPullRequestCommitStatus(lhClient scmprovider.SCMClient, pr *scm.PullRequest, contexts []string, desiredStatuses ...string) {
	gomega.Expect(pr.Sha).ShouldNot(gomega.Equal(""))
	repo := pr.Repository()
	checkPRStatuses := func() error {
		statuses, err := lhClient.ListStatuses(repo.Namespace, repo.Name, pr.Sha)
		if err != nil {
			logInfof("error fetching commit statuses for PR %s/%s/%d: %s\n", repo.Namespace, repo.Name, pr.Number, err)
			return err
		}
		contextStatuses := make(map[string]*scm.Status)
		// For GitHub, only set the status if it's the first one we see for the context, which is always the newest
		// For GitLab, ordering is actually the inverse,

		var orderedStatuses []*scm.Status
		if GitKind() == "gitlab" {
			for i := len(statuses) - 1; i >= 0; i-- {
				orderedStatuses = append(orderedStatuses, statuses[i])
			}
		} else {
			orderedStatuses = append(orderedStatuses, statuses...)
		}
		for _, status := range orderedStatuses {
			if status == nil {
				return err
			}
			if _, exists := contextStatuses[status.Label]; !exists {
				contextStatuses[status.Label] = status
			}
		}

		var matchedStatus *scm.Status
		var wrongStatuses []string

		for _, c := range contexts {
			status, ok := contextStatuses[c]
			if !ok || status == nil {
				wrongStatuses = append(wrongStatuses, fmt.Sprintf("%s: missing", c))
			} else if !isADesiredStatus(status.State.String(), desiredStatuses) {
				wrongStatuses = append(wrongStatuses, fmt.Sprintf("%s: %s", c, status.State.String()))
			} else {
				matchedStatus = status
			}
		}

		if len(wrongStatuses) > 0 || matchedStatus == nil {
			errMsg := fmt.Sprintf("wrong or missing status for PR %s/%s/%d context(s): %s, expected %s", repo.Namespace, repo.Name, pr.Number, strings.Join(wrongStatuses, ", "), strings.Join(desiredStatuses, ","))
			logInfof("WARNING: %s\n", errMsg)
			return errors.New(errMsg)
		}

		return nil
	}

	exponentialBackOff := backoff.NewExponentialBackOff()
	exponentialBackOff.MaxElapsedTime = 15 * time.Minute
	exponentialBackOff.MaxInterval = 10 * time.Second
	exponentialBackOff.Reset()
	err := backoff.Retry(checkPRStatuses, exponentialBackOff)

	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}

func isADesiredStatus(status string, desiredStatuses []string) bool {
	for _, s := range desiredStatuses {
		if status == s {
			return true
		}
	}
	return false
}

const infoPrefix = "      "

// logInfo info logging
func logInfo(message string) {
	fmt.Fprintln(ginkgo.GinkgoWriter, infoPrefix+message)
}

// logInfof info logging
func logInfof(format string, args ...interface{}) {
	fmt.Fprintf(ginkgo.GinkgoWriter, infoPrefix+fmt.Sprintf(format, args...))
}

func RunWithReporters(t *testing.T, suiteId string) {
	reportsDir := os.Getenv("REPORTS_DIR")
	if reportsDir == "" {
		reportsDir = filepath.Join("../", "build", "reports")
	}
	err := os.MkdirAll(reportsDir, 0700)
	if err != nil {
		t.Errorf("cannot create %s because %v", reportsDir, err)
	}
	reporters := make([]ginkgo.Reporter, 0)

	slowSpecThresholdStr := os.Getenv("SLOW_SPEC_THRESHOLD")
	if slowSpecThresholdStr == "" {
		slowSpecThresholdStr = "50000"
		_ = os.Setenv("SLOW_SPEC_THRESHOLD", slowSpecThresholdStr)

	}
	slowSpecThreshold, err := strconv.ParseFloat(slowSpecThresholdStr, 64)
	if err != nil {
		panic(err.Error())
	}
	ginkgoconfig.DefaultReporterConfig.SlowSpecThreshold = slowSpecThreshold
	ginkgoconfig.DefaultReporterConfig.Verbose = testing.Verbose()
	reporters = append(reporters, gr.NewJUnitReporter(filepath.Join(reportsDir, fmt.Sprintf("%s.junit.xml", suiteId))))
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, fmt.Sprintf("Lighthouse E2E tests: %s", suiteId), reporters)
}

// AttemptToLGTMOwnPullRequest return an error if the /lgtm fails to add the lgtm label to PR
func AttemptToLGTMOwnPullRequest(lhClient scmprovider.SCMClient, pr *scm.PullRequest) error {
	repo := pr.Repository()

	err := lhClient.CreateComment(repo.Namespace, repo.Name, pr.Number, true, "/lgtm")
	if err != nil {
		return err
	}

	return ExpectThatPullRequestHasCommentMatching(lhClient, pr, func(comments []*scm.Comment) error {
		for _, c := range comments {
			if strings.Contains(c.Body, "you cannot LGTM your own PR.") {
				return nil
			}
		}
		return fmt.Errorf("couldn't find comment containing the expected message")
	})
}

// AddReviewerToPullRequestWithChatOpsCommand returns an error of the command fails to add the reviewer to either the reviewers list or the assignees list
func AddReviewerToPullRequestWithChatOpsCommand(lhClient scmprovider.SCMClient, pr *scm.PullRequest, reviewer string) error {
	ginkgo.By(fmt.Sprintf("Adding the '/cc %s' comment and waiting for %s to be a reviewer", reviewer, reviewer))
	repo := pr.Repository()
	err := lhClient.CreateComment(repo.Namespace, repo.Name, pr.Number, true, fmt.Sprintf("/cc %s", reviewer))

	err = ExpectThatPullRequestMatches(lhClient, pr.Number, repo.Namespace, repo.Name, func(request *scm.PullRequest) error {
		if len(request.Assignees) == 0 && len(request.Reviewers) == 0 {
			return fmt.Errorf("expected %s as reviewer, but no reviewers or assignees set on PR")
		}
		for _, r := range request.Reviewers {
			if r.Login == reviewer {
				return nil
			}
		}
		for _, a := range request.Assignees {
			if a.Login == reviewer {
				return nil
			}
		}
		return fmt.Errorf("expected %s as a reviewer, but the user is not present in reviewers or assignees on the PR", reviewer)
	})
	if err != nil {
		return err
	}

	ginkgo.By(fmt.Sprintf("Adding the '/uncc %s' comment and waiting for the user to be gone from reviewers", reviewer))
	err = lhClient.CreateComment(repo.Namespace, repo.Name, pr.Number, true, fmt.Sprintf("/uncc %s", reviewer))
	if err != nil {
		return err
	}

	return ExpectThatPullRequestMatches(lhClient, pr.Number, repo.Namespace, repo.Name, func(request *scm.PullRequest) error {
		if len(request.Assignees) == 0 && len(request.Reviewers) == 0 {
			return nil
		}
		for _, r := range request.Reviewers {
			if r.Login == reviewer {
				return fmt.Errorf("expected %s to be removed from reviewers but is still present", reviewer)
			}
		}
		for _, a := range request.Assignees {
			if a.Login == reviewer {
				return fmt.Errorf("expected %s to be removed from assignees but is still present", reviewer)
			}
		}
		return nil
	})
}

// ExpectThatPullRequestMatches returns an error if the PR does not satisfy the provided funciton
func ExpectThatPullRequestMatches(lhClient scmprovider.SCMClient, pullRequestNumber int, owner, repo string, matchFunc func(request *scm.PullRequest) error) error {
	f := func() error {
		pullRequest, err := lhClient.GetPullRequest(owner, repo, pullRequestNumber)
		if err != nil {
			return err
		}
		return matchFunc(pullRequest)
	}

	return retryExponentialBackoff(lighthouseActionTimeout, f)
}

// ExpectThatPullRequestHasLabel returns an error if the PR does not have the specified label
func ExpectThatPullRequestHasLabel(lhClient scmprovider.SCMClient, pullRequestNumber int, owner, repo, label string) error {
	return ExpectThatPullRequestMatches(lhClient, pullRequestNumber, owner, repo, func(request *scm.PullRequest) error {
		if len(request.Labels) < 1 {
			return fmt.Errorf("the pull request has no labels")
		}
		for _, l := range request.Labels {
			if l.Name == label {
				return nil
			}
		}
		return fmt.Errorf("the pull request does not have the specified label: %s", label)

	})
}

// ExpectThatPullRequestDoesNotHaveLabel returns an error if the PR does have the specified label
func ExpectThatPullRequestDoesNotHaveLabel(lhClient scmprovider.SCMClient, pullRequestNumber int, owner, repo, label string) error {
	return ExpectThatPullRequestMatches(lhClient, pullRequestNumber, owner, repo, func(request *scm.PullRequest) error {
		if len(request.Labels) < 1 {
			return nil
		}
		for _, l := range request.Labels {
			if l.Name == label {
				return fmt.Errorf("the pull request has the specified label %s but shouldn't", label)
			}
		}
		return nil

	})
}

// AddHoldLabelToPullRequestWithChatOpsCommand returns an error of the command fails to add the do-not-merge/hold label
func AddHoldLabelToPullRequestWithChatOpsCommand(lhClient scmprovider.SCMClient, pr *scm.PullRequest) error {
	repo := pr.Repository()
	ginkgo.By("Adding the /hold comment and waiting for the label to be present")
	err := lhClient.CreateComment(repo.Namespace, repo.Name, pr.Number, true, "/hold")
	if err != nil {
		return err
	}

	err = ExpectThatPullRequestHasLabel(lhClient, pr.Number, repo.Namespace, repo.Name, "do-not-merge/hold")
	if err != nil {
		return err
	}

	ginkgo.By("Adding the /hold cancel comment and waiting for the label to be gone")
	err = lhClient.CreateComment(repo.Namespace, repo.Name, pr.Number, true, "/hold cancel")
	if err != nil {
		return err
	}

	return ExpectThatPullRequestDoesNotHaveLabel(lhClient, pr.Number, repo.Namespace, repo.Name, "do-not-merge/hold")
}

// AddWIPLabelToPullRequestByUpdatingTitle adds the WIP label by adding WIP to a pull request's title
func AddWIPLabelToPullRequestByUpdatingTitle(lhClient scmprovider.SCMClient, scmClient *scm.Client, pr *scm.PullRequest) error {
	repo := pr.Repository()
	originalTitle := pr.Title

	ginkgo.By("Changing the pull request title to start with WIP and waiting for the label to be present")

	input := &scm.PullRequestInput{
		Title: fmt.Sprintf("WIP %s", originalTitle),
	}
	_, _, err := scmClient.PullRequests.Update(context.Background(), fmt.Sprintf("%s/%s", repo.Namespace, repo.Name), pr.Number, input)
	if err != nil {
		return err
	}
	err = ExpectThatPullRequestHasLabel(lhClient, pr.Number, repo.Namespace, repo.Name, "do-not-merge/work-in-progress")
	if err != nil {
		return err
	}

	ginkgo.By("Changing the pull request title to remove the WIP and waiting for the label to be gone")
	input = &scm.PullRequestInput{
		Title: originalTitle,
	}
	_, _, err = scmClient.PullRequests.Update(context.Background(), fmt.Sprintf("%s/%s", repo.Namespace, repo.Name), pr.Number, input)
	if err != nil {
		return err
	}

	return ExpectThatPullRequestDoesNotHaveLabel(lhClient, pr.Number, repo.Namespace, repo.Name, "do-not-merge/work-in-progress")
}

// ApprovePullRequest attempts to /approve a PR with the given approver git provider, then verify the label is there with the default provider
func ApprovePullRequest(lhClient scmprovider.SCMClient, approverclient scmprovider.SCMClient, pr *scm.PullRequest) error {
	repo := pr.Repository()

	ginkgo.By("approving the PR")
	approveCmd := "approve"
	if GitKind() == "gitlab" {
		approveCmd = "lh-" + approveCmd
	}

	err := approverclient.CreateComment(repo.Namespace, repo.Name, pr.Number, true, fmt.Sprintf("/%s", approveCmd))
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	ginkgo.By("waiting for the approved label to appear")
	return ExpectThatPullRequestHasLabel(lhClient, pr.Number, repo.Namespace, repo.Name, "approved")
}

// WaitForPullRequestToMerge checks the PR's status until it's merged or timed out.
func WaitForPullRequestToMerge(lhClient scmprovider.SCMClient, pr *scm.PullRequest) {
	repo := pr.Repository()
	waitForMergeFunc := func() error {
		updatedPR, err := lhClient.GetPullRequest(repo.Namespace, repo.Name, pr.Number)
		if err != nil {
			logInfof("WARNING: Error getting pull request: %s\n", err)
			return err
		}
		if updatedPR == nil {
			err = fmt.Errorf("got a nil PR for %s", pr.Link)
			logInfof("WARNING: %s\n", err)
			return err
		}
		if updatedPR.Merged {
			return nil
		} else {
			err = fmt.Errorf("PR %s not yet merged", pr.Link)
			logInfof("WARNING: %s, sleeping and retrying\n", err)
			return err
		}
	}

	err := retryExponentialBackoff(15*time.Minute, waitForMergeFunc)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}
