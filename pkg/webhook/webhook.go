package webhook

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/bitbucket"
	"github.com/jenkins-x/go-scm/scm/driver/gitea"
	"github.com/jenkins-x/go-scm/scm/driver/github"
	"github.com/jenkins-x/go-scm/scm/driver/gitlab"
	"github.com/jenkins-x/go-scm/scm/driver/gogs"
	"github.com/jenkins-x/go-scm/scm/driver/stash"
	"github.com/jenkins-x/jx/pkg/cmd/clients"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/cmd/helper"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	"github.com/jenkins-x/lighthouse/pkg/prow/hook"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/metrics"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	helloMessage = "hello from the Jenkins X Lighthouse\n"

	// HealthPath is the URL path for the HTTP endpoint that returns health status.
	HealthPath = "/health"
	// ReadyPath URL path for the HTTP endpoint that returns ready status.
	ReadyPath = "/ready"

	noGitServerURLMessage = "No Git Server URI defined for $GIT_SERVER"

	ProwConfigMapName           = "config"
	ProwPluginsConfigMapName    = "plugins"
	ProwExternalPluginsFilename = "external-plugins.yaml"
	ProwConfigFilename          = "config.yaml"
	ProwPluginsFilename         = "plugins.yaml"
)

// WebhookOptions holds the command line arguments
type WebhookOptions struct {
	BindAddress string
	Path        string
	Port        int
	JSONLog     bool

	factory        clients.Factory
	namespace      string
	pluginFilename string
	configFilename string
	server         *hook.Server
	botName        string
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
	cmd.Flags().StringVar(&options.botName, "bot-name", "", "The name of the bot user to run as. Defaults to $BOT_NAME if not specified.")

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

	_, _, err = o.createSCMClient()
	if err != nil {
		return errors.Wrapf(err, "failed to create ScmClient")
	}

	mux := http.NewServeMux()
	mux.Handle(HealthPath, http.HandlerFunc(o.health))
	mux.Handle(ReadyPath, http.HandlerFunc(o.ready))

	indexPaths := []string{"/", "/index.html"}
	for _, p := range indexPaths {
		if o.Path != p {
			mux.Handle(p, http.HandlerFunc(o.getIndex))
		}
	}
	mux.Handle(o.Path, http.HandlerFunc(o.handleWebHookRequests))

	logrus.Infof("Environment Controller is now listening on path %s for WebHooks", o.Path)
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

// getIndex returns a simple home page
func (o *WebhookOptions) getIndex(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("GET index")
	w.Write([]byte(helloMessage))
}

func (o *WebhookOptions) isReady() bool {
	// TODO a better readiness check
	return true
}

// handle request for pipeline runs
func (o *WebhookOptions) handleWebHookRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// liveness probe etc
		o.getIndex(w, r)
		return
	}
	logrus.Infof("about to parse webhook")

	scmClient, token, err := o.createSCMClient()
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
	gitClient, _ := git.NewClient()

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

	server.OnRequest()

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

		err := o.updatePlumberClient(l, server, pushHook.Repository(), w)
		if err != nil {
			return
		}

		server.HandlePushEvent(l, pushHook)
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

		err := o.updatePlumberClient(l, server, issueCommentHook.Repository(), w)
		if err != nil {
			return
		}
		server.HandleIssueCommentEvent(l, *issueCommentHook)
		w.Write([]byte("OK"))
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

		w.Write([]byte("OK"))
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
		w.Write([]byte("OK"))
		return
	} else {
		l.Info("invoking webhook handler")
		w.Write([]byte("OK"))
		return
	}
}

func (o *WebhookOptions) misingSourceRepository(hook scm.Webhook, w http.ResponseWriter) {
	repoText := repositoryToString(hook.Repository())
	logrus.Errorf("cannot trigger a pipeline on %s as there is no SourceRepository", repoText)
	responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: No source repository for %s", repoText))
}

// GetFactory lazily creates a Factory if its not already created
func (o *WebhookOptions) GetFactory() clients.Factory {
	if o.factory == nil {
		o.factory = clients.NewFactory()
	}
	return o.factory
}

func (o *WebhookOptions) createCommonOptions(ns string) *opts.CommonOptions {
	factory := o.GetFactory()
	options := opts.NewCommonOptionsWithFactory(factory)
	options.SetDevNamespace(ns)
	options.SetCurrentNamespace(ns)
	options.Verbose = true
	options.BatchMode = true
	options.Out = os.Stdout
	options.Err = os.Stderr
	return &options
}

func repositoryToString(repo scm.Repository) string {
	return fmt.Sprintf("%s/%s branch %s", repo.Namespace, repo.Name, repo.Branch)
}

func (o *WebhookOptions) secretFn(webhook scm.Webhook) (string, error) {
	return os.Getenv("HMAC_TOKEN"), nil
}

func (o *WebhookOptions) createSCMClient() (*scm.Client, string, error) {
	kind := os.Getenv("GIT_KIND")
	if kind == "" {
		kind = "github"
	}
	serverURL := os.Getenv("GIT_SERVER")

	var client *scm.Client
	var err error

	switch kind {
	case "bitbucket":
		if serverURL != "" {
			client, err = bitbucket.New(serverURL)
		} else {
			client = bitbucket.NewDefault()
		}
	case "gitea":
		if serverURL == "" {
			return nil, "", fmt.Errorf(noGitServerURLMessage)
		}
		client, err = gitea.New(serverURL)
	case "github":
		if serverURL != "" {
			client, err = github.New(serverURL)
		} else {
			client = github.NewDefault()
		}
	case "gitlab":
		if serverURL != "" {
			client, err = gitlab.New(serverURL)
		} else {
			client = gitlab.NewDefault()
		}
	case "gogs":
		if serverURL == "" {
			return nil, "", fmt.Errorf(noGitServerURLMessage)
		}
		client, err = gogs.New(serverURL)
	case "stash":
		if serverURL == "" {
			return nil, "", fmt.Errorf(noGitServerURLMessage)
		}
		client, err = stash.New(serverURL)
	default:
		return nil, "", fmt.Errorf("Unsupported $GIT_KIND value: %s", kind)
	}
	if err != nil {
		return client, "", err
	}
	token, err := o.createSCMToken(kind)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	client.Client = oauth2.NewClient(context.Background(), ts)
	return client, token, err
}

func (o *WebhookOptions) GetBotName() string {
	o.botName = os.Getenv("BOT_NAME")
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
	err := o.createConfigFiles()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to lazy create the prow config files from ConfigMaps")
	}

	configAgent := &config.Agent{}
	configFilename := o.configFilename
	pluginFilename := o.pluginFilename

	logrus.WithField("file", configFilename).Info("loading ChatOps Config")

	if err := configAgent.Start(configFilename, pluginFilename); err != nil {
		logrus.WithError(err).Fatal("Error starting config agent.")
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

	gitClient, err := git.NewClient()
	if err != nil {
		logrus.WithError(err).Fatal("Error getting git client.")
	}
	defer gitClient.Clean()

	pluginAgent := &plugins.ConfigAgent{}

	logrus.WithField("file", pluginFilename).Info("loading ChatOps Plugins configuration")

	err = pluginAgent.Load(pluginFilename)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load configuration from %s", pluginFilename)
	}

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

func (o *WebhookOptions) updatePlumberClient(l *logrus.Entry, server hook.Server, repository scm.Repository, w http.ResponseWriter) error {
	plumberClient, err := plumber.NewPlumber(repository, o.createCommonOptions(o.namespace))
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
