package webhook

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/git"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/metrics"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/version"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// WebhooksController holds the command line arguments
type WebhooksController struct {
	ConfigMapWatcher *watcher.ConfigMapWatcher

	path           string
	namespace      string
	pluginFilename string
	configFilename string
	server         *Server
	botName        string
	gitServerURL   string
	gitClient      git.Client
	launcher       launcher.PipelineLauncher
}

// NewWebhooksController creates and configures the controller
func NewWebhooksController(path, namespace, botName, pluginFilename, configFilename string) (*WebhooksController, error) {
	o := &WebhooksController{
		path:           path,
		namespace:      namespace,
		pluginFilename: pluginFilename,
		configFilename: configFilename,
		botName:        botName,
	}
	var err error
	o.server, err = o.createHookServer()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Hook Server")
	}

	cfg := o.server.ConfigAgent.Config
	o.gitServerURL = util.GetGitServer(cfg)
	gitClient, err := git.NewClient(o.gitServerURL, util.GitKind(cfg))
	if err != nil {
		logrus.WithError(err).Fatal("Error getting git client.")
	}
	o.gitClient = gitClient

	_, _, lhClient, _, err := clients.GetAPIClients()
	if err != nil {
		return nil, errors.Wrap(err, "Error creating kubernetes resource clients.")
	}
	o.launcher = launcher.NewLauncher(lhClient, o.namespace)

	return o, nil
}

// CleanupGitClientDir cleans up the git client's working directory
func (o *WebhooksController) CleanupGitClientDir() {
	err := o.gitClient.Clean()
	if err != nil {
		logrus.WithError(err).Fatal("Error cleaning the git client.")
	}
}

// Health returns either HTTP 204 if the service is healthy, otherwise nothing ('cos it's dead).
func (o *WebhooksController) Health(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Health check")
	w.WriteHeader(http.StatusNoContent)
}

// Ready returns either HTTP 204 if the service is Ready to serve requests, otherwise HTTP 503.
func (o *WebhooksController) Ready(w http.ResponseWriter, r *http.Request) {
	logrus.Debug("Ready check")
	if o.isReady() {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
}

// DefaultHandler responds to requests without a specific handler
func (o *WebhooksController) DefaultHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == o.path || strings.HasPrefix(path, o.path+"/") {
		o.HandleWebhookRequests(w, r)
		return
	}
	path = strings.TrimPrefix(path, "/")
	if path == "" || path == "index.html" {
		return
	}
	http.Error(w, fmt.Sprintf("unknown path %s", path), 404)
}

func (o *WebhooksController) isReady() bool {
	// TODO a better readiness check
	return true
}

// HandleWebhookRequests handles incoming events
func (o *WebhooksController) HandleWebhookRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// liveness probe etc
		logrus.WithField("method", r.Method).Debug("invalid http method so returning 200")
		return
	}
	logrus.Debug("about to parse webhook")

	cfg := o.server.ConfigAgent.Config

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Errorf("failed to Read Body: %s", err.Error())
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Read Body: %s", err.Error()))
		return
	}

	err = r.Body.Close() // must close
	if err != nil {
		logrus.Errorf("failed to Close Body: %s", err.Error())
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Read Close: %s", err.Error()))
		return
	}

	r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	_, scmClient, serverURL, _, err := util.GetSCMClient("", cfg)
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
		gitCloneUser = util.GetBotName(cfg)
		token, err = util.GetSCMToken(util.GitKind(cfg))
		if err != nil {
			logrus.Errorf("no scm token specified: %s", err.Error())
			responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: no scm token specified: %s", err.Error()))
			return
		}
	}
	_, kubeClient, lhClient, _, err := clients.GetAPIClients()
	if err != nil {
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: %s", err.Error()))
	}

	o.gitClient.SetCredentials(gitCloneUser, func() []byte {
		return []byte(token)
	})
	util.AddAuthToSCMClient(scmClient, token, ghaSecretDir != "")

	o.server.ClientAgent = &plugins.ClientAgent{
		BotName:           util.GetBotName(cfg),
		SCMProviderClient: scmClient,
		KubernetesClient:  kubeClient,
		GitClient:         o.gitClient,
		LighthouseClient:  lhClient.LighthouseV1alpha1().LighthouseJobs(o.namespace),
		LauncherClient:    o.launcher,
	}
	l, output, err := o.ProcessWebHook(logrus.WithField("Webhook", webhook.Kind()), webhook)
	if err != nil {
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: %s", err.Error()))
	}
	// Demux events only to external plugins that require this event.
	if external := util.ExternalPluginsForEvent(o.server.Plugins, string(webhook.Kind()), webhook.Repository().FullName); len(external) > 0 {
		go util.CallExternalPluginsWithWebhook(l, external, webhook, util.HMACToken(), &o.server.wg)
	}

	_, err = w.Write([]byte(output))
	if err != nil {
		l.Debugf("failed to process the webhook: %v", err)
	}
}

// ProcessWebHook process a webhook
func (o *WebhooksController) ProcessWebHook(l *logrus.Entry, webhook scm.Webhook) (*logrus.Entry, string, error) {
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
	// If we are in GitHub App mode and have a populated config, check if the repository for this webhook is one we actually
	// know about and error out if not.
	if util.GetGitHubAppSecretDir() != "" && o.server.ConfigAgent != nil {
		cfg := o.server.ConfigAgent.Config()
		if cfg != nil {
			if len(cfg.GetPostsubmits(repository)) == 0 && len(cfg.GetPresubmits(repository)) == 0 {
				l.Infof("webhook from unconfigured repository %s, returning error", repository.Link)
				return l, "", fmt.Errorf("repository not configured: %s", repository.Link)
			}
		}
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

		o.server.handlePushEvent(l, pushHook)
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

		o.server.handlePullRequestEvent(l, prHook)
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

		o.server.handleBranchEvent(l, branchHook)
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

		o.server.handleIssueCommentEvent(l, *issueCommentHook)
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

		o.server.handlePullRequestCommentEvent(l, *prCommentHook)
		return l, "processed PR comment hook", nil
	}
	prReviewHook, ok := webhook.(*scm.ReviewHook)
	if ok {
		action := prReviewHook.Action
		fields["Action"] = action.String()
		pr := prReviewHook.PullRequest
		fields["PR.Number"] = pr.Number
		fields["PR.Ref"] = pr.Ref
		fields["PR.Sha"] = pr.Sha
		fields["PR.Title"] = pr.Title
		fields["PR.Body"] = pr.Body
		fields["Review.State"] = prReviewHook.Review.State
		fields["Reviewer.Name"] = prReviewHook.Review.Author.Name
		fields["Reviewer.Login"] = prReviewHook.Review.Author.Login
		fields["Reviewer.Avatar"] = prReviewHook.Review.Author.Avatar

		l.Info("invoking PR Review handler")

		o.server.handleReviewEvent(l, *prReviewHook)
		return l, "processed PR review hook", nil
	}
	l.Debugf("unknown kind %s webhook %#v", webhook.Kind(), webhook)
	return l, fmt.Sprintf("unknown hook %s", webhook.Kind()), nil
}

func (o *WebhooksController) secretFn(webhook scm.Webhook) (string, error) {
	return util.HMACToken(), nil
}

func (o *WebhooksController) createHookServer() (*Server, error) {
	configAgent := &config.Agent{}
	pluginAgent := &plugins.ConfigAgent{}

	var err error
	o.ConfigMapWatcher, err = watcher.SetupConfigMapWatchers(o.namespace, configAgent, pluginAgent)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create ConfigMap watcher")
	}

	promMetrics := NewMetrics()

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

	serverURL, err := url.Parse(o.gitServerURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse server URL %s", o.gitServerURL)
	}
	server := &Server{
		ConfigAgent: configAgent,
		Plugins:     pluginAgent,
		Metrics:     promMetrics,
		ServerURL:   serverURL,
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
