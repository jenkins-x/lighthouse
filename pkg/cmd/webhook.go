package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/bitbucket"
	"github.com/drone/go-scm/scm/driver/gitea"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/driver/gitlab"
	"github.com/drone/go-scm/scm/driver/gogs"
	"github.com/drone/go-scm/scm/driver/stash"
	"github.com/jenkins-x/jx/pkg/cmd/clients"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/lighthouse/pkg/builder"
	"github.com/jenkins-x/lighthouse/pkg/cmd/helper"
	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"

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

	Builder   builder.Builder
	factory   clients.Factory
	namespace string
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

	client, err := o.createSCMClient(r)
	if err != nil {
		logrus.Errorf("failed to create SCM client: %s", err.Error())
		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}

	webhook, err := client.Webhooks.Parse(r, o.secretFn)
	if err != nil {
		logrus.Errorf("failed to parse webhook: %s", err.Error())

		responseHTTPError(w, http.StatusInternalServerError, fmt.Sprintf("500 Internal Server Error: Failed to parse webhook: %s", err.Error()))
		return
	}

	logrus.Infof("parsed webhook")

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
	if ok {
		fields["Ref"] = pushHook.Ref
		fields["BaseRef"] = pushHook.BaseRef
		fields["Commit.Sha"] = pushHook.Commit.Sha
		fields["Commit.Link"] = pushHook.Commit.Link
		fields["Commit.Author"] = pushHook.Commit.Author
		fields["Commit.Message"] = pushHook.Commit.Message
		fields["Commit.Committer.Name"] = pushHook.Commit.Committer.Name

		logrus.WithFields(logrus.Fields(fields)).Info("invoking push handler")

		o.startBuild(pushHook, r, w)
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

		logrus.WithFields(logrus.Fields(fields)).Info("invoking PR handler")

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

		logrus.WithFields(logrus.Fields(fields)).Info("invoking PR Comment handler")
		w.Write([]byte("OK"))
		return
	} else {
		logrus.WithFields(logrus.Fields(fields)).Info("invoking webhook handler")
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

func (o *WebhookOptions) createSCMClient(request *http.Request) (*scm.Client, error) {
	kind := os.Getenv("GIT_KIND")
	if kind == "" {
		kind = "github"
	}
	server := os.Getenv("GIT_SERVER")

	switch kind {
	case "bitbucket":
		if server != "" {
			return bitbucket.New(server)
		}
		return bitbucket.NewDefault(), nil
	case "gitea":
		if server == "" {
			return nil, fmt.Errorf(noGitServerURLMessage)
		}
		return gitea.New(server)
	case "github":
		if server != "" {
			github.New(server)
		}
		return github.NewDefault(), nil
	case "gitlab":
		if server != "" {
			return gitlab.New(server)
		}
		return gitlab.NewDefault(), nil
	case "gogs":
		if server == "" {
			return nil, fmt.Errorf(noGitServerURLMessage)
		}
		return gogs.New(server)
	case "stash":
		if server == "" {
			return nil, fmt.Errorf(noGitServerURLMessage)
		}
		return stash.New(server)
	default:
		return nil, fmt.Errorf("Unsupported $GIT_KIND value: %s", kind)
	}
}

func (o *WebhookOptions) returnError(err error, message string, w http.ResponseWriter, r *http.Request) {
	logrus.Errorf("returning error: %v %s", err, message)
	responseHTTPError(w, http.StatusInternalServerError, "500 Internal Error: "+message+" "+err.Error())
}

func responseHTTPError(w http.ResponseWriter, statusCode int, response string) {
	logrus.WithFields(logrus.Fields{
		"response":    response,
		"status-code": statusCode,
	}).Info(response)
	http.Error(w, response, statusCode)
}
