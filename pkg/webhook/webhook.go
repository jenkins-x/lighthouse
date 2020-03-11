package webhook

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/lighthouse/pkg/cmd/helper"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	"github.com/jenkins-x/lighthouse/pkg/prow/hook"
	"github.com/jenkins-x/lighthouse/pkg/prow/logrusutil"
	"github.com/jenkins-x/lighthouse/pkg/prow/metrics"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/version"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
)

const (
	// HealthPath is the URL path for the HTTP endpoint that returns health status.
	HealthPath = "/health"
	// ReadyPath URL path for the HTTP endpoint that returns ready status.
	ReadyPath = "/ready"

	// ProwConfigMapName name of the ConfgMap holding the config
	ProwConfigMapName = "config"
	// ProwPluginsConfigMapName name of the ConfigMap holding the plugins config
	ProwPluginsConfigMapName = "plugins"
	// ProwConfigFilename config file name
	ProwConfigFilename = "config.yaml"
	// ProwPluginsFilename plugins file name
	ProwPluginsFilename = "plugins.yaml"
)

// Options holds the command line arguments
type Options struct {
	BindAddress string
	Path        string
	Port        int
	JSONLog     bool

	factory          jxfactory.Factory
	namespace        string
	pluginFilename   string
	configFilename   string
	server           *hook.Server
	botName          string
	gitServerURL     string
	configMapWatcher *watcher.ConfigMapWatcher
}

// NewCmdWebhook creates the command
func NewCmdWebhook() *cobra.Command {
	options := Options{}

	cmd := &cobra.Command{
		Use:   "lighthouse",
		Short: "Runs the lighthouse webhook handler",
		Run: func(cmd *cobra.Command, args []string) {
			err := options.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().BoolVarP(&options.JSONLog, "json", "", true, "Enable JSON logging")
	cmd.Flags().IntVarP(&options.Port, "port", "", 8080, "The TCP port to listen on.")
	cmd.Flags().StringVarP(&options.BindAddress, "bind", "", "",
		"The interface address to bind to (by default, will listen on all interfaces/addresses).")
	cmd.Flags().StringVarP(&options.Path, "path", "", "/hook",
		"The path to listen on for requests to trigger a pipeline run.")
	cmd.Flags().StringVar(&options.pluginFilename, "plugin-file", "", "Path to the plugins.yaml file. If not specified it is loaded from the 'plugins' ConfigMap")
	cmd.Flags().StringVar(&options.configFilename, "config-file", "", "Path to the config.yaml file. If not specified it is loaded from the 'config' ConfigMap")
	cmd.Flags().StringVar(&options.botName, "bot-name", "", "The name of the bot user to run as. Defaults to $GIT_USER if not specified.")

	return cmd
}

// NewWebhook creates a new webhook handler
func NewWebhook(factory jxfactory.Factory, server *hook.Server) *Options {
	return &Options{
		factory: factory,
		server:  server,
	}
}

// Run will implement this command
func (o *Options) Run() error {
	if o.JSONLog {
		logrus.SetFormatter(logrusutil.CreateDefaultFormatter())
	}

	_, ns, err := o.GetFactory().CreateJXClient()
	if err != nil {
		return errors.Wrapf(err, "failed to create JX Client")
	}
	o.namespace = ns
	o.server, err = o.createHookServer()
	if err != nil {
		return errors.Wrapf(err, "failed to create Hook Server")
	}

	_, o.gitServerURL, err = o.createSCMClient()
	if err != nil {
		return errors.Wrapf(err, "failed to create ScmClient")
	}

	mux := http.NewServeMux()
	mux.Handle(HealthPath, http.HandlerFunc(o.health))
	mux.Handle(ReadyPath, http.HandlerFunc(o.ready))

	mux.Handle("/", http.HandlerFunc(o.defaultHandler))
	mux.Handle(o.Path, http.HandlerFunc(o.handleWebHookRequests))

	logrus.Infof("Lighthouse is now listening on path %s and port %d for WebHooks", o.Path, o.Port)
	return http.ListenAndServe(":"+strconv.Itoa(o.Port), mux)
}

// health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *Options) health(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *Options) ready(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Ready check")
	if o.isReady() {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (o *Options) defaultHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == o.Path || strings.HasPrefix(path, o.Path+"/") {
		o.handleWebHookRequests(w, r)
		return
	}
	path = strings.TrimPrefix(path, "/")
	if path == "" || path == "index.html" {
		o.getIndex(w, r)
		return
	}
	http.Error(w, fmt.Sprintf("unknown path %s", path), 404)
}

// getIndex returns a simple home page
func (o *Options) getIndex(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("GET index")
	message := fmt.Sprintf(`Hello from Jenkins X Lighthouse version: %s

For more information see: https://github.com/jenkins-x/lighthouse
`, version.Version)

	_, err := w.Write([]byte(message))
	if err != nil {
		logrus.Debugf("failed to write the index: %v", err)
	}
}

func (o *Options) isReady() bool {
	// TODO a better readiness check
	return true
}

// handle request for pipeline runs
func (o *Options) handleWebHookRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// liveness probe etc
		logrus.WithField("method", r.Method).Debug("invalid http method so returning index")
		o.getIndex(w, r)
		return
	}
	logrus.Debug("about to parse webhook")

	scmClient, serverURL, err := o.createSCMClient()
	if err != nil {
		logrus.Errorf("failed to create SCM scmClient: %s", err.Error())
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}

	webhook, err := scmClient.Webhooks.Parse(r, o.secretFn)
	if err != nil {
		logrus.Warnf("failed to parse webhook: %s", err.Error())

		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}
	if webhook == nil {
		logrus.Error("no webhook was parsed")

		responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: No webhook could be parsed")
		return
	}

	ghaSecretDir := util.GetGitHubAppSecretDir()

	var gitCloneUser string
	var token string
	if ghaSecretDir != "" {
		gitCloneUser = util.GitHubAppGitRemoteUsername
		tokenFinder := util.NewOwnerTokensDir(serverURL, ghaSecretDir)
		token, err = tokenFinder.FindToken(webhook.Repository().Namespace)
		if err != nil {
			logrus.Errorf("failed to read owner token: %s", err.Error())
			responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: failed to read owner token: %s", err.Error()))
			return
		}
	} else {
		gitCloneUser = o.GetBotName()
		token, err = o.createSCMToken(o.gitKind())
		if err != nil {
			logrus.Errorf("no scm token specified: %s", err.Error())
			responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: no scm token specified: %s", err.Error()))
			return
		}
	}
	kubeClient, _, _ := o.GetFactory().CreateKubeClient()
	gitClient, _ := git.NewClient(serverURL, o.gitKind())

	gitClient.SetCredentials(gitCloneUser, func() []byte {
		return []byte(token)
	})
	util.AddAuthToSCMClient(scmClient, token, ghaSecretDir != "")

	o.server.ClientAgent = &plugins.ClientAgent{
		BotName:           o.GetBotName(),
		SCMProviderClient: scmClient,
		KubernetesClient:  kubeClient,
		GitClient:         gitClient,
	}
	l, output, err := o.ProcessWebHook(logrus.WithField("Webhook", webhook.Kind()), webhook)
	if err != nil {
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: %s", err.Error()))
	}
	_, err = w.Write([]byte(output))
	if err != nil {
		l.Debugf("failed to process the webhook: %v", err)
	}
}

// ProcessWebHook process a webhook
func (o *Options) ProcessWebHook(l *logrus.Entry, webhook scm.Webhook) (*logrus.Entry, string, error) {
	repository := webhook.Repository()
	fields := map[string]interface{}{
		"Namespace": repository.Namespace,
		"Name":      repository.Name,
		"Branch":    repository.Branch,
		"Link":      repository.Link,
		"ID":        repository.ID,
		"Clone":     repository.Clone,
		"Webhook":   webhook.Kind(),
	}
	l = l.WithFields(logrus.Fields(fields))
	_, ok := webhook.(*scm.PingHook)
	if ok {
		l.Info("received ping")
		return l, fmt.Sprintf("pong from lighthouse %s", version.Version), nil
	}
	pushHook, ok := webhook.(*scm.PushHook)
	if ok {
		fields["Ref"] = pushHook.Ref
		fields["BaseRef"] = pushHook.BaseRef
		fields["Commit.Sha"] = pushHook.Commit.Sha
		fields["Commit.Link"] = pushHook.Commit.Link
		fields["Commit.Author"] = pushHook.Commit.Author
		fields["Commit.Message"] = pushHook.Commit.Message
		fields["Commit.Committer.Name"] = pushHook.Commit.Committer.Name

		l.Info("invoking Push handler")

		err := o.updatePlumberClientAndReturnError(l, o.server, pushHook.Repository())
		if err != nil {
			return l, "", err
		}

		o.server.HandlePushEvent(l, pushHook)
		return l, "processed push hook", nil
	}
	prHook, ok := webhook.(*scm.PullRequestHook)
	if ok {
		action := prHook.Action
		fields["Action"] = action.String()
		pr := prHook.PullRequest
		fields["PR.Number"] = pr.Number
		fields["PR.Ref"] = pr.Ref
		fields["PR.Sha"] = pr.Sha
		fields["PR.Title"] = pr.Title
		fields["PR.Body"] = pr.Body

		l.Info("invoking PR handler")

		err := o.updatePlumberClientAndReturnError(l, o.server, prHook.Repository())
		if err != nil {
			return l, "", err
		}

		o.server.HandlePullRequestEvent(l, prHook)
		return l, "processed PR hook", nil
	}
	branchHook, ok := webhook.(*scm.BranchHook)
	if ok {
		action := branchHook.Action
		ref := branchHook.Ref
		sender := branchHook.Sender
		fields["Action"] = action.String()
		fields["Ref.Sha"] = ref.Sha
		fields["Sender.Name"] = sender.Name

		l.Info("invoking branch handler")

		err := o.updatePlumberClientAndReturnError(l, o.server, branchHook.Repository())
		if err != nil {
			return l, "", err
		}

		o.server.HandleBranchEvent(l, branchHook)
		return l, "processed branch hook", nil
	}
	issueCommentHook, ok := webhook.(*scm.IssueCommentHook)
	if ok {
		action := issueCommentHook.Action
		issue := issueCommentHook.Issue
		comment := issueCommentHook.Comment
		sender := issueCommentHook.Sender
		fields["Action"] = action.String()
		fields["Issue.Number"] = issue.Number
		fields["Issue.Title"] = issue.Title
		fields["Issue.Body"] = issue.Body
		fields["Comment.Body"] = comment.Body
		fields["Sender.Body"] = sender.Name
		fields["Sender.Login"] = sender.Login
		fields["Kind"] = "IssueCommentHook"

		l.Info("invoking Issue Comment handler")

		err := o.updatePlumberClientAndReturnError(l, o.server, issueCommentHook.Repository())
		if err != nil {
			return l, "", err
		}
		o.server.HandleIssueCommentEvent(l, *issueCommentHook)
		return l, "processed issue comment hook", nil
	}
	prCommentHook, ok := webhook.(*scm.PullRequestCommentHook)
	if ok {
		action := prCommentHook.Action
		fields["Action"] = action.String()
		pr := prCommentHook.PullRequest
		fields["PR.Number"] = pr.Number
		fields["PR.Ref"] = pr.Ref
		fields["PR.Sha"] = pr.Sha
		fields["PR.Title"] = pr.Title
		fields["PR.Body"] = pr.Body
		comment := prCommentHook.Comment
		fields["Comment.Body"] = comment.Body
		author := comment.Author
		fields["Author.Name"] = author.Name
		fields["Author.Login"] = author.Login
		fields["Author.Avatar"] = author.Avatar

		l.Info("invoking PR Comment handler")

		l.Info("invoking Issue Comment handler")

		err := o.updatePlumberClientAndReturnError(l, o.server, prCommentHook.Repository())
		if err != nil {
			return l, "", err
		}
		o.server.HandlePullRequestCommentEvent(l, *prCommentHook)
		return l, "processed PR comment hook", nil
	}
	l.Debugf("unknown kind %s webhook %#v", webhook.Kind(), webhook)
	return l, fmt.Sprintf("unknown hook %s", webhook.Kind()), nil
}

// GetFactory lazily creates a Factory if its not already created
func (o *Options) GetFactory() jxfactory.Factory {
	if o.factory == nil {
		o.factory = jxfactory.NewFactory()
	}
	return o.factory
}

func (o *Options) secretFn(webhook scm.Webhook) (string, error) {
	return os.Getenv("HMAC_TOKEN"), nil
}

func (o *Options) createSCMClient() (*scm.Client, string, error) {
	kind := o.gitKind()
	serverURL := os.Getenv("GIT_SERVER")

	client, err := factory.NewClient(kind, serverURL, "")
	return client, serverURL, err
}

func (o *Options) gitKind() string {
	kind := os.Getenv("GIT_KIND")
	if kind == "" {
		kind = "github"
	}
	return kind
}

// GetBotName returns the bot name
func (o *Options) GetBotName() string {
	if util.GetGitHubAppSecretDir() != "" {
		ghaBotName, err := util.GetGitHubAppAPIUser()
		// TODO: Probably should handle error cases here better, but for now, just fall through.
		if err == nil && ghaBotName != "" {
			return ghaBotName
		}
	}
	o.botName = os.Getenv("GIT_USER")
	if o.botName == "" {
		o.botName = "jenkins-x-bot"
	}
	return o.botName
}

func (o *Options) createSCMToken(gitKind string) (string, error) {
	envName := "GIT_TOKEN"
	value := os.Getenv(envName)
	if value == "" {
		return value, fmt.Errorf("No token available for git kind %s at environment variable $%s", gitKind, envName)
	}
	return value, nil
}

func (o *Options) createHookServer() (*hook.Server, error) {
	configAgent := &config.Agent{}
	pluginAgent := &plugins.ConfigAgent{}

	onConfigYamlChange := func(text string) {
		if text != "" {
			config, err := config.LoadYAMLConfig([]byte(text))
			if err != nil {
				logrus.WithError(err).Error("Error processing the prow Config YAML")
			} else {
				logrus.Info("updating the prow core configuration")
				configAgent.Set(config)
			}
		}
	}

	onPluginsYamlChange := func(text string) {
		if text != "" {
			config, err := pluginAgent.LoadYAMLConfig([]byte(text))
			if err != nil {
				logrus.WithError(err).Error("Error processing the prow Plugins YAML")
			} else {
				logrus.Info("updating the prow plugins configuration")
				pluginAgent.Set(config)
			}
		}
	}

	clientFactory := o.GetFactory()
	kubeClient, _, err := clientFactory.CreateKubeClient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Kube client")
	}

	callbacks := []watcher.ConfigMapCallback{
		&watcher.ConfigMapEntryCallback{
			Name:     ProwConfigMapName,
			Key:      ProwConfigFilename,
			Callback: onConfigYamlChange,
		},
		&watcher.ConfigMapEntryCallback{
			Name:     ProwPluginsConfigMapName,
			Key:      ProwPluginsFilename,
			Callback: onPluginsYamlChange,
		},
	}
	o.configMapWatcher, err = watcher.NewConfigMapWatcher(kubeClient, o.namespace, callbacks)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ConfigMap watcher")
	}

	gitClient, err := git.NewClient(o.gitServerURL, o.gitKind())
	if err != nil {
		logrus.WithError(err).Fatal("Error getting git client.")
	}
	defer func() {
		err := gitClient.Clean()
		if err != nil {
			logrus.WithError(err).Fatal("Error cleaning the git client.")
		}
	}()

	promMetrics := hook.NewMetrics()

	// Push metrics to the configured prometheus pushgateway endpoint.
	agentConfig := configAgent.Config()
	if agentConfig != nil {
		pushGateway := agentConfig.PushGateway
		if pushGateway.Endpoint != "" {
			logrus.WithField("gateway", pushGateway.Endpoint).Infof("using push gateway")
			go metrics.ExposeMetrics("hook", pushGateway)
		} else {
			logrus.Warn("not pushing metrics as there is no push_gateway defined in the config.yaml")
		}
	} else {
		logrus.Warn("no configAgent configuration")
	}

	metapipelineClient, err := plumber.NewMetaPipelineClient(clientFactory)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create metapipeline client")
	}

	server := &hook.Server{
		ClientFactory:      clientFactory,
		ConfigAgent:        configAgent,
		Plugins:            pluginAgent,
		Metrics:            promMetrics,
		MetapipelineClient: metapipelineClient,
		//TokenGenerator: secretAgent.GetTokenGenerator(o.webhookSecretFile),
	}
	return server, nil
}

func (o *Options) updatePlumberClientAndReturnError(l *logrus.Entry, server *hook.Server, repository scm.Repository) error {
	jxClient, _, err := o.GetFactory().CreateJXClient()
	if err != nil {
		err = errors.Wrapf(err, "failed to create JX client")
		l.Errorf("%s", err.Error())
		return err
	}
	plumberClient, err := plumber.NewPlumber(jxClient, o.namespace)
	if err != nil {
		err = errors.Wrapf(err, "failed to create Plumber client")
		l.Errorf("%s", err.Error())
		return err
	}
	server.ClientAgent.PlumberClient = plumberClient
	return nil
}

func responseHTTPError(w http.ResponseWriter, statusCode int, response string) {
	logrus.WithFields(logrus.Fields{
		"response":    response,
		"status-code": statusCode,
	}).Info(response)
	http.Error(w, response, statusCode)
}
