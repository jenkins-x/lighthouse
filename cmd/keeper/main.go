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
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/lighthouse/pkg/clients"
	"github.com/jenkins-x/lighthouse/pkg/config"
	configutil "github.com/jenkins-x/lighthouse/pkg/config/util"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/git"
	gitv2 "github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/jenkins-x/lighthouse/pkg/interrupts"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/keeper"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/jenkins-x/lighthouse/pkg/metrics"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/pkg/errors"
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
	c, err := newKeeperController(configAgent, botName, gitKind, gitToken, serverURL, o.maxRecordsPerPool, o.historyURI, o.statusURI, o.namespace)
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
	metrics.ExposeMetrics("keeper", cfg().PushGateway)

	// serve data
	logrus.WithField("port", o.port).Info("Starting HTTP server")
	interrupts.ListenAndServe(server, 10*time.Second)

	interrupts.WaitForGracefulShutdown()
}

func sync(c keeper.Controller) {
	if err := c.Sync(); err != nil {
		logrus.WithError(err).Error("Error syncing.")
	}
}

func newKeeperController(configAgent *config.Agent, botName string, gitKind string, gitToken string, serverURL string, maxRecordsPerPool int, historyURI string, statusURI string, ns string) (keeper.Controller, error) {
	var scmClient *scm.Client
	var err error
	if gitKind == "gitea" || gitKind == "bitbucketcloud" {
		// gitea returns 403 if the gitToken isn't passed here
		scmClient, err = factory.NewClient(gitKind, serverURL, gitToken, factory.SetUsername(botName))
	} else {
		scmClient, err = factory.NewClient(gitKind, serverURL, "", factory.SetUsername(botName))
	}
	if err != nil {
		return nil, errors.Wrap(err, "cannot create SCM client")
	}
	util.AddAuthToSCMClient(scmClient, gitToken)
	gitproviderClient := scmprovider.ToClient(scmClient, botName)
	gitClient, err := git.NewClient(serverURL, gitKind)
	if err != nil {
		return nil, errors.Wrap(err, "creating git client")
	}
	gitClient.SetCredentials(botName, func() []byte {
		return []byte(gitToken)
	})

	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", serverURL)
	}

	gitCloneUser := botName

	configureOpts := func(opts *gitv2.ClientFactoryOpts) {
		opts.Token = func() []byte {
			return []byte(gitToken)
		}
		opts.GitUser = func() (name, email string, err error) {
			name = gitCloneUser
			return
		}
		opts.Username = func() (login string, err error) {
			login = gitCloneUser
			return
		}
		if u.Host != "" {
			opts.Host = u.Host
		}
		if u.Scheme != "" {
			opts.Scheme = u.Scheme
		}
	}
	gitFactory, err := gitv2.NewClientFactory(configureOpts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create git client factory for server %s", serverURL)
	}
	fb := filebrowser.NewFileBrowserFromGitClient(gitFactory)
	fileBrowsers, err := filebrowser.NewFileBrowsers(serverURL, fb)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create git file browser")
	}

	tektonClient, _, lhClient, _, err := clients.GetAPIClients()
	if err != nil {
		return nil, errors.Wrap(err, "Error creating kubernetes resource clients.")
	}
	launcherClient := launcher.NewLauncher(lhClient, ns)
	c, err := keeper.NewController(gitproviderClient, gitproviderClient, fileBrowsers, launcherClient, tektonClient, lhClient, ns, configAgent.Config, gitClient, maxRecordsPerPool, historyURI, statusURI, nil)
	return c, err
}
