package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const github = "github.com"

// Client represents a git client
type Client interface {
	Clean() error
	SetRemote(remote string)
	SetCredentials(user string, tokenGenerator func() []byte)
	Clone(repo string) (*Repo, error)
}

// Repo is a clone of a git repository. Create with Client.Clone, and don't
// forget to clean it up after.
type Repo struct {
	// Dir is the location of the git repo.
	Dir string

	// git is the path to the git binary.
	git string
	// base is the base path for remote git fetch calls.
	base string
	// repo is the full repo name: "org/repo".
	repo string
	// user is used for pushing to the remote repo.
	user string
	// pass is used for pushing to the remote repo.
	pass string

	logger *logrus.Entry
}

// Clean deletes the repo. It is unusable after calling.
func (r *Repo) Clean() error {
	return os.RemoveAll(r.Dir)
}

func (r *Repo) gitCommand(arg ...string) *exec.Cmd {
	cmd := exec.Command(r.git, arg...)
	cmd.Dir = r.Dir
	return cmd
}

// Checkout runs git checkout.
func (r *Repo) Checkout(commitlike string) error {
	r.logger.Infof("Checkout %s.", commitlike)
	co := r.gitCommand("checkout", commitlike)
	if b, err := co.CombinedOutput(); err != nil {
		return fmt.Errorf("error checking out %s: %v. output: %s", commitlike, err, string(b))
	}
	return nil
}

// RevParse runs git rev-parse.
func (r *Repo) RevParse(commitlike string) (string, error) {
	r.logger.Infof("RevParse %s.", commitlike)
	b, err := r.gitCommand("rev-parse", commitlike).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error rev-parsing %s: %v. output: %s", commitlike, err, string(b))
	}
	return string(b), nil
}

// CheckoutNewBranch creates a new branch and checks it out.
func (r *Repo) CheckoutNewBranch(branch string) error {
	r.logger.Infof("Create and checkout %s.", branch)
	co := r.gitCommand("checkout", "-b", branch)
	if b, err := co.CombinedOutput(); err != nil {
		return fmt.Errorf("error checking out %s: %v. output: %s", branch, err, string(b))
	}
	return nil
}

// Merge attempts to merge commitlike into the current branch. It returns true
// if the merge completes. It returns an error if the abort fails.
func (r *Repo) Merge(commitlike string) (bool, error) {
	r.logger.Infof("Merging %s.", commitlike)
	co := r.gitCommand("merge", "--no-ff", "--no-stat", "-m merge", commitlike)

	b, err := co.CombinedOutput()
	if err == nil {
		return true, nil
	}
	r.logger.WithError(err).Warningf("Merge failed with output: %s", string(b))

	if b, err := r.gitCommand("merge", "--abort").CombinedOutput(); err != nil {
		return false, fmt.Errorf("error aborting merge for commitlike %s: %v. output: %s", commitlike, err, string(b))
	}

	return false, nil
}

// Am tries to apply the patch in the given path into the current branch
// by performing a three-way merge (similar to git cherry-pick). It returns
// an error if the patch cannot be applied.
func (r *Repo) Am(path string) error {
	r.logger.Infof("Applying %s.", path)
	co := r.gitCommand("am", "--3way", path)
	b, err := co.CombinedOutput()
	if err == nil {
		return nil
	}
	output := string(b)
	r.logger.WithError(err).Warningf("Patch apply failed with output: %s", output)
	if b, abortErr := r.gitCommand("am", "--abort").CombinedOutput(); err != nil {
		r.logger.WithError(abortErr).Warningf("Aborting patch apply failed with output: %s", string(b))
	}
	applyMsg := "The copy of the patch that failed is found in: .git/rebase-apply/patch"
	if strings.Contains(output, applyMsg) {
		i := strings.Index(output, applyMsg)
		err = fmt.Errorf("%s", output[:i])
	}
	return err
}

// Push pushes over https to the provided owner/repo#branch using a password
// for basic auth.
func (r *Repo) Push(repo, branch string) error {
	if r.user == "" || r.pass == "" {
		return errors.New("cannot push without credentials - configure your git client")
	}
	r.logger.Infof("Pushing to '%s/%s (branch: %s)'.", r.user, repo, branch)
	remote := fmt.Sprintf("https://%s:%s@%s/%s/%s", r.user, r.pass, github, r.user, repo)
	co := r.gitCommand("push", remote, branch)
	_, err := co.CombinedOutput()
	return err
}

// CheckoutPullRequest does exactly that.
func (r *Repo) CheckoutPullRequest(number int) error {
	r.logger.Infof("Fetching and checking out %s#%d.", r.repo, number)
	if b, err := retryCmd(r.logger, r.Dir, r.git, "fetch", r.base+"/"+r.repo, fmt.Sprintf("pull/%d/head:pull%d", number, number)); err != nil {
		return fmt.Errorf("git fetch failed for PR %d: %v. output: %s", number, err, string(b))
	}
	co := r.gitCommand("checkout", fmt.Sprintf("pull%d", number))
	if b, err := co.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed for PR %d: %v. output: %s", number, err, string(b))
	}
	return nil
}

// Config runs git config.
func (r *Repo) Config(key, value string) error {
	r.logger.Infof("Running git config %s %s", key, value)
	if b, err := r.gitCommand("config", key, value).CombinedOutput(); err != nil {
		return fmt.Errorf("git config %s %s failed: %v. output: %s", key, value, err, string(b))
	}
	return nil
}

// retryCmd will retry the command a few times with backoff. Use this for any
// commands that will be talking to GitHub, such as clones or fetches.
func retryCmd(l *logrus.Entry, dir, cmd string, arg ...string) ([]byte, error) {
	var b []byte
	var err error
	sleepyTime := time.Second
	for i := 0; i < 3; i++ {
		c := exec.Command(cmd, arg...)
		c.Dir = dir
		b, err = c.CombinedOutput()
		if err != nil {
			l.Warningf("Running %s %v returned error %v with output %s.", cmd, arg, err, string(b))
			time.Sleep(sleepyTime)
			sleepyTime *= 2
			continue
		}
		break
	}
	return b, err
}
