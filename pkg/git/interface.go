package git

// Provider is a generic interface that hides provider-specific details
type Provider interface {
	Name() string
	URL() string
	ParseWebhook(webhook string)
}
