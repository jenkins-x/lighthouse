/*
Copyright 2017 The Kubernetes Authors.

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

package approve

import (
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/plugins/approve/approvers"
	"github.com/jenkins-x/lighthouse/pkg/repoowners"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// PluginName defines this plugin's registered name.
	PluginName = "approve"

	approveCommand  = "APPROVE"
	cancelArgument  = "cancel"
	lgtmCommand     = "LGTM"
	noIssueArgument = "no-issue"
)

var (
	associatedIssueRegexFormat = `(?:%s/[^/]+/issues/|#)(\d+)`
	commandRegex               = regexp.MustCompile(`(?m)^/([^\s]+)[\t ]*([^\n\r]*)`)
	notificationRegex          = regexp.MustCompile(`(?is)^\[` + approvers.ApprovalNotificationName + `\] *?([^\n]*)(?:\n\n(.*))?`)

	// deprecatedBotNames are the names of the bots that previously handled approvals.
	// Each can be removed once every PR approved by the old bot has been merged or unapproved.
	deprecatedBotNames = []string{"k8s-merge-robot", "openshift-merge-robot"}

	// handleFunc is used to allow mocking out the behavior of 'handle' while testing.
	handleFunc = handle
)

type scmProviderClient interface {
	GetPullRequest(org, repo string, number int) (*scm.PullRequest, error)
	GetPullRequestChanges(org, repo string, number int) ([]*scm.Change, error)
	GetIssueLabels(org, repo string, number int, pr bool) ([]*scm.Label, error)
	ListIssueComments(org, repo string, number int) ([]*scm.Comment, error)
	ListReviews(org, repo string, number int) ([]*scm.Review, error)
	ListPullRequestComments(org, repo string, number int) ([]*scm.Comment, error)
	DeleteComment(org, repo string, number, ID int, pr bool) error
	CreateComment(org, repo string, number int, pr bool, comment string) error
	BotName() (string, error)
	AddLabel(org, repo string, number int, label string, pr bool) error
	RemoveLabel(org, repo string, number int, label string, pr bool) error
	ListIssueEvents(org, repo string, num int) ([]*scm.ListedIssueEvent, error)
	ProviderType() string
}

type ownersClient interface {
	LoadRepoOwners(org, repo, base string) (repoowners.RepoOwner, error)
}

type state struct {
	org    string
	repo   string
	branch string
	number int

	body      string
	author    string
	assignees []scm.User
	htmlURL   string
}

func init() {
	plugins.RegisterGenericCommentHandler(PluginName, handleGenericCommentEvent, helpProvider)
	plugins.RegisterReviewEventHandler(PluginName, handleReviewEvent, helpProvider)
	plugins.RegisterPullRequestHandler(PluginName, handlePullRequestEvent, helpProvider)
}

func helpProvider(config *plugins.Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	doNot := func(b bool) string {
		if b {
			return ""
		}
		return "do not "
	}
	willNot := func(b bool) string {
		if b {
			return "will "
		}
		return "will not "
	}

	approveConfig := map[string]string{}
	for _, repo := range enabledRepos {
		parts := strings.Split(repo, "/")
		var opts *plugins.Approve
		switch len(parts) {
		case 1:
			opts = optionsForRepo(config, repo, "")
		case 2:
			opts = optionsForRepo(config, parts[0], parts[1])
		default:
			return nil, fmt.Errorf("invalid repo in enabledRepos: %q", repo)
		}
		approveConfig[repo] = fmt.Sprintf("Pull requests %s require an associated issue.<br>Pull request authors %s implicitly approve their own PRs.<br>The /lgtm [cancel] command(s) %s act as approval.<br>A GitHub approved or changes requested review %s act as approval or cancel respectively.", doNot(opts.IssueRequired), doNot(opts.HasSelfApproval()), willNot(opts.LgtmActsAsApprove), willNot(opts.ConsiderReviewState()))
	}
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The approve plugin implements a pull request approval process that manages the '` + labels.Approved + `' label and an approval notification comment. Approval is achieved when the set of users that have approved the PR is capable of approving every file changed by the PR. A user is able to approve a file if their username or an alias they belong to is listed in the 'approvers' section of an OWNERS file in the directory of the file or higher in the directory tree.
<br>
<br>Per-repo configuration may be used to require that PRs link to an associated issue before approval is granted. It may also be used to specify that the PR authors implicitly approve their own PRs.
<br>For more information see <a href="https://git.github.com/jenkins-x/lighthouse/pkg/prow/plugins/approve/approvers/README.md">here</a>.`,
		Config: approveConfig,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/approve [no-issue|cancel]",
		Description: "Approves a pull request",
		Featured:    true,
		WhoCanUse:   "Users listed as 'approvers' in appropriate OWNERS files.",
		Examples:    []string{"/approve", "/approve no-issue", "/lh-approve"},
	})
	return pluginHelp, nil
}

func handleGenericCommentEvent(pc plugins.Agent, ce scmprovider.GenericCommentEvent) error {
	return handleGenericComment(
		pc.Logger,
		pc.SCMProviderClient,
		pc.OwnersClient,
		pc.ServerURL,
		pc.PluginConfig,
		&ce,
	)
}

func handleGenericComment(log *logrus.Entry, spc scmProviderClient, oc ownersClient, serverURL *url.URL, config *plugins.Configuration, ce *scmprovider.GenericCommentEvent) error {
	if ce.Action != scm.ActionCreate || !ce.IsPR || ce.IssueState == "closed" {
		return nil
	}

	botName, err := spc.BotName()
	if err != nil {
		return err
	}

	opts := optionsForRepo(config, ce.Repo.Namespace, ce.Repo.Name)
	if !isApprovalCommand(botName, opts.LgtmActsAsApprove, &comment{Body: ce.Body, Author: ce.Author.Login}) {
		return nil
	}

	pr, err := spc.GetPullRequest(ce.Repo.Namespace, ce.Repo.Name, ce.Number)
	if err != nil {
		return err
	}

	log.Warnf("pr: %+v", pr)
	repo, err := oc.LoadRepoOwners(ce.Repo.Namespace, ce.Repo.Name, pr.Base.Ref)
	if err != nil {
		return err
	}

	return handleFunc(
		log,
		spc,
		repo,
		serverURL,
		opts,
		&state{
			org:       ce.Repo.Namespace,
			repo:      ce.Repo.Name,
			branch:    pr.Base.Ref,
			number:    ce.Number,
			body:      ce.IssueBody,
			author:    ce.IssueAuthor.Login,
			assignees: ce.Assignees,
			htmlURL:   ce.IssueLink,
		},
	)
}

// handleReviewEvent should only handle reviews that have no approval command.
// Reviews with approval commands will be handled by handleGenericCommentEvent.
func handleReviewEvent(pc plugins.Agent, re scm.ReviewHook) error {
	return handleReview(
		pc.Logger,
		pc.SCMProviderClient,
		pc.OwnersClient,
		pc.ServerURL,
		pc.PluginConfig,
		&re,
	)
}

func handleReview(log *logrus.Entry, spc scmProviderClient, oc ownersClient, serverURL *url.URL, config *plugins.Configuration, re *scm.ReviewHook) error {
	if re.Action != scm.ActionSubmitted && re.Action != scm.ActionDismissed {
		return nil
	}

	botName, err := spc.BotName()
	if err != nil {
		return err
	}

	opts := optionsForRepo(config, re.Repo.Namespace, re.Repo.Name)

	// Check for an approval command is in the body. If one exists, let the
	// genericCommentEventHandler handle this event. Approval commands override
	// review state.
	if isApprovalCommand(botName, opts.LgtmActsAsApprove, &comment{Body: re.Review.Body, Author: re.Review.Author.Login}) {
		return nil
	}

	// Check for an approval command via review state. If none exists, don't
	// handle this event.
	if !isApprovalState(botName, opts.ConsiderReviewState(), &comment{Author: re.Review.Author.Login, ReviewState: re.Review.State}) {
		return nil
	}

	repo, err := oc.LoadRepoOwners(re.Repo.Namespace, re.Repo.Name, re.PullRequest.Base.Ref)
	if err != nil {
		return err
	}

	return handleFunc(
		log,
		spc,
		repo,
		serverURL,
		optionsForRepo(config, re.Repo.Namespace, re.Repo.Name),
		&state{
			org:       re.Repo.Namespace,
			repo:      re.Repo.Name,
			branch:    re.PullRequest.Base.Ref,
			number:    re.PullRequest.Number,
			body:      re.PullRequest.Body,
			author:    re.PullRequest.Author.Login,
			assignees: re.PullRequest.Assignees,
			htmlURL:   re.PullRequest.Link,
		},
	)

}

func handlePullRequestEvent(pc plugins.Agent, pre scm.PullRequestHook) error {
	return handlePullRequest(
		pc.Logger,
		pc.SCMProviderClient,
		pc.OwnersClient,
		pc.ServerURL,
		pc.PluginConfig,
		&pre,
	)
}

func handlePullRequest(log *logrus.Entry, spc scmProviderClient, oc ownersClient, serverURL *url.URL, config *plugins.Configuration, pre *scm.PullRequestHook) error {
	if pre.Action != scm.ActionOpen &&
		pre.Action != scm.ActionReopen &&
		pre.Action != scm.ActionSync &&
		pre.Action != scm.ActionLabel {
		return nil
	}
	botName, err := spc.BotName()
	if err != nil {
		return err
	}
	if pre.Action == scm.ActionLabel &&
		(pre.Label.Name != labels.Approved || pre.Sender.Login == botName || pre.PullRequest.State == "closed") {
		return nil
	}

	ref := pre.PullRequest.Base.Ref
	log.Warnf("pre PR: %+v", pre.PullRequest)
	repo, err := oc.LoadRepoOwners(pre.Repo.Namespace, pre.Repo.Name, ref)
	if err != nil {
		return err
	}

	return handleFunc(
		log,
		spc,
		repo,
		serverURL,
		optionsForRepo(config, pre.Repo.Namespace, pre.Repo.Name),
		&state{
			org:       pre.Repo.Namespace,
			repo:      pre.Repo.Name,
			branch:    ref,
			number:    pre.PullRequest.Number,
			body:      pre.PullRequest.Body,
			author:    pre.PullRequest.Author.Login,
			assignees: pre.PullRequest.Assignees,
			htmlURL:   pre.PullRequest.Link,
		},
	)
}

// Returns associated issue, or 0 if it can't find any.
// This is really simple, and could be improved later.
func findAssociatedIssue(body, org string) (int, error) {
	associatedIssueRegex, err := regexp.Compile(fmt.Sprintf(associatedIssueRegexFormat, org))
	if err != nil {
		return 0, err
	}
	match := associatedIssueRegex.FindStringSubmatch(body)
	if len(match) == 0 {
		return 0, nil
	}
	v, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, err
	}
	return v, nil
}

// handle is the workhorse the will actually make updates to the PR.
// The algorithm goes as:
// - Initially, we build an approverSet
//   - Go through all comments in order of creation.
//     - (Issue/PR comments, PR review comments, and PR review bodies are considered as comments)
//   - If anyone said "/approve", add them to approverSet.
//   - If anyone said "/lgtm" AND LgtmActsAsApprove is enabled, add them to approverSet.
//   - If anyone created an approved review AND ReviewActsAsApprove is enabled, add them to approverSet.
// - Then, for each file, we see if any approver of this file is in approverSet and keep track of files without approval
//   - An approver of a file is defined as:
//     - Someone listed as an "approver" in an OWNERS file in the files directory OR
//     - in one of the file's parent directories
// - Iff all files have been approved, the bot will add the "approved" label.
// - Iff a cancel command is found, that reviewer will be removed from the approverSet
// 	and the munger will remove the approved label if it has been applied
func handle(log *logrus.Entry, spc scmProviderClient, repo approvers.Repo, serverURL *url.URL, opts *plugins.Approve, pr *state) error {
	fetchErr := func(context string, err error) error {
		return fmt.Errorf("failed to get %s for %s/%s#%d: %v", context, pr.org, pr.repo, pr.number, err)
	}

	changes, err := spc.GetPullRequestChanges(pr.org, pr.repo, pr.number)
	if err != nil {
		return fetchErr("PR file changes", err)
	}
	var filenames []string
	for _, change := range changes {
		filenames = append(filenames, change.Path)
	}
	issueLabels, err := spc.GetIssueLabels(pr.org, pr.repo, pr.number, true)
	if err != nil {
		return fetchErr("issue labels", err)
	}
	hasApprovedLabel := false
	for _, label := range issueLabels {
		if label.Name == labels.Approved {
			hasApprovedLabel = true
			break
		}
	}
	botName, err := spc.BotName()
	if err != nil {
		return fetchErr("bot name", err)
	}
	var issueComments []*scm.Comment
	// Get issue comments _only_ if this is GitHub. Otherwise just get PR comments.
	if spc.ProviderType() == "github" {
		issueComments, err = spc.ListIssueComments(pr.org, pr.repo, pr.number)
		if err != nil {
			return fetchErr("issue comments", err)
		}
	}
	reviewComments, err := spc.ListPullRequestComments(pr.org, pr.repo, pr.number)
	if err != nil {
		return fetchErr("review comments", err)
	}
	reviews, err := spc.ListReviews(pr.org, pr.repo, pr.number)
	if err != nil && err.Error() != scm.ErrNotSupported.Error() {
		return fetchErr("reviews", err)
	}

	approversHandler := approvers.NewApprovers(
		approvers.NewOwners(
			log,
			filenames,
			repo,
			int64(pr.number),
		),
	)
	approversHandler.AssociatedIssue, err = findAssociatedIssue(pr.body, pr.org)
	if err != nil {
		log.WithError(err).Errorf("Failed to find associated issue from PR body: %v", err)
	}
	approversHandler.RequireIssue = opts.IssueRequired
	approversHandler.ManuallyApproved = humanAddedApproved(spc, log, pr.org, pr.repo, pr.number, botName, hasApprovedLabel)

	// Author implicitly approves their own PR if config allows it
	if opts.HasSelfApproval() {
		approversHandler.AddAuthorSelfApprover(pr.author, pr.htmlURL+"#", false)
	} else {
		// Treat the author as an assignee, and suggest them if possible
		approversHandler.AddAssignees(pr.author)
	}

	commentsFromIssueComments := commentsFromIssueComments(issueComments)
	comments := append(commentsFromReviewComments(reviewComments), commentsFromIssueComments...)
	comments = append(comments, commentsFromReviews(reviews)...)
	sort.SliceStable(comments, func(i, j int) bool {
		return comments[i].Created.Before(comments[j].Created)
	})
	approveComments := filterComments(comments, approvalMatcher(botName, opts.LgtmActsAsApprove, opts.ConsiderReviewState()))
	addApprovers(&approversHandler, approveComments, pr.author, opts.ConsiderReviewState())

	for _, user := range pr.assignees {
		approversHandler.AddAssignees(user.Login)
	}

	notifications := filterComments(comments, notificationMatcher(botName))
	latestNotification := getLast(notifications)
	usePrefix := spc.ProviderType() == "gitlab"
	newMessage := updateNotification(serverURL, pr.org, pr.repo, pr.branch, latestNotification, approversHandler, usePrefix)
	if newMessage != nil {
		for _, notif := range notifications {
			if err := spc.DeleteComment(pr.org, pr.repo, pr.number, notif.ID, true); err != nil {
				log.WithError(err).Errorf("Failed to delete comment from %s/%s#%d, ID: %d.", pr.org, pr.repo, pr.number, notif.ID)
			}
		}
		if err := spc.CreateComment(pr.org, pr.repo, pr.number, true, *newMessage); err != nil {
			log.WithError(err).Errorf("Failed to create comment on %s/%s#%d: %q.", pr.org, pr.repo, pr.number, *newMessage)
		}
	}

	if !approversHandler.IsApproved() {
		if hasApprovedLabel {
			if err := spc.RemoveLabel(pr.org, pr.repo, pr.number, labels.Approved, true); err != nil {
				log.WithError(err).Errorf("Failed to remove %q label from %s/%s#%d.", labels.Approved, pr.org, pr.repo, pr.number)
			}
		}
	} else if !hasApprovedLabel {
		if err := spc.AddLabel(pr.org, pr.repo, pr.number, labels.Approved, true); err != nil {
			log.WithError(err).Errorf("Failed to add %q label to %s/%s#%d.", labels.Approved, pr.org, pr.repo, pr.number)
		}
	}
	return nil
}

func humanAddedApproved(spc scmProviderClient, log *logrus.Entry, org, repo string, number int, botName string, hasLabel bool) func() bool {
	findOut := func() bool {
		if !hasLabel {
			return false
		}
		events, err := spc.ListIssueEvents(org, repo, number)
		if err != nil {
			log.WithError(err).Errorf("Failed to list issue events for %s/%s#%d.", org, repo, number)
			return false
		}
		lastAdded := scm.ListedIssueEvent{}
		for _, event := range events {
			// Only consider "approved" label added events.
			if event.Event != scmprovider.IssueActionLabeled || event.Label.Name != labels.Approved {
				continue
			}
			lastAdded = *event
		}

		if lastAdded.Actor.Login == "" || lastAdded.Actor.Login == botName || isDeprecatedBot(lastAdded.Actor.Login) {
			return false
		}
		return true
	}

	var cache *bool
	return func() bool {
		if cache == nil {
			val := findOut()
			cache = &val
		}
		return *cache
	}
}

func approvalMatcher(botName string, lgtmActsAsApprove, reviewActsAsApprove bool) func(*comment) bool {
	return func(c *comment) bool {
		return isApprovalCommand(botName, lgtmActsAsApprove, c) || isApprovalState(botName, reviewActsAsApprove, c)
	}
}

func isApprovalCommand(botName string, lgtmActsAsApprove bool, c *comment) bool {
	if c.Author == botName || isDeprecatedBot(c.Author) {
		return false
	}

	for _, match := range commandRegex.FindAllStringSubmatch(c.Body, -1) {
		cmd := removeLighthouseCommandPrefix(match[1])
		if (cmd == lgtmCommand && lgtmActsAsApprove) || cmd == approveCommand {
			return true
		}
	}
	return false
}

func removeLighthouseCommandPrefix(cmd string) string {
	cmd = strings.ToUpper(cmd)
	if strings.HasPrefix(cmd, strings.ToUpper(util.LighthouseCommandPrefix)) {
		cmd = strings.TrimPrefix(cmd, strings.ToUpper(util.LighthouseCommandPrefix))
	}
	return cmd
}

func isApprovalState(botName string, reviewActsAsApprove bool, c *comment) bool {
	if c.Author == botName || isDeprecatedBot(c.Author) {
		return false
	}

	// The review webhook returns state as lowercase, while the review API
	// returns state as uppercase. Uppercase the value here so it always
	// matches the constant.
	reviewState := strings.ToUpper(c.ReviewState)

	// ReviewStateApproved = /approve
	// ReviewStateChangesRequested = /approve cancel
	// ReviewStateDismissed = remove previous approval or disapproval
	// (Reviews can go from Approved or ChangesRequested to Dismissed
	// state if the Dismiss action is used)
	if reviewActsAsApprove && (reviewState == scm.ReviewStateApproved ||
		reviewState == scm.ReviewStateChangesRequested ||
		reviewState == scm.ReviewStateDismissed) {
		return true
	}
	return false
}

func notificationMatcher(botName string) func(*comment) bool {
	return func(c *comment) bool {
		if c.Author != botName && !isDeprecatedBot(c.Author) {
			return false
		}
		match := notificationRegex.FindStringSubmatch(c.Body)
		return len(match) > 0
	}
}

func updateNotification(linkURL *url.URL, org, repo, branch string, latestNotification *comment, approversHandler approvers.Approvers, usePrefix bool) *string {
	message := approvers.GetMessage(approversHandler, linkURL, org, repo, branch, usePrefix)
	if message == nil || (latestNotification != nil && strings.Contains(latestNotification.Body, *message)) {
		return nil
	}
	return message
}

// addApprovers iterates through the list of comments on a PR
// and identifies all of the people that have said /approve and adds
// them to the Approvers.  The function uses the latest approve or cancel comment
// to determine the Users intention. A review in requested changes state is
// considered a cancel.
func addApprovers(approversHandler *approvers.Approvers, approveComments []*comment, author string, reviewActsAsApprove bool) {
	for _, c := range approveComments {
		if c.Author == "" {
			continue
		}

		if reviewActsAsApprove && c.ReviewState == scm.ReviewStateApproved {
			approversHandler.AddApprover(
				c.Author,
				c.Link,
				false,
			)
		}
		if reviewActsAsApprove && c.ReviewState == scm.ReviewStateChangesRequested {
			approversHandler.RemoveApprover(c.Author)
		}

		for _, match := range commandRegex.FindAllStringSubmatch(c.Body, -1) {
			name := removeLighthouseCommandPrefix(match[1])
			if name != approveCommand && name != lgtmCommand {
				continue
			}
			args := strings.ToLower(strings.TrimSpace(match[2]))
			if strings.Contains(args, cancelArgument) {
				approversHandler.RemoveApprover(c.Author)
				continue
			}

			if c.Author == author {
				approversHandler.AddAuthorSelfApprover(
					c.Author,
					c.Link,
					args == noIssueArgument,
				)
			}

			if name == approveCommand {
				approversHandler.AddApprover(
					c.Author,
					c.Link,
					args == noIssueArgument,
				)
			} else {
				approversHandler.AddLGTMer(
					c.Author,
					c.Link,
					args == noIssueArgument,
				)
			}

		}
	}
}

// optionsForRepo gets the plugins.Approve struct that is applicable to the indicated repo.
func optionsForRepo(config *plugins.Configuration, org, repo string) *plugins.Approve {
	fullName := fmt.Sprintf("%s/%s", org, repo)

	a := func() *plugins.Approve {
		// First search for repo config
		for _, c := range config.Approve {
			if !sets.NewString(c.Repos...).Has(fullName) {
				continue
			}
			return &c
		}

		// If you don't find anything, loop again looking for an org config
		for _, c := range config.Approve {
			if !sets.NewString(c.Repos...).Has(org) {
				continue
			}
			return &c
		}

		// Return an empty config, and use plugin defaults
		return &plugins.Approve{}
	}()
	if a.DeprecatedImplicitSelfApprove == nil && a.RequireSelfApproval == nil && config.UseDeprecatedSelfApprove {
		no := false
		a.DeprecatedImplicitSelfApprove = &no
	}
	if a.DeprecatedReviewActsAsApprove == nil && a.IgnoreReviewState == nil && config.UseDeprecatedReviewApprove {
		no := false
		a.DeprecatedReviewActsAsApprove = &no
	}
	return a
}

type comment struct {
	Body        string
	Author      string
	Created     time.Time
	Link        string
	ID          int
	ReviewState string
}

func commentFromIssueComment(ic *scm.Comment) *comment {
	if ic == nil {
		return nil
	}
	return &comment{
		Body:    ic.Body,
		Author:  ic.Author.Login,
		Created: ic.Created,
		Link:    ic.Link,
		ID:      ic.ID,
	}
}

func commentsFromIssueComments(ics []*scm.Comment) []*comment {
	comments := make([]*comment, 0, len(ics))
	for i := range ics {
		comments = append(comments, commentFromIssueComment(ics[i]))
	}
	return comments
}

func commentFromReviewComment(rc *scm.Comment) *comment {
	if rc == nil {
		return nil
	}
	return &comment{
		Body:    rc.Body,
		Author:  rc.Author.Login,
		Created: rc.Created,
		Link:    rc.Link,
		ID:      rc.ID,
	}
}

func commentsFromReviewComments(rcs []*scm.Comment) []*comment {
	comments := make([]*comment, 0, len(rcs))
	for i := range rcs {
		comments = append(comments, commentFromReviewComment(rcs[i]))
	}
	return comments
}

func commentFromReview(review *scm.Review) *comment {
	if review == nil {
		return nil
	}
	return &comment{
		Body:        review.Body,
		Author:      review.Author.Login,
		Created:     review.Created,
		Link:        review.Link,
		ID:          review.ID,
		ReviewState: review.State,
	}
}

func commentsFromReviews(reviews []*scm.Review) []*comment {
	comments := make([]*comment, 0, len(reviews))
	for i := range reviews {
		comments = append(comments, commentFromReview(reviews[i]))
	}
	return comments
}

func filterComments(comments []*comment, filter func(*comment) bool) []*comment {
	filtered := make([]*comment, 0, len(comments))
	for _, c := range comments {
		if filter(c) {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func getLast(cs []*comment) *comment {
	if len(cs) == 0 {
		return nil
	}
	return cs[len(cs)-1]
}

func isDeprecatedBot(login string) bool {
	for _, deprecated := range deprecatedBotNames {
		if deprecated == login {
			return true
		}
	}
	return false
}
