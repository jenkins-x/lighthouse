/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugins

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/jenkins-x/go-scm/scm"
	lighthouseclient "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/commentpruner"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/repoowners"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

var (
	plugins = map[string]Plugin{}
)

// RegisterPlugin registers a plugin.
func RegisterPlugin(name string, plugin Plugin) {
	plugins[name] = plugin
}

// HelpProvider defines the function type that construct a pluginhelp.PluginHelp for enabled
// plugins. It takes into account the plugins configuration and enabled repositories.
type HelpProvider func(config *Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error)

// ConfigHelpProvider defines the function type that constructs help about a plugin configuration.
type ConfigHelpProvider func(config *Configuration, enabledRepos []string) (map[string]string, error)

// IssueHandler defines the function contract for a scm.Issue handler.
type IssueHandler func(Agent, scm.Issue) error

// PullRequestHandler defines the function contract for a scm.PullRequest handler.
type PullRequestHandler func(Agent, scm.PullRequestHook) error

// StatusEventHandler defines the function contract for a scm.Status handler.
type StatusEventHandler func(Agent, scm.Status) error

type DeploymentStatusHandler func(Agent, scm.DeploymentStatusHook) error

// PushEventHandler defines the function contract for a scm.PushHook handler.
type PushEventHandler func(Agent, scm.PushHook) error

// ReviewEventHandler defines the function contract for a ReviewHook handler.
type ReviewEventHandler func(Agent, scm.ReviewHook) error

// GenericCommentHandler defines the function contract for a scm.Comment handler.
type GenericCommentHandler func(Agent, scmprovider.GenericCommentEvent) error

// CommandEventHandler defines the function contract for a command handler.
type CommandEventHandler func(CommandMatch, Agent, scmprovider.GenericCommentEvent) error

// HelpProviders returns the map of registered plugins with their associated HelpProvider.
func HelpProviders() map[string]HelpProvider {
	pluginHelp := make(map[string]HelpProvider)
	for k, v := range plugins {
		pluginHelp[k] = func(config *Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error) {
			return v.GetHelp(config, enabledRepos)
		}
	}
	return pluginHelp
}

// Agent may be used concurrently, so each entry must be thread-safe.
type Agent struct {
	SCMProviderClient *scmprovider.Client
	LauncherClient    launcher.PipelineLauncher
	GitClient         git.Client
	KubernetesClient  kubernetes.Interface
	LighthouseClient  lighthouseclient.LighthouseJobInterface
	ServerURL         *url.URL
	/*
		SlackClient      *slack.Client
	*/

	OwnersClient *repoowners.Client

	// Config provides information about the jobs
	// that we know how to run for repos.
	Config *config.Config
	// PluginConfig provides plugin-specific options
	PluginConfig *Configuration

	Logger *logrus.Entry

	// may be nil if not initialized
	Commentpruner *commentpruner.EventClient
}

// NewAgent bootstraps a new Agent struct from the passed dependencies.
func NewAgent(configAgent *config.Agent, pluginConfigAgent *ConfigAgent, clientAgent *ClientAgent, serverURL *url.URL, logger *logrus.Entry) Agent {
	prowConfig := configAgent.Config()
	pluginConfig := pluginConfigAgent.Config()
	scmClient := scmprovider.ToClient(clientAgent.SCMProviderClient, clientAgent.BotName)
	return Agent{
		SCMProviderClient: scmClient,
		GitClient:         clientAgent.GitClient,
		KubernetesClient:  clientAgent.KubernetesClient,
		LauncherClient:    clientAgent.LauncherClient,
		LighthouseClient:  clientAgent.LighthouseClient,
		ServerURL:         serverURL,

		/*
			SlackClient:   clientAgent.SlackClient,
		*/
		OwnersClient: repoowners.NewClient(
			clientAgent.GitClient, scmClient,
			prowConfig, pluginConfig.MDYAMLEnabled,
			pluginConfig.SkipCollaborators,
		),
		Config:       prowConfig,
		PluginConfig: pluginConfig,
		Logger:       logger,
	}
}

// InitializeCommentPruner attaches a commentpruner.EventClient to the agent to handle
// pruning comments.
func (a *Agent) InitializeCommentPruner(org, repo string, pr int) {
	a.Commentpruner = commentpruner.NewEventClient(
		a.SCMProviderClient, a.Logger.WithField("client", "commentpruner"),
		org, repo, pr,
	)
}

// CommentPruner will return the commentpruner.EventClient attached to the agent or an error
// if one is not attached.
func (a *Agent) CommentPruner() (*commentpruner.EventClient, error) {
	if a.Commentpruner == nil {
		return nil, errors.New("comment pruner client never initialized")
	}
	return a.Commentpruner, nil
}

// ClientAgent contains the various clients that are attached to the Agent.
type ClientAgent struct {
	BotName           string
	SCMProviderClient *scm.Client

	KubernetesClient kubernetes.Interface
	GitClient        git.Client
	LauncherClient   launcher.PipelineLauncher
	LighthouseClient lighthouseclient.LighthouseJobInterface

	/*	SlackClient      *slack.Client
	 */
}

// ConfigAgent contains the agent mutex and the Agent configuration.
type ConfigAgent struct {
	mut           sync.Mutex
	configuration *Configuration
}

// Load attempts to load config from the path. It returns an error if either
// the file can't be read or the configuration is invalid.
func (pa *ConfigAgent) Load(path string) error {
	b, err := os.ReadFile(path) // #nosec
	if err != nil {
		return err
	}
	np := &Configuration{}
	if err := yaml.Unmarshal(b, np); err != nil {
		return err
	}
	if err := np.Validate(); err != nil {
		return err
	}
	presentPlugins := make(map[string]interface{}, len(plugins))
	for k, v := range plugins {
		cp := v
		presentPlugins[k] = &cp
	}
	if err := np.ValidatePluginsArePresent(presentPlugins); err != nil {
		return err
	}

	pa.Set(np)
	return nil
}

// LoadYAMLConfig loads the configuration from the given data
func (pa *ConfigAgent) LoadYAMLConfig(data []byte) (*Configuration, error) {
	c := &Configuration{}
	if err := yaml.Unmarshal(data, c); err != nil {
		return c, err
	}
	if err := c.Validate(); err != nil {
		return c, err
	}
	presentPlugins := make(map[string]interface{}, len(plugins))
	for k, v := range plugins {
		cp := v
		presentPlugins[k] = &cp
	}
	if err := c.ValidatePluginsArePresent(presentPlugins); err != nil {
		return c, err
	}
	return c, nil
}

// Config returns the agent current Configuration.
func (pa *ConfigAgent) Config() *Configuration {
	pa.mut.Lock()
	defer pa.mut.Unlock()
	return pa.configuration
}

// Set attempts to set the plugins that are enabled on repos. Plugins are listed
// as a map from repositories to the list of plugins that are enabled on them.
// Specifying simply an org name will also work, and will enable the plugin on
// all repos in the org.
func (pa *ConfigAgent) Set(pc *Configuration) {
	pa.mut.Lock()
	defer pa.mut.Unlock()
	pa.configuration = pc
}

// getPlugins returns a list of plugins that are enabled on a given (org, repository).
func (pa *ConfigAgent) getPlugins(owner, repo string) []string {
	var plugins []string

	// on bitbucket server the owner can be the ProjectKey which is upper case - so lets also check for the case
	// of a lower case project key matching projects
	owners := []string{owner}
	lowerOwner := strings.ToLower(owner)
	if lowerOwner != owner {
		owners = append(owners, lowerOwner)
	}
	for _, o := range owners {
		fullName := fmt.Sprintf("%s/%s", o, repo)
		plugins = append(plugins, pa.configuration.Plugins[o]...)
		plugins = append(plugins, pa.configuration.Plugins[fullName]...)
	}
	logrus.Infof("found plugins %s\n", strings.Join(plugins, ", "))
	return plugins
}

// GetPlugins returns a map of plugin names to plugins for the repo.
func (pa *ConfigAgent) GetPlugins(owner, repo, provider string) map[string]Plugin {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]Plugin{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := plugins[p]; ok && !plugins[p].IsProviderExcluded(provider) {
			hs[p] = h
		}
	}
	return hs
}
