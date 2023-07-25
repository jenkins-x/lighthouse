package plugins

import (
	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Plugin defines a plugin and its handlers
type Plugin struct {
	Description             string
	ExcludedProviders       sets.String
	ConfigHelpProvider      ConfigHelpProvider
	IssueHandler            IssueHandler
	PullRequestHandler      PullRequestHandler
	PushEventHandler        PushEventHandler
	ReviewEventHandler      ReviewEventHandler
	StatusEventHandler      StatusEventHandler
	DeploymentStatusHandler DeploymentStatusHandler
	GenericCommentHandler   GenericCommentHandler
	Commands                []Command
}

// InvokeCommandHandler calls InvokeHandler on all commands
func (plugin Plugin) InvokeCommandHandler(ce *scmprovider.GenericCommentEvent, handler func(CommandEventHandler, *scmprovider.GenericCommentEvent, CommandMatch) error) error {
	for _, cmd := range plugin.Commands {
		if err := cmd.InvokeCommandHandler(ce, handler); err != nil {
			return err
		}
	}
	return nil
}

// GetHelp returns plugin help
func (plugin Plugin) GetHelp(config *Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
	var err error
	h := &pluginhelp.PluginHelp{
		Description:       plugin.Description,
		Events:            plugin.getEvents(),
		ExcludedProviders: plugin.ExcludedProviders.List(),
	}
	if plugin.ConfigHelpProvider != nil {
		h.Config, err = plugin.ConfigHelpProvider(config, enabledRepos)
	}
	for _, c := range plugin.Commands {
		h.AddCommand(c.GetHelp())
	}
	return h, err
}

// IsProviderExcluded returns true if the given provider is excluded, false otherwise
func (plugin Plugin) IsProviderExcluded(provider string) bool {
	return plugin.ExcludedProviders.Has(provider)
}

func (plugin Plugin) getEvents() []string {
	var events []string
	if plugin.IssueHandler != nil {
		events = append(events, "issue")
	}
	if plugin.PullRequestHandler != nil {
		events = append(events, "pull_request")
	}
	if plugin.PushEventHandler != nil {
		events = append(events, "push")
	}
	if plugin.ReviewEventHandler != nil {
		events = append(events, "pull_request_review")
	}
	if plugin.StatusEventHandler != nil {
		events = append(events, "status")
	}
	if plugin.GenericCommentHandler != nil {
		events = append(events, "GenericCommentEvent (any event for user text)")
	}
	return events
}
