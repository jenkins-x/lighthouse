package job

type Deployment struct {
	Base
	Reporter
	// The deployment state that trigger this pipeline
	// Can be one of: error, failure, inactive, in_progress, queued, pending, success
	// If not set all deployment state event triggers
	State string `json:"state,omitempty"`
	// Deployment for this environment trigger this pipeline
	// If not set deployments for all environments trigger
	Environment string `json:"environment,omitempty"`
}
