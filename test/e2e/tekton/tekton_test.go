package tekton

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
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
	"github.com/jenkins-x/lighthouse/test/e2e"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/pod"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
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
	e2e.RunWithReporters(t, "TektonIntegrationTest")
}

var _ = ChatOpsTests()

func ChatOpsTests() bool {
	return Describe("Lighthouse Tekton support", func() {
		BeforeEach(func() {
			var err error
			By("creating HMAC token")
			hmacToken, err = e2e.CreateHMACToken()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(hmacToken).ShouldNot(BeEmpty())

			By("creating primary SCM client")
			scmClient, spc, gitServerURL, err = e2e.CreateSCMClient(e2e.GetBotName, e2e.GetPrimarySCMToken)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(scmClient).ShouldNot(BeNil())
			Expect(spc).ShouldNot(BeNil())
			Expect(gitServerURL).ShouldNot(BeEmpty())

			By("creating approver SCM client")
			approverClient, approverSpc, _, err = e2e.CreateSCMClient(e2e.GetApproverName, e2e.GetApproverSCMToken)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(approverClient).ShouldNot(BeNil())
			Expect(approverSpc).ShouldNot(BeNil())

			By("creating git client")
			gitClient, err = e2e.CreateGitClient(gitServerURL, e2e.GetBotName, e2e.GetPrimarySCMToken)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(gitClient).ShouldNot(BeNil())

			By("creating repository")
			repo, localClone, err = e2e.CreateBaseRepository(e2e.GetBotName(), e2e.GetApproverName(), scmClient, gitClient)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(repo).ShouldNot(BeNil())
			Expect(localClone).ShouldNot(BeNil())
			repoFullName = fmt.Sprintf("%s/%s", repo.Namespace, repo.Name)

			By(fmt.Sprintf("adding %s to new repository", e2e.GetApproverName()))
			err = e2e.AddCollaborator(e2e.GetApproverName(), repo, scmClient, approverClient)
			Expect(err).ShouldNot(HaveOccurred())

			By("adding the Pipeline and Task definitions to the cluster")
			err = applyPipelineAndTask()
			Expect(err).ShouldNot(HaveOccurred())
			e2e.ExpectCommandExecution(localClone.Dir, 1, 0, "kubectl", "apply", "-f",
				"https://raw.githubusercontent.com/tektoncd/catalog/master/task/git-batch-merge/0.2/git-batch-merge.yaml")

			By(fmt.Sprintf("creating and populating Lighthouse config for %s", repo.Clone))
			cfg, pluginCfg, err := e2e.ProcessConfigAndPlugins(repo.Namespace, repo.Name, ns, job.TektonPipelineAgent, "")
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

			err = e2e.ApplyConfigAndPluginsConfigMaps(cfg, pluginCfg)
			Expect(err).ShouldNot(HaveOccurred())

			By(fmt.Sprintf("setting up webhooks for %s", repo.Clone))
			err = e2e.CreateWebHook(scmClient, repo, hmacToken)
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
				err = os.WriteFile(newFile, []byte("Hello world"), 0600)
				e2e.ExpectCommandExecution(localClone.Dir, 1, 0, "git", "add", newFile)

				changedScriptFile := filepath.Join("test_data", "passingRepoScript.sh")
				changedScript, err := os.ReadFile(changedScriptFile) /* #nosec */
				Expect(err).ShouldNot(HaveOccurred())

				scriptOutputFile := filepath.Join(localClone.Dir, "script.sh")
				err = os.WriteFile(scriptOutputFile, changedScript, 0600)
				Expect(err).ShouldNot(HaveOccurred())

				e2e.ExpectCommandExecution(localClone.Dir, 1, 0, "git", "commit", "-a", "-m", "Adding for test PR")

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
				err = e2e.ExpectThatPullRequestHasCommentMatching(scmClient, pr, func(comments []*scm.Comment) error {
					for _, c := range comments {
						if strings.Contains(c.Body, "[APPROVALNOTIFIER]") {
							ownerRegex := regexp.MustCompile(`(?m).*\[OWNERS]\((.*)\).*`)
							matches := ownerRegex.FindStringSubmatch(c.Body)
							if len(matches) == 0 {
								return backoff.Permanent(fmt.Errorf("could not find OWNERS link in:\n%s", c.Body))
							}
							expected := e2e.URLForFile(e2e.GitKind(), gitServerURL, repo.Namespace, repo.Name, "OWNERS")
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
				e2e.WaitForPullRequestCommitStatus(scmClient, pr, []string{defaultContext}, "success")
			})

			By("changing the PR to fail", func() {
				failScriptFile := filepath.Join("test_data", "failingRepoScript.sh")
				failScript, err := os.ReadFile(failScriptFile) /* #nosec */
				Expect(err).ShouldNot(HaveOccurred())

				scriptOutputFile := filepath.Join(localClone.Dir, "script.sh")
				err = os.WriteFile(scriptOutputFile, failScript, 0600)
				Expect(err).ShouldNot(HaveOccurred())

				e2e.ExpectCommandExecution(localClone.Dir, 1, 0, "git", "commit", "-a", "-m", "Updating to fail")

				err = localClone.Push(repo.Name, prBranch)
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("waiting for the PR build to fail", func() {
				e2e.WaitForPullRequestCommitStatus(scmClient, pr, []string{defaultContext}, "failure")
			})

			By("attempting to LGTM our own PR", func() {
				err = e2e.AttemptToLGTMOwnPullRequest(scmClient, pr)
				Expect(err).ShouldNot(HaveOccurred())
			})

			if e2e.GitKind() != "stash" {
				By("requesting and unrequesting a reviewer", func() {
					err = e2e.AddReviewerToPullRequestWithChatOpsCommand(spc, pr, e2e.GetApproverName())
					Expect(err).NotTo(HaveOccurred())
				})
			}

			By("adding a hold label", func() {
				err = e2e.AddHoldLabelToPullRequestWithChatOpsCommand(spc, pr)
				Expect(err).NotTo(HaveOccurred())
			})

			// Adding WIP to a MR title is hijacked by GitLab and currently doesn't send a webhook event, so skip for now.
			if e2e.GitKind() != "gitlab" {
				By("adding a WIP label", func() {
					err = e2e.AddWIPLabelToPullRequestByUpdatingTitle(spc, scmClient, pr)
					Expect(err).NotTo(HaveOccurred())
				})
			}

			By("approving pull request", func() {
				err = e2e.ApprovePullRequest(spc, approverSpc, pr)
				Expect(err).ShouldNot(HaveOccurred())
			})

			// '/retest' and '/test this' need to be done by a user other than the bot, as best as I can tell. (APB)

			By("retest failed context with it failing again", func() {
				err = approverSpc.CreateComment(repo.Namespace, repo.Name, pr.Number, true, "/retest")
				Expect(err).ShouldNot(HaveOccurred())

				// Wait until we see a pending or running status, meaning we've got a new build
				e2e.WaitForPullRequestCommitStatus(scmClient, pr, []string{defaultContext}, "pending", "running", "in-progress")

				// Wait until we see the build fail.
				e2e.WaitForPullRequestCommitStatus(scmClient, pr, []string{defaultContext}, "failure")
			})

			By("'/test this' with it failing again", func() {
				err = approverSpc.CreateComment(repo.Namespace, repo.Name, pr.Number, true, "/test this")
				Expect(err).ShouldNot(HaveOccurred())

				// Wait until we see a pending or running status, meaning we've got a new build
				e2e.WaitForPullRequestCommitStatus(scmClient, pr, []string{defaultContext}, "pending", "running", "in-progress")

				// Wait until we see the build fail.
				e2e.WaitForPullRequestCommitStatus(scmClient, pr, []string{defaultContext}, "failure")
			})

			// '/override' has to be done by a repo admin, so use the bot user.

			By("override failed context, see status as success, wait for it to merge", func() {
				err = spc.CreateComment(repo.Namespace, repo.Name, pr.Number, true, fmt.Sprintf("/override %s", defaultContext))
				Expect(err).ShouldNot(HaveOccurred())

				// Wait until we see a success status
				e2e.WaitForPullRequestCommitStatus(scmClient, pr, []string{defaultContext}, "success")

				e2e.WaitForPullRequestToMerge(spc, pr)
			})
		})
	})
}

func generatePipelineRunSpec() *pipelinev1.PipelineRunSpec {
	return &pipelinev1.PipelineRunSpec{
		PipelineRef: &pipelinev1.PipelineRef{
			Name: "lh-test-pipeline",
		},
		TaskRunTemplate: pipelinev1.PipelineTaskRunTemplate{
			ServiceAccountName: "tekton-bot",
			PodTemplate:        &pod.PodTemplate{},
		},

		Workspaces: []pipelinev1.WorkspaceBinding{{
			Name: "shared-data",
			VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
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
	tmpDir, err := os.MkdirTemp("", "pipeline-and-task")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir) //nolint: errcheck

	pAndTFile := filepath.Join("test_data", "tekton", "pipelineAndTask.tmpl.yaml")
	rawPAndT, err := os.ReadFile(pAndTFile)
	if err != nil {
		return errors.Wrapf(err, "reading pipeline/task template %s", pAndTFile)
	}
	pAndTTmpl, err := template.New("pAndT").Parse(string(rawPAndT))
	if err != nil {
		return errors.Wrapf(err, "parsing pipeline/task template from %s", pAndTFile)
	}

	rawToken, err := e2e.GetPrimarySCMToken()
	if err != nil {
		return errors.Wrapf(err, "getting git token for user %s", e2e.GetBotName())
	}

	input := pipelineCRDInput{
		Namespace:  ns,
		BaseGitURL: gitServerURL,
		GitUser:    base64.StdEncoding.EncodeToString([]byte(e2e.GetBotName())),
		GitToken:   base64.StdEncoding.EncodeToString([]byte(rawToken)),
	}

	var pAndTBuf bytes.Buffer

	err = pAndTTmpl.Execute(&pAndTBuf, &input)
	if err != nil {
		return errors.Wrapf(err, "applying pipeline/task template from %s", pAndTFile)
	}

	outputFile := filepath.Join(tmpDir, "pipelineAndTask.yaml")
	err = os.WriteFile(outputFile, pAndTBuf.Bytes(), 0644)
	if err != nil {
		return errors.Wrapf(err, "writing to output file %s", outputFile)
	}

	e2e.ExpectCommandExecution(tmpDir, 1, 0, "kubectl", "apply", "-f", "pipelineAndTask.yaml")
	return nil
}
