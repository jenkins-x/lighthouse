package util

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/scmauth"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func credentialsDir() string {
	return os.Getenv(credentialsDirEnvVar)
}

// AuthMode returns the configured credential mode, or empty when unset.
func AuthMode() scmauth.Mode {
	return scmauth.Mode(os.Getenv(AuthModeEnvVar))
}

// NewTokenSource resolves the credential source for the configured mode.
func NewTokenSource(cfg config.Getter) (scmauth.TokenSource, error) {
	mode := AuthMode()
	if mode == "" {
		return nil, errors.Errorf("$%s is not set; want one of %s, %s or %s",
			AuthModeEnvVar, scmauth.ModeStaticToken, scmauth.ModeOwnerTokens, scmauth.ModeGitHubApp)
	}

	opts := scmauth.Options{
		Mode:      mode,
		ServerURL: GetGitServer(cfg),
		BotName:   configuredBotName(cfg),
		SecretDir: credentialsDir(),
	}

	if mode == scmauth.ModeStaticToken {
		token, err := GetSCMToken(GitKind(cfg))
		if err != nil {
			return nil, err
		}
		opts.StaticToken = token
	}

	src, err := scmauth.New(opts)
	if err != nil {
		return nil, err
	}
	logrus.WithField("mode", mode).Info("resolved git credential mode")
	return src, nil
}

// GetGitServer returns the git server base URL from the environment
func GetGitServer(cfg config.Getter) string {
	serverURL := os.Getenv("GIT_SERVER")

	actualConfig := cfg()
	if serverURL == "" && actualConfig != nil && actualConfig.ProviderConfig != nil {
		serverURL = actualConfig.ProviderConfig.Server
	}
	if serverURL == "" {
		serverURL = "https://github.com"
	}
	return serverURL
}

// GitKind gets the git kind from the environment
func GitKind(cfg config.Getter) string {
	kind := os.Getenv("GIT_KIND")
	actualConfig := cfg()
	if kind == "" && actualConfig != nil && actualConfig.ProviderConfig != nil {
		kind = actualConfig.ProviderConfig.Kind
	}
	if kind == "" {
		kind = "github"
	}
	return kind
}

func configuredBotName(cfg config.Getter) string {
	botName := os.Getenv("GIT_USER")
	actualConfig := cfg()
	if botName == "" && actualConfig != nil && actualConfig.ProviderConfig != nil {
		botName = actualConfig.ProviderConfig.BotUser
	}
	if botName == "" && AuthMode() != scmauth.ModeGitHubApp {
		botName = "jenkins-x-bot"
	}
	return botName
}

// GetSCMToken gets the SCM secret from the environment
func GetSCMToken(gitKind string) (string, error) {
	envName := "GIT_TOKEN"
	value := os.Getenv(envName)
	var err error
	if value == "" {
		err = fmt.Errorf("no token available for git kind %s at environment variable $%s", gitKind, envName)
	}
	// If we could not retrieve the Git token from the environment then attempt
	// to read it from the filesystem
	if err != nil {
		value, pathErr := getSCMTokenFromPath(gitKind)
		if pathErr == nil {
			return value, nil
		}
		// Construct multi error to avoid hiding issues
		multiErr := multierror.Error{
			Errors: []error{err, pathErr},
		}
		err = multiErr.ErrorOrNil()
	}
	return value, err
}

// getSCMTokenFromPath retrieves the SCM secret from the filesystem
func getSCMTokenFromPath(gitKind string) (string, error) {
	envName := "GIT_TOKEN_PATH"
	value := os.Getenv(envName)
	if value == "" {
		return value, fmt.Errorf("no token path available for git kind %s at environment variable $%s", gitKind, envName)
	}
	b, err := os.ReadFile(value)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// HMACToken gets the HMAC token from the environment or filesystem
func HMACToken() string {
	hmacToken := os.Getenv("HMAC_TOKEN")
	// For backwards compatibility we only attempt to read from the filesystem
	// if the HMAC token is not set in the environment
	if len(hmacToken) == 0 {
		// If HMAC_TOKEN_PATH is specified then attempt to read from the filesystem
		hmacTokenPath := os.Getenv("HMAC_TOKEN_PATH")
		if len(hmacTokenPath) > 0 {
			b, err := os.ReadFile(hmacTokenPath)
			if err != nil {
				logrus.Errorf("failed to read HMAC_TOKEN_PATH %s: %s", hmacTokenPath, err)
				return hmacToken
			}
			hmacToken = string(b)
		}
	}
	return hmacToken
}

// BlobURLForProvider gets the link to the blob for an individual file in a commit or branch
func BlobURLForProvider(providerType string, baseURL *url.URL, owner, repo, branch string, fullPath string) string {
	switch providerType {
	case "stash":
		u := fmt.Sprintf("%s/projects/%s/repos/%s/browse/%v", strings.TrimSuffix(baseURL.String(), "/"), strings.ToUpper(owner), repo, fullPath)
		if branch != "master" {
			u = fmt.Sprintf("%s?at=%s", u, url.QueryEscape("refs/heads/"+branch))
		}
		return u
	case "gitlab":
		return fmt.Sprintf("%s/%s/%s/-/blob/%s/%v", strings.TrimSuffix(baseURL.String(), "/"), owner, repo, branch, fullPath)
	case "gitea":
		return fmt.Sprintf("%s/%s/%s/src/branch/%s/%v", strings.TrimSuffix(baseURL.String(), "/"), owner, repo, branch, fullPath)
	default:
		return fmt.Sprintf("%s/%s/%s/blob/%s/%v", strings.TrimSuffix(baseURL.String(), "/"), owner, repo, branch, fullPath)
	}
}
