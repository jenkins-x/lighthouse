package jenkins

import (
	"bytes"
	"context"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/hashicorp/go-multierror"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/test/e2e"
	"github.com/jenkins-x/lighthouse/test/e2e/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"
)

const (
	jenkinsTestJobName = "pr-build"
	prBranch           = "pr-branch"
	defaultContext     = "pr-build"
)

var (
	hmacToken       string
	gitClient       git.Client
	scmClient       *scm.Client
	approverClient  *scm.Client
	jenkinsClient   Client
	gitServerURL    string
	testRepo        *scm.Repository
	testPullRequest *scm.PullRequest
	localClone      *git.Repo
	err             error
)

func TestJenkins(t *testing.T) {
	e2e.RunWithReporters(t, "JenkinsIntegrationTest")
}

var _ = BeforeSuite(func() {
	By("setting up logging")
	if os.Getenv("E2E_QUIET_LOG") == "true" {
		logrus.SetOutput(ioutil.Discard)
	}

	By("ensuring environment configured")
	err = ensureEnvironment()
	Expect(err).ShouldNot(HaveOccurred())

	By("creating webhook secret")
	hmacToken, err = e2e.CreateHMACToken()
	Expect(err).ShouldNot(HaveOccurred())
	Expect(hmacToken).ShouldNot(BeEmpty())

	By("creating primary SCM client")
	scmClient, _, gitServerURL, err = e2e.CreateSCMClient(e2e.GetBotName, e2e.GetPrimarySCMToken)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(scmClient).ShouldNot(BeNil())
	Expect(gitServerURL).ShouldNot(BeEmpty())

	By("creating approver SCM client")
	approverClient, _, _, err = e2e.CreateSCMClient(e2e.GetApproverName, e2e.GetApproverSCMToken)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(approverClient).ShouldNot(BeNil())

	By("creating Git client")
	gitClient, err = e2e.CreateGitClient(gitServerURL, e2e.GetBotName, e2e.GetPrimarySCMToken)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(gitClient).ShouldNot(BeNil())

	By("creating test repository")
	testRepo, localClone, err = e2e.CreateBaseRepository(e2e.GetBotName(), e2e.GetApproverName(), scmClient, gitClient)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(testRepo).ShouldNot(BeNil())
	Expect(localClone).ShouldNot(BeNil())
	repoFullName := fmt.Sprintf("%s/%s", testRepo.Namespace, testRepo.Name)
	logrus.Infof("%s", repoFullName)

	By(fmt.Sprintf("adding %s to new repository", e2e.GetApproverName()))
	err = e2e.AddCollaborator(e2e.GetApproverName(), testRepo, scmClient, approverClient)
	Expect(err).ShouldNot(HaveOccurred())

	By(fmt.Sprintf("creating and populating Lighthouse config for %s", testRepo.Clone))
	testNamespace := os.Getenv(e2e.TestNamespace)
	jenkinsURL := os.Getenv(e2e.JenkinsURL)
	cfg, pluginCfg, err := e2e.ProcessConfigAndPlugins(testRepo.Namespace, testRepo.Name, testNamespace, job.JenkinsAgent, jenkinsURL)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(cfg).ShouldNot(BeNil())
	Expect(pluginCfg).ShouldNot(BeNil())

	err = e2e.ApplyConfigAndPluginsConfigMaps(cfg, pluginCfg)
	Expect(err).ShouldNot(HaveOccurred())

	By(fmt.Sprintf("setting up webhooks for %s", testRepo.Clone))
	err = e2e.CreateWebHook(scmClient, testRepo, hmacToken)
	Expect(err).ShouldNot(HaveOccurred())

	By(fmt.Sprintf("creating Jenkins Job for %s", testRepo.Clone))
	jenkinsClient = NewJenkinsClient(os.Getenv(e2e.JenkinsURL), os.Getenv(e2e.JenkinsUser), os.Getenv(e2e.JenkinsAPIToken))
	jobExists, err := jenkinsClient.JobExists(jenkinsTestJobName)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(jobExists).Should(Equal(false))

	err = createJenkinsJob(jenkinsClient, jenkinsTestJobName)
	Expect(err).ShouldNot(HaveOccurred())

	jobExists, err = jenkinsClient.JobExists(jenkinsTestJobName)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(jobExists).Should(Equal(true))
})

var _ = AfterSuite(func() {
	// Delete is currently not implemented in go-scm!?
	//_, err = scmClient.Repositories.Delete(context.Background(), testRepo.FullName)
	//Expect(err).ShouldNot(HaveOccurred())

	err = jenkinsClient.DeleteJob(jenkinsTestJobName)
	Expect(err).ShouldNot(HaveOccurred())

	err = gitClient.Clean()
	Expect(err).ShouldNot(HaveOccurred())
})

var _ = ChatOpsTests()

func ChatOpsTests() bool {
	return Describe("Lighthouse Jenkins support", func() {
		It("creates PR and triggers successful Jenkins pipeline run", func() {
			succeedingPRBranch := prBranch + "-success"

			By("creating pr branch, and pushing it")
			err = localClone.CheckoutNewBranch(succeedingPRBranch)
			Expect(err).ShouldNot(HaveOccurred())

			newFile := filepath.Join(localClone.Dir, "README")
			err = ioutil.WriteFile(newFile, []byte("Hello world"), 0600)
			e2e.ExpectCommandExecution(localClone.Dir, 1, 0, "git", "add", newFile)
			e2e.ExpectCommandExecution(localClone.Dir, 1, 0, "git", "commit", "-a", "-m", "Adding for test PR")

			err = localClone.Push(testRepo.Name, succeedingPRBranch)
			Expect(err).ShouldNot(HaveOccurred())

			By("creating a pull request")
			prInput := &scm.PullRequestInput{
				Title: "Lighthouse Succeeding Test PR",
				Head:  succeedingPRBranch,
				Base:  "master",
				Body:  "Test PR for Lighthouse",
			}
			testPullRequest, _, err = scmClient.PullRequests.Create(context.Background(), testRepo.FullName, prInput)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(testPullRequest).ShouldNot(BeNil())

			By("verifying OWNERS link in APPROVALNOTIFIER comment is correct")
			matchFunc := func(comments []*scm.Comment) error {
				for _, c := range comments {
					if strings.Contains(c.Body, "[APPROVALNOTIFIER]") {
						ownerRegex := regexp.MustCompile(`(?m).*\[OWNERS]\((.*)\).*`)
						matches := ownerRegex.FindStringSubmatch(c.Body)
						if len(matches) == 0 {
							return backoff.Permanent(fmt.Errorf("could not find OWNERS link in:\n%s", c.Body))
						}
						expected := e2e.URLForFile(e2e.GitKind(), gitServerURL, testRepo.Namespace, testRepo.Name, "OWNERS")
						if expected != matches[1] {
							return backoff.Permanent(fmt.Errorf("expected OWNERS URL %s, but got %s", expected, matches[1]))
						}
						return nil
					}
				}
				return fmt.Errorf("couldn't find comment containing APPROVALNOTIFIER")
			}
			err = e2e.ExpectThatPullRequestHasCommentMatching(scmClient, testPullRequest, matchFunc)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for build to succeed")
			e2e.WaitForPullRequestCommitStatus(scmClient, testPullRequest, []string{defaultContext}, "success")

			status, err := e2e.GetCommitStatus(scmClient, testRepo.FullName, testPullRequest.Sha, defaultContext)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).NotTo(BeNil())
			Expect(status.Target).NotTo(BeEmpty())
			Expect(status.Target).To(HavePrefix(os.Getenv(e2e.JenkinsURL)))

			log, err := e2e.BuildLog(status.Target)
			Expect(err).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Finished: SUCCESS"))
		})

		It("creates PR and triggers failing Jenkins pipeline run", func() {
			failingPRBranch := prBranch + "-failure"
			By("creating pr branch, and pushing it")
			err = localClone.CheckoutNewBranch(failingPRBranch)
			Expect(err).ShouldNot(HaveOccurred())

			By("creating a failing go file")
			failFile := filepath.Join("test_data", "main.go.failing")
			failFileContent, err := ioutil.ReadFile(filepath.Clean(failFile))
			Expect(err).ShouldNot(HaveOccurred())

			outFile := filepath.Join(localClone.Dir, "main.go")
			err = ioutil.WriteFile(outFile, failFileContent, 0600)
			Expect(err).ShouldNot(HaveOccurred())

			e2e.ExpectCommandExecution(localClone.Dir, 1, 0, "git", "commit", "-a", "-m", "Updating main.go for failing test PR")

			err = localClone.Push(testRepo.Name, failingPRBranch)
			Expect(err).ShouldNot(HaveOccurred())

			By("creating a pull request")
			prInput := &scm.PullRequestInput{
				Title: "Lighthouse Failing Test PR",
				Head:  failingPRBranch,
				Base:  "master",
				Body:  "Test PR for Lighthouse",
			}
			testPullRequest, _, err = scmClient.PullRequests.Create(context.Background(), testRepo.FullName, prInput)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(testPullRequest).ShouldNot(BeNil())

			By("waiting for the PR build to fail")
			e2e.WaitForPullRequestCommitStatus(scmClient, testPullRequest, []string{defaultContext}, "failure")

			status, err := e2e.GetCommitStatus(scmClient, testRepo.FullName, testPullRequest.Sha, defaultContext)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).NotTo(BeNil())
			Expect(status.Target).NotTo(BeEmpty())
			Expect(status.Target).To(HavePrefix(os.Getenv(e2e.JenkinsURL)))

			log, err := e2e.BuildLog(status.Target)
			Expect(err).NotTo(HaveOccurred())
			Expect(log).To(ContainSubstring("Finished: FAILURE"))
		})
	})
}

func createJenkinsJob(jc Client, jobName string) error {
	testJobFileName := filepath.Join("test_data", "jenkins", "config.xml")
	jobTemplate, err := template.ParseFiles(filepath.Clean(testJobFileName))
	if err != nil {
		return errors.Wrapf(err, "parsing job template from %s", testJobFileName)
	}

	var buffer bytes.Buffer
	envContext := util.Env()
	err = jobTemplate.Execute(&buffer, envContext)
	if err != nil {
		return errors.Wrapf(err, "applying job template from %s", testJobFileName)
	}

	err = jc.CreateJob(jobName, bytes.NewReader(buffer.Bytes()))
	if err != nil {
		return errors.Wrap(err, "error posting job config")
	}

	return nil
}

func ensureEnvironment() error {
	var missingEnvVarErrors []error

	requiredEnvVars := []string{
		e2e.PrimarySCMUserEnvVar,
		e2e.PrimarySCMTokenEnvVar,
		e2e.GitServerEnvVar,
		e2e.ApproverSCMUserEnvVar,
		e2e.ApproverSCMTokenEnvVar,
		e2e.TestNamespace,
		e2e.JenkinsURL,
		e2e.JenkinsUser,
		e2e.JenkinsAPIToken,
		e2e.JenkinsGitCredentialID,
	}

	for _, envVar := range requiredEnvVars {
		_, exist := os.LookupEnv(envVar)
		if !exist {
			err := fmt.Errorf("the environment variable %s needs to be set for executing this test", envVar)
			missingEnvVarErrors = append(missingEnvVarErrors, err)
		}
	}

	multiErr := multierror.Error{
		Errors: missingEnvVarErrors,
	}
	return multiErr.ErrorOrNil()
}
