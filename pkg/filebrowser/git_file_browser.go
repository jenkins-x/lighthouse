package filebrowser

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type gitFileBrowser struct {
	clientFactory git.ClientFactory
	clientsLock   sync.RWMutex
	clients       map[string]*repoClientFacade
	pullTimes     map[string]int64
}

const headBranchPrefix = "HEAD branch:"

var (
	shaRegex = regexp.MustCompile("\\b[0-9a-f]{7,40}\\b") //nolint:gosimple
)

// NewFileBrowserFromGitClient creates a new file browser from an Scm client
func NewFileBrowserFromGitClient(clientFactory git.ClientFactory) Interface {
	return &gitFileBrowser{
		clientFactory: clientFactory,
		clients:       map[string]*repoClientFacade{},
		pullTimes:     map[string]int64{},
	}
}

func (f *gitFileBrowser) WithDir(owner, repo, ref string, fc FetchCache, sparseCheckoutPatterns []string, fn func(dir string) error) error {
	return f.withRepoClient(owner, repo, ref, fc, sparseCheckoutPatterns, func(repoClient git.RepoClient) error {
		dir := repoClient.Directory()
		return fn(dir)
	})
}

func (f *gitFileBrowser) GetMainAndCurrentBranchRefs(_, _, eventRef string) ([]string, error) {
	return []string{"", eventRef}, nil
}

func (f *gitFileBrowser) GetFile(owner, repo, path, ref string, fc FetchCache) (answer []byte, err error) {
	err = f.withRepoClient(owner, repo, ref, fc, []string{"/" + path}, func(repoClient git.RepoClient) error {
		f := repoPath(repoClient, path)
		exists, err := util.FileExists(f)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", f)
		}
		if !exists {
			answer = nil
			return nil
		}
		answer, err = os.ReadFile(f) // #nosec
		return err
	})
	return
}

func (f *gitFileBrowser) ListFiles(owner, repo, path, ref string, fc FetchCache) (answer []*scm.FileEntry, err error) {
	err = f.withRepoClient(owner, repo, ref, fc, []string{"/" + path + "/*"}, func(repoClient git.RepoClient) error {
		dir := repoPath(repoClient, path)
		exists, err := util.DirExists(dir)
		if err != nil {
			return errors.Wrapf(err, "failed to check dir exists %s", dir)
		}
		if !exists {
			return nil
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return errors.Wrapf(err, "failed to list files in directory %s", dir)
		}
		infos := make([]os.FileInfo, 0, len(entries))
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			infos = append(infos, info)
		}
		for _, f := range infos {
			name := f.Name()
			if name == ".git" {
				continue
			}
			t := "file"
			if f.IsDir() {
				t = "dir"
			}
			path := filepath.Join(dir, name)
			answer = append(answer, &scm.FileEntry{
				Name: name,
				Path: path,
				Type: t,
				Size: int(f.Size()),
				Sha:  ref,
				Link: path,
			})
		}
		return nil
	})
	return
}

func repoPath(repoClient git.RepoClient, path string) string {
	dir := repoClient.Directory()
	if path == "" || path == "/" {
		return dir
	}
	return filepath.Join(dir, path)
}

func (f *gitFileBrowser) withRepoClient(owner, repo, ref string, fc FetchCache, sparseCheckoutPatterns []string, fn func(repoClient git.RepoClient) error) error {
	client := f.getOrCreateClient(owner, repo)
	sparseCheckout, _ := strconv.ParseBool(os.Getenv("SPARSE_CHECKOUT"))
	var err error
	client.lock.Lock()
	defer client.lock.Unlock()
	if client.repoClient == nil {
		client.repoClient, err = f.clientFactory.ClientFor(owner, repo, sparseCheckoutPatterns)
		if err != nil {
			return errors.Wrapf(err, "failed to create repo client")
		}
	} else if sparseCheckout && len(sparseCheckoutPatterns) > 0 {
		if err := client.repoClient.SetSparseCheckoutPatterns(sparseCheckoutPatterns); err != nil {
			return errors.Wrapf(err, "failed to set sparse checkout patterns")
		}
	}
	if client.mainBranch == "" {
		var err error
		client.mainBranch, err = getMainBranch(client.repoClient.Directory())
		if err != nil {
			return errors.Wrapf(err, "failed to detect the main branch")
		}
	}
	if err == nil {
		err = client.UseRef(ref, fc)
		if err != nil {
			err = errors.Wrapf(err, "failed to switch to ref %s", ref)
		}
		if err == nil {
			err = fn(client.repoClient)
			if err != nil {
				err = errors.Wrapf(err, "failed to process repo %s/%s refref %s", owner, repo, ref)
			}
		}
	}
	return err
}

// getOrCreateClient lazily creates a repo client and lock
func (f *gitFileBrowser) getOrCreateClient(owner string, repo string) *repoClientFacade {
	fullName := scm.Join(owner, repo)
	f.clientsLock.Lock()
	client := f.clients[fullName]
	if client == nil {
		client = &repoClientFacade{
			fullName: fullName,
		}
		f.clients[fullName] = client
	}
	f.clientsLock.Unlock()
	return client
}

// repoClientFacade a repo client and a lock to create/use it
type repoClientFacade struct {
	lock       sync.RWMutex
	repoClient git.RepoClient
	fullName   string
	mainBranch string
	ref        string
}

// UseRef this method should only be used within the lock
func (c *repoClientFacade) UseRef(ref string, fc FetchCache) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		ref = c.mainBranch
	}
	// let's remove the bitbucket cloud refs prefix
	if strings.HasPrefix(ref, "refs/heads/") {
		ref = "origin/" + strings.TrimPrefix(ref, "refs/heads/")
	}

	shouldFetch := fc.ShouldFetch(c.fullName, ref)
	isSHA := IsSHA(ref)
	if shouldFetch && isSHA {
		// lets check if we've already fetched this sha
		sha, err := c.repoClient.HasSHA(ref)
		if err == nil && sha != "" {
			shouldFetch = false
			logrus.StandardLogger().WithFields(map[string]interface{}{
				"Name": c.fullName,
				"Ref":  ref,
				"File": "git_file_browser",
			}).Debug("not fetching ref as we already have it")
		}
	}

	if c.ref == ref && !shouldFetch {
		return nil
	}

	start := time.Now()

	// let's detach branch first before we go to a sha/ref
	err := c.repoClient.Checkout("--detach")
	if err != nil {
		return errors.Wrapf(err, "failed to detach repository %s", c.fullName)
	}

	if shouldFetch {
		if ref != c.mainBranch {
			err := c.repoClient.FetchRef(ref)
			if err != nil {
				return errors.Wrapf(err, "failed to fetch repository %s", c.fullName)
			}
		} else {
			// let's merge any new changes into the main branch. ref aught to always be the main branch unless it's a sha
			_, err := runCmd(c.repoClient.Directory(), "git", "fetch", "origin", "--force", c.mainBranch+":"+c.mainBranch)
			if err != nil {
				return errors.Wrapf(err, "failed to update local branch %s for repository %s", c.mainBranch, c.fullName)
			}
		}
	}
	c.ref = ref
	err = c.repoClient.Checkout(ref)
	if err != nil {
		return errors.Wrapf(err, "failed to checkout repository %s ref %s", c.fullName, ref)
	}

	duration := time.Since(start)
	logrus.StandardLogger().WithFields(map[string]interface{}{
		"Name":     c.fullName,
		"Ref":      ref,
		"File":     "git_file_browser",
		"Duration": duration.String(),
	}).Debug("fetched and checked out ref")
	return nil
}

// getMainBranch returns the main branch name such as 'master' or 'main'
func getMainBranch(dir string) (string, error) {
	remoteName := "origin"
	text, err := runCmd(dir, "git", "rev-parse", "--abbrev-ref", remoteName+"/HEAD")
	if err != nil {
		text, err = runCmd(dir, "git", "remote", "show", remoteName)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get the remote branch name for %s", remoteName)
		}
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, headBranchPrefix) {
				mainBranch := strings.TrimSpace(strings.TrimPrefix(line, headBranchPrefix))
				if mainBranch != "" {
					return mainBranch, nil
				}
			}
		}
		return "", errors.Errorf("output of git remote show %s has no prefix %s as was: %s", remoteName, headBranchPrefix, text)
	}
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, remoteName)
	return strings.TrimPrefix(text, "/"), nil
}

func runCmd(dir, cmd string, arg ...string) (string, error) {
	c := exec.Command(cmd, arg...) // #nosec
	c.Dir = dir
	b, err := c.CombinedOutput()
	text := strings.TrimSpace(string(b))
	if err != nil {
		return text, errors.Wrapf(err, "failed to run command in dir %s: %s, %v: %s", dir, cmd, arg, text)
	}
	return text, nil
}

// IsSHA returns true if the given ref is a git sha
func IsSHA(ref string) bool {
	return shaRegex.MatchString(ref)
}
