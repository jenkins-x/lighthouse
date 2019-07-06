package git

// Webhook provides a generic type for webhooks emitted by git providers
type Webhook struct {
	URL     string
	Payload string
}

// Provider is a generic interface that hides provider-specific details
type Provider interface {
	Name() string
	URL() string
	ParseWebhook(webhook string) Webhook
}
