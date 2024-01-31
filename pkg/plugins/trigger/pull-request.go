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
	"net/url"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/errorutil"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func handlePR(c Client, trigger *plugins.Trigger, pr scm.PullRequestHook) error {
	if len(c.Config.GetPresubmits(pr.PullRequest.Base.Repo)) == 0 {
		return nil
	}

	org, repo, a := orgRepoAuthor(pr.PullRequest)
	author := string(a)
	num := pr.PullRequest.Number
	switch pr.Action {
	case scm.ActionOpen:
		// When a PR is opened, if the author is in the org then build it.
		// Otherwise, ask for "/ok-to-test". There's no need to look for previous
		// "/ok-to-test" comments since the PR was just opened!
		member, err := TrustedUser(c.SCMProviderClient, trigger, author, org, repo)
		if err != nil {
			return fmt.Errorf("could not check membership: %s", err)
		}
		if !member {
			c.Logger.Infof("Author is not a member, Welcome message to PR author %q.", author)
			if err = welcomeMsg(c.SCMProviderClient, trigger, pr.PullRequest); err != nil {
				return fmt.Errorf("could not welcome non-org member %q: %v", author, err)
			}
			return nil
		}

		if err = infoMsg(c, pr.PullRequest); err != nil {
			return err
		}
		return buildAllIfTrustedOrDraft(c, trigger, pr)
	case scm.ActionEdited, scm.ActionUpdate:
		// if someone changes the base of their PR, we will get this
		// event and the changes field will list that the base SHA and
		// ref changes so we can detect such a case and retrigger tests
		changes := pr.Changes
		if changes.Base.Ref.From != "" || changes.Base.Sha.From != "" {
			// the base of the PR changed and we need to re-test it
			return buildAllIfTrustedOrDraft(c, trigger, pr)
		}
	case scm.ActionReopen, scm.ActionSync:
		return buildAllIfTrustedOrDraft(c, trigger, pr)
	case scm.ActionReadyForReview, scm.ActionConvertedToDraft:
		if trigger.SkipDraftPR {
			return buildAllIfTrustedOrDraft(c, trigger, pr)
		}
	case scm.ActionLabel:
		// When a PR is LGTMd, if it is untrusted then build it once.
		if pr.Label.Name == labels.LGTM {
			_, toTrigger, err := TrustedOrDraftPullRequest(c.SCMProviderClient, trigger, author, org, repo, num, pr.PullRequest.Draft, nil)
			if err != nil {
				return fmt.Errorf("could not validate PR: %s", err)
			} else if !toTrigger {
				c.Logger.Info("Starting all jobs for untrusted or draft PR with LGTM.")
				return buildAll(c, &pr.PullRequest, pr.GUID, trigger.ElideSkippedContexts)
			}
		}
	default:
		c.Logger.Warnf("unknown PR Action %d of %s", int(pr.Action), pr.Action.String())
	}
	return nil
}

type login string

func orgRepoAuthor(pr scm.PullRequest) (string, string, login) {
	org := pr.Base.Repo.Namespace
	repo := pr.Base.Repo.Name
	author := pr.Author.Login
	return org, repo, login(author)
}

func buildAllIfTrustedOrDraft(c Client, trigger *plugins.Trigger, pr scm.PullRequestHook) error {
	// When a PR is updated, check if the user is trusted or if the PR is a draft.
	org, repo, a := orgRepoAuthor(pr.PullRequest)
	author := string(a)
	num := pr.PullRequest.Number
	l, toTrigger, err := TrustedOrDraftPullRequest(c.SCMProviderClient, trigger, author, org, repo, num, pr.PullRequest.Draft, nil)
	if err != nil {
		return fmt.Errorf("could not validate PR: %s", err)
	}

	hasOkToTestLabel := scmprovider.HasLabel(labels.OkToTest, l)

	if toTrigger {
		// Eventually remove needs-ok-to-test
		// Will not work for org members since labels are not fetched in this case
		if scmprovider.HasLabel(labels.NeedsOkToTest, l) {
			if err := c.SCMProviderClient.RemoveLabel(org, repo, num, labels.NeedsOkToTest, true); err != nil {
				return err
			}
		}

		// We want to avoid launching exactly the same pipelines that were already launched just before (Draft or not)
		// So we skip pipelines if trigger.SkipDraftPR, ActionReadyForReview or ActionConvertedToDraft and with ok-to-test label
		if (pr.Action == scm.ActionReadyForReview || pr.Action == scm.ActionConvertedToDraft) && trigger.SkipDraftPR && hasOkToTestLabel {
			return nil
		}

		c.Logger.Info("Starting all jobs for new/updated PR.")
		return buildAll(c, &pr.PullRequest, pr.GUID, trigger.ElideSkippedContexts)
	}

	if trigger.SkipDraftPR && pr.PullRequest.Draft {
		// Welcome Message for Draft PR
		if err = welcomeMsgForDraftPR(c.SCMProviderClient, trigger, pr.PullRequest); err != nil {
			return fmt.Errorf("could not welcome for draft pr %q: %v", author, err)
		}
		// Skip pipelines trigger if SkipDraftPR enabled and PR is a draft without OkToTest label
		if !hasOkToTestLabel {
			c.Logger.Infof("Skipping build for draft PR %d unless converted to `ready for review` or `/ok-to-test` comment is added.", num)
		}
	}

	return nil
}

func infoMsg(c Client, pr scm.PullRequest) error {
	if isSyntaxDeprecated := isPipelinesSyntaxDeprecated(c.Config, pr.Repository()); !isSyntaxDeprecated {
		return nil
	}

	org, repo, _ := orgRepoAuthor(pr)

	comment := `[jx-info] Hi, we've detected that the pipelines in this repository are using a syntax that will soon be deprecated.
We'll continue to update you through PRs as we progress. Please check [#8589](https://www.github.com/jenkins-x/jx/issues/8589) for further information.
`

	if err := c.SCMProviderClient.CreateComment(org, repo, pr.Number, true, comment); err != nil {
		return errors.Wrap(err, "failed to comment info message")
	}
	return nil
}

func isPipelinesSyntaxDeprecated(cfg *config.Config, repo scm.Repository) bool {
	logger := logrus.WithField("repo", repo.FullName)
	for _, pre := range cfg.GetPresubmits(repo) {
		if pre.PipelineRunSpec == nil {
			err := pre.LoadPipeline(logger)
			if err != nil {
				return false
			}
		}
		if pre.IsResolvedWithUsesSyntax {
			return true
		}
	}
	for _, post := range cfg.GetPostsubmits(repo) {
		if post.PipelineRunSpec == nil {
			err := post.LoadPipeline(logger)
			if err != nil {
				return false
			}
		}
		if post.IsResolvedWithUsesSyntax {
			return true
		}
	}
	return false
}

func welcomeMsg(spc scmProviderClient, trigger *plugins.Trigger, pr scm.PullRequest) error {
	var errors []error
	org, repo, a := orgRepoAuthor(pr)
	author := string(a)
	encodedRepoFullName := url.QueryEscape(pr.Base.Repo.FullName)
	var more string
	if trigger.TrustedOrg != "" && trigger.TrustedOrg != org {
		more = fmt.Sprintf("or [%s](https://github.com/orgs/%s/people) ", trigger.TrustedOrg, trigger.TrustedOrg)
	}

	var joinOrgURL string
	if trigger.JoinOrgURL != "" {
		joinOrgURL = trigger.JoinOrgURL
	} else {
		joinOrgURL = fmt.Sprintf("https://github.com/orgs/%s/people", org)
	}

	var comment string
	if trigger.IgnoreOkToTest {
		comment = fmt.Sprintf(`Hi @%s. Thanks for your PR.

PRs from untrusted users cannot be marked as trusted with `+"`/ok-to-test`"+` in this repo meaning untrusted PR authors can never trigger tests themselves. Collaborators can still trigger tests on the PR using `+"`/test all`"+`.

I understand the commands that are listed [here](https://jenkins-x.io/v3/develop/reference/chatops/?repo=%s).

<details>

%s
</details>
`, author, encodedRepoFullName, plugins.AboutThisBotWithoutCommands)
	} else {
		comment = fmt.Sprintf(`Hi @%s. Thanks for your PR.

I'm waiting for a [%s](https://github.com/orgs/%s/people) %smember to verify that this patch is reasonable to test. If it is, they should reply with `+"`/ok-to-test`"+` on its own line. Until that is done, I will not automatically test new commits in this PR, but the usual testing commands by org members will still work. Regular contributors should [join the org](%s) to skip this step.

Once the patch is verified, the new status will be reflected by the `+"`%s`"+` label.

I understand the commands that are listed [here](https://jenkins-x.io/v3/develop/reference/chatops/?repo=%s).

<details>

%s
</details>
`, author, org, org, more, joinOrgURL, labels.OkToTest, encodedRepoFullName, plugins.AboutThisBotWithoutCommands)
		if err := spc.AddLabel(org, repo, pr.Number, labels.NeedsOkToTest, true); err != nil {
			errors = append(errors, err)
		}
	}

	if err := spc.CreateComment(org, repo, pr.Number, true, comment); err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return errorutil.NewAggregate(errors...)
	}
	return nil
}

func welcomeMsgForDraftPR(spc scmProviderClient, trigger *plugins.Trigger, pr scm.PullRequest) error {
	org, repo, _ := orgRepoAuthor(pr)

	comment := "This is a draft PR with the scheduler `trigger.SkipDraftPR` parameter enabled on this repository.\n"

	var errors []error

	if trigger.IgnoreOkToTest {
		comment += "Draft PRs cannot trigger pipelines with `/ok-to-test` in this repository. Collaborators can still trigger pipelines on the Draft PR using `/test all`."
	} else {
		comment += "To trigger pipelines on this draft PR, please mark it as ready for review by removing the draft status if you are trusted. Collaborators can also trigger tests on draft PRs using `/ok-to-test`."

		if err := spc.AddLabel(org, repo, pr.Number, labels.NeedsOkToTest, true); err != nil {
			errors = append(errors, err)
		}
	}

	comment += fmt.Sprintf(`
<details>

%s
</details>
`, plugins.AboutThisBotWithoutCommands)

	comments, err := spc.ListPullRequestComments(org, repo, pr.Number)
	if err != nil {
		errors = append(errors, err)
	}

	toComment := true
	for _, c := range comments {
		if c.Body == comment {
			toComment = false
			break
		}
	}

	if toComment {
		if err := spc.CreateComment(org, repo, pr.Number, true, comment); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return errorutil.NewAggregate(errors...)
	}
	return nil
}

// TrustedOrDraftPullRequest returns whether or not the given PR should be tested.
// It first checks if the author is in the org, then looks for "ok-to-test" label.
func TrustedOrDraftPullRequest(spc scmProviderClient, trigger *plugins.Trigger, author, org, repo string, num int, isDraft bool, l []*scm.Label) ([]*scm.Label, bool, error) {
	// First get PR labels
	if l == nil {
		var err error
		l, err = spc.GetIssueLabels(org, repo, num, true)
		if err != nil {
			return l, false, fmt.Errorf("error getting issue labels: %v", err)
		}
	}
	// Then check if the author is a member of the org and if trigger.SkipDraftPR disabled or PR not in draft
	if orgMember, err := TrustedUser(spc, trigger, author, org, repo); err != nil {
		return l, false, fmt.Errorf("error checking %s for trust: %v", author, err)
	} else if orgMember && (!trigger.SkipDraftPR || !isDraft) {
		return l, true, nil
	}
	// Then check if PR has ok-to-test label (untrusted user or trigger.SkipDraftPR enabled && draft PR)
	return l, scmprovider.HasLabel(labels.OkToTest, l), nil
}

// buildAll ensures that all builds that should run and will be required are built
func buildAll(c Client, pr *scm.PullRequest, eventGUID string, elideSkippedContexts bool) error {
	org, repo, number, branch := pr.Base.Repo.Namespace, pr.Base.Repo.Name, pr.Number, pr.Base.Ref
	changes := job.NewGitHubDeferredChangedFilesProvider(c.SCMProviderClient, org, repo, number)
	toTest, toSkip, err := jobutil.FilterPresubmits(jobutil.TestAllFilter(), changes, branch, c.Config.GetPresubmits(pr.Base.Repo), c.Logger)
	if err != nil {
		return err
	}
	return RunAndSkipJobs(c, pr, toTest, toSkip, eventGUID, elideSkippedContexts)
}
