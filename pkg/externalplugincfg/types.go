package externalplugincfg

const (
	// ConfigMapName name of the config map for external plugins
	ConfigMapName = "lighthouse-external-plugins"
)

// ExternalPlugin holds configuration for registering an external
// plugin in prow.
type ExternalPlugin struct {
	// Name of the plugin.
	Name string `json:"name"`
	// RequiredResources the kubernetes resources required to enable this external plugin
	RequiredResources []Resource `json:"requiredResources,omitempty"`
}

// Resource represents a kubernetes resource
type Resource struct {
	// Kind of the resource.
	Kind string `json:"kind"`
	// Name of the resource.
	Name string `json:"name"`
	// Namespace of the resource.
	Namespace string `json:"namespace"`
}

func (r *Resource) String() string {
	if r.Namespace != "" {
		return r.Kind + "/" + r.Namespace + "/" + r.Name
	}
	return r.Kind + "/" + r.Name
}
