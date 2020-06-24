package plugins

import "github.com/jenkins-x/lighthouse-config/pkg/plugins"

// This contains type aliases from lighthouse-config to simplify the transition here

type (
	// Configuration is an alias to the configuration type in lighthouse-config
	Configuration = plugins.Configuration

	// Approve is an alias to the type in lighthouse-config
	Approve = plugins.Approve

	// Blockade is an alias to the type in lighthouse-config
	Blockade = plugins.Blockade

	// Blunderbuss is an alias to the type in lighthouse-config
	Blunderbuss = plugins.Blunderbuss

	// Golint is an alias to the type in lighthouse-config
	Golint = plugins.Golint

	// ExternalPlugin is an alias to the type in lighthouse-config
	ExternalPlugin = plugins.ExternalPlugin

	// Owners is an alias to the type in lighthouse-config
	Owners = plugins.Owners

	// RequireSIG is an alias to the type in lighthouse-config
	RequireSIG = plugins.RequireSIG

	// SigMention is an alias to the type in lighthouse-config
	SigMention = plugins.SigMention

	// Size is an alias to the type in lighthouse-config
	Size = plugins.Size

	// Lgtm is an alias to the type in lighthouse-config
	Lgtm = plugins.Lgtm

	// Cat is an alias to the type in lighthouse-config
	Cat = plugins.Cat

	// Label is an alias to the type in lighthouse-config
	Label = plugins.Label

	// Trigger is an alias to the type in lighthouse-config
	Trigger = plugins.Trigger

	// Heart is an alias to the type in lighthouse-config
	Heart = plugins.Heart

	// Milestone is an alias to the type in lighthouse-config
	Milestone = plugins.Milestone

	// Slack is an alias to the type in lighthouse-config
	Slack = plugins.Slack

	// ConfigMapSpec is an alias to the type in lighthouse-config
	ConfigMapSpec = plugins.ConfigMapSpec

	// ConfigUpdate is an alias to the type in lighthouse-config
	ConfigUpdate = plugins.ConfigUpdater

	// MergeWarning is an alias to the type in lighthouse-config
	MergeWarning = plugins.MergeWarning

	// Welcome is an alias to the type in lighthouse-config
	Welcome = plugins.Welcome

	// CherryPickUnapproved is an alias to the type in lighthouse-config
	CherryPickUnapproved = plugins.CherryPickUnapproved

	// RequireMatchingLabel is an alias to the type in lighthouse-config
	RequireMatchingLabel = plugins.RequireMatchingLabel
)
