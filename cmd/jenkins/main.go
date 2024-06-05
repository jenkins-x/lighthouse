/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/version"

	"github.com/NYTimes/gziphandler"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/secret"
	"github.com/jenkins-x/lighthouse/pkg/engines/jenkins"
	"github.com/jenkins-x/lighthouse/pkg/interrupts"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/jenkins-x/lighthouse/pkg/watcher"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

type options struct {
	selector  string
	namespace string

	jenkinsURL             string
	jenkinsUserName        string
	jenkinsTokenFile       string
	jenkinsBearerTokenFile string
	certFile               string
	keyFile                string
	caCertFile             string
	csrfProtect            bool

	dryRun bool
}

func (o *options) Validate() error {
	if _, err := url.ParseRequestURI(o.jenkinsURL); err != nil {
		return fmt.Errorf("invalid -jenkins-url URI: %q", o.jenkinsURL)
	}

	if o.jenkinsTokenFile == "" && o.jenkinsBearerTokenFile == "" {
		return errors.New("either --jenkins-token-file or --jenkins-bearer-token-file must be set")
	} else if o.jenkinsTokenFile != "" && o.jenkinsBearerTokenFile != "" {
		return errors.New("only one of --jenkins-token-file or --jenkins-bearer-token-file can be set")
	}

	var transportSecretsProvided int
	if o.certFile == "" {
		transportSecretsProvided = transportSecretsProvided + 1
	}
	if o.keyFile == "" {
		transportSecretsProvided = transportSecretsProvided + 1
	}
	if o.caCertFile == "" {
		transportSecretsProvided = transportSecretsProvided + 1
	}
	if transportSecretsProvided != 0 && transportSecretsProvided != 3 {
		return errors.New("either --cert-file, --key-file, and --ca-cert-file must all be provided or none of them must be provided")
	}
	return nil
}

func gatherOptions() (options, error) {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&o.selector, "label-selector", labels.Everything().String(), "Label selector to select correct Jenkins configuration from Lighthouse configuration. See https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors for constructing a label selector.")
	fs.StringVar(&o.namespace, "namespace", "lighthouse", "The namespace in which Lighthouse is installed. Defaults to 'lighthouse'.")

	fs.StringVar(&o.jenkinsURL, "jenkins-url", "http://jenkins-proxy", "Jenkins URL")
	fs.StringVar(&o.jenkinsUserName, "jenkins-user", "jenkins-trigger", "Jenkins username")
	fs.StringVar(&o.jenkinsTokenFile, "jenkins-token-file", "", "Path to the file containing the Jenkins API token.")
	fs.StringVar(&o.jenkinsBearerTokenFile, "jenkins-bearer-token-file", "", "Path to the file containing the Jenkins API bearer token.")
	fs.StringVar(&o.certFile, "cert-file", "", "Path to a PEM-encoded certificate file.")
	fs.StringVar(&o.keyFile, "key-file", "", "Path to a PEM-encoded key file.")
	fs.StringVar(&o.caCertFile, "ca-cert-file", "", "Path to a PEM-encoded CA certificate file.")
	fs.BoolVar(&o.csrfProtect, "csrf-protect", false, "Request a CSRF protection token from Jenkins that will be used in all subsequent requests to Jenkins.")

	fs.BoolVar(&o.dryRun, "dry-run", false, "Whether or not to make mutating API calls to GitHub/Kubernetes/Jenkins.")
	err := fs.Parse(os.Args[1:])
	if err != nil {
		return options{}, err
	}
	return o, nil
}

func main() {
	logrusutil.ComponentInit("lighthouse-jenkins-controller")
	logrus.WithField("version", fmt.Sprintf("%v", version.Version)).Info("Lighthouse Jenkins Controller")

	o, err := gatherOptions()
	if err != nil {
		logrus.Fatalf("Unable to parse command line arguments: %v", err)
	}

	if err := o.Validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	defer interrupts.WaitForGracefulShutdown()

	if _, err := labels.Parse(o.selector); err != nil {
		logrus.WithError(err).Fatal("Error parsing label selector.")
	}

	configAgent := &config.Agent{}
	watcher, err := watcher.SetupConfigMapWatchers(o.namespace, configAgent, nil)
	if err != nil {
		logrus.WithError(err).Fatal(err, "Error loading configuration.")
	}
	defer watcher.Stop()

	cfg := configAgent.Config

	_, _, lighthouseClientSet, _, err := clients.GetAPIClients()
	if err != nil {
		logrus.WithError(err).Fatal(err, "Error creating kubernetes resource clients.")
	}

	authConfig := &jenkins.AuthConfig{
		CSRFProtect: o.csrfProtect,
	}

	var tokens []string
	if o.jenkinsTokenFile != "" {
		tokens = append(tokens, o.jenkinsTokenFile)
	}

	if o.jenkinsBearerTokenFile != "" {
		tokens = append(tokens, o.jenkinsBearerTokenFile)
	}

	// Start the secret agent.
	secretAgent := &secret.Agent{}
	if err := secretAgent.Start(tokens); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	if o.jenkinsTokenFile != "" {
		authConfig.Basic = &jenkins.BasicAuthConfig{
			User:     o.jenkinsUserName,
			GetToken: secretAgent.GetTokenGenerator(o.jenkinsTokenFile),
		}
	} else if o.jenkinsBearerTokenFile != "" {
		authConfig.BearerToken = &jenkins.BearerTokenAuthConfig{
			GetToken: secretAgent.GetTokenGenerator(o.jenkinsBearerTokenFile),
		}
	}
	var tlsConfig *tls.Config
	if o.certFile != "" && o.keyFile != "" {
		config, err := loadCerts(o.certFile, o.keyFile, o.caCertFile)
		if err != nil {
			logrus.WithError(err).Fatalf("Could not read certificate files.")
		}
		tlsConfig = config
	}
	metrics := jenkins.NewMetrics()
	jc, err := jenkins.NewClient(o.jenkinsURL, o.dryRun, tlsConfig, authConfig, nil, metrics.ClientMetrics)
	if err != nil {
		logrus.WithError(err).Fatalf("Could not setup Jenkins client.")
	}

	c, err := jenkins.NewController(lighthouseClientSet.LighthouseV1alpha1().LighthouseJobs(o.namespace), jc, nil, cfg, o.selector)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to instantiate Jenkins controller.")
	}

	// Serve Jenkins logs here and proxy deck to use this endpoint
	// instead of baking agent-specific logic in deck
	logMux := http.NewServeMux()
	logMux.Handle("/", gziphandler.GzipHandler(handleLog(jc)))
	server := &http.Server{Addr: ":8080", Handler: logMux}
	interrupts.ListenAndServe(server, 5*time.Second)

	// run the controller
	interrupts.TickLiteral(func() {
		start := time.Now()
		if err := c.Sync(); err != nil {
			logrus.WithError(err).Error("Error syncing.")
		}
		duration := time.Since(start)
		logrus.WithField("duration", fmt.Sprintf("%v", duration)).Info("Synced")
		metrics.ResyncPeriod.Observe(duration.Seconds())
	}, 30*time.Second)
}

func loadCerts(certFile, keyFile, caCertFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	if caCertFile != "" { // #nosec
		caCert, err := os.ReadFile(caCertFile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}

func handleLog(jenkinsClient *jenkins.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")

		// Needs to be a GET request.
		if r.Method != http.MethodGet {
			http.Error(w, "405 Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Needs to get Jenkins logs.
		if !strings.HasSuffix(r.URL.Path, "consoleText") {
			http.Error(w, "403 Forbidden: Request may only access raw Jenkins logs", http.StatusForbidden)
			return
		}

		log, err := jenkinsClient.GetSkipMetrics(r.URL.Path)
		if err != nil {
			http.Error(w, fmt.Sprintf("Log not found: %v", err), http.StatusNotFound)
			logrus.WithError(err).Warning(fmt.Sprintf("Cannot get logs from Jenkins (GET %s).", r.URL.Path))
			return
		}

		if _, err = w.Write(log); err != nil {
			logrus.WithError(err).Warning("Error writing log.")
		}
	}
}
