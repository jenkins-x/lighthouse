package util

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/sirupsen/logrus"
)

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

// GetBotName returns the bot name from the environment
func GetBotName(cfg config.Getter) string {
	botName := os.Getenv("GIT_USER")
	actualConfig := cfg()
	if botName == "" && actualConfig != nil && actualConfig.ProviderConfig != nil {
		botName = actualConfig.ProviderConfig.BotUser
	}
	if botName == "" {
		botName = "jenkins-x-bot"
	}
	return botName
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
