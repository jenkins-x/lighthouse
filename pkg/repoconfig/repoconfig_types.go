package repoconfig

import (
	"github.com/jenkins-x/lighthouse/pkg/config/branchprotection"
	"github.com/jenkins-x/lighthouse/pkg/config/keeper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RepositoryConfig represents local repository configurations
type RepositoryConfig struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the RepositoryConfig from the client
	// +optional
	Spec RepositoryConfigSpec `json:"spec"`
}

// RepositoryConfigSpec defines the desired state of RepositoryConfig.
type RepositoryConfigSpec struct {
	// Presubmit zero or more presubmits
	Presubmits []Presubmit `json:"presubmits,omitempty"`

	// Postsubmit zero or more postsubmits
	Postsubmits []Postsubmit `json:"postsubmits,omitempty"`

	// Plugins the plugin names to enable for this repository
	Plugins []string `json:"plugins,omitempty"`

	// BranchProtections to configure branch protection
	BranchProtection *branchprotection.ContextPolicy `json:"branchProtection,omitempty"`

	// PluginConfig the configuration for the plugins
	PluginConfig *PluginConfig `json:"pluginConfig,omitempty"`

	// Keeper any keeper/tide related configurations
	Keeper *Keeper `json:"keeper,omitempty"`
}

// RepositoryConfigList contains a list of RepositoryConfig
type RepositoryConfigList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RepositoryConfig `json:"items"`
}

// Keeper for keeper specific queries
type Keeper struct {
	// Query the query to add for the repository
	Query *keeper.Query `json:"query,omitempty"`
}

// PluginConfig configuration for plugins
type PluginConfig struct {
	// Plugins the list of plugins
	Plugins []string `json:"plugins,omitempty"`

	// Approve approve plugin configuration
	Approve *Approve `json:"approve,omitempty"`
}
