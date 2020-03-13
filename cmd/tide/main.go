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
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/interrupts"
	"github.com/jenkins-x/lighthouse/pkg/prow/logrusutil"
	"github.com/jenkins-x/lighthouse/pkg/prow/metrics"
	"github.com/jenkins-x/lighthouse/pkg/prow/pjutil"
	"github.com/jenkins-x/lighthouse/pkg/tide"
	"github.com/jenkins-x/lighthouse/pkg/tide/githubapp"
	"github.com/sirupsen/logrus"
)

type options struct {
	port int

	configPath    string
	jobConfigPath string
	botName       string
	gitServerURL  string
	gitKind       string

	syncThrottle   int
	statusThrottle int

	dryRun  bool
	runOnce bool

	maxRecordsPerPool int
	// historyURI where Tide should store its action history.
	// Can be a /local/path or gs://path/to/object.
	// GCS writes will use the bucket's default acl for new objects. Ensure both that
	// a) the gcs credentials can write to this bucket
	// b) the default acls do not expose any private info
	historyURI string

	// statusURI where Tide store status update state.
	// Can be a /local/path or gs://path/to/object.
	// GCS writes will use the bucket's default acl for new objects. Ensure both that
	// a) the gcs credentials can write to this bucket
	// b) the default acls do not expose any private info
	statusURI string
}

func (o *options) Validate() error {
	return nil
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options
	fs.IntVar(&o.port, "port", 8888, "Port to listen on.")
	fs.StringVar(&o.configPath, "config-path", "", "Path to config.yaml.")
	fs.StringVar(&o.jobConfigPath, "job-config-path", "", "Path to prow job configs.")
	fs.StringVar(&o.botName, "bot-name", "jenkins-x-bot", "The bot name")
	fs.StringVar(&o.gitServerURL, "git-url", "", "The git provider URL")
	fs.StringVar(&o.gitKind, "git-kind", "", "The git provider kind (e.g. github, gitlab, bitbucketserver")
	fs.BoolVar(&o.dryRun, "dry-run", true, "Whether to mutate any real-world state.")
	fs.BoolVar(&o.runOnce, "run-once", false, "If true, run only once then quit.")
	fs.IntVar(&o.syncThrottle, "sync-hourly-tokens", 800, "The maximum number of tokens per hour to be used by the sync controller.")
	fs.IntVar(&o.statusThrottle, "status-hourly-tokens", 400, "The maximum number of tokens per hour to be used by the status controller.")

	fs.IntVar(&o.maxRecordsPerPool, "max-records-per-pool", 1000, "The maximum number of history records stored for an individual Tide pool.")
	fs.StringVar(&o.historyURI, "history-uri", "", "The /local/path or gs://path/to/object to store tide action history. GCS writes will use the default object ACL for the bucket")
	fs.StringVar(&o.statusURI, "status-path", "", "The /local/path or gs://path/to/object to store status controller state. GCS writes will use the default object ACL for the bucket.")

	err := fs.Parse(args)
	if err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}
	o.configPath = config.Path(o.configPath)
	return o
}

func main() {
	logrusutil.ComponentInit("tide")

	defer interrupts.WaitForGracefulShutdown()

	pjutil.ServePProf()

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	configAgent := &config.Agent{}
	if err := configAgent.Start(o.configPath, o.jobConfigPath); err != nil {
		logrus.WithError(err).Fatal("Error starting config agent.")
	}

	botName := o.botName
	if botName == "" {
		botName = os.Getenv("GIT_USER")
	}
	if botName == "" {
		logrus.Fatal("no $GIT_USER defined")
	}
	serverURL := o.gitServerURL
	if serverURL == "" {
		serverURL = os.Getenv("GIT_SERVER")
	}
	if serverURL == "" {
		serverURL = "https://github.com"
	}
	gitKind := o.gitKind
	if gitKind == "" {
		gitKind = os.Getenv("GIT_KIND")
	}
	if gitKind == "" {
		gitKind = "github"
	}
	gitToken := os.Getenv("GIT_TOKEN")

	cfg := configAgent.Config
	c, err := githubapp.NewTideController(configAgent, botName, gitKind, gitToken, serverURL, o.maxRecordsPerPool, o.historyURI, o.statusURI)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating Tide controller.")
	}
	defer c.Shutdown()
	http.Handle("/", c)
	http.Handle("/history", c.GetHistory())
	server := &http.Server{Addr: ":" + strconv.Itoa(o.port)}

	start := time.Now()
	sync(c)
	if o.runOnce {
		return
	}

	// run the controller, but only after one sync period expires after our first run
	time.Sleep(time.Until(start.Add(cfg().Tide.SyncPeriod)))
	interrupts.Tick(func() {
		sync(c)
	}, func() time.Duration {
		return cfg().Tide.SyncPeriod
	})

	// Push metrics to the configured prometheus pushgateway endpoint or serve them
	gateway := cfg().PushGateway
	if gateway.Endpoint != "" {
		logrus.WithField("gateway", gateway.Endpoint).Infof("using push gateway")
		go metrics.ExposeMetrics("tide", gateway)

		// serve data
		interrupts.ListenAndServe(server, 10*time.Second)
	} else {
		logrus.Warn("not pushing metrics as there is no push_gateway defined in the config.yaml")

		// serve data
		err := server.ListenAndServe()
		logrus.WithError(err).Errorf("failed to server HTTP")
	}
}

func sync(c tide.Controller) {
	if err := c.Sync(); err != nil {
		logrus.WithError(err).Error("Error syncing.")
	}
}
