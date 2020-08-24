package e2e

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"text/template"

	"github.com/cenkalti/backoff"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	ns             = "lh-test"
	prBranch       = "for-pr"
	defaultContext = "pr-build"
)

var (
	hmacToken      string
	gitClient      git.Client
	scmClient      *scm.Client
	spc            scmprovider.SCMClient
	approverClient *scm.Client
	approverSpc    scmprovider.SCMClient
	gitServerURL   string
	repo           *scm.Repository
	repoFullName   string
	localClone     *git.Repo
)

var _ = AfterSuite(func() {
	err := gitClient.Clean()
	if err != nil {
		logrus.WithError(err).Fatal("Error cleaning the git client.")
	}

})

func TestTekton(t *testing.T) {
	RunWithReporters(t, "Tekton integration")
}

var _ = ChatOpsTests()

func ChatOpsTests() bool {
	return Describe("Lighthouse Tekton support", func() {
		BeforeEach(func() {
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
			approverClient, approverSpc, _, err = CreateSCMClient(GetApproverName, GetApproverSCMToken)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(approverClient).ShouldNot(BeNil())
			Expect(approverSpc).ShouldNot(BeNil())

			By("creating git client")
			gitClient, err = CreateGitClient(gitServerURL, GetBotName, GetPrimarySCMToken)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gitClient).ShouldNot(BeNil())

			By("creating repository")
			repo, localClone, err = CreateBaseRepository(GetBotName(), GetApproverName(), scmClient, gitClient)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(repo).ShouldNot(BeNil())
			Expect(localClone).ShouldNot(BeNil())
			repoFullName = fmt.Sprintf("%s/%s", repo.Namespace, repo.Name)

			By(fmt.Sprintf("adding %s to new repository", GetApproverName()))
			err = AddCollaborator(GetApproverName(), repo, scmClient, approverClient)
			Expect(err).ShouldNot(HaveOccurred())

			By("adding the Pipeline and Task definitions to the cluster")
			err = applyPipelineAndTask()
			Expect(err).ShouldNot(HaveOccurred())
			ExpectCommandExecution(localClone.Dir, 1, 0, "kubectl", "apply", "-f",
				"https://raw.githubusercontent.com/tektoncd/catalog/master/task/git-batch-merge/0.2/git-batch-merge.yaml")

			By(fmt.Sprintf("creating and populating Lighthouse config for %s", repo.Clone))
			cfg, pluginCfg, err := ProcessConfigAndPlugins(repo.Namespace, repo.Name, ns, job.TektonPipelineAgent)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(cfg).ShouldNot(BeNil())
			Expect(pluginCfg).ShouldNot(BeNil())

			cfg.Presubmits[repoFullName][0].PipelineRunSpec = generatePipelineRunSpec()
			cfg.Presubmits[repoFullName][0].PipelineRunParams = []job.PipelineRunParam{
				{
					Name:          "batch-refs",
					ValueTemplate: "{{ range $i, $v := .Refs.Pulls }}{{if $i}} {{end}}{{ $v.Ref }}{{ end }}",
				},
				{
					Name:          "branch-name",
					ValueTemplate: "{{ .Refs.BaseRef }}",
				},
				{
					Name:          "repo-url",
					ValueTemplate: "{{ .Refs.CloneURI }}",
				},
			}

			err = ApplyConfigAndPluginsConfigMaps(cfg, pluginCfg)
			Expect(err).ShouldNot(HaveOccurred())

			By(fmt.Sprintf("setting up webhooks for %s", repo.Clone))
			err = CreateWebHook(scmClient, repo, hmacToken)
			Expect(err).ShouldNot(HaveOccurred())
		})
		var (
			err error
			pr  *scm.PullRequest
		)

		It("verifies Lighthouse triggers and reports Tekton pipeline runs properly", func() {
			By("cloning, creating the new branch, and pushing it", func() {
				err = localClone.CheckoutNewBranch(prBranch)
				Expect(err).ShouldNot(HaveOccurred())

				newFile := filepath.Join(localClone.Dir, "README")
				err = ioutil.WriteFile(newFile, []byte("Hello world"), 0600)
				ExpectCommandExecution(localClone.Dir, 1, 0, "git", "add", newFile)

				changedScriptFile := filepath.Join("test_data", "passingRepoScript.sh")
				changedScript, err := ioutil.ReadFile(changedScriptFile) /* #nosec */
				Expect(err).ShouldNot(HaveOccurred())

				scriptOutputFile := filepath.Join(localClone.Dir, "script.sh")
				err = ioutil.WriteFile(scriptOutputFile, changedScript, 0600)
				Expect(err).ShouldNot(HaveOccurred())

				ExpectCommandExecution(localClone.Dir, 1, 0, "git", "commit", "-a", "-m", "Adding for test PR")

				err = localClone.Push(repo.Name, prBranch)
				Expect(err).ShouldNot(HaveOccurred())
			})
			By("creating a pull request", func() {
				prInput := &scm.PullRequestInput{
					Title: "Lighthouse Test PR",
					Head:  prBranch,
					Base:  "master",
					Body:  "Test PR for Lighthouse",
				}
				pr, _, err = scmClient.PullRequests.Create(context.Background(), repoFullName, prInput)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(pr).ShouldNot(BeNil())
			})
			By("verifying OWNERS link in APPROVALNOTIFIER comment is correct", func() {
				err = ExpectThatPullRequestHasCommentMatching(spc, pr, func(comments []*scm.Comment) error {
					for _, c := range comments {
						if strings.Contains(c.Body, "[APPROVALNOTIFIER]") {
							ownerRegex := regexp.MustCompile(`(?m).*\[OWNERS]\((.*)\).*`)
							matches := ownerRegex.FindStringSubmatch(c.Body)
							if len(matches) == 0 {
								return backoff.Permanent(fmt.Errorf("could not find OWNERS link in:\n%s", c.Body))
							}
							expected := urlForProvider(GitKind(), gitServerURL, repo.Namespace, repo.Name)
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
			By("waiting for build to succeed", func() {
				WaitForPullRequestCommitStatus(spc, pr, []string{defaultContext}, "success")
			})

			By("changing the PR to fail", func() {
				failScriptFile := filepath.Join("test_data", "failingRepoScript.sh")
				failScript, err := ioutil.ReadFile(failScriptFile) /* #nosec */
				Expect(err).ShouldNot(HaveOccurred())

				scriptOutputFile := filepath.Join(localClone.Dir, "script.sh")
				err = ioutil.WriteFile(scriptOutputFile, failScript, 0600)
				Expect(err).ShouldNot(HaveOccurred())

				ExpectCommandExecution(localClone.Dir, 1, 0, "git", "commit", "-a", "-m", "Updating to fail")

				err = localClone.Push(repo.Name, prBranch)
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("waiting for the PR build to fail", func() {
				WaitForPullRequestCommitStatus(spc, pr, []string{defaultContext}, "failure")
			})

			By("attempting to LGTM our own PR", func() {
				err = AttemptToLGTMOwnPullRequest(spc, pr)
				Expect(err).ShouldNot(HaveOccurred())
			})

			if GitKind() != "stash" {
				By("requesting and unrequesting a reviewer", func() {
					err = AddReviewerToPullRequestWithChatOpsCommand(spc, pr, GetApproverName())
					Expect(err).NotTo(HaveOccurred())
				})
			}

			By("adding a hold label", func() {
				err = AddHoldLabelToPullRequestWithChatOpsCommand(spc, pr)
				Expect(err).NotTo(HaveOccurred())
			})

			// Adding WIP to a MR title is hijacked by GitLab and currently doesn't send a webhook event, so skip for now.
			if GitKind() != "gitlab" {
				By("adding a WIP label", func() {
					err = AddWIPLabelToPullRequestByUpdatingTitle(spc, scmClient, pr)
					Expect(err).NotTo(HaveOccurred())
				})
			}

			By("approving pull request", func() {
				err = ApprovePullRequest(spc, approverSpc, pr)
				Expect(err).ShouldNot(HaveOccurred())
			})

			// '/retest' and '/test this' need to be done by a user other than the bot, as best as I can tell. (APB)

			By("retest failed context with it failing again", func() {
				err = approverSpc.CreateComment(repo.Namespace, repo.Name, pr.Number, true, "/retest")
				Expect(err).ShouldNot(HaveOccurred())

				// Wait until we see a pending or running status, meaning we've got a new build
				WaitForPullRequestCommitStatus(spc, pr, []string{defaultContext}, "pending", "running", "in-progress")

				// Wait until we see the build fail.
				WaitForPullRequestCommitStatus(spc, pr, []string{defaultContext}, "failure")
			})

			By("'/test this' with it failing again", func() {
				err = approverSpc.CreateComment(repo.Namespace, repo.Name, pr.Number, true, "/test this")
				Expect(err).ShouldNot(HaveOccurred())

				// Wait until we see a pending or running status, meaning we've got a new build
				WaitForPullRequestCommitStatus(spc, pr, []string{defaultContext}, "pending", "running", "in-progress")

				// Wait until we see the build fail.
				WaitForPullRequestCommitStatus(spc, pr, []string{defaultContext}, "failure")
			})

			// '/override' has to be done by a repo admin, so use the bot user.

			By("override failed context, see status as success, wait for it to merge", func() {
				err = spc.CreateComment(repo.Namespace, repo.Name, pr.Number, true, fmt.Sprintf("/override %s", defaultContext))
				Expect(err).ShouldNot(HaveOccurred())

				// Wait until we see a success status
				WaitForPullRequestCommitStatus(spc, pr, []string{defaultContext}, "success")

				WaitForPullRequestToMerge(spc, pr)
			})
		})
	})
}

func urlForProvider(providerType string, serverURL string, owner string, repo string) string {
	switch providerType {
	case "stash":
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
		ServiceAccountName: "tekton-bot",
		Workspaces: []tektonv1beta1.WorkspaceBinding{{
			Name: "shared-data",
			VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
					VolumeName:       "",
					StorageClassName: nil,
					VolumeMode:       nil,
					DataSource:       nil,
				},
			},
		}},
	}
}

type pipelineCRDInput struct {
	Namespace  string
	BaseGitURL string
	GitUser    string
	GitToken   string
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

	rawToken, err := GetPrimarySCMToken()
	if err != nil {
		return errors.Wrapf(err, "getting git token for user %s", GetBotName())
	}

	input := pipelineCRDInput{
		Namespace:  ns,
		BaseGitURL: gitServerURL,
		GitUser:    base64.StdEncoding.EncodeToString([]byte(GetBotName())),
		GitToken:   base64.StdEncoding.EncodeToString([]byte(rawToken)),
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
