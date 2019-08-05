package webhook

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/jx/pkg/jxfactory"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/cmd/helper"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	"github.com/jenkins-x/lighthouse/pkg/prow/hook"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
	"github.com/jenkins-x/lighthouse/pkg/version"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/metrics"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// HealthPath is the URL path for the HTTP endpoint that returns health status.
	HealthPath = "/health"
	// ReadyPath URL path for the HTTP endpoint that returns ready status.
	ReadyPath = "/ready"

	ProwConfigMapName        = "config"
	ProwPluginsConfigMapName = "plugins"
	ProwConfigFilename       = "config.yaml"
	ProwPluginsFilename      = "plugins.yaml"
)

// WebhookOptions holds the command line arguments
type WebhookOptions struct {
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
	options := WebhookOptions{}

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

// Run will implement this command
func (o *WebhookOptions) Run() error {
	if o.JSONLog {
		logrus.SetFormatter(&logrus.JSONFormatter{})
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

	_, o.gitServerURL, _, err = o.createSCMClient()
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
func (o *WebhookOptions) health(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Health check")
	w.WriteHeader(http.StatusNoContent)
}

// ready returns either HTTP 204 if the service is ready to serve requests, otherwise HTTP 503.
func (o *WebhookOptions) ready(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Ready check")
	if o.isReady() {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

func (o *WebhookOptions) defaultHandler(w http.ResponseWriter, r *http.Request) {
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
func (o *WebhookOptions) getIndex(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("GET index")
	message := fmt.Sprintf(`Hello from Jenkins X Lighthouse version: %s

For more information see: https://github.com/jenkins-x/lighthouse
`, version.Version)

	w.Write([]byte(message))
}

func (o *WebhookOptions) isReady() bool {
	// TODO a better readiness check
	return true
}

// handle request for pipeline runs
func (o *WebhookOptions) handleWebHookRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// liveness probe etc
		logrus.WithField("method", r.Method).Debug("invalid http method so returning index")
		o.getIndex(w, r)
		return
	}
	logrus.Infof("about to parse webhook")

	scmClient, serverURL, token, err := o.createSCMClient()
	if err != nil {
		logrus.Errorf("failed to create SCM scmClient: %s", err.Error())
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}

	webhook, err := scmClient.Webhooks.Parse(r, o.secretFn)
	if err != nil {
		logrus.Errorf("failed to parse webhook: %s", err.Error())

		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}
	if webhook == nil {
		logrus.Error("no webhook was parsed")

		responseHTTPError(w, http.StatusInternalServerError, "500 Internal Server Error: No webhook could be parsed")
		return
	}

	repository := webhook.Repository()
	fields := map[string]interface{}{
		"Namespace": repository.Namespace,
		"Name":      repository.Name,
		"Branch":    repository.Branch,
		"Link":      repository.Link,
		"ID":        repository.ID,
		"Clone":     repository.Clone,
		"CloneSSH":  repository.CloneSSH,
	}

	kubeClient, _, _ := o.GetFactory().CreateKubeClient()
	gitClient, _ := git.NewClient(serverURL, o.gitKind())

	user := o.GetBotName()
	gitClient.SetCredentials(user, func() []byte {
		return []byte(token)
	})

	server := *o.server

	server.ClientAgent = &plugins.ClientAgent{
		BotName:          o.GetBotName(),
		GitHubClient:     scmClient,
		KubernetesClient: kubeClient,
		GitClient:        gitClient,
	}

	pushHook, ok := webhook.(*scm.PushHook)
	l := logrus.WithFields(logrus.Fields(fields))
	if ok {
		fields["Ref"] = pushHook.Ref
		fields["BaseRef"] = pushHook.BaseRef
		fields["Commit.Sha"] = pushHook.Commit.Sha
		fields["Commit.Link"] = pushHook.Commit.Link
		fields["Commit.Author"] = pushHook.Commit.Author
		fields["Commit.Message"] = pushHook.Commit.Message
		fields["Commit.Committer.Name"] = pushHook.Commit.Committer.Name

		l.Info("invoking Push handler")

		err := o.updatePlumberClientAndReturnError(l, server, pushHook.Repository(), w)
		if err != nil {
			return
		}

		server.HandlePushEvent(l, pushHook)
		w.Write([]byte("processed push hook"))
		return
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

		err := o.updatePlumberClientAndReturnError(l, server, prHook.Repository(), w)
		if err != nil {
			return
		}

		server.HandlePullRequestEvent(l, prHook)
		w.Write([]byte("processed PR hook"))
		return
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

		err := o.updatePlumberClientAndReturnError(l, server, branchHook.Repository(), w)
		if err != nil {
			return
		}

		server.HandleBranchEvent(l, branchHook)
		w.Write([]byte("processed Branch hook"))
		return
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

		err := o.updatePlumberClientAndReturnError(l, server, issueCommentHook.Repository(), w)
		if err != nil {
			return
		}
		server.HandleIssueCommentEvent(l, *issueCommentHook)
		w.Write([]byte("processed issue comment hook"))
		return
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

		err := o.updatePlumberClientAndReturnError(l, server, prCommentHook.Repository(), w)
		if err != nil {
			return
		}
		server.HandlePullRequestCommentEvent(l, *prCommentHook)

		w.Write([]byte("processed PR comment hook"))
		return
	}

	l.Infof("unknown webhook %#v", webhook)
	w.Write([]byte("ignored unknown hook"))
}

func (o *WebhookOptions) missingSourceRepository(hook scm.Webhook, w http.ResponseWriter) {
	repoText := repositoryToString(hook.Repository())
	logrus.Errorf("cannot trigger a pipeline on %s as there is no SourceRepository", repoText)
	responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: No source repository for %s", repoText))
}

// GetFactory lazily creates a Factory if its not already created
func (o *WebhookOptions) GetFactory() jxfactory.Factory {
	if o.factory == nil {
		o.factory = jxfactory.NewFactory()
	}
	return o.factory
}

func repositoryToString(repo scm.Repository) string {
	return fmt.Sprintf("%s/%s branch %s", repo.Namespace, repo.Name, repo.Branch)
}

func (o *WebhookOptions) secretFn(webhook scm.Webhook) (string, error) {
	return os.Getenv("HMAC_TOKEN"), nil
}

func (o *WebhookOptions) createSCMClient() (*scm.Client, string, string, error) {
	kind := o.gitKind()
	serverURL := os.Getenv("GIT_SERVER")

	token, err := o.createSCMToken(kind)
	if err != nil {
		return nil, serverURL, token, err
	}
	client, err := factory.NewClient(kind, serverURL, token)
	return client, serverURL, token, err
}

func (o *WebhookOptions) gitKind() string {
	kind := os.Getenv("GIT_KIND")
	if kind == "" {
		kind = "github"
	}
	return kind
}

func (o *WebhookOptions) GetBotName() string {
	o.botName = os.Getenv("GIT_USER")
	if o.botName == "" {
		o.botName = "jenkins-x-bot"
	}
	return o.botName
}

func (o *WebhookOptions) createSCMToken(gitKind string) (string, error) {
	envName := "GIT_TOKEN"
	value := os.Getenv(envName)
	if value == "" {
		return value, fmt.Errorf("No token available for git kind %s at environment variable $%s", gitKind, envName)
	}
	return value, nil
}

func (o *WebhookOptions) returnError(err error, message string, w http.ResponseWriter, r *http.Request) {
	logrus.Errorf("returning error: %v %s", err, message)
	responseHTTPError(w, http.StatusInternalServerError, "500 Internal Error: "+message+" "+err.Error())
}

func (o *WebhookOptions) createHookServer() (*hook.Server, error) {
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

	kubeClient, _, err := o.GetFactory().CreateKubeClient()
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

	/*
		secretAgent := &config.SecretAgent{}
		if err := secretAgent.Start(tokens); err != nil {
			logrus.WithError(err).Fatal("Error starting secrets agent.")
		}

		var githubClient *github.Client
		var kubeClient *kube.Client
		if o.dryRun {
			githubClient = github.NewDryRunClient(secretAgent.GetTokenGenerator(o.githubTokenFile), o.githubEndpoint.Strings()...)
			kubeClient = kube.NewFakeClient(o.deckURL)
		} else {
			githubClient = github.NewClient(secretAgent.GetTokenGenerator(o.githubTokenFile), o.githubEndpoint.Strings()...)
			if o.cluster == "" {
				kubeClient, err = kube.NewClientInCluster(configAgent.Config().PlumberJobNamespace)
				if err != nil {
					logrus.WithError(err).Fatal("Error getting kube client.")
				}
			} else {
				kubeClient, err = kube.NewClientFromFile(o.cluster, configAgent.Config().PlumberJobNamespace)
				if err != nil {
					logrus.WithError(err).Fatal("Error getting kube client.")
				}
			}
		}

	*/

	/*	var slackClient *slack.Client
		if !o.dryRun && string(secretAgent.GetSecret(o.slackTokenFile)) != "" {
			logrus.Info("Using real slack client.")
			slackClient = slack.NewClient(secretAgent.GetTokenGenerator(o.slackTokenFile))
		}
		if slackClient == nil {
			logrus.Info("Using fake slack client.")
			slackClient = slack.NewFakeClient()
		}
	*/

	gitClient, err := git.NewClient(o.gitServerURL, o.gitKind())
	if err != nil {
		logrus.WithError(err).Fatal("Error getting git client.")
	}
	defer gitClient.Clean()

	/*
			// Get the bot's name in order to set credentials for the git client.
			botName, err := githubClient.BotName()
			if err != nil {
				logrus.WithError(err).Fatal("Error getting bot name.")
			}
			gitClient.SetCredentials(botName, secretAgent.GetTokenGenerator(o.githubTokenFile))


		ownersClient := repoowners.NewClient(
			configAgent, pluginAgent.MDYAMLEnabled,
			pluginAgent.SkipCollaborators,
		)

			pluginAgent.PluginClient = plugins.PluginClient{
				GitHubClient: githubClient,
				KubeClient:   kubeClient,
				GitClient:    gitClient,
				// TODO
				//SlackClient:  slackClient,
				OwnersClient: ownersClient,
				Logger:       logrus.WithField("agent", "plugin"),
			}
			if err := pluginAgent.Start(o.pluginConfig); err != nil {
				logrus.WithError(err).Fatal("Error starting plugins.")
			}
	*/

	promMetrics := hook.NewMetrics()

	// Push metrics to the configured prometheus pushgateway endpoint.
	pushGateway := configAgent.Config().PushGateway
	if pushGateway.Endpoint != "" {
		go metrics.PushMetrics("hook", pushGateway.Endpoint, pushGateway.Interval)
	}

	server := &hook.Server{
		ConfigAgent: configAgent,
		Plugins:     pluginAgent,
		Metrics:     promMetrics,
		//TokenGenerator: secretAgent.GetTokenGenerator(o.webhookSecretFile),
	}
	return server, nil
}

// createConfigFiles if no configuration files are defined on the CLI we dynamically load them from the ConfigMaps
// to simplify running things locally
func (o *WebhookOptions) createConfigFiles() error {
	if o.configFilename != "" && o.pluginFilename != "" {
		return nil
	}
	kubeClient, _, err := o.GetFactory().CreateKubeClient()
	if err != nil {
		return errors.Wrapf(err, "failed to create KubeClient")
	}
	ns := o.namespace
	configMapInterface := kubeClient.CoreV1().ConfigMaps(ns)

	if o.configFilename == "" {
		ccm, err := configMapInterface.Get(ProwConfigMapName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to load ConfigMap %s in namespace %s", ProwConfigMapName, ns)
		}
		cyml := ""
		if ccm.Data != nil {
			cyml = ccm.Data[ProwConfigFilename]
		}
		if cyml == "" {
			return errors.Wrapf(err, "no entry %s in ConfigMap %s in namespace %s", ProwConfigFilename, ProwConfigMapName, ns)
		}
		cf, err := ioutil.TempFile("", ProwConfigMapName+"-")
		if err != nil {
			return errors.Wrapf(err, "failed to create a temporary file for %s", ProwConfigFilename)
		}
		cfile := cf.Name()
		err = ioutil.WriteFile(cfile, []byte(cyml), util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save filer %s", cfile)
		}
		o.configFilename = cfile
	}

	if o.pluginFilename == "" {
		pcm, err := configMapInterface.Get(ProwPluginsConfigMapName, metav1.GetOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to load ConfigMap %s in namespace %s", ProwPluginsConfigMapName, ns)
		}
		pyml := ""
		if pcm.Data != nil {
			pyml = pcm.Data[ProwPluginsFilename]
		}
		if pyml == "" {
			return errors.Wrapf(err, "no entry %s in ConfigMap %s in namespace %s", ProwPluginsFilename, ProwPluginsConfigMapName, ns)
		}
		pf, err := ioutil.TempFile("", ProwPluginsConfigMapName+"-")
		if err != nil {
			return errors.Wrapf(err, "failed to create a temporary file for %s", ProwPluginsConfigMapName)
		}
		pfile := pf.Name()
		err = ioutil.WriteFile(pfile, []byte(pyml), util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save filer %s", pfile)
		}
		o.pluginFilename = pfile
	}
	return nil
}

func (o *WebhookOptions) updatePlumberClientAndReturnError(l *logrus.Entry, server hook.Server, repository scm.Repository, w http.ResponseWriter) error {
	plumberClient, err := plumber.NewPlumber(repository)
	if err != nil {
		l.Errorf("failed to create Plumber webhook: %s", err.Error())

		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to create Plumber: %s", err.Error()))
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
