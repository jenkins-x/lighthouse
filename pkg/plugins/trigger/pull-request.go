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
	"net/url"
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
		c.Logger.Infof("Author %q is a member, Starting all jobs for new PR.", author)
		return buildAll(c, &pr.PullRequest, pr.GUID, trigger.ElideSkippedContexts)
	case scm.ActionReopen:
		// When a PR is reopened, check that the user is in the org or that an org
		// member had said "/ok-to-test" before building, resulting in label ok-to-test.
		l, trusted, err := TrustedPullRequest(c.SCMProviderClient, trigger, author, org, repo, num, nil)
		if err != nil {
			return fmt.Errorf("could not validate PR: %s", err)
		} else if trusted {
			// Eventually remove need-ok-to-test
			// Does not work for TrustedUser() == true since labels are not fetched in this case
			if scmprovider.HasLabel(labels.NeedsOkToTest, l) {
				if err := c.SCMProviderClient.RemoveLabel(org, repo, num, labels.NeedsOkToTest, true); err != nil {
					return err
				}
			}
			c.Logger.Info("Starting all jobs for updated PR.")
			return buildAll(c, &pr.PullRequest, pr.GUID, trigger.ElideSkippedContexts)
		}
	case scm.ActionEdited, scm.ActionUpdate:
		// if someone changes the base of their PR, we will get this
		// event and the changes field will list that the base SHA and
		// ref changes so we can detect such a case and retrigger tests
		changes := pr.Changes
		if changes.Base.Ref.From != "" || changes.Base.Sha.From != "" {
			// the base of the PR changed and we need to re-test it
			return buildAllIfTrusted(c, trigger, pr)
		}
	case scm.ActionSync:
		return buildAllIfTrusted(c, trigger, pr)
	case scm.ActionLabel:
		// When a PR is LGTMd, if it is untrusted then build it once.
		if pr.Label.Name == labels.LGTM {
			_, trusted, err := TrustedPullRequest(c.SCMProviderClient, trigger, author, org, repo, num, nil)
			if err != nil {
				return fmt.Errorf("could not validate PR: %s", err)
			} else if !trusted {
				c.Logger.Info("Starting all jobs for untrusted PR with LGTM.")
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

func buildAllIfTrusted(c Client, trigger *plugins.Trigger, pr scm.PullRequestHook) error {
	// When a PR is updated, check that the user is in the org or that an org
	// member has said "/ok-to-test" before building. There's no need to ask
	// for "/ok-to-test" because we do that once when the PR is created.
	org, repo, a := orgRepoAuthor(pr.PullRequest)
	author := string(a)
	num := pr.PullRequest.Number
	l, trusted, err := TrustedPullRequest(c.SCMProviderClient, trigger, author, org, repo, num, nil)
	if err != nil {
		return fmt.Errorf("could not validate PR: %s", err)
	} else if trusted {
		// Eventually remove needs-ok-to-test
		// Will not work for org members since labels are not fetched in this case
		if scmprovider.HasLabel(labels.NeedsOkToTest, l) {
			if err := c.SCMProviderClient.RemoveLabel(org, repo, num, labels.NeedsOkToTest, true); err != nil {
				return err
			}
		}
		c.Logger.Info("Starting all jobs for updated PR.")
		return buildAll(c, &pr.PullRequest, pr.GUID, trigger.ElideSkippedContexts)
	}
	return nil
}

func infoMsg(c Client, pr scm.PullRequest) error {
	if isSyntaxDeprecated := isPipelinesSyntaxDeprecated(c.Config, pr.Repository()); !isSyntaxDeprecated {
		return nil
	}

	org, repo, a := orgRepoAuthor(pr)
	author := string(a)

	comment := fmt.Sprintf(`[jx-info] Hi @%s. We've detected that the pipelines in this repository are using a syntax that will soon be deprecated.
We'll continue to update you through PRs as we progress. Please check [#8589](https://www.github.com/jenkins-x/jx/issues/8589) for further information.
`, author)

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

// TrustedPullRequest returns whether or not the given PR should be tested.
// It first checks if the author is in the org, then looks for "ok-to-test" label.
func TrustedPullRequest(spc scmProviderClient, trigger *plugins.Trigger, author, org, repo string, num int, l []*scm.Label) ([]*scm.Label, bool, error) {
	// First check if the author is a member of the org.
	if orgMember, err := TrustedUser(spc, trigger, author, org, repo); err != nil {
		return l, false, fmt.Errorf("error checking %s for trust: %v", author, err)
	} else if orgMember {
		return l, true, nil
	}
	// Then check if PR has ok-to-test label
	if l == nil {
		var err error
		l, err = spc.GetIssueLabels(org, repo, num, true)
		if err != nil {
			return l, false, fmt.Errorf("error getting issue labels: %v", err)
		}
	}
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
