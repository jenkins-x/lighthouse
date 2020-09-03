package util

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/go-scm/scm/transport"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// AddAuthToSCMClient configures an existing go-scm client with transport and authorization using the given token,
// depending on whether the token is a GitHub App token
func AddAuthToSCMClient(client *scm.Client, token string, isGitHubApp bool) {
	if isGitHubApp {
		defaultScmTransport(client)
		tr := &transport.Custom{
			Base: http.DefaultTransport,
			Before: func(r *http.Request) {
				r.Header.Set("Authorization", "token "+token)
				r.Header.Set("Accept", "application/vnd.github.machine-man-preview+json")
			},
		}
		client.Client.Transport = tr
		return
	}
	if client.Driver.String() == "gitea" {
		client.Client = &http.Client{
			Transport: &transport.Authorization{
				Scheme:      "token",
				Credentials: token,
			},
		}
	} else if client.Driver.String() == "gitlab" || client.Driver.String() == "bitbucketcloud" {
		client.Client = &http.Client{
			Transport: &transport.PrivateToken{
				Token: token,
			},
		}
	} else {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		client.Client = oauth2.NewClient(context.Background(), ts)
	}
}

func defaultScmTransport(scmClient *scm.Client) {
	if scmClient.Client == nil {
		scmClient.Client = http.DefaultClient
	}
	if scmClient.Client.Transport == nil {
		scmClient.Client.Transport = http.DefaultTransport
	}
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

// GetSCMClient gets the Lighthouse SCM client, go-scm client, server URL, and token for the current user and server
func GetSCMClient(owner string, cfg config.Getter) (scmprovider.SCMClient, *scm.Client, string, string, error) {
	kind := GitKind(cfg)
	serverURL := GetGitServer(cfg)
	ghaSecretDir := GetGitHubAppSecretDir()

	var token string
	var err error
	if ghaSecretDir != "" && owner != "" {
		tokenFinder := NewOwnerTokensDir(serverURL, ghaSecretDir)
		token, err = tokenFinder.FindToken(owner)
		if err != nil {
			logrus.Errorf("failed to read owner token: %s", err.Error())
			return nil, nil, "", "", errors.Wrapf(err, "failed to read owner token for owner %s", owner)
		}
	} else {
		token, err = GetSCMToken(kind)
		if err != nil {
			return nil, nil, serverURL, token, err
		}
	}

	client, err := factory.NewClient(kind, serverURL, token)
	scmClient := scmprovider.ToClient(client, GetBotName(cfg))
	return scmClient, client, serverURL, token, err
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
	if GetGitHubAppSecretDir() != "" {
		ghaBotName, err := GetGitHubAppAPIUser()
		// TODO: Probably should handle error cases here better, but for now, just fall through.
		if err == nil && ghaBotName != "" {
			return ghaBotName
		}
	}
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

// GetSCMToken gets the SCM secret from the environment
func GetSCMToken(gitKind string) (string, error) {
	envName := "GIT_TOKEN"
	value := os.Getenv(envName)
	if value == "" {
		return value, fmt.Errorf("no token available for git kind %s at environment variable $%s", gitKind, envName)
	}
	return value, nil
}

// HMACToken gets the HMAC token from the environment
func HMACToken() string {
	return os.Getenv("HMAC_TOKEN")
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
	default:
		return fmt.Sprintf("%s/%s/%s/blob/%s/%v", strings.TrimSuffix(baseURL.String(), "/"), owner, repo, branch, fullPath)
	}
}
