package scmclients

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
)

// Credentials authenticate Lighthouse against a git server, both for API calls and for git over HTTPS.
type Credentials interface {
	scm.TokenSource
	// BotName is the login Lighthouse acts as, used to recognise its own comments and labels.
	BotName() string
	// CloneUser is the username sent alongside the token for git over HTTPS.
	CloneUser() string
}

// NewTokenCredentials returns Credentials for a token that never changes, such as a personal access token.
func NewTokenCredentials(botName, token string) Credentials {
	return &tokenCredentials{botName: botName, token: token}
}

// CredentialsFromEnv reads the token from $GIT_TOKEN, falling back to the file named by $GIT_TOKEN_PATH.
func CredentialsFromEnv(botName string) (Credentials, error) {
	if token := os.Getenv("GIT_TOKEN"); token != "" {
		return NewTokenCredentials(botName, token), nil
	}
	path := os.Getenv("GIT_TOKEN_PATH")
	if path == "" {
		return nil, errors.New("no git token: neither $GIT_TOKEN nor $GIT_TOKEN_PATH is set")
	}
	c := &fileCredentials{botName: botName, path: path}
	if _, err := c.read(); err != nil {
		return nil, err
	}
	return c, nil
}

type tokenCredentials struct {
	botName string
	token   string
}

func (c *tokenCredentials) Token(context.Context) (*scm.Token, error) {
	return &scm.Token{Token: c.token}, nil
}

func (c *tokenCredentials) BotName() string   { return c.botName }
func (c *tokenCredentials) CloneUser() string { return c.botName }

// fileCredentials re-reads the token on every call so a rotated secret is picked up without a restart.
type fileCredentials struct {
	botName string
	path    string
}

func (c *fileCredentials) Token(context.Context) (*scm.Token, error) {
	token, err := c.read()
	if err != nil {
		return nil, err
	}
	return &scm.Token{Token: token}, nil
}

func (c *fileCredentials) read() (string, error) {
	b, err := os.ReadFile(c.path)
	if err != nil {
		return "", fmt.Errorf("reading git token from %s: %w", c.path, err)
	}
	return strings.TrimSpace(string(b)), nil
}

func (c *fileCredentials) BotName() string   { return c.botName }
func (c *fileCredentials) CloneUser() string { return c.botName }
