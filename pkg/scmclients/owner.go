package scmclients

import (
	"net/url"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/git"
	gitv2 "github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// OwnerClients holds the clients for one owner. Git clients are built on first use, as
// each one costs a working directory.
type OwnerClients struct {
	Provider  *scmprovider.Client
	SCM       *scm.Client
	BotName   string
	CloneUser string

	owner  string
	parent *Clients

	mu         sync.Mutex
	git        git.Client
	gitFactory gitv2.ClientFactory
	browser    filebrowser.Interface
	browsers   *filebrowser.FileBrowsers
}

// Token returns a callback resolving a credential for this owner on every call.
func (o *OwnerClients) Token() func() []byte {
	return func() []byte {
		token, err := o.parent.src.Token(o.owner)
		if err != nil {
			logrus.WithError(err).WithField("owner", o.owner).Error("failed to resolve git token")
			return nil
		}
		return []byte(token)
	}
}

// CheckCredential reports whether a credential can be resolved for this owner.
func (o *OwnerClients) CheckCredential() error {
	_, err := o.parent.src.Token(o.owner)
	return errors.Wrapf(err, "resolving credential for owner %q", o.owner)
}

// Git returns the git client for this owner, creating it on first use.
func (o *OwnerClients) Git() (git.Client, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.git != nil {
		return o.git, nil
	}
	client, err := git.NewClient(o.parent.serverURL, o.parent.gitKind)
	if err != nil {
		return nil, errors.Wrap(err, "creating git client")
	}
	client.SetCredentials(o.CloneUser, o.Token())
	o.git = client
	return client, nil
}

// Browser returns the file browser for this owner, creating it on first use.
func (o *OwnerClients) Browser() (filebrowser.Interface, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.fileBrowser()
}

// Browsers returns the file browsers for this owner, creating them on first use.
func (o *OwnerClients) Browsers() (*filebrowser.FileBrowsers, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.browsers != nil {
		return o.browsers, nil
	}
	fb, err := o.fileBrowser()
	if err != nil {
		return nil, err
	}
	browsers, err := filebrowser.NewFileBrowsers(o.parent.serverURL, fb)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create git filebrowser %s", o.parent.serverURL)
	}
	o.browsers = browsers
	return browsers, nil
}

// fileBrowser must be called with the mutex held.
func (o *OwnerClients) fileBrowser() (filebrowser.Interface, error) {
	if o.browser != nil {
		return o.browser, nil
	}

	serverURL := o.parent.serverURL
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", serverURL)
	}

	tokenFn := o.Token()
	configureOpts := func(opts *gitv2.ClientFactoryOpts) {
		opts.Token = tokenFn
		opts.GitUser = func() (name, email string, err error) {
			name = o.CloneUser
			return
		}
		opts.Username = func() (login string, err error) {
			login = o.CloneUser
			return
		}
		if u.Host != "" {
			opts.Host = u.Host
		}
		if u.Scheme != "" {
			opts.Scheme = u.Scheme
		}
		opts.UseUserInURL = o.parent.userInURL
	}

	newFactory := gitv2.NewNoMirrorClientFactory
	if o.parent.mirror {
		newFactory = gitv2.NewClientFactory
	}
	gitFactory, err := newFactory(configureOpts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create git client factory for server %s", serverURL)
	}

	o.gitFactory = gitFactory
	o.browser = filebrowser.NewFileBrowserFromGitClient(gitFactory)
	return o.browser, nil
}

func (o *OwnerClients) clean() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	var errs *multierror.Error
	if o.git != nil {
		if err := o.git.Clean(); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	if o.gitFactory != nil {
		if err := o.gitFactory.Clean(); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs.ErrorOrNil()
}
