package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	gitv2 "github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/jenkins-x/lighthouse/pkg/poller"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/config"
	configutil "github.com/jenkins-x/lighthouse/pkg/config/util"
	"github.com/jenkins-x/lighthouse/pkg/interrupts"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/logrusutil"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/jenkins-x/lighthouse/pkg/watcher"
	"github.com/sirupsen/logrus"
)

type options struct {
	configPath    string
	jobConfigPath string
	botName       string
	gitServerURL  string
	gitKind       string
	namespace     string
	repoNames     string
	hookEndpoint  string
	runOnce       bool
	dryRun        bool
	syncPeriod    time.Duration
}

func (o *options) Validate() error {
	return nil
}

func gatherOptions(fs *flag.FlagSet, args ...string) options {
	var o options
	fs.StringVar(&o.configPath, "config-path", "", "Path to config.yaml.")
	fs.StringVar(&o.jobConfigPath, "job-config-path", "", "Path to prow job configs.")
	fs.StringVar(&o.botName, "bot-name", "", "The bot name")
	fs.StringVar(&o.gitServerURL, "git-url", "", "The git provider URL")
	fs.StringVar(&o.gitKind, "git-kind", "", "The git provider kind (e.g. github, gitlab, bitbucketserver")
	fs.BoolVar(&o.runOnce, "run-once", false, "If true, run only once then quit.")
	fs.BoolVar(&o.dryRun, "dry-run", false, "Disable POSTing to the webhook service and just log the webhooks instead.")

	fs.StringVar(&o.namespace, "namespace", "jx", "The namespace to listen in")
	fs.StringVar(&o.repoNames, "repo", "", "The git repository names to poll. If not specified all the repositories are polled")
	fs.StringVar(&o.hookEndpoint, "hook", "", "The hook endpoint to post to")

	fs.DurationVar(&o.syncPeriod, "period", 20*time.Second, "The time period between polls")

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
	if o.hookEndpoint == "" {
		logrus.Fatal("no hook endpoint defined")
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

	gitCloneUser := o.botName

	u, err := url.Parse(serverURL)
	if err != nil {
		logrus.WithError(err).Fatalf("failed to parse git server %s", serverURL)
	}

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
		opts.Host = u.Host
		opts.Scheme = u.Scheme
	}
	gitFactory, err := gitv2.NewClientFactory(configureOpts)
	if err != nil {
		logrus.WithError(err).Fatalf("failed to create git client factory for server %s", o.gitServerURL)
	}
	fb := filebrowser.NewFileBrowserFromGitClient(gitFactory)

	var repoNames []string
	if o.repoNames != "" {
		repoNames = strings.Split(o.repoNames, ",")
	}
	if len(repoNames) == 0 {
		cfg := configAgent.Config
		repoNames = findAllRepoNames(cfg())
	}
	if len(repoNames) == 0 {
		logrus.Fatal("no repositories found")
	}

	c, err := poller.NewPollingController(repoNames, serverURL, fb, o.notifier)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating Keeper controller.")
	}

	start := time.Now()
	c.Sync()
	if o.runOnce {
		return
	}

	// run the controller, but only after one sync period expires after our first run
	time.Sleep(time.Until(start.Add(o.syncPeriod)))
	interrupts.Tick(func() {
		c.Sync()
	}, func() time.Duration {
		return o.syncPeriod
	})
}

func findAllRepoNames(c *config.Config) []string {
	m := map[string]bool{}

	for fullName := range c.Presubmits {
		m[fullName] = true
	}
	for fullName := range c.Postsubmits {
		m[fullName] = true
	}
	for fullName := range c.InRepoConfig.Enabled {
		if !strings.Contains(fullName, "*") && strings.Contains(fullName, "/") {
			m[fullName] = true
		}
	}
	var repoNames []string
	for fullName := range m {
		repoNames = append(repoNames, fullName)
	}
	return repoNames
}

func (o *options) notifier(hook scm.Webhook) error {
	if o.dryRun {
		logrus.WithField("Hook", hook).Info("notify")
		return nil
	}

	data, err := json.Marshal(hook)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal hook %#v", hook)
	}

	req, err := http.NewRequest("POST", o.hookEndpoint, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if err != nil {
			return errors.Wrapf(err, "failed to invoke endpoint %s", o.hookEndpoint)
		}
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	l := logrus.WithFields(map[string]interface{}{
		"Status":   resp.Status,
		"Endpoint": o.hookEndpoint,
		"Body":     string(body),
	})
	if resp.StatusCode >= 300 {
		l.Warnf("failed to notify")
		return errors.Errorf("status when invoking endpoint")
	}
	l.Infof("notified")
	return nil
}
