// Package scmclients builds the clients Lighthouse uses to talk to a git server from a single set of credentials.
package scmclients

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/git"
	gitv2 "github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
)

// Provider returns the clients to use for a repository owner.
type Provider interface {
	ForOwner(owner string) (*ClientSet, error)
}

// ClientSet holds every client for one git server identity. Build it once per process and share it. Each client
// resolves its token from the Credentials on use, so nothing needs rebuilding when a token changes.
type ClientSet struct {
	GitKind   string
	ServerURL *url.URL
	BotName   string

	SCMClient         *scm.Client // TODO: remove. Use SCMProviderClient, or ToScmClient() where the raw client is needed.
	SCMProviderClient *scmprovider.Client
	GitClient         git.Client // TODO: remove. pkg/git is superseded by pkg/git/v2, use GitFactory.
	GitFactory        gitv2.ClientFactory
	FileBrowsers      *filebrowser.FileBrowsers
}

// Options override how a ClientSet is built.
type Options struct {
	// GitKind, ServerURL and BotName override the environment and Lighthouse config, typically from CLI flags. Empty means no override.
	GitKind   string
	ServerURL string
	BotName   string

	// NoMirror makes the file browsers clone directly rather than through a local mirror.
	NoMirror bool
	// UseUserInURL sets the v2 factory option of the same name
	UseUserInURL bool
}

// New builds a ClientSet for the configured git server.
func New(cfg config.Getter, o Options) (*ClientSet, error) {
	kind := cmp.Or(o.GitKind, util.GitKind(cfg))
	server := cmp.Or(o.ServerURL, util.GetGitServer(cfg))

	creds, err := credentialsFromEnv(cmp.Or(o.BotName, util.GetBotName(cfg)))
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(server)
	if err != nil {
		return nil, fmt.Errorf("parsing git server URL %s: %w", server, err)
	}

	scmClient, err := factory.NewClientWithTokenSource(kind, server, creds, factory.SetUsername(creds.BotName()))
	if err != nil {
		return nil, fmt.Errorf("creating %s client for %s: %w", kind, server, err)
	}

	token := gitToken(creds)

	gitClient, err := git.NewClient(server, kind)
	if err != nil {
		return nil, fmt.Errorf("creating git client: %w", err)
	}
	gitClient.SetCredentials(creds.CloneUser(), token)

	configure := func(opts *gitv2.ClientFactoryOpts) {
		opts.Token = token
		opts.GitUser = func() (string, string, error) {
			return creds.BotName(), "", nil
		}
		opts.Username = func() (string, error) {
			return creds.CloneUser(), nil
		}
		if u.Host != "" {
			opts.Host = u.Host
		}
		if u.Scheme != "" {
			opts.Scheme = u.Scheme
		}
		opts.UseUserInURL = o.UseUserInURL
	}
	newFactory := gitv2.NewClientFactory
	if o.NoMirror {
		newFactory = gitv2.NewNoMirrorClientFactory
	}
	gitFactory, err := newFactory(configure)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("creating git client factory for %s: %w", server, err), gitClient.Clean())
	}

	fileBrowsers, err := filebrowser.NewFileBrowsers(server, filebrowser.NewFileBrowserFromGitClient(gitFactory))
	if err != nil {
		return nil, errors.Join(fmt.Errorf("creating file browsers for %s: %w", server, err), gitClient.Clean(), gitFactory.Clean())
	}

	return &ClientSet{
		GitKind:           kind,
		ServerURL:         u,
		BotName:           creds.BotName(),
		SCMClient:         scmClient,
		SCMProviderClient: scmprovider.ToClient(scmClient, creds.BotName()),
		GitClient:         gitClient,
		GitFactory:        gitFactory,
		FileBrowsers:      fileBrowsers,
	}, nil
}

// ForOwner returns c for every owner, as a single identity serves them all.
func (c *ClientSet) ForOwner(string) (*ClientSet, error) {
	return c, nil
}

// Clean removes the git clients' local caches. The ClientSet is unusable afterwards.
func (c *ClientSet) Clean() error {
	var errs []error
	if c.GitClient != nil {
		errs = append(errs, c.GitClient.Clean())
	}
	if c.GitFactory != nil {
		errs = append(errs, c.GitFactory.Clean())
	}
	return errors.Join(errs...)
}

func gitToken(creds Credentials) func() []byte {
	return func() []byte {
		t, err := creds.Token(context.Background())
		if err != nil {
			logrus.WithError(err).Error("failed to resolve git token")
			return nil
		}
		if t == nil {
			return nil
		}
		return []byte(t.Token)
	}
}
