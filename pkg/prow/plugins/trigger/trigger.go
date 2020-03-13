/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package trigger

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/errorutil"
	"github.com/jenkins-x/lighthouse/pkg/prow/gitprovider"
	"github.com/jenkins-x/lighthouse/pkg/prow/pjutil"
	"github.com/jenkins-x/lighthouse/pkg/prow/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// PluginName is the name of the trigger plugin
	PluginName = "trigger"
)

func init() {
	plugins.RegisterGenericCommentHandler(PluginName, handleGenericCommentEvent, helpProvider)
	plugins.RegisterPullRequestHandler(PluginName, handlePullRequest, helpProvider)
	plugins.RegisterPushEventHandler(PluginName, handlePush, helpProvider)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	configInfo := map[string]string{}
	for _, orgRepo := range enabledRepos {
		parts := strings.Split(orgRepo, "/")
		var trigger *plugins.Trigger
		switch len(parts) {
		case 1:
			trigger = config.TriggerFor(orgRepo, "")
		case 2:
			trigger = config.TriggerFor(parts[0], parts[1])
		default:
			return nil, fmt.Errorf("invalid repo in enabledRepos: %q", orgRepo)
		}
		org := parts[0]
		if trigger.TrustedOrg != "" {
			org = trigger.TrustedOrg
		}
		configInfo[orgRepo] = fmt.Sprintf("The trusted GitHub organization for this repository is %q.", org)
	}
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The trigger plugin starts tests in reaction to commands and pull request events. It is responsible for ensuring that test jobs are only run on trusted PRs. A PR is considered trusted if the author is a member of the 'trusted organization' for the repository or if such a member has left an '/ok-to-test' command on the PR.
<br>Trigger starts jobs automatically when a new trusted PR is created or when an untrusted PR becomes trusted, but it can also be used to start jobs manually via the '/test' command.
<br>The '/retest' command can be used to rerun jobs that have reported failure.`,
		Config: configInfo,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/ok-to-test",
		Description: "Marks a PR as 'trusted' and starts tests.",
		Featured:    false,
		WhoCanUse:   "Members of the trusted organization for the repo.",
		Examples:    []string{"/ok-to-test"},
	})
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/test (<job name>|all)",
		Description: "Manually starts a/all test job(s).",
		Featured:    true,
		WhoCanUse:   "Anyone can trigger this command on a trusted PR.",
		Examples:    []string{"/test all", "/test pull-bazel-test"},
	})
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/retest",
		Description: "Rerun test jobs that have failed.",
		Featured:    true,
		WhoCanUse:   "Anyone can trigger this command on a trusted PR.",
		Examples:    []string{"/retest"},
	})
	return pluginHelp, nil
}

type scmProviderClient interface {
	AddLabel(org, repo string, number int, label string, pr bool) error
	BotName() (string, error)
	IsCollaborator(org, repo, user string) (bool, error)
	IsMember(org, user string) (bool, error)
	GetPullRequest(org, repo string, number int) (*scm.PullRequest, error)
	GetRef(org, repo, ref string) (string, error)
	CreateComment(owner, repo string, number int, pr bool, comment string) error
	ListIssueComments(owner, repo string, issue int) ([]*scm.Comment, error)
	CreateStatus(org, repo, ref string, s *scm.StatusInput) (*scm.Status, error)
	GetCombinedStatus(org, repo, ref string) (*scm.CombinedStatus, error)
	GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error)
	RemoveLabel(org, repo string, number int, label string, pr bool) error
	DeleteStaleComments(org, repo string, number int, comments []*scm.Comment, pr bool, isStale func(*scm.Comment) bool) error
	GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error)
}

type launcher interface {
	Launch(*v1alpha1.LighthouseJob, metapipeline.Client, scm.Repository) (*v1alpha1.LighthouseJob, error)
}

// Client holds the necessary structures to work with prow via logging, github, kubernetes and its configuration.
//
// TODO(fejta): consider exporting an interface rather than a struct
type Client struct {
	SCMProviderClient  scmProviderClient
	LauncherClient     launcher
	Config             *config.Config
	Logger             *logrus.Entry
	MetapipelineClient metapipeline.Client
}

type trustedUserClient interface {
	IsCollaborator(org, repo, user string) (bool, error)
	IsMember(org, user string) (bool, error)
	BotName() (string, error)
}

func getClient(pc plugins.Agent) Client {
	return Client{
		SCMProviderClient:  pc.SCMProviderClient,
		Config:             pc.Config,
		LauncherClient:     pc.LauncherClient,
		Logger:             pc.Logger,
		MetapipelineClient: pc.MetapipelineClient,
	}
}

func handlePullRequest(pc plugins.Agent, pr scm.PullRequestHook) error {
	org, repo, _ := orgRepoAuthor(pr.PullRequest)
	return handlePR(getClient(pc), pc.PluginConfig.TriggerFor(org, repo), pr)
}

func handleGenericCommentEvent(pc plugins.Agent, gc gitprovider.GenericCommentEvent) error {
	return handleGenericComment(getClient(pc), pc.PluginConfig.TriggerFor(gc.Repo.Namespace, gc.Repo.Name), gc)
}

func handlePush(pc plugins.Agent, pe scm.PushHook) error {
	return handlePE(getClient(pc), pe)
}

// TrustedUser returns true if user is trusted in repo.
//
// Trusted users are either repo collaborators, org members or trusted org members.
// Whether repo collaborators and/or a second org is trusted is configured by trigger.
func TrustedUser(spc trustedUserClient, trigger *plugins.Trigger, user, org, repo string) (bool, error) {
	botUser, err := spc.BotName()
	if err == nil && user == botUser {
		return true, nil
	}
	// First check if user is a collaborator, assuming this is allowed
	if !trigger.OnlyOrgMembers {
		if ok, err := spc.IsCollaborator(org, repo, user); err != nil {
			return false, fmt.Errorf("error in IsCollaborator: %v", err)
		} else if ok {
			return true, nil
		}
	}

	// TODO(fejta): consider dropping support for org checks in the future.

	// Next see if the user is an org member
	if member, err := spc.IsMember(org, user); err != nil {
		return false, fmt.Errorf("error in IsMember(%s): %v", org, err)
	} else if member {
		return true, nil
	}

	// Determine if there is a second org to check
	if trigger.TrustedOrg == "" || trigger.TrustedOrg == org {
		return false, nil // No trusted org and/or it is the same
	}

	// Check the second trusted org.
	member, err := spc.IsMember(trigger.TrustedOrg, user)
	if err != nil {
		return false, fmt.Errorf("error in IsMember(%s): %v", trigger.TrustedOrg, err)
	}
	return member, nil
}

func skippedStatusFor(context string) *scm.StatusInput {
	return &scm.StatusInput{
		State: scm.StateSuccess,
		Label: context,
		Desc:  "Skipped.",
	}
}

func failedStatusForMetapipelineCreation(context string, err error) *scm.StatusInput {
	return &scm.StatusInput{
		State: scm.StateError,
		Label: context,
		Desc:  fmt.Sprintf("Error creating metapipeline: %s", err),
	}
}

// RunAndSkipJobs executes the config.Presubmits that are requested and posts skipped statuses
// for the reporting jobs that are skipped
func RunAndSkipJobs(c Client, pr *scm.PullRequest, requestedJobs []config.Presubmit, skippedJobs []config.Presubmit, eventGUID string, elideSkippedContexts bool) error {
	if err := validateContextOverlap(requestedJobs, skippedJobs); err != nil {
		c.Logger.WithError(err).Warn("Could not run or skip requested jobs, overlapping contexts.")
		return err
	}
	runErr := runRequested(c, pr, requestedJobs, eventGUID)
	var skipErr error
	if !elideSkippedContexts {
		skipErr = skipRequested(c, pr, skippedJobs)
	}

	return errorutil.NewAggregate(runErr, skipErr)
}

// validateContextOverlap ensures that there will be no overlap in contexts between a set of jobs running and a set to skip
func validateContextOverlap(toRun, toSkip []config.Presubmit) error {
	requestedContexts := sets.NewString()
	for _, job := range toRun {
		requestedContexts.Insert(job.Context)
	}
	skippedContexts := sets.NewString()
	for _, job := range toSkip {
		skippedContexts.Insert(job.Context)
	}
	if overlap := requestedContexts.Intersection(skippedContexts).List(); len(overlap) > 0 {
		return fmt.Errorf("the following contexts are both triggered and skipped: %s", strings.Join(overlap, ", "))
	}

	return nil
}

// runRequested executes the config.Presubmits that are requested
func runRequested(c Client, pr *scm.PullRequest, requestedJobs []config.Presubmit, eventGUID string) error {
	baseSHA, err := c.SCMProviderClient.GetRef(pr.Base.Repo.Namespace, pr.Base.Repo.Name, "heads/"+pr.Base.Ref)
	if err != nil {
		return err
	}

	var errors []error
	for _, job := range requestedJobs {
		c.Logger.Infof("Starting %s build.", job.Name)
		pj := pjutil.NewPresubmit(pr, baseSHA, job, eventGUID)
		c.Logger.WithFields(pjutil.LighthouseJobFields(&pj)).Info("Creating a new LighthouseJob.")
		if _, err := c.LauncherClient.Launch(&pj, c.MetapipelineClient, pr.Repository()); err != nil {
			c.Logger.WithError(err).Error("Failed to create LighthouseJob.")
			errors = append(errors, err)
			if _, statusErr := c.SCMProviderClient.CreateStatus(pr.Base.Repo.Namespace, pr.Base.Repo.Name, pr.Head.Ref, failedStatusForMetapipelineCreation(job.Context, err)); statusErr != nil {
				errors = append(errors, statusErr)
			}
		}
	}
	return errorutil.NewAggregate(errors...)
}

// skipRequested posts skipped statuses for the config.Presubmits that are requested
func skipRequested(c Client, pr *scm.PullRequest, skippedJobs []config.Presubmit) error {
	var errors []error
	for _, job := range skippedJobs {
		if job.SkipReport {
			continue
		}
		c.Logger.Infof("Skipping %s build.", job.Name)
		if _, err := c.SCMProviderClient.CreateStatus(pr.Base.Repo.Namespace, pr.Base.Repo.Name, pr.Head.Ref, skippedStatusFor(job.Context)); err != nil {
			errors = append(errors, err)
		}
	}
	return errorutil.NewAggregate(errors...)
}
