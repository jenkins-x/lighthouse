package bitbucketserver

import (
	"encoding/json"
	"strings"

	apiv1 "github.com/foghornci/foghorn/pkg/apis/foghornci.io/v1"
)

// Provider is an implementation of the git.Provider interface for Bitbucket Server
type Provider struct {
	URL  string
	Name string
}

// ParseWebhook encapsulates provider-specific logic for creating a Webhook CRD
func (bbs *Provider) ParseWebhook(rawWebhook []byte) *apiv1.Webhook {
	var parsedWebhook webhook
	json.Unmarshal(rawWebhook, &parsedWebhook)
	if strings.HasPrefix(parsedWebhook.EventKey, "repo") {
		return processRepoWebhook(parsedWebhook, rawWebhook)
	} else if strings.HasPrefix(parsedWebhook.EventKey, "pr") {
		return processPRWebhook(parsedWebhook, rawWebhook)
	}
	return nil
}
