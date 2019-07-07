package bitbucketserver

import (
	"github.com/jenkins-x/lighthouse/pkg/git"
)

// BitbucketServer is an implementation of the git.Provider interface
type BitbucketServer struct {
	url  string
	name string
}

// URL returns the provider's URL
func (bbs *BitbucketServer) URL() string {
	return bbs.url
}

// Name returns the provider's name
func (bbs *BitbucketServer) Name() string {
	return bbs.name
}

// ParseWebhook encapsulates provider-specific logic for creating a generic Webhook structure
func (bbs *BitbucketServer) ParseWebhook(webhook string) git.Webhook {
	return git.Webhook{}
}
