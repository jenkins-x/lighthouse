// Package scmclients builds the SCM and git clients for each owner from one credential source.
package scmclients

import (
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/scmauth"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/pkg/errors"
)

// Clients builds clients for each owner from one credential source.
type Clients struct {
	src       scmauth.TokenSource
	serverURL string
	gitKind   string
	botName   string
	mirror    bool
	userInURL bool

	mu      sync.Mutex
	byOwner map[string]*OwnerClients
}

// An Option overrides a configured value. Empty values are ignored.
type Option func(*Clients)

// WithServerURL overrides the git server base URL.
func WithServerURL(serverURL string) Option {
	return func(c *Clients) {
		if serverURL != "" {
			c.serverURL = serverURL
		}
	}
}

// WithGitKind overrides the git provider kind.
func WithGitKind(gitKind string) Option {
	return func(c *Clients) {
		if gitKind != "" {
			c.gitKind = gitKind
		}
	}
}

// WithBotName overrides the login taken from the credential source.
func WithBotName(botName string) Option {
	return func(c *Clients) {
		if botName != "" {
			c.botName = botName
		}
	}
}

// WithMirroredClones clones through a local mirror.
func WithMirroredClones() Option {
	return func(c *Clients) {
		c.mirror = true
	}
}

// WithUserInURL puts the clone username in the clone URL rather than in a header.
func WithUserInURL() Option {
	return func(c *Clients) {
		c.userInURL = true
	}
}

// New returns clients for the configured auth mode. Share one per process, since minted
// tokens and git working directories are cached on it.
func New(cfg config.Getter, opts ...Option) (*Clients, error) {
	src, err := util.NewTokenSource(cfg)
	if err != nil {
		return nil, err
	}
	return NewWithSource(src, cfg, opts...), nil
}

// NewWithSource returns clients backed by an already-resolved credential source.
func NewWithSource(src scmauth.TokenSource, cfg config.Getter, opts ...Option) *Clients {
	c := &Clients{
		src:       src,
		serverURL: util.GetGitServer(cfg),
		gitKind:   util.GitKind(cfg),
		byOwner:   map[string]*OwnerClients{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// TokenSource returns the source these clients authenticate with.
func (c *Clients) TokenSource() scmauth.TokenSource {
	return c.src
}

// RequiresOwner reports whether credentials are scoped per owner.
func (c *Clients) RequiresOwner() bool {
	return c.src.RequiresOwner()
}

// ServerURL returns the git server base URL.
func (c *Clients) ServerURL() string {
	return c.serverURL
}

// GitKind returns the git provider kind.
func (c *Clients) GitKind() string {
	return c.gitKind
}

// BotName returns the login Lighthouse acts as.
func (c *Clients) BotName() (string, error) {
	if c.botName != "" {
		return c.botName, nil
	}
	return c.src.BotName()
}

// Unauthenticated returns a client with no credentials.
func (c *Clients) Unauthenticated() (*scm.Client, error) {
	return factory.NewClient(c.gitKind, c.serverURL, "")
}

// ForOwner returns the clients for owner, building them on first use.
func (c *Clients) ForOwner(owner string) (*OwnerClients, error) {
	if owner == "" && c.RequiresOwner() {
		return nil, errors.New("the configured credentials are scoped per owner, so an owner is required")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if oc, ok := c.byOwner[owner]; ok {
		return oc, nil
	}

	botName, err := c.BotName()
	if err != nil {
		return nil, errors.Wrap(err, "resolving bot name")
	}

	scmClient, err := factory.NewClientWithTokenSource(c.gitKind, c.serverURL,
		scmauth.ForOwner(c.src, owner), factory.SetUsername(botName))
	if err != nil {
		return nil, errors.Wrapf(err, "creating SCM client for %s", c.serverURL)
	}

	cloneUser := botName
	if c.src.UsesAppInstallationToken() {
		cloneUser = util.GitHubAppGitRemoteUsername
	}

	oc := &OwnerClients{
		Provider:  scmprovider.ToClient(scmClient, botName),
		SCM:       scmClient,
		BotName:   botName,
		CloneUser: cloneUser,
		owner:     owner,
		parent:    c,
	}
	c.byOwner[owner] = oc
	return oc, nil
}

// Clean removes the working directories and clone caches of every client built so far.
func (c *Clients) Clean() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errs *multierror.Error
	for _, oc := range c.byOwner {
		if err := oc.clean(); err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs.ErrorOrNil()
}
