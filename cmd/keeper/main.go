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

	"github.com/jenkins-x/lighthouse/pkg/config"
	configutil "github.com/jenkins-x/lighthouse/pkg/config/util"
	"github.com/jenkins-x/lighthouse/pkg/interrupts"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/keeper"
	"github.com/jenkins-x/lighthouse/pkg/keeper/githubapp"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/jenkins-x/lighthouse/pkg/metrics"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/sirupsen/logrus"
)

type options struct {
	port int

	configPath    string
	jobConfigPath string
	botName       string
	gitServerURL  string
	gitKind       string
	namespace     string

	runOnce bool

	maxRecordsPerPool int
	// historyURI where Keeper should store its action history.
	// Can be a /local/path or gs://path/to/object.
	// GCS writes will use the bucket's default acl for new objects. Ensure both that
	// a) the gcs credentials can write to this bucket
	// b) the default acls do not expose any private info
	historyURI string

	// statusURI where Keeper store status update state.
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
	fs.StringVar(&o.botName, "bot-name", "", "The bot name")
	fs.StringVar(&o.gitServerURL, "git-url", "", "The git provider URL")
	fs.StringVar(&o.gitKind, "git-kind", "", "The git provider kind (e.g. github, gitlab, bitbucketserver")
	fs.BoolVar(&o.runOnce, "run-once", false, "If true, run only once then quit.")

	fs.IntVar(&o.maxRecordsPerPool, "max-records-per-pool", 1000, "The maximum number of history records stored for an individual Keeper pool.")
	fs.StringVar(&o.historyURI, "history-uri", "", "The /local/path or gs://path/to/object to store keeper action history. GCS writes will use the default object ACL for the bucket")
	fs.StringVar(&o.statusURI, "status-path", "", "The /local/path or gs://path/to/object to store status controller state. GCS writes will use the default object ACL for the bucket.")
	fs.StringVar(&o.namespace, "namespace", "", "The namespace to listen in")

	err := fs.Parse(args)
	if err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}
	o.configPath = configutil.PathOrDefault(o.configPath)
	return o
}

func main() {
	logrusutil.ComponentInit("keeper")

	defer interrupts.WaitForGracefulShutdown()

	jobutil.ServePProf()

	o := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)
	if err := o.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid options")
	}

	configAgent := &config.Agent{}
	cfgMapWatcher, err := watcher.SetupConfigMapWatchers(o.namespace, configAgent, nil)
	if err != nil {
		logrus.WithError(err).Fatal("error starting config map watcher")
	}
	defer cfgMapWatcher.Stop()

	botName := o.botName
	if botName == "" {
		botName = util.GetBotName(configAgent.Config)
	}
	if util.GetGitHubAppSecretDir() != "" {
		botName, err = util.GetGitHubAppAPIUser()
		if err != nil {
			logrus.WithError(err).Fatal("unable to read API user for GitHub App integration")
		}
	}
	if botName == "" {
		logrus.Fatal("no $GIT_USER defined")
	}
	serverURL := o.gitServerURL
	if serverURL == "" {
		serverURL = util.GetGitServer(configAgent.Config)
	}
	gitKind := o.gitKind
	if gitKind == "" {
		gitKind = util.GitKind(configAgent.Config)
	}
	gitToken, err := util.GetSCMToken(gitKind)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating Keeper controller.")
	}

	cfg := configAgent.Config
	c, err := githubapp.NewKeeperController(configAgent, botName, gitKind, gitToken, serverURL, o.maxRecordsPerPool, o.historyURI, o.statusURI, o.namespace)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating Keeper controller.")
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
	time.Sleep(time.Until(start.Add(cfg().Keeper.SyncPeriod)))
	interrupts.Tick(func() {
		sync(c)
	}, func() time.Duration {
		return cfg().Keeper.SyncPeriod
	})

	// Push metrics to the configured prometheus pushgateway endpoint or serve them
	gateway := cfg().PushGateway
	if gateway.Endpoint != "" {
		logrus.WithField("gateway", gateway.Endpoint).Infof("using push gateway")
		go metrics.ExposeMetrics("keeper", gateway)

		// serve data
		interrupts.ListenAndServe(server, 10*time.Second)
	} else {
		logrus.Warn("not pushing metrics as there is no push_gateway defined in the config.yaml")

		// serve data
		err := server.ListenAndServe()
		logrus.WithError(err).Errorf("failed to server HTTP")
	}
}

func sync(c keeper.Controller) {
	if err := c.Sync(); err != nil {
		logrus.WithError(err).Error("Error syncing.")
	}
}
