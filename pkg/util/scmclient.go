package util

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/go-scm/scm/transport"
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
	if client.Driver.String() == "gitlab" || client.Driver.String() == "bitbucketcloud" {
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
func GetGitServer() string {
	serverURL := os.Getenv("GIT_SERVER")

	if serverURL == "" {
		serverURL = "https://github.com"
	}
	return serverURL
}

// GetSCMClient gets the Lighthouse SCM client, go-scm client, server URL, and token for the current user and server
func GetSCMClient(owner string) (scmprovider.SCMClient, *scm.Client, string, string, error) {
	kind := GitKind()
	serverURL := GetGitServer()
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
	scmClient := scmprovider.ToClient(client, GetBotName())
	return scmClient, client, serverURL, token, err
}

// GitKind gets the git kind from the environment
func GitKind() string {
	kind := os.Getenv("GIT_KIND")
	if kind == "" {
		kind = "github"
	}
	return kind
}

// GetBotName returns the bot name from the environment
func GetBotName() string {
	if GetGitHubAppSecretDir() != "" {
		ghaBotName, err := GetGitHubAppAPIUser()
		// TODO: Probably should handle error cases here better, but for now, just fall through.
		if err == nil && ghaBotName != "" {
			return ghaBotName
		}
	}
	botName := os.Getenv("GIT_USER")
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
