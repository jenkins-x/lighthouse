package poller

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/jenkins-x/lighthouse/pkg/poller/pollstate"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strings"
)

var (
	censor = func(content []byte) []byte { return content }
)

type pollingController struct {
	repositoryNames []string
	gitServer       string
	scmClient       *scmprovider.Client
	fb              filebrowser.Interface
	pollstate       pollstate.Interface
	logger          *logrus.Entry
	notifier        func(webhook scm.Webhook) error
}

func NewPollingController(repositoryNames []string, gitServer string, fb filebrowser.Interface, notifier func(webhook scm.Webhook) error) (*pollingController, error) {
	logger := logrus.NewEntry(logrus.StandardLogger())
	if gitServer == "" {
		gitServer = "https://github.com"
	}
	return &pollingController{
		repositoryNames: repositoryNames,
		gitServer:       gitServer,
		logger:          logger,
		fb:              fb,
		notifier:        notifier,
		pollstate:       pollstate.NewMemoryPollState(),
	}, nil
}

func (c *pollingController) PollReleases() {
	for _, fullName := range c.repositoryNames {
		l := c.logger.WithField("Repo", fullName)

		// lets git clone and see if the latest git commit sha is new...
		owner, repo := scm.Split(fullName)
		ref := ""
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
			sha := strings.TrimSpace(string(out))
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

			l.Infof("triggering release webhook")

			before := ""
			pushHook, err := c.createPushHook(fullName, owner, repo, before, sha, branch)
			if err != nil {
				return errors.Wrapf(err, "failed to create PushHook")
			}

			err = c.notifier(pushHook)
			if err != nil {
				return errors.Wrapf(err, "failed to notify PushHook")
			}
			l.Infof("notified PushHook")
			return nil
		})
		if err != nil {
			l.WithError(err).Warn("failed to poll release")
		}
	}
}

func (c *pollingController) createPushHook(fullName, owner, repo, before, after, branch string) (*scm.PushHook, error) {
	return &scm.PushHook{
		Ref: "refs/heads/" + branch,
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
