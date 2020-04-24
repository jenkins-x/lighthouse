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
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	lighthouseclient "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/commentpruner"
	"github.com/jenkins-x/lighthouse/pkg/config"
	git2 "github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/pluginhelp"
	"github.com/jenkins-x/lighthouse/pkg/repoowners"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

var (
	pluginHelp                 = map[string]HelpProvider{}
	genericCommentHandlers     = map[string]GenericCommentHandler{}
	issueHandlers              = map[string]IssueHandler{}
	issueCommentHandlers       = map[string]IssueCommentHandler{}
	pullRequestHandlers        = map[string]PullRequestHandler{}
	pushEventHandlers          = map[string]PushEventHandler{}
	reviewEventHandlers        = map[string]ReviewEventHandler{}
	reviewCommentEventHandlers = map[string]ReviewCommentEventHandler{}
	statusEventHandlers        = map[string]StatusEventHandler{}
)

// HelpProvider defines the function type that construct a pluginhelp.PluginHelp for enabled
// plugins. It takes into account the plugins configuration and enabled repositories.
type HelpProvider func(config *Configuration, enabledRepos []string) (*pluginhelp.PluginHelp, error)

// HelpProviders returns the map of registered plugins with their associated HelpProvider.
func HelpProviders() map[string]HelpProvider {
	return pluginHelp
}

// IssueHandler defines the function contract for a scm.Issue handler.
type IssueHandler func(Agent, scm.Issue) error

// RegisterIssueHandler registers a plugin's scm.Issue handler.
func RegisterIssueHandler(name string, fn IssueHandler, help HelpProvider) {
	pluginHelp[name] = help
	issueHandlers[name] = fn
}

// IssueCommentHandler defines the function contract for a scm.Comment handler.
type IssueCommentHandler func(Agent, scm.IssueCommentHook) error

// RegisterIssueCommentHandler registers a plugin's scm.Comment handler.
func RegisterIssueCommentHandler(name string, fn IssueCommentHandler, help HelpProvider) {
	pluginHelp[name] = help
	issueCommentHandlers[name] = fn
}

// PullRequestHandler defines the function contract for a scm.PullRequest handler.
type PullRequestHandler func(Agent, scm.PullRequestHook) error

// RegisterPullRequestHandler registers a plugin's scm.PullRequest handler.
func RegisterPullRequestHandler(name string, fn PullRequestHandler, help HelpProvider) {
	pluginHelp[name] = help
	pullRequestHandlers[name] = fn
}

// StatusEventHandler defines the function contract for a scm.Status handler.
type StatusEventHandler func(Agent, scm.Status) error

// RegisterStatusEventHandler registers a plugin's scm.Status handler.
func RegisterStatusEventHandler(name string, fn StatusEventHandler, help HelpProvider) {
	pluginHelp[name] = help
	statusEventHandlers[name] = fn
}

// PushEventHandler defines the function contract for a scm.PushHook handler.
type PushEventHandler func(Agent, scm.PushHook) error

// RegisterPushEventHandler registers a plugin's scm.PushHook handler.
func RegisterPushEventHandler(name string, fn PushEventHandler, help HelpProvider) {
	pluginHelp[name] = help
	pushEventHandlers[name] = fn
}

// ReviewEventHandler defines the function contract for a ReviewHook handler.
type ReviewEventHandler func(Agent, scm.ReviewHook) error

// RegisterReviewEventHandler registers a plugin's ReviewHook handler.
func RegisterReviewEventHandler(name string, fn ReviewEventHandler, help HelpProvider) {
	pluginHelp[name] = help
	reviewEventHandlers[name] = fn
}

// ReviewCommentEventHandler defines the function contract for a scm.PullRequestCommentHook handler.
type ReviewCommentEventHandler func(Agent, scm.PullRequestCommentHook) error

// RegisterReviewCommentEventHandler registers a plugin's scm.PullRequestCommentHook handler.
func RegisterReviewCommentEventHandler(name string, fn ReviewCommentEventHandler, help HelpProvider) {
	pluginHelp[name] = help
	reviewCommentEventHandlers[name] = fn
}

// GenericCommentHandler defines the function contract for a scm.Comment handler.
type GenericCommentHandler func(Agent, scmprovider.GenericCommentEvent) error

// RegisterGenericCommentHandler registers a plugin's scm.Comment handler.
func RegisterGenericCommentHandler(name string, fn GenericCommentHandler, help HelpProvider) {
	pluginHelp[name] = help
	genericCommentHandlers[name] = fn
}

// Agent may be used concurrently, so each entry must be thread-safe.
type Agent struct {
	ClientFactory      jxfactory.Factory
	SCMProviderClient  *scmprovider.Client
	LauncherClient     launcher.PipelineLauncher
	MetapipelineClient metapipeline.Client
	GitClient          git2.Client
	KubernetesClient   kubernetes.Interface
	LighthouseClient   lighthouseclient.LighthouseJobInterface
	ServerURL          *url.URL
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
	commentPruner *commentpruner.EventClient
}

// NewAgent bootstraps a new Agent struct from the passed dependencies.
func NewAgent(clientFactory jxfactory.Factory, configAgent *config.Agent, pluginConfigAgent *ConfigAgent, clientAgent *ClientAgent, metapipelineClient metapipeline.Client, serverURL *url.URL, logger *logrus.Entry) Agent {
	prowConfig := configAgent.Config()
	pluginConfig := pluginConfigAgent.Config()
	scmClient := scmprovider.ToClient(clientAgent.SCMProviderClient, clientAgent.BotName)
	return Agent{
		ClientFactory:      clientFactory,
		SCMProviderClient:  scmClient,
		GitClient:          clientAgent.GitClient,
		LauncherClient:     clientAgent.LauncherClient,
		MetapipelineClient: metapipelineClient,
		LighthouseClient:   clientAgent.LighthouseClient,
		ServerURL:          serverURL,

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
	a.commentPruner = commentpruner.NewEventClient(
		a.SCMProviderClient, a.Logger.WithField("client", "commentpruner"),
		org, repo, pr,
	)
}

// CommentPruner will return the commentpruner.EventClient attached to the agent or an error
// if one is not attached.
func (a *Agent) CommentPruner() (*commentpruner.EventClient, error) {
	if a.commentPruner == nil {
		return nil, errors.New("comment pruner client never initialized")
	}
	return a.commentPruner, nil
}

// ClientAgent contains the various clients that are attached to the Agent.
type ClientAgent struct {
	BotName           string
	SCMProviderClient *scm.Client

	KubernetesClient   kubernetes.Interface
	GitClient          git2.Client
	LauncherClient     launcher.PipelineLauncher
	MetapipelineClient metapipeline.Client
	LighthouseClient   lighthouseclient.LighthouseJobInterface

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
	b, err := ioutil.ReadFile(path) // #nosec
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

// Start starts polling path for plugin config. If the first attempt fails,
// then start returns the error. Future errors will halt updates but not stop.
func (pa *ConfigAgent) Start(path string) error {
	if err := pa.Load(path); err != nil {
		return err
	}
	ticker := time.Tick(1 * time.Minute)
	go func() {
		for range ticker {
			if err := pa.Load(path); err != nil {
				logrus.WithField("path", path).WithError(err).Error("Error loading plugin config.")
			}
		}
	}()
	return nil
}

// GenericCommentHandlers returns a map of plugin names to handlers for the repo.
func (pa *ConfigAgent) GenericCommentHandlers(owner, repo string) map[string]GenericCommentHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]GenericCommentHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := genericCommentHandlers[p]; ok {
			hs[p] = h
		}
	}
	return hs
}

// IssueHandlers returns a map of plugin names to handlers for the repo.
func (pa *ConfigAgent) IssueHandlers(owner, repo string) map[string]IssueHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]IssueHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := issueHandlers[p]; ok {
			hs[p] = h
		}
	}
	return hs
}

// IssueCommentHandlers returns a map of plugin names to handlers for the repo.
func (pa *ConfigAgent) IssueCommentHandlers(owner, repo string) map[string]IssueCommentHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]IssueCommentHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := issueCommentHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// PullRequestHandlers returns a map of plugin names to handlers for the repo.
func (pa *ConfigAgent) PullRequestHandlers(owner, repo string) map[string]PullRequestHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]PullRequestHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := pullRequestHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// ReviewEventHandlers returns a map of plugin names to handlers for the repo.
func (pa *ConfigAgent) ReviewEventHandlers(owner, repo string) map[string]ReviewEventHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]ReviewEventHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := reviewEventHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// ReviewCommentEventHandlers returns a map of plugin names to handlers for the repo.
func (pa *ConfigAgent) ReviewCommentEventHandlers(owner, repo string) map[string]ReviewCommentEventHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]ReviewCommentEventHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := reviewCommentEventHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// StatusEventHandlers returns a map of plugin names to handlers for the repo.
func (pa *ConfigAgent) StatusEventHandlers(owner, repo string) map[string]StatusEventHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]StatusEventHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := statusEventHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
}

// PushEventHandlers returns a map of plugin names to handlers for the repo.
func (pa *ConfigAgent) PushEventHandlers(owner, repo string) map[string]PushEventHandler {
	pa.mut.Lock()
	defer pa.mut.Unlock()

	hs := map[string]PushEventHandler{}
	for _, p := range pa.getPlugins(owner, repo) {
		if h, ok := pushEventHandlers[p]; ok {
			hs[p] = h
		}
	}

	return hs
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

	// until we have the configuration stuff setup nicely - lets add a simple way to enable plugins
	pluginNames := os.Getenv("LIGHTHOUSE_PLUGINS")
	if pluginNames != "" {
		names := strings.Split(pluginNames, ",")
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name != "" {
				found := false
				for _, p := range plugins {
					if p == name {
						found = true
						break
					}
				}
				if !found {
					plugins = append(plugins, name)
				}
			}
		}
	}
	logrus.Infof("found plugins %s\n", strings.Join(plugins, ", "))
	return plugins
}

// EventsForPlugin returns the registered events for the passed plugin.
func EventsForPlugin(name string) []string {
	var events []string
	if _, ok := issueHandlers[name]; ok {
		events = append(events, "issue")
	}
	if _, ok := issueCommentHandlers[name]; ok {
		events = append(events, "issue_comment")
	}
	if _, ok := pullRequestHandlers[name]; ok {
		events = append(events, "pull_request")
	}
	if _, ok := pushEventHandlers[name]; ok {
		events = append(events, "push")
	}
	if _, ok := reviewEventHandlers[name]; ok {
		events = append(events, "pull_request_review")
	}
	if _, ok := reviewCommentEventHandlers[name]; ok {
		events = append(events, "pull_request_review_comment")
	}
	if _, ok := statusEventHandlers[name]; ok {
		events = append(events, "status")
	}
	if _, ok := genericCommentHandlers[name]; ok {
		events = append(events, "GenericCommentEvent (any event for user text)")
	}
	return events
}
