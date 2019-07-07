package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	clientv1 "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned/typed/jenkins.io/v1"
	"github.com/jenkins-x/lighthouse/pkg/git/bitbucketserver"
	"github.com/sirupsen/logrus"
	rest "k8s.io/client-go/rest"
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

	namespace string
}

func main() {
	o := WebhookOptions{
		Path:        "/",
		Port:        8080,
		JSONLog:     true,
		BindAddress: "localhost",
	}

	if o.JSONLog {
		logrus.SetFormatter(&logrus.JSONFormatter{})
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

	logrus.Infof("Catcher is now listening on path %s for WebHooks", o.Path)
	if err := http.ListenAndServe(":"+strconv.Itoa(o.Port), mux); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
	gitProvider := bitbucketserver.Provider{
		URL:  "bitbucket.beescloud.com",
		Name: "bitbucketserver",
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Fatalf("error reading webhook request body: %s", err)
	}

	webhook := gitProvider.ParseWebhook(body)

	config, err := rest.InClusterConfig()
	if err != nil {
		logrus.Fatalf("config creation failed: %s", err)
	}

	client, err := clientv1.NewForConfig(config)
	if err != nil {
		logrus.Fatalf("client initialization failed: %s", err)
	}

	webhookInterface := client.Webhooks("lighthouse")

	result, err := webhookInterface.Create(webhook)
	if err != nil {
		logrus.Fatalf("Webhook CRD creation failed: %s", err)
	}
	logrus.Infof("Webhook CRD created with type: %s", result.Spec.EventType)
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
