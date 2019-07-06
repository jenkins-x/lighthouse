package bitbucketserver

import (
	apiv1 "github.com/jenkins-x/lighthouse/pkg/apis/jenkins.io/v1"
)

// BitbucketServer is an implementation of the git.Provider interface
type Provider struct {
	URL  string
	Name string
}

// ParseWebhook encapsulates provider-specific logic for creating a Webhook CRD
func (bbs *Provider) ParseWebhook(rawWebhook []byte) *apiv1.Webhook {
	return &apiv1.Webhook{
		Spec: apiv1.WebhookSpec{
			EventType: "pull_request",
			Ref:       "abc123",
			Provider:  "Bitbucket Server",
			URL:       "bitbucket.beescloud.com",
			Payload:   "{\"foo\": \"bar\"}",
		},
	}
}
