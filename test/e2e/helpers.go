package e2e

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net/http"
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
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/repoowners"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	util2 "github.com/jenkins-x/lighthouse/test/e2e/util"
	"github.com/onsi/gomega/gexec"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/onsi/ginkgo"
	ginkgoconfig "github.com/onsi/ginkgo/config"
	gr "github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
)

const (
	// TestNamespace the namespace in which to run the test.
	TestNamespace = "E2E_TEST_NAMESPACE" /* #nosec */
	// PrimarySCMTokenEnvVar is the name of the environment variable containing the Git SCM token for the bot user.
	PrimarySCMTokenEnvVar = "E2E_PRIMARY_SCM_TOKEN" /* #nosec */
	// PrimarySCMUserEnvVar is the name of the environment variable containing the Git SCM username of the bot user.
	PrimarySCMUserEnvVar = "E2E_PRIMARY_SCM_USER"
	// ApproverSCMTokenEnvVar is the name of the environment variable containing the Git SCM token for the approver.
	ApproverSCMTokenEnvVar = "E2E_APPROVER_SCM_TOKEN" /* #nosec */
	// ApproverSCMUserEnvVar is the name of the environment variable containing the Git SCM username of the approver.
	ApproverSCMUserEnvVar = "E2E_APPROVER_SCM_USER"
	// HmacTokenEnvVar is the name of the environment variable containing the webhook secret.
	HmacTokenEnvVar = "E2E_HMAC_TOKEN" /* #nosec */
	// GitServerEnvVar is the name of the environment variable containing URL to the Git SCM provider
	GitServerEnvVar = "E2E_GIT_SERVER"
	// GitKindEnvVar is the name of the environment variable containing the Git SCM kind
	GitKindEnvVar = "E2E_GIT_KIND"
	// BaseRepoName is the name of the environment variable containing is the base name for the test repo. Will be suffixed with a random seed.
	BaseRepoName = "lh-e2e-test"
	// TestRepoName name of the test repo.
	TestRepoName = "E2E_TEST_REPO"
	// JenkinsURL the URL to the Jenkins test instance.
	JenkinsURL = "E2E_JENKINS_URL"
	// JenkinsUser username for making Jenkins API requests.
	JenkinsUser = "E2E_JENKINS_USER"
	// JenkinsAPIToken API token for Jenkins.
	JenkinsAPIToken = "E2E_JENKINS_API_TOKEN" /* #nosec */
	// JenkinsGitCredentialID id of the global Jenkins Git credentials
	JenkinsGitCredentialID = "E2E_JENKINS_GIT_CREDENTIAL_ID"
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
	serverURL := os.Getenv(GitServerEnvVar)

	token, err := tokenFunc()
	if err != nil {
		return nil, nil, "", err
	}

	botName := GetBotName()
	client, err := factory.NewClient(kind, serverURL, token, factory.SetUsername(botName))

	util.AddAuthToSCMClient(client, token, false)

	spc := scmprovider.ToClient(client, userFunc())
	return client, spc, serverURL, err
}

// GitKind returns the git provider flavor being used
func GitKind() string {
	kind := os.Getenv(GitKindEnvVar)
	if kind == "" {
		kind = "github"
	}
	return kind
}

// CreateHMACToken creates an HMAC token for use in webhooks, defaulting to the E2E_HMAC_TOKEN env var if set
func CreateHMACToken() (string, error) {
	fromEnv := os.Getenv(HmacTokenEnvVar)
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
	botName := os.Getenv(PrimarySCMUserEnvVar)
	if botName == "" {
		botName = "jenkins-x-bot"
	}
	return botName
}

// GetApproverName gets the approver user's username
func GetApproverName() string {
	botName := os.Getenv(ApproverSCMUserEnvVar)
	if botName == "" {
		botName = "jenkins-x-bot"
	}
	return botName
}

// GetPrimarySCMToken gets the token used by the bot/primary user
func GetPrimarySCMToken() (string, error) {
	return getSCMToken(PrimarySCMTokenEnvVar, GitKind())
}

// GetApproverSCMToken gets the token used by the approver
func GetApproverSCMToken() (string, error) {
	return getSCMToken(ApproverSCMTokenEnvVar, GitKind())
}

func getSCMToken(envName, gitKind string) (string, error) {
	value := os.Getenv(envName)
	if value == "" {
		return value, fmt.Errorf("no token available for git kind %s at environment variable $%s", gitKind, envName)
	}
	return value, nil
}

// CreateBaseRepository creates the repository that will be used for tests
func CreateBaseRepository(botUser, approver string, botClient *scm.Client, gitClient git.Client) (*scm.Repository, *git.Repo, error) {
	repoName := BaseRepoName + "-" + strconv.FormatInt(ginkgo.GinkgoRandomSeed(), 10)

	input := &scm.RepositoryInput{
		Name:    repoName,
		Private: true,
	}

	repo, _, err := botClient.Repositories.Create(context.Background(), input)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create repository")
	}
	_ = os.Setenv(TestRepoName, repo.Clone)

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

	testRepoDir := filepath.Join("test_data", "repo")
	err = util2.Copy(testRepoDir, r.Dir, true)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	ExpectCommandExecution(r.Dir, 1, 0, "git", "add", ".")

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

	err = os.WriteFile(ownersFile, ownersYaml, 0600)
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
	_, alreadyCollaborator, _, err := botClient.Repositories.AddCollaborator(context.Background(), fmt.Sprintf("%s/%s", repo.Namespace, repo.Name), approver, scm.WritePermission)
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
		session, err := gexec.Start(command, logrus.StandardLogger().Out, logrus.StandardLogger().Out)
		if err != nil {
			return err
		}
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
	Owner      string
	Repo       string
	Namespace  string
	Agent      string
	JenkinsURL string
}

// ProcessConfigAndPlugins reads the templates for the config and plugins config maps and replaces the owner, repo, and namespace in them
func ProcessConfigAndPlugins(owner, repo, namespace, agent, jenkinsURL string) (*config.Config, *plugins.Configuration, error) {
	cfgFile := filepath.Join("test_data", "example-config.tmpl.yml")
	pluginFile := filepath.Join("test_data", "example-plugins.tmpl.yml")

	rawConfig, err := os.ReadFile(cfgFile) /* #nosec */
	if err != nil {
		return nil, nil, errors.Wrapf(err, "reading config template %s", cfgFile)
	}
	rawPlugins, err := os.ReadFile(pluginFile) /* #nosec */
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
		Owner:      owner,
		Repo:       repo,
		Namespace:  namespace,
		Agent:      agent,
		JenkinsURL: jenkinsURL,
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
	output, err := exec.Command("kubectl", "get", "ingress", "-l", "app=lighthouse-webhooks", "-o", "jsonpath={.items[0].spec.rules[0].host}").CombinedOutput()
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
	if scmClient.Driver.String() == "gitea" {
		input.Events.Issue = true
		input.Events.PullRequest = true
		input.Events.Branch = true
		input.Events.IssueComment = true
		input.Events.PullRequestComment = true
		input.Events.Push = true
		input.Events.ReviewComment = true
		input.Events.Tag = true
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

	tmpDir, err := os.MkdirTemp("", "kubectl")
	if err != nil {
		return errors.Wrapf(err, "creating temp directory")
	}
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

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

	err = os.WriteFile(cfgFile, cfgYaml, 0600)
	if err != nil {
		return errors.Wrapf(err, "writing config map to %s", cfgFile)
	}
	err = os.WriteFile(pluginFile, pluginYaml, 0600)
	if err != nil {
		return errors.Wrapf(err, "writing plugins map to %s", pluginFile)
	}

	ExpectCommandExecution(tmpDir, 1, 0, "kubectl", "apply", "-f", "config-map.yaml")
	ExpectCommandExecution(tmpDir, 1, 0, "kubectl", "apply", "-f", "plugins-map.yaml")

	return nil
}

// ExpectThatPullRequestHasCommentMatching returns an error if the PR does not have a comment matching the provided function
func ExpectThatPullRequestHasCommentMatching(scmClient *scm.Client, pr *scm.PullRequest, matchFunc func(comments []*scm.Comment) error) error {
	f := func() error {
		comments, _, err := scmClient.PullRequests.ListComments(context.Background(), pr.Repository().FullName, pr.Number, &scm.ListOptions{})
		if err != nil {
			return err
		}
		return matchFunc(comments)
	}

	return retryExponentialBackoff(lighthouseActionTimeout, f)
}

// WaitForPullRequestCommitStatus checks a pull request until either it reaches a given status in all the contexts supplied
// or a timeout is reached.
func WaitForPullRequestCommitStatus(scmClient *scm.Client, pr *scm.PullRequest, contexts []string, desiredStatuses ...string) {
	gomega.Expect(pr.Sha).ShouldNot(gomega.Equal(""))
	repo := pr.Repository()

	checkPRStatuses := func() error {
		updatedPR, _, err := scmClient.PullRequests.Find(context.Background(), repo.FullName, pr.Number)
		if err != nil {
			return err
		}
		statuses, _, err := scmClient.Repositories.ListStatus(context.Background(), repo.FullName, updatedPR.Sha, &scm.ListOptions{})
		if err != nil {
			logInfof("error fetching commit statuses for PR %s/%s/%d: %s\n", repo.Namespace, repo.Name, updatedPR.Number, err)
			return err
		}
		contextStatuses := commitStatusesByContext(statuses)

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
			errMsg := fmt.Sprintf("wrong or missing status for PR %s/%s/%d context(s): %s, expected %s", repo.Namespace, repo.Name, updatedPR.Number, strings.Join(wrongStatuses, ", "), strings.Join(desiredStatuses, ","))
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

// GetCommitStatus retrieves the status check with the specified name for the specified sha
func GetCommitStatus(scmClient *scm.Client, repoName string, sha string, statusName string) (*scm.Status, error) {
	statuses, _, err := scmClient.Repositories.ListStatus(context.Background(), repoName, sha, &scm.ListOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list commit status for %s in %s", sha, repoName)
	}
	contextStatuses := commitStatusesByContext(statuses)
	return contextStatuses[statusName], nil
}

// BuildLog retrieves the build log as string from the specified URL.
func BuildLog(url string) (string, error) {
	resp, err := http.Get(url) // #nosec G107
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("unexpected HTTP response status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	log := string(bodyBytes)
	return log, nil
}

func commitStatusesByContext(statuses []*scm.Status) map[string]*scm.Status {
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
			logInfo("WARNING: nil commit status retrieved")
			continue
		}
		if _, exists := contextStatuses[status.Label]; !exists {
			contextStatuses[status.Label] = status
		}
	}

	return contextStatuses
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
	_, _ = fmt.Fprint(logrus.StandardLogger().Out, infoPrefix+fmt.Sprint(message))
}

// logInfof info logging
func logInfof(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(logrus.StandardLogger().Out, infoPrefix+fmt.Sprintf(format, args...))
}

// RunWithReporters runs a suite with better logging and gathering of test results
func RunWithReporters(t *testing.T, suiteID string) {
	reportsDir := os.Getenv("REPORTS_DIR")
	if reportsDir == "" {
		reportsDir = filepath.Join("..", "..", "..", "build", "reports")
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
	reporters = append(reporters, gr.NewJUnitReporter(filepath.Join(reportsDir, fmt.Sprintf("%s.junit.xml", suiteID))))
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, fmt.Sprintf("Lighthouse E2E tests: %s", suiteID), reporters)
}

// AttemptToLGTMOwnPullRequest return an error if the /lgtm fails to add the lgtm label to PR
func AttemptToLGTMOwnPullRequest(scmClient *scm.Client, pr *scm.PullRequest) error {
	repo := pr.Repository()

	_, _, err := scmClient.PullRequests.CreateComment(context.Background(), repo.FullName, pr.Number, &scm.CommentInput{Body: "/lgtm"})
	if err != nil {
		return err
	}

	return ExpectThatPullRequestHasCommentMatching(scmClient, pr, func(comments []*scm.Comment) error {
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
	_ = lhClient.CreateComment(repo.Namespace, repo.Name, pr.Number, true, fmt.Sprintf("/cc %s", reviewer))

	err := ExpectThatPullRequestMatches(lhClient, pr.Number, repo.Namespace, repo.Name, func(request *scm.PullRequest) error {
		if len(request.Assignees) == 0 && len(request.Reviewers) == 0 {
			return fmt.Errorf("expected %s as reviewer, but no reviewers or assignees set on PR", reviewer)
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

// ExpectThatPullRequestMatches returns an error if the PR does not satisfy the provided function
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
func ApprovePullRequest(lhClient scmprovider.SCMClient, approverClient scmprovider.SCMClient, pr *scm.PullRequest) error {
	repo := pr.Repository()

	ginkgo.By("approving the PR")
	approveCmd := "approve"
	if GitKind() == "gitlab" {
		approveCmd = "lh-" + approveCmd
	}

	err := approverClient.CreateComment(repo.Namespace, repo.Name, pr.Number, true, fmt.Sprintf("/%s", approveCmd))
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
		}
		err = fmt.Errorf("PR %s not yet merged", pr.Link)
		logInfof("WARNING: %s, sleeping and retrying\n", err)
		return err
	}

	err := retryExponentialBackoff(15*time.Minute, waitForMergeFunc)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
}

// URLForFile returns the SCM provider URL for a specific file in the repository.
func URLForFile(providerType string, serverURL string, owner string, repo string, path string) string {
	switch providerType {
	case "stash":
		return fmt.Sprintf("%s/projects/%s/repos/%s/browse/%s", serverURL, strings.ToUpper(owner), repo, path)
	case "gitlab":
		return fmt.Sprintf("%s/%s/%s/-/blob/master/%s", serverURL, owner, repo, path)
	case "gitea":
		return fmt.Sprintf("%s/%s/%s/src/branch/master/%s", serverURL, owner, repo, path)
	default:
		return fmt.Sprintf("%s/%s/%s/blob/master/%s", serverURL, owner, repo, path)
	}
}
