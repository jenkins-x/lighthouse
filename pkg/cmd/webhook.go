package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/bitbucket"
	"github.com/jenkins-x/go-scm/scm/driver/gitea"
	"github.com/jenkins-x/go-scm/scm/driver/github"
	"github.com/jenkins-x/go-scm/scm/driver/gitlab"
	"github.com/jenkins-x/go-scm/scm/driver/gogs"
	"github.com/jenkins-x/go-scm/scm/driver/stash"
	"github.com/jenkins-x/jx/pkg/cmd/clients"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/lighthouse/pkg/builder"
	"github.com/jenkins-x/lighthouse/pkg/cmd/helper"
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/git"
	"github.com/jenkins-x/lighthouse/pkg/prow/hook"
	"github.com/jenkins-x/lighthouse/pkg/prow/plugins"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/metrics"

	"github.com/spf13/cobra"
)

const (
	helloMessage = "hello from the Jenkins X Lighthouse\n"

	// HealthPath is the URL path for the HTTP endpoint that returns health status.
	HealthPath = "/health"
	// ReadyPath URL path for the HTTP endpoint that returns ready status.
	ReadyPath = "/ready"

	noGitServerURLMessage = "No Git Server URI defined for $GIT_SERVER"
)

// WebhookOptions holds the command line arguments
type WebhookOptions struct {
	BindAddress string
	Path        string
	Port        int
	JSONLog     bool

	Builder       builder.Builder
	factory       clients.Factory
	namespace     string
	configPath    string
	jobConfigPath string
	server        *hook.Server
}

// NewCmdWebhook creates the command
func NewCmdWebhook() *cobra.Command {
	options := WebhookOptions{}

	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Runs the webhook handler",
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
	cmd.Flags().StringVar(&options.configPath, "config-path", "config.yaml", "Path to config.yaml.")
	cmd.Flags().StringVar(&options.jobConfigPath, "job-config-path", "", "Path to prow job configs.")

	return cmd
}

// Run will implement this command
func (o *WebhookOptions) Run() error {
	if o.JSONLog {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	jxClient, ns, err := o.GetFactory().CreateJXClient()
	if err != nil {
		return errors.Wrapf(err, "failed to create JX Client")
	}
	o.namespace = ns
	o.Builder, err = builder.NewBuilder(jxClient, ns)
	if err != nil {
		return errors.Wrapf(err, "failed to create Builder")
	}
	o.server, err = o.createHookServer()
	if err != nil {
		return errors.Wrapf(err, "failed to create Hook Server")
	}

	_, err = o.createSCMClient()
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

	scmClient, err := o.createSCMClient()
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

		l.Info("invoking push handler")

		o.startBuild(pushHook, r, w)
		return
	}

	// lets try invoke prow commands...
	kubeClient, _, _ := o.GetFactory().CreateKubeClient()
	gitClient, _ := git.NewClient()

	server := *o.server
	server.ClientAgent = &plugins.ClientAgent{
		GitHubClient:     scmClient,
		KubernetesClient: kubeClient,
		GitClient:        gitClient,
	}

	server.OnRequest()

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

		server.HandleIssueCommentEvent(l, *issueCommentHook)
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

func (o *WebhookOptions) startBuild(hook *scm.PushHook, r *http.Request, w http.ResponseWriter) {
	message, err := o.Builder.StartBuild(hook, o.createCommonOptions(o.namespace))
	if err != nil {
		logrus.Errorf("failed to start build on %s: %s", repositoryToString(hook.Repository()), err.Error())

		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}
	w.Write([]byte(message))

	logrus.Infof("triggering META pipeline")

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

func (o *WebhookOptions) createSCMClient() (*scm.Client, error) {
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
			return nil, fmt.Errorf(noGitServerURLMessage)
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
			return nil, fmt.Errorf(noGitServerURLMessage)
		}
		client, err = gogs.New(serverURL)
	case "stash":
		if serverURL == "" {
			return nil, fmt.Errorf(noGitServerURLMessage)
		}
		client, err = stash.New(serverURL)
	default:
		return nil, fmt.Errorf("Unsupported $GIT_KIND value: %s", kind)
	}
	if err != nil {
		return client, err
	}
	token, err := o.createSCMToken(kind)
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	client.Client = oauth2.NewClient(context.Background(), ts)
	return client, err
}

func (o *WebhookOptions) createSCMToken(gitKind string) (string, error) {
	envName := "JX_" + strings.ToUpper(gitKind) + "_TOKEN"
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
	if err := configAgent.Start(o.configPath, o.jobConfigPath); err != nil {
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
				kubeClient, err = kube.NewClientInCluster(configAgent.Config().ProwJobNamespace)
				if err != nil {
					logrus.WithError(err).Fatal("Error getting kube client.")
				}
			} else {
				kubeClient, err = kube.NewClientFromFile(o.cluster, configAgent.Config().ProwJobNamespace)
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
	err = pluginAgent.Load(o.configPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load configuration from %s", o.configPath)
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

func responseHTTPError(w http.ResponseWriter, statusCode int, response string) {
	logrus.WithFields(logrus.Fields{
		"response":    response,
		"status-code": statusCode,
	}).Info(response)
	http.Error(w, response, statusCode)
}
