package bitbucketserver

import (
	apiv1 "github.com/jenkins-x/lighthouse/pkg/apis/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Provider is an implementation of the git.Provider interface for Bitbucket Server
type Provider struct {
	URL  string
	Name string
}

// ParseWebhook encapsulates provider-specific logic for creating a Webhook CRD
func (bbs *Provider) ParseWebhook(rawWebhook []byte) *apiv1.Webhook {
	return &apiv1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "webhook-",
		},
		Spec: apiv1.WebhookSpec{
			EventType: "pull_request",
			Ref:       "abc123",
			Provider:  "Bitbucket Server",
			URL:       "bitbucket.beescloud.com",
			Payload:   "{\"foo\": \"bar\"}",
		},
	}
}
