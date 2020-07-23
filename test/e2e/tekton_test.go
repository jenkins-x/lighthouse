package e2e

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse-config/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/engines/tekton"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	ns = "lh-test"
)

var (
	hmacToken      string
	gitClient      git.Client
	scmClient      *scm.Client
	spc            scmprovider.SCMClient
	approverClient *scm.Client
	gitServerURL   string
	repo           *scm.Repository
	repoDir        string
)

var _ = BeforeSuite(func() {
	var err error
	By("creating HMAC token")
	hmacToken, err = CreateHMACToken()
	Expect(err).ShouldNot(HaveOccurred())
	Expect(hmacToken).ShouldNot(BeEmpty())

	By("creating primary SCM client")
	scmClient, spc, gitServerURL, err = CreateSCMClient(GetBotName, GetPrimarySCMToken)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(scmClient).ShouldNot(BeNil())
	Expect(spc).ShouldNot(BeNil())
	Expect(gitServerURL).ShouldNot(BeEmpty())

	By("creating approver SCM client")
	approverClient, _, _, err = CreateSCMClient(GetApproverName, GetApproverSCMToken)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(approverClient).ShouldNot(BeNil())

	By("creating git client")
	gitClient, err = CreateGitClient(gitServerURL, GetBotName, GetPrimarySCMToken)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(gitClient).ShouldNot(BeNil())

	By("creating repository")
	repo, repoDir, err = CreateBaseRepository(GetBotName(), GetApproverName(), scmClient, gitClient)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(repo).ShouldNot(BeNil())
	Expect(repoDir).ShouldNot(BeEmpty())

	By("adding the Pipeline and Task definitions to the cluster")
	err = applyPipelineAndTask()
	Expect(err).ShouldNot(HaveOccurred())

	By(fmt.Sprintf("creating and populating Lighthouse config for %s", repo.Clone))
	cfg, pluginCfg, err := ProcessConfigAndPlugins(repo.Namespace, repo.Name, ns)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(cfg).ShouldNot(BeNil())
	Expect(pluginCfg).ShouldNot(BeNil())

	cfg.Presubmits[fmt.Sprintf("%s/%s", repo.Namespace, repo.Name)][0].PipelineRunSpec = generatePipelineRunSpec()
	cfg.Presubmits[fmt.Sprintf("%s/%s", repo.Namespace, repo.Name)][0].Agent = config.TektonPipelineAgent

	err = ApplyConfigAndPluginsConfigMaps(cfg, pluginCfg)
	Expect(err).ShouldNot(HaveOccurred())

	By(fmt.Sprintf("setting up webhooks for %s", repo.Clone))
	err = CreateWebHook(scmClient, repo, hmacToken)
	Expect(err).ShouldNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	err := gitClient.Clean()
	if err != nil {
		logrus.WithError(err).Fatal("Error cleaning the git client.")
	}

})

func TestTekton(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lighthouse Tekton")
}

var _ = ChatOpsTests()

func ChatOpsTests() bool {
	return Describe("Lighthouse Tekton support", func() {
/*		var (
			T                helpers.TestOptions
			err              error
			approverProvider gits.GitProvider
		)

		BeforeEach(func() {
			provider, err = T.GetGitProvider()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(provider).ShouldNot(BeNil())

			approverProvider, err = T.GetApproverGitProvider()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(approverProvider).ShouldNot(BeNil())

			qsNameParts := strings.Split(lhQuickstart, "-")
			qsAbbr := ""
			for s := range qsNameParts {
				qsAbbr = qsAbbr + qsNameParts[s][:1]

			}
			applicationName := helpers.TempDirPrefix + qsAbbr + "-" + strconv.FormatInt(GinkgoRandomSeed(), 10)
			T = helpers.TestOptions{
				ApplicationName: applicationName,
				WorkDir:         helpers.WorkDir,
			}
			T.GitProviderURL()

			utils.LogInfof("Creating application %s in dir %s\n", util.ColorInfo(applicationName), util.ColorInfo(helpers.WorkDir))
		})

		Describe("Create a repository", func() {
			Context(fmt.Sprintf("by running jx create quickstart %s", lhQuickstart), func() {
				It("creates a new source repository", func() {

					args := []string{"create", "quickstart", "-b", "--org", T.GetGitOrganisation(), "-p", T.ApplicationName, "-f", lhQuickstart}

					gitProviderUrl, err := T.GitProviderURL()
					Expect(err).NotTo(HaveOccurred())
					if gitProviderUrl != "" {
						utils.LogInfof("Using Git provider URL %s\n", gitProviderUrl)
						args = append(args, "--git-provider-url", gitProviderUrl)
					}
					argsStr := strings.Join(args, " ")
					By(fmt.Sprintf("calling jx %s", argsStr), func() {
						T.ExpectJxExecution(T.WorkDir, helpers.TimeoutSessionWait, 0, args...)
					})

					By("adding the approver to OWNERS", func() {
						createdPR := T.CreatePullRequestWithLocalChange(fmt.Sprintf("Adding %s to OWNERS", helpers.PullRequestApproverUsername), func(workDir string) {
							// overwrite the existing OWNERS with a new one containing the approver user
							fileName := "OWNERS"
							owners := filepath.Join(workDir, fileName)

							data := []byte(fmt.Sprintf("approvers:\n- %s\n- %s\nreviewers:\n- %s\n- %s\n",
								provider.UserAuth().Username, helpers.PullRequestApproverUsername,
								provider.UserAuth().Username, helpers.PullRequestApproverUsername))
							err := ioutil.WriteFile(owners, data, util.DefaultWritePermissions)
							if err != nil {
								panic(err)
							}

							T.ExpectCommandExecution(workDir, time.Minute, 0, "git", "add", fileName)
						})

						ownersPR, err := T.GetPullRequestByNumber(provider, createdPR.Owner, createdPR.Repository, createdPR.PullRequestNumber)
						Expect(err).NotTo(HaveOccurred())
						Expect(ownersPR).ShouldNot(BeNil())

						By("merging the OWNERS PR")
						// GitLab seems to want us to sleep a bit after creation
						if provider.Kind() == "gitlab" {
							time.Sleep(30 * time.Second)
						}
						err = provider.MergePullRequest(ownersPR, "PR merge")
						Expect(err).ShouldNot(HaveOccurred())

						T.WaitForPullRequestToMerge(provider, ownersPR.Owner, ownersPR.Repo, *ownersPR.Number, ownersPR.URL)
					})

					prTitle := "My First PR commit"
					var pr *gits.GitPullRequest
					By("performing a pull request on the source and making sure it fails", func() {
						createdPR := T.CreatePullRequestWithLocalChange(prTitle, func(workDir string) {
							// overwrite the existing jenkins-x.yml with a failing one
							fileName := "jenkins-x.yml"
							jxYml := filepath.Join(workDir, fileName)

							data := []byte(brokenJenkinsXYml)
							err := ioutil.WriteFile(jxYml, data, util.DefaultWritePermissions)
							if err != nil {
								panic(err)
							}

							T.ExpectCommandExecution(workDir, time.Minute, 0, "git", "add", fileName)
						})

						pr, err = T.GetPullRequestByNumber(provider, createdPR.Owner, createdPR.Repository, createdPR.PullRequestNumber)
						Expect(err).NotTo(HaveOccurred())
						Expect(pr).ShouldNot(BeNil())

						By("verifying OWNERS link in APPROVALNOTIFIER comment is correct", func() {
							err = T.ExpectThatPullRequestHasCommentMatching(provider, createdPR.PullRequestNumber, createdPR.Owner, createdPR.Repository, func(comments []*scm.Comment) error {
								for _, c := range comments {
									if strings.Contains(c.Body, "[APPROVALNOTIFIER]") {
										ownerRegex := regexp.MustCompile(`(?m).*\[OWNERS]\((.*)\).*`)
										matches := ownerRegex.FindStringSubmatch(c.Body)
										if len(matches) == 0 {
											return backoff.Permanent(fmt.Errorf("could not find OWNERS link in:\n%s", c.Body))
										}
										expected := urlForProvider(provider.Kind(), provider.ServerURL(), createdPR.Owner, createdPR.Repository)
										if expected != matches[1] {
											return backoff.Permanent(fmt.Errorf("expected OWNERS URL %s, but got %s", expected, matches[1]))
										}
										return nil
									}
								}
								return fmt.Errorf("couldn't find comment containing APPROVALNOTIFIER")
							})
							Expect(err).NotTo(HaveOccurred())
						})
						By("waiting for build to fail", func() {
							T.WaitForPullRequestCommitStatus(provider, pr, []string{defaultContext}, "failure")
						})

						By("getting build log for a completed build", func() {
							// Verify that we can get the build log for a completed build.
							jobName := createdPR.Owner + "/" + createdPR.Repository + "/PR-" + strconv.Itoa(createdPR.PullRequestNumber)
							T.TailSpecificBuildLog(jobName, 1, helpers.TimeoutBuildCompletes)
						})
					})

					By("attempting to LGTM our own PR", func() {
						err = T.AttemptToLGTMOwnPullRequest(provider, pr)
						Expect(err).NotTo(HaveOccurred())
					})

					// TODO: Figure out if this something that we can actually fix for BitBucket Server or if we should just ignore it forever
					if provider.Kind() != gits.KindBitBucketServer {
						By("requesting and unrequesting a reviewer", func() {
							err = T.AddReviewerToPullRequestWithChatOpsCommand(provider, approverProvider, pr, helpers.PullRequestApproverUsername)
							Expect(err).NotTo(HaveOccurred())
						})
					}

					By("adding a hold label", func() {
						err = T.AddHoldLabelToPullRequestWithChatOpsCommand(provider, pr)
						Expect(err).NotTo(HaveOccurred())
					})

					// Adding WIP to a MR title is hijacked by GitLab and currently doesn't send a webhook event, so skip for now.
					if provider.Kind() != "gitlab" {
						By("adding a WIP label", func() {
							err = T.AddWIPLabelToPullRequestByUpdatingTitle(provider, pr)
							Expect(err).NotTo(HaveOccurred())
						})
					}

					By("approving pull request", func() {
						err = T.ApprovePullRequest(provider, approverProvider, pr)
						Expect(err).ShouldNot(HaveOccurred())
					})

					// '/retest' and '/test this' need to be done by a user other than the bot, as best as I can tell. (APB)

					By("retest failed context with it failing again", func() {
						err = approverProvider.AddPRComment(pr, "/retest")
						Expect(err).ShouldNot(HaveOccurred())

						// Wait until we see a pending or running status, meaning we've got a new build
						T.WaitForPullRequestCommitStatus(provider, pr, []string{defaultContext}, "pending", "running", "in-progress")

						// Wait until we see the build fail.
						T.WaitForPullRequestCommitStatus(provider, pr, []string{defaultContext}, "failure")
					})

					By("'/test this' with it failing again", func() {
						err = approverProvider.AddPRComment(pr, "/test this")
						Expect(err).ShouldNot(HaveOccurred())

						// Wait until we see a pending or running status, meaning we've got a new build
						T.WaitForPullRequestCommitStatus(provider, pr, []string{defaultContext}, "pending", "running", "in-progress")

						// Wait until we see the build fail.
						T.WaitForPullRequestCommitStatus(provider, pr, []string{defaultContext}, "failure")
					})

					// '/override' has to be done by a repo admin, so use the bot user.

					By("override failed context, see status as success, wait for it to merge", func() {
						err = provider.AddPRComment(pr, fmt.Sprintf("/override %s", defaultContext))
						Expect(err).ShouldNot(HaveOccurred())

						// Wait until we see a success status
						T.WaitForPullRequestCommitStatus(provider, pr, []string{defaultContext}, "success")

						T.WaitForPullRequestToMerge(provider, pr.Owner, pr.Repo, *pr.Number, pr.URL)
					})

					// TODO: Later: add multiple contexts, one more required, one more optional

					if provider.Kind() == "github" {
						By("creating an issue and assigning it to a valid user", func() {
							issue := &gits.GitIssue{
								Owner: T.GetGitOrganisation(),
								Repo:  T.GetApplicationName(),
								Title: "Test the /assign command",
								Body:  "This tests assigning a user using a ChatOps command",
							}
							err = T.CreateIssueAndAssignToUserWithChatOpsCommand(issue, provider)
							Expect(err).NotTo(HaveOccurred())
						})
					}

					if T.DeleteApplications() {
						args = []string{"delete", "application", "-b", T.ApplicationName}
						argsStr := strings.Join(args, " ")
						By(fmt.Sprintf("calling %s to delete the application", argsStr), func() {
							T.ExpectJxExecution(T.WorkDir, helpers.TimeoutSessionWait, 0, args...)
						})
					}

					if T.DeleteRepos() {
						args = []string{"delete", "repo", "-b", "--github", "-o", T.GetGitOrganisation(), "-n", T.ApplicationName}
						argsStr = strings.Join(args, " ")

						By(fmt.Sprintf("calling %s to delete the repository", os.Args), func() {
							T.ExpectJxExecution(T.WorkDir, helpers.TimeoutSessionWait, 0, args...)
						})
					}
				})
			})
		})
*/	})
}

func urlForProvider(providerType string, serverURL string, owner string, repo string) string {
	switch providerType {
	case "bitbucketserver":
		return fmt.Sprintf("%s/projects/%s/repos/%s/browse/OWNERS", serverURL, strings.ToUpper(owner), repo)
	case "gitlab":
		return fmt.Sprintf("%s/%s/%s/-/blob/master/OWNERS", serverURL, owner, repo)
	default:
		return fmt.Sprintf("%s/%s/%s/blob/master/OWNERS", serverURL, owner, repo)
	}
}

func generatePipelineRunSpec() *tektonv1beta1.PipelineRunSpec {
	return &tektonv1beta1.PipelineRunSpec{
		PipelineRef: &tektonv1beta1.PipelineRef{
			Name: "lh-test-pipeline",
		},
		Resources: []tektonv1beta1.PipelineResourceBinding{{
			Name: "pipeline-git",
			ResourceRef: &tektonv1beta1.PipelineResourceRef{
				Name: tekton.ProwImplicitGitResource,
			},
		}},
		ServiceAccountName: "default",
	}
}

type pipelineCRDInput struct {
	Namespace string
}

func applyPipelineAndTask() error {
	tmpDir, err := ioutil.TempDir("", "pipeline-and-task")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	pAndTFile := filepath.Join("test_data", "tekton", "pipelineAndTask.tmpl.yaml")
	rawPAndT, err := ioutil.ReadFile(pAndTFile)
	if err != nil {
		return errors.Wrapf(err, "reading pipeline/task template %s", pAndTFile)
	}
	pAndTTmpl, err := template.New("pAndT").Parse(string(rawPAndT))
	if err != nil {
		return errors.Wrapf(err, "parsing pipeline/task template from %s", pAndTFile)
	}

	input := pipelineCRDInput{
		Namespace: ns,
	}

	var pAndTBuf bytes.Buffer

	err = pAndTTmpl.Execute(&pAndTBuf, &input)
	if err != nil {
		return errors.Wrapf(err, "applying pipeline/task template from %s", pAndTFile)
	}

	outputFile := filepath.Join(tmpDir, "pipelineAndTask.yaml")
	err = ioutil.WriteFile(outputFile, pAndTBuf.Bytes(), 0644)
	if err != nil {
		return errors.Wrapf(err, "writing to output file %s", outputFile)
	}

	ExpectCommandExecution(tmpDir, 1, 0, "kubectl", "apply", "-f", "pipelineAndTask.yaml")
	return nil
}
