package util

import (
	"context"
	"net/http"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/transport"
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
