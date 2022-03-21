package poller

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/jenkins-x/lighthouse/pkg/poller/pollstate"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	censor = func(content []byte) []byte { return content }
)

type pollingController struct {
	DisablePollRelease          bool
	DisablePollPullRequest      bool
	repositoryNames             []string
	gitServer                   string
	scmClient                   *scm.Client
	fb                          filebrowser.Interface
	pollstate                   pollstate.Interface
	logger                      *logrus.Entry
	contextMatchPatternCompiled *regexp.Regexp
	requireReleaseSuccess       bool
	notifier                    func(webhook *scm.WebhookWrapper) error
}

func (c *pollingController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello from lighthouse poller\n"))
}

func NewPollingController(repositoryNames []string, gitServer string, scmClient *scm.Client, contextMatchPatternCompiled *regexp.Regexp, requireReleaseSuccess bool, fb filebrowser.Interface, notifier func(webhook *scm.WebhookWrapper) error) (*pollingController, error) {
	logger := logrus.NewEntry(logrus.StandardLogger())
	if gitServer == "" {
		gitServer = "https://github.com"
	}
	return &pollingController{
		repositoryNames:             repositoryNames,
		gitServer:                   gitServer,
		logger:                      logger,
		scmClient:                   scmClient,
		contextMatchPatternCompiled: contextMatchPatternCompiled,
		requireReleaseSuccess:       requireReleaseSuccess,
		fb:                          fb,
		notifier:                    notifier,
		pollstate:                   pollstate.NewMemoryPollState(),
	}, nil
}

func (c *pollingController) Logger() *logrus.Entry {
	return c.logger
}

func (c *pollingController) SyncReleases() {
	if !c.DisablePollRelease {
		c.PollReleases()
	}
}

func (c *pollingController) SyncPullRequests() {
	if !c.DisablePollPullRequest {
		c.PollPullRequests()
	}
}

func (c *pollingController) PollReleases() {
	ctx := context.TODO()

	for _, fullName := range c.repositoryNames {
		l := c.logger.WithField("Repo", fullName)

		l.Info("polling for new commit on main branch")

		// lets git clone and see if the latest git commit sha is new...
		owner, repo := scm.Split(fullName)
		ref := ""
		sha := ""
		fc := filebrowser.NewFetchCache()
		err := c.fb.WithDir(owner, repo, ref, fc, func(dir string) error {
			executor, err := git.NewCensoringExecutor(dir, censor, l)
			if err != nil {
				return errors.Wrapf(err, "failed to create git executor")
			}

			out, err := executor.Run("rev-parse", "HEAD")
			if err != nil {
				return errors.Wrapf(err, "failed to get latest git commit sha")
			}
			sha = strings.TrimSpace(string(out))
			if sha == "" {
				return errors.Errorf("could not find latest git commit sha")
			}
			out, err = executor.Run("rev-parse", "--abbrev-ref", "HEAD")
			if err != nil {
				return errors.Wrapf(err, "failed to get current git branch")
			}
			branch := strings.TrimSpace(string(out))

			l = l.WithField("SHA", sha).WithField("Branch", branch)

			newValue, err := c.pollstate.IsNew(fullName, "release", sha)
			if err != nil {
				return errors.Wrapf(err, "failed to check if sha %s is new", sha)
			}
			if !newValue {
				return nil
			}

			// lets check we have not triggered this before
			hasStatus, err := c.hasStatusForSHA(ctx, l, fullName, sha, true)
			if err != nil {
				return errors.Wrapf(err, "failed to check for status ")
			}
			if hasStatus {
				return nil
			}

			if c.requireReleaseSuccess {
				// We have not been able to find a successful release status so invalidate cache to retry
				c.pollstate.Invalidate(fullName, "release", sha)
			}

			l.Infof("triggering release webhook")

			before := ""
			pushHook, err := c.createPushHook(fullName, owner, repo, before, sha, branch, branch)
			if err != nil {
				return errors.Wrapf(err, "failed to create PushHook")
			}

			wh := &scm.WebhookWrapper{
				PushHook: pushHook,
			}
			err = c.notifier(wh)
			if err != nil {
				return errors.Wrapf(err, "failed to notify PushHook")
			}
			l.Infof("notified PushHook")
			return nil
		})
		if err != nil {
			c.pollstate.Invalidate(fullName, "release", sha)
			l.WithError(err).Warn("failed to poll release")
		}
	}
}

func (c *pollingController) PollPullRequests() {
	ctx := context.TODO()

	for _, fullName := range c.repositoryNames {
		l := c.logger.WithField("Repo", fullName)

		l.Info("polling for new commit on main branch")

		opts := &scm.PullRequestListOptions{
			Open: true,

			// TODO use last update poll?
		}
		prs, _, err := c.scmClient.PullRequests.List(ctx, fullName, opts)
		if err != nil {
			l.WithError(err).Error("failed to list open pull requests")
			continue
		}
		if len(prs) == 0 {
			l.Info("no open Pull Requests")
			continue
		}

		for _, pr := range prs {
			sha := pr.Sha
			if sha == "" {
				l.Infof("no SHA for PullRequest")
				continue
			}
			prName := "PR-" + strconv.Itoa(pr.Number)

			l2 := l.WithFields(map[string]interface{}{
				"SHA":         sha,
				"PullRequest": pr.Number,
			})
			err = c.pollPullRequest(ctx, l2, fullName, pr, prName, sha)
			if err != nil {
				c.pollstate.Invalidate(fullName, prName, "created")
				l2.WithError(err).Error("failed to check for PullRequestHook")
				continue
			}
			err = c.pollPullRequestPushHook(ctx, l2, fullName, pr, prName, sha)
			if err != nil {
				c.pollstate.Invalidate(fullName, prName+"-push", sha)
				l2.WithError(err).Error("failed to check for PullRequestHook")
				continue
			}
		}
	}
}

func (c *pollingController) pollPullRequest(ctx context.Context, l *logrus.Entry, fullName string, pr *scm.PullRequest, prName, sha string) error {
	newValue, err := c.pollstate.IsNew(fullName, prName, "created")
	if err != nil {
		return errors.Wrapf(err, "failed to check if sha %s is new", sha)
	}
	if !newValue {
		return nil
	}

	hasStatus, err := c.hasStatusForSHA(ctx, l, fullName, sha, false)
	if err != nil {
		return errors.Wrapf(err, "failed to check for status ")
	}
	if hasStatus {
		return nil
	}

	l.Infof("triggering pull request webhook")

	pullRequestHook, err := c.createPullRequestHook(fullName, pr)
	if err != nil {
		return errors.Wrapf(err, "failed to create PullRequestHook")
	}

	wh := &scm.WebhookWrapper{
		PullRequestHook: pullRequestHook,
	}
	err = c.notifier(wh)
	if err != nil {
		return errors.Wrapf(err, "failed to notify PullRequestHook")
	}
	l.Infof("notified PullRequestHook")
	return nil
}

func (c *pollingController) pollPullRequestPushHook(ctx context.Context, l *logrus.Entry, fullName string, pr *scm.PullRequest, prName, sha string) error {
	newValue, err := c.pollstate.IsNew(fullName, prName+"-push", sha)
	if err != nil {
		return errors.Wrapf(err, "failed to check if sha %s is new", sha)
	}
	if !newValue {
		return nil
	}

	hasStatus, err := c.hasStatusForSHA(ctx, l, fullName, sha, false)
	if err != nil {
		return errors.Wrapf(err, "failed to check for status ")
	}
	if hasStatus {
		return nil
	}

	l.Infof("triggering pull request push webhook")

	owner, repo := scm.Split(fullName)
	branch := pr.Repository().Branch
	if branch == "" {
		branch = pr.Head.Ref
	}
	l = l.WithField("Branch", branch)
	before := ""
	refBranch := pr.Source
	pushHook, err := c.createPushHook(fullName, owner, repo, before, sha, branch, refBranch)
	if err != nil {
		return errors.Wrapf(err, "failed to create PushHook")
	}

	wh := &scm.WebhookWrapper{
		PushHook: pushHook,
	}
	err = c.notifier(wh)
	if err != nil {
		return errors.Wrapf(err, "failed to notify PR PushHook")
	}
	l.Infof("notified PR PushHook")
	return nil
}

func (c *pollingController) hasStatusForSHA(ctx context.Context, l *logrus.Entry, fullName string, sha string, isRelease bool) (bool, error) {
	statuses, err := c.ListAllStatuses(ctx, fullName, sha)
	if err != nil {
		return false, errors.Wrapf(err, "failed to list status")
	}
	for _, s := range statuses {
		if c.isMatchingStatus(s, isRelease) {
			l.WithField("Statuses", statuses).Info("the SHA has CI statuses so not triggering")
			return true, nil
		}
	}
	return false, nil
}

func (c *pollingController) ListAllStatuses(ctx context.Context, fullName string, sha string) ([]*scm.Status, error) {
	allStatuses := []*scm.Status{}
	page := 1
	for {
		opts := scm.ListOptions{
			Page: page,
			Size: 100,
		}
		statuses, response, err := c.scmClient.Repositories.ListStatus(ctx, fullName, sha, opts)
		if err != nil {
			return allStatuses, err
		}
		allStatuses = append(allStatuses, statuses...)
		if page == response.Page.Next || response.Page.Next < 1 {
			break
		}
		page = response.Page.Next
	}
	return allStatuses, nil
}

func (c *pollingController) isMatchingStatus(s *scm.Status, isRelease bool) bool {
	if isRelease && c.requireReleaseSuccess {
		if s.State != scm.StateSuccess {
			return false
		}
	}

	if c.contextMatchPatternCompiled != nil {
		return c.contextMatchPatternCompiled.MatchString(s.Label)
	}
	return !strings.HasPrefix(s.Label, "Lighthouse")
}

func (c *pollingController) createPushHook(fullName, owner, repo, before, after, branch, refBranch string) (*scm.PushHook, error) {
	return &scm.PushHook{
		Ref: "refs/heads/" + refBranch,
		Repo: scm.Repository{
			Namespace: owner,
			Name:      repo,
			FullName:  fullName,
			Branch:    branch,
			Clone:     c.gitServer + "/" + fullName + ".git",
			Link:      c.gitServer + "/" + fullName,
		},
		Before:       before,
		After:        after,
		Commits:      nil,
		Commit:       scm.Commit{},
		Sender:       scm.User{},
		GUID:         after,
		Installation: nil,
	}, nil
}

func (c *pollingController) createPullRequestHook(fullName string, pr *scm.PullRequest) (*scm.PullRequestHook, error) {
	repo := pr.Repository()
	return &scm.PullRequestHook{
		Action:       scm.ActionOpen,
		Repo:         repo,
		Label:        scm.Label{},
		PullRequest:  *pr,
		Sender:       scm.User{},
		Changes:      scm.PullRequestHookChanges{},
		GUID:         "",
		Installation: nil,
	}, nil
}
