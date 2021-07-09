package git

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"sync"
	"time"
)

// NewNoMirrorClientFactory creates a client factory which does not use mirroring
func NewNoMirrorClientFactory(opts ...ClientFactoryOpt) (ClientFactory, error) {
	o := ClientFactoryOpts{}
	defaultClientFactoryOpts(&o)
	for _, opt := range opts {
		opt(&o)
	}

	cacheDir, err := ioutil.TempDir(*o.CacheDirBase, "gitcache")
	if err != nil {
		return nil, err
	}
	var remotes RemoteResolverFactory
	if o.UseSSH != nil && *o.UseSSH {
		remotes = &sshRemoteResolverFactory{
			host:     o.Host,
			username: o.Username,
		}
	} else {
		remotes = &httpResolverFactory{
			scheme:   o.Scheme,
			host:     o.Host,
			username: o.Username,
			token:    o.Token,
			urlUser:  o.UseUserInURL,
		}
	}
	return &noMirrorClientFactory{
		cacheDir:     cacheDir,
		cacheDirBase: *o.CacheDirBase,
		remotes:      remotes,
		gitUser:      o.GitUser,
		censor:       o.Censor,
		masterLock:   &sync.Mutex{},
		repoLocks:    map[string]*sync.Mutex{},
		logger:       logrus.WithField("client", "git"),
	}, nil
}

type noMirrorClientFactory struct {
	remotes RemoteResolverFactory
	gitUser UserGetter
	censor  Censor
	logger  *logrus.Entry

	// cacheDir is the root under which cached clones of repos are created
	cacheDir string
	// cacheDirBase is the basedir under which create tempdirs
	cacheDirBase string
	// masterLock guards mutations to the repoLocks records
	masterLock *sync.Mutex
	// repoLocks guard mutating access to subdirectories under the cacheDir
	repoLocks map[string]*sync.Mutex
}

// bootstrapClients returns a repository client and cloner for a dir.
func (c *noMirrorClientFactory) bootstrapClients(org, repo, dir string) (cacher, cloner, RepoClient, error) {
	if dir == "" {
		workdir, err := os.Getwd()
		if err != nil {
			return nil, nil, nil, err
		}
		dir = workdir
	}
	logger := c.logger.WithFields(logrus.Fields{"org": org, "repo": repo})
	logger.WithField("dir", dir).Debug("Creating a pre-initialized client.")
	executor, err := NewCensoringExecutor(dir, c.censor, logger)
	if err != nil {
		return nil, nil, nil, err
	}
	client := &repoClient{
		publisher: publisher{
			remotes: remotes{
				publishRemote: c.remotes.PublishRemote(org, repo),
				centralRemote: c.remotes.CentralRemote(org, repo),
			},
			executor: executor,
			info:     c.gitUser,
			logger:   logger,
		},
		interactor: interactor{
			dir:      dir,
			remote:   c.remotes.CentralRemote(org, repo),
			executor: executor,
			logger:   logger,
		},
	}
	return client, client, client, nil
}

// ClientFromDir returns a repository client for a directory that's already initialized with content.
// If the directory isn't specified, the current working directory is used.
func (c *noMirrorClientFactory) ClientFromDir(org, repo, dir string) (RepoClient, error) {
	_, _, client, err := c.bootstrapClients(org, repo, dir)
	return client, err
}

// ClientFor returns a repository client for the specified repository.
// This function may take a long time if it is the first time cloning the repo.
// In that case, it must do a full git mirror clone. For large repos, this can
// take a while. Once that is done, it will do a git fetch instead of a clone,
// which will usually take at most a few seconds.
func (c *noMirrorClientFactory) ClientFor(org, repo string) (RepoClient, error) {
	start := time.Now()
	cacheDir := path.Join(c.cacheDir, org, repo)
	l := c.logger.WithFields(logrus.Fields{"org": org, "repo": repo, "dir": cacheDir})
	l.Debug("Creating a client from the cache.")

	repoDir, err := ioutil.TempDir(c.cacheDirBase, "gitrepo")
	if err != nil {
		return nil, err
	}
	_, repoClientCloner, repoClient, err := c.bootstrapClients(org, repo, repoDir)
	if err != nil {
		return nil, err
	}
	c.masterLock.Lock()
	if _, exists := c.repoLocks[cacheDir]; !exists {
		c.repoLocks[cacheDir] = &sync.Mutex{}
	}
	c.masterLock.Unlock()
	c.repoLocks[cacheDir].Lock()
	defer c.repoLocks[cacheDir].Unlock()
	if _, err := os.Stat(path.Join(cacheDir, "HEAD")); os.IsNotExist(err) {
		// we have not yet cloned this repo, we need to do a full clone
		if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil && !os.IsExist(err) {
			return nil, err
		}

		remote, err := c.remotes.CentralRemote(org, repo)()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve remote for %s/%s", org, repo)
		}
		if err := repoClientCloner.Clone(remote); err != nil {
			return nil, err
		}
	} else if err != nil {
		// something unexpected happened
		return nil, err
	} else {
		// we have cloned the repo previously, but will refresh it
		if err := repoClient.Fetch(); err != nil {
			return nil, err
		}
	}
	duration := time.Now().Sub(start)
	l.WithField("Duration", duration.String()).Info("cloned repository")
	return repoClient, nil
}

// Clean removes the caches used to generate clients
func (c *noMirrorClientFactory) Clean() error {
	return os.RemoveAll(c.cacheDir)
}
