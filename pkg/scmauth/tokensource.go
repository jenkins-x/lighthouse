package scmauth

import (
	"github.com/pkg/errors"
)

// Mode selects how credentials are obtained.
type Mode string

const (
	// ModeStaticToken uses one token for every owner.
	ModeStaticToken Mode = "staticToken"

	// ModeOwnerTokens looks up a pre-supplied token per owner.
	ModeOwnerTokens Mode = "ownerTokens"

	// ModeGitHubApp mints installation tokens from an app ID and private key.
	ModeGitHubApp Mode = "githubApp"
)

// ErrNoToken is returned when a source holds no credential for an owner.
var ErrNoToken = errors.New("no token available")

// TokenSource resolves a credential for an owner. Implementations must be safe for
// concurrent use and cheap enough to call on every request.
type TokenSource interface {
	// Token returns a credential valid for owner.
	Token(owner string) (string, error)

	// RequiresOwner reports whether Token needs a non-empty owner.
	RequiresOwner() bool

	// BotName returns the login this credential acts as.
	BotName() (string, error)

	// UsesAppInstallationToken reports whether Token returns a GitHub App installation token.
	UsesAppInstallationToken() bool
}

type staticTokenSource struct {
	token   string
	botName string
}

// NewStaticTokenSource returns a TokenSource that returns token for every owner.
func NewStaticTokenSource(token, botName string) TokenSource {
	return &staticTokenSource{token: token, botName: botName}
}

func (s *staticTokenSource) Token(string) (string, error) {
	if s.token == "" {
		return "", ErrNoToken
	}
	return s.token, nil
}

func (s *staticTokenSource) RequiresOwner() bool { return false }

func (s *staticTokenSource) BotName() (string, error) {
	if s.botName == "" {
		return "", errors.New("no bot name configured")
	}
	return s.botName, nil
}

func (s *staticTokenSource) UsesAppInstallationToken() bool { return false }

type ownerTokensSource struct {
	dir     *OwnerTokensDir
	botName string
}

// NewOwnerTokensSource returns a TokenSource backed by a directory of per-owner tokens.
func NewOwnerTokensSource(serverURL, dir, botName string) TokenSource {
	return &ownerTokensSource{
		dir:     NewOwnerTokensDir(serverURL, dir),
		botName: botName,
	}
}

func (s *ownerTokensSource) Token(owner string) (string, error) {
	if owner == "" {
		return "", errors.New("owner is required to resolve a per-owner token")
	}
	return s.dir.FindToken(owner)
}

func (s *ownerTokensSource) RequiresOwner() bool { return true }

func (s *ownerTokensSource) BotName() (string, error) {
	if s.botName == "" {
		return "", errors.New("no bot name configured")
	}
	return s.botName, nil
}

func (s *ownerTokensSource) UsesAppInstallationToken() bool { return false }
