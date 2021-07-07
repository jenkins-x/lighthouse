package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"

	"github.com/jenkins-x/lighthouse/pkg/interrupts"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/jenkins-x/lighthouse/pkg/webhook"
	"github.com/sirupsen/logrus"
)

const (
	// HealthPath is the URL path for the HTTP endpoint that returns Health status.
	HealthPath = "/Health"
	// ReadyPath URL path for the HTTP endpoint that returns Ready status.
	ReadyPath = "/Ready"
)

type options struct {
	bindAddress string
	path        string
	pollPath    string
	port        int
	jsonLog     bool

	namespace      string
	pluginFilename string
	configFilename string
	botName        string
}

func (o *options) Validate() error {
	return nil
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options
	fs.BoolVar(&o.jsonLog, "json", true, "Enable JSON logging")
	fs.IntVar(&o.port, "port", 8080, "The TCP port to listen on.")
	fs.StringVar(&o.bindAddress, "bind", "",
		"The interface address to bind to (by default, will listen on all interfaces/addresses).")
	fs.StringVar(&o.path, "path", "/hook",
		"The path to listen on for webhook to trigger a pipeline run.")
	fs.StringVar(&o.pollPath, "pollPath", "/poll",
		"The path to listen on for polling requests to trigger a pipeline run.")
	fs.StringVar(&o.pluginFilename, "plugin-file", "", "Path to the plugins.yaml file. If not specified it is loaded from the 'plugins' ConfigMap")
	fs.StringVar(&o.configFilename, "config-file", "", "Path to the config.yaml file. If not specified it is loaded from the 'config' ConfigMap")
	fs.StringVar(&o.botName, "bot-name", "", "The name of the bot user to run as. Defaults to $GIT_USER if not specified.")
	fs.StringVar(&o.namespace, "namespace", "", "The namespace to listen in")

	err := fs.Parse(args)
	if err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	return o
}

// Entrypoint for the command
func main() {
	defer interrupts.WaitForGracefulShutdown()

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	if o.jsonLog {
		logrus.SetFormatter(logrusutil.CreateDefaultFormatter())
	}

	controller, err := webhook.NewWebhooksController(o.path, o.namespace, o.botName, o.pluginFilename, o.configFilename)
	if err != nil {
		logrus.WithError(err).Fatal("failed to set up controller")
	}
	defer func() {
		controller.CleanupGitClientDir()
		controller.ConfigMapWatcher.Stop()
	}()

	mux := http.NewServeMux()
	mux.Handle(HealthPath, http.HandlerFunc(controller.Health))
	mux.Handle(ReadyPath, http.HandlerFunc(controller.Ready))

	mux.Handle("/", http.HandlerFunc(controller.DefaultHandler))
	mux.Handle(o.path, http.HandlerFunc(controller.HandleWebhookRequests))
	mux.Handle(o.pollPath, http.HandlerFunc(controller.HandlePollingRequests))

	// lets serve metrics
	metricsHandler := http.HandlerFunc(controller.Metrics)
	go serveMetrics(metricsHandler)

	logrus.Infof("Lighthouse is now listening on path %s and port %d for WebHooks", o.path, o.port)
	err = http.ListenAndServe(":"+strconv.Itoa(o.port), mux)
	logrus.WithError(err).Errorf("failed to serve HTTP")
}

func serveMetrics(metricsHandler http.Handler) {
	logrus.Info("Lighthouse is serving prometheus metrics on port 2112")
	err := http.ListenAndServe(":2112", metricsHandler)
	logrus.WithError(err).Errorf("failed to serve HTTP")
}
