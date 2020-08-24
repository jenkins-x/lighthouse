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

// Package config knows how to read and parse config.yaml.
package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config/org"
	"github.com/sirupsen/logrus"
	"gopkg.in/robfig/cron.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"
)

const (
	logMountName            = "logs"
	logMountPath            = "/logs"
	codeMountName           = "code"
	codeMountPath           = "/home/prow/go"
	toolsMountName          = "tools"
	toolsMountPath          = "/tools"
	gcsCredentialsMountName = "gcs-credentials"
	gcsCredentialsMountPath = "/secrets/gcs"
)

// Config is a read-only snapshot of the config.
type Config struct {
	JobConfig
	ProwConfig
}

// JobConfig is config for all prow jobs
type JobConfig struct {
	// Presets apply to all job types.
	Presets []Preset `json:"presets,omitempty"`
	// Full repo name (such as "kubernetes/kubernetes") -> list of jobs.
	Presubmits  map[string][]Presubmit  `json:"presubmits,omitempty"`
	Postsubmits map[string][]Postsubmit `json:"postsubmits,omitempty"`

	// Periodics are not associated with any repo.
	Periodics []Periodic `json:"periodics,omitempty"`
}

// ProwConfig is config for all prow controllers
type ProwConfig struct {
	Keeper           Keeper                `json:"tide,omitempty"`
	Plank            Plank                 `json:"plank,omitempty"`
	BranchProtection BranchProtection      `json:"branch-protection,omitempty"`
	Orgs             map[string]org.Config `json:"orgs,omitempty"`
	Gerrit           Gerrit                `json:"gerrit,omitempty"`

	// TODO: Move this out of the main config.
	JenkinsOperators []JenkinsOperator `json:"jenkins_operators,omitempty"`

	// LighthouseJobNamespace is the namespace in the cluster that prow
	// components will use for looking up LighthouseJobs. The namespace
	// needs to exist and will not be created by prow.
	// Defaults to "default".
	LighthouseJobNamespace string `json:"prowjob_namespace,omitempty"`
	// PodNamespace is the namespace in the cluster that prow
	// components will use for looking up Pods owned by LighthouseJobs.
	// The namespace needs to exist and will not be created by prow.
	// Defaults to "default".
	PodNamespace string `json:"pod_namespace,omitempty"`

	// LogLevel enables dynamically updating the log level of the
	// standard logger that is used by all prow components.
	//
	// Valid values:
	//
	// "debug", "info", "warn", "warning", "error", "fatal", "panic"
	//
	// Defaults to "info".
	LogLevel string `json:"log_level,omitempty"`

	// PushGateway is a prometheus push gateway.
	PushGateway PushGateway `json:"push_gateway,omitempty"`

	// OwnersDirExcludes is used to configure which directories to ignore when
	// searching for OWNERS{,_ALIAS} files in a repo.
	OwnersDirExcludes *OwnersDirExcludes `json:"owners_dir_excludes,omitempty"`

	// OwnersDirExcludes is DEPRECATED in favor of OwnersDirExcludes
	OwnersDirBlacklist *OwnersDirExcludes `json:"owners_dir_blacklist,omitempty"`

	// Pub/Sub Subscriptions that we want to listen to
	PubSubSubscriptions PubsubSubscriptions `json:"pubsub_subscriptions,omitempty"`

	// GitHubOptions allows users to control how prow applications display GitHub website links.
	GitHubOptions GitHubOptions `json:"github,omitempty"`

	// ProviderConfig contains optional SCM provider information
	ProviderConfig *ProviderConfig `json:"providerConfig,omitempty"`
}

// ProviderConfig is optionally used to configure information about the SCM provider being used. These values will be
// used as fallbacks if environment variables aren't set.
type ProviderConfig struct {
	// Kind is the go-scm driver name
	Kind string `json:"kind,omitempty"`
	// Server is the base URL for the provider, like https://github.com
	Server string `json:"server,omitempty"`
	// BotUser is the username on the provider the bot will use
	BotUser string `json:"botUser,omitempty"`
}

// OwnersDirExcludes is used to configure which directories to ignore when
// searching for OWNERS{,_ALIAS} files in a repo.
type OwnersDirExcludes struct {
	// Repos configures a directory blacklist per repo (or org)
	Repos map[string][]string `json:"repos"`
	// Default configures a default blacklist for repos (or orgs) not
	// specifically configured
	Default []string `json:"default"`
}

// PushGateway is a prometheus push gateway.
type PushGateway struct {
	// Endpoint is the location of the prometheus pushgateway
	// where prow will push metrics to.
	Endpoint string `json:"endpoint,omitempty"`
	// IntervalString compiles into Interval at load time.
	IntervalString string `json:"interval,omitempty"`
	// Interval specifies how often prow will push metrics
	// to the pushgateway. Defaults to 1m.
	Interval time.Duration `json:"-"`
	// ServeMetrics tells if or not the components serve metrics
	ServeMetrics bool `json:"serve_metrics"`
}

// Controller holds configuration applicable to all agent-specific
// prow controllers.
type Controller struct {
	// JobURLTemplateString compiles into JobURLTemplate at load time.
	JobURLTemplateString string `json:"job_url_template,omitempty"`
	// JobURLTemplate is compiled at load time from JobURLTemplateString. It
	// will be passed a builder.PipelineOptions and is used to set the URL for the
	// "Details" link on GitHub as well as the link from deck.
	JobURLTemplate *template.Template `json:"-"`

	// ReportTemplateString compiles into ReportTemplate at load time.
	ReportTemplateString string `json:"report_template,omitempty"`
	// ReportTemplate is compiled at load time from ReportTemplateString. It
	// will be passed a builder.PipelineOptions and can provide an optional blurb below
	// the test failures comment.
	ReportTemplate *template.Template `json:"-"`

	// MaxConcurrency is the maximum number of tests running concurrently that
	// will be allowed by the controller. 0 implies no limit.
	MaxConcurrency int `json:"max_concurrency,omitempty"`

	// MaxGoroutines is the maximum number of goroutines spawned inside the
	// controller to handle tests. Defaults to 20. Needs to be a positive
	// number.
	MaxGoroutines int `json:"max_goroutines,omitempty"`

	// AllowCancellations enables aborting presubmit jobs for commits that
	// have been superseded by newer commits in Github pull requests.
	AllowCancellations bool `json:"allow_cancellations,omitempty"`
}

// Plank is config for the plank controller.
type Plank struct {

	// ReportTemplateString compiles into ReportTemplate at load time.
	ReportTemplateString string `json:"report_template,omitempty"`
	// ReportTemplate is compiled at load time from ReportTemplateString. It
	// will be passed a builder.PipelineOptions and can provide an optional blurb below
	// the test failures comment.
	ReportTemplate *template.Template `json:"-"`
}

// Gerrit is config for the gerrit controller.
type Gerrit struct {
	// TickInterval is how often we do a sync with binded gerrit instance
	TickIntervalString string        `json:"tick_interval,omitempty"`
	TickInterval       time.Duration `json:"-"`
	// RateLimit defines how many changes to query per gerrit API call
	// default is 5
	RateLimit int `json:"ratelimit,omitempty"`
}

// JenkinsOperator is config for the jenkins-operator controller.
type JenkinsOperator struct {
	Controller `json:",inline"`
	// LabelSelectorString compiles into LabelSelector at load time.
	// If set, this option needs to match --label-selector used by
	// the desired jenkins-operator. This option is considered
	// invalid when provided with a single jenkins-operator config.
	//
	// For label selector syntax, see below:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	LabelSelectorString string `json:"label_selector,omitempty"`
	// LabelSelector is used so different jenkins-operator replicas
	// can use their own configuration.
	LabelSelector labels.Selector `json:"-"`
}

// PubsubSubscriptions maps GCP projects to a list of Topics.
type PubsubSubscriptions map[string][]string

// GitHubOptions allows users to control how prow applications display GitHub website links.
type GitHubOptions struct {
	// LinkURLFromConfig is the string representation of the link_url config parameter.
	// This config parameter allows users to override the default GitHub link url for all plugins.
	// If this option is not set, we assume "https://github.com".
	LinkURLFromConfig string `json:"link_url,omitempty"`

	// LinkURL is the url representation of LinkURLFromConfig. This variable should be used
	// in all places internally.
	LinkURL *url.URL
}

// Load loads and parses the config at path.
func Load(prowConfig, jobConfig string) (c *Config, err error) {
	// we never want config loading to take down the prow components
	defer func() {
		if r := recover(); r != nil {
			c, err = nil, fmt.Errorf("panic loading config: %v", r)
		}
	}()
	c, err = loadConfigFromFiles(prowConfig, jobConfig)
	if err != nil {
		return nil, err
	}

	return c.finalizeAndValidate()
}

// LoadYAMLConfig loads the configuration from the given data
func LoadYAMLConfig(data []byte) (*Config, error) {
	c := &Config{}
	if err := yaml.Unmarshal(data, c); err != nil {
		return c, err
	}
	if err := parseProwConfig(c); err != nil {
		return c, err
	}

	return c.finalizeAndValidate()
}

// loadConfigFromFiles loads one or multiple config files and returns a config object.
func loadConfigFromFiles(prowConfig, jobConfig string) (*Config, error) {
	stat, err := os.Stat(prowConfig)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("prowConfig cannot be a dir - %s", prowConfig)
	}

	var nc Config
	if err := yamlToConfig(prowConfig, &nc); err != nil {
		return nil, err
	}
	if err := parseProwConfig(&nc); err != nil {
		return nil, err
	}

	// TODO(krzyzacy): temporary allow empty jobconfig
	//                 also temporary allow job config in prow config
	if jobConfig == "" {
		return &nc, nil
	}

	stat, err = os.Stat(jobConfig)
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		// still support a single file
		var jc JobConfig
		if err := yamlToConfig(jobConfig, &jc); err != nil {
			return nil, err
		}
		if err := nc.mergeJobConfig(jc); err != nil {
			return nil, err
		}
		return &nc, nil
	}

	// we need to ensure all config files have unique basenames,
	// since updateconfig plugin will use basename as a key in the configmap
	uniqueBasenames := sets.String{}

	err = filepath.Walk(jobConfig, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logrus.WithError(err).Errorf("walking path %q.", path)
			// bad file should not stop us from parsing the directory
			return nil
		}

		if strings.HasPrefix(info.Name(), "..") {
			// kubernetes volumes also include files we
			// should not look be looking into for keys
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		if uniqueBasenames.Has(base) {
			return fmt.Errorf("duplicated basename is not allowed: %s", base)
		}
		uniqueBasenames.Insert(base)

		var subConfig JobConfig
		if err := yamlToConfig(path, &subConfig); err != nil {
			return err
		}
		return nc.mergeJobConfig(subConfig)
	})

	if err != nil {
		return nil, err
	}

	return &nc, nil
}

// yamlToConfig converts a yaml file into a Config object
func yamlToConfig(path string, nc interface{}) error {
	b, err := ioutil.ReadFile(path) // #nosec
	if err != nil {
		return fmt.Errorf("error reading %s: %v", path, err)
	}
	if err := yaml.Unmarshal(b, nc); err != nil {
		return fmt.Errorf("error unmarshaling %s: %v", path, err)
	}
	var jc *JobConfig
	switch v := nc.(type) {
	case *JobConfig:
		jc = v
	case *Config:
		jc = &v.JobConfig
	}
	for rep := range jc.Presubmits {
		fix := func(job *Presubmit) {
			job.SourcePath = path
		}
		for i := range jc.Presubmits[rep] {
			fix(&jc.Presubmits[rep][i])
		}
	}
	for rep := range jc.Postsubmits {
		fix := func(job *Postsubmit) {
			job.SourcePath = path
		}
		for i := range jc.Postsubmits[rep] {
			fix(&jc.Postsubmits[rep][i])
		}
	}

	fix := func(job *Periodic) {
		job.SourcePath = path
	}
	for i := range jc.Periodics {
		fix(&jc.Periodics[i])
	}
	return nil
}

// mergeConfig merges two JobConfig together
// It will try to merge:
//	- Presubmits
//	- Postsubmits
// 	- Periodics
//	- PodPresets
func (c *Config) mergeJobConfig(jc JobConfig) error {
	// Merge everything
	// *** Presets ***
	c.Presets = append(c.Presets, jc.Presets...)

	// validate no duplicated preset key-value pairs
	validLabels := map[string]bool{}
	for _, preset := range c.Presets {
		for label, val := range preset.Labels {
			pair := label + ":" + val
			if _, ok := validLabels[pair]; ok {
				return fmt.Errorf("duplicated preset 'label:value' pair : %s", pair)
			}
			validLabels[pair] = true
		}
	}

	// *** Periodics ***
	c.Periodics = append(c.Periodics, jc.Periodics...)

	// *** Presubmits ***
	if c.Presubmits == nil {
		c.Presubmits = make(map[string][]Presubmit)
	}
	for repo, jobs := range jc.Presubmits {
		c.Presubmits[repo] = append(c.Presubmits[repo], jobs...)
	}

	// *** Postsubmits ***
	if c.Postsubmits == nil {
		c.Postsubmits = make(map[string][]Postsubmit)
	}
	for repo, jobs := range jc.Postsubmits {
		c.Postsubmits[repo] = append(c.Postsubmits[repo], jobs...)
	}

	return nil
}

func setPresubmitDecorationDefaults(c *Config, ps *Presubmit) {
	/*	if ps.Decorate {
			ps.DecorationConfig = ps.DecorationConfig.ApplyDefault(c.Plank.DefaultDecorationConfig)
		}
	*/
}

func setPostsubmitDecorationDefaults(c *Config, ps *Postsubmit) {
	/*
		if ps.Decorate {
			ps.DecorationConfig = ps.DecorationConfig.ApplyDefault(c.Plank.DefaultDecorationConfig)
		}

	*/
}

func setPeriodicDecorationDefaults(c *Config, ps *Periodic) {
	/*
		if ps.Decorate {
			ps.DecorationConfig = ps.DecorationConfig.ApplyDefault(c.Plank.DefaultDecorationConfig)
		}
	*/
}

// finalizeAndValidate sets default configurations, validates the configuration, etc
func (c *Config) finalizeAndValidate() (*Config, error) {
	if err := c.finalizeJobConfig(); err != nil {
		return nil, err
	}
	if err := c.validateJobConfig(); err != nil {
		return nil, err
	}
	return c, nil
}

// finalizeJobConfig mutates and fixes entries for jobspecs
func (c *Config) finalizeJobConfig() error {
	if c.decorationRequested() {
		/*		if c.Plank.DefaultDecorationConfig == nil {
					return errors.New("no default decoration config provided for plank")
				}
				if c.Plank.DefaultDecorationConfig.UtilityImages == nil {
					return errors.New("no default decoration image pull specs provided for plank")
				}
				if c.Plank.DefaultDecorationConfig.GCSConfiguration == nil {
					return errors.New("no default GCS decoration config provided for plank")
				}
				if c.Plank.DefaultDecorationConfig.GCSCredentialsSecret == "" {
					return errors.New("no default GCS credentials secret provided for plank")
				}
		*/
		for _, vs := range c.Presubmits {
			for i := range vs {
				setPresubmitDecorationDefaults(c, &vs[i])
			}
		}

		for _, js := range c.Postsubmits {
			for i := range js {
				setPostsubmitDecorationDefaults(c, &js[i])
			}
		}

		for i := range c.Periodics {
			setPeriodicDecorationDefaults(c, &c.Periodics[i])
		}
	}

	// Ensure that regexes are valid and set defaults.
	for _, vs := range c.Presubmits {
		c.defaultPresubmitFields(vs)
		if err := SetPresubmitRegexes(vs); err != nil {
			return fmt.Errorf("could not set regex: %v", err)
		}
	}
	for _, js := range c.Postsubmits {
		c.defaultPostsubmitFields(js)
		if err := SetPostsubmitRegexes(js); err != nil {
			return fmt.Errorf("could not set regex: %v", err)
		}
	}

	c.defaultPeriodicFields(c.Periodics)

	for _, v := range c.AllPresubmits(nil) {
		if err := resolvePresets(v.Name, v.Labels, v.Spec, c.Presets); err != nil {
			return err
		}
	}

	for _, v := range c.AllPostsubmits(nil) {
		if err := resolvePresets(v.Name, v.Labels, v.Spec, c.Presets); err != nil {
			return err
		}
	}

	for _, v := range c.AllPeriodics() {
		if err := resolvePresets(v.Name, v.Labels, v.Spec, c.Presets); err != nil {
			return err
		}
	}

	if c.OwnersDirExcludes == nil {
		c.OwnersDirExcludes = c.OwnersDirBlacklist
	}

	return nil
}

var jobNameRegex = regexp.MustCompile(`^[A-Za-z0-9-._]+$`)

func validateJobBase(v JobBase, jobType PipelineKind, podNamespace string) error {
	if !jobNameRegex.MatchString(v.Name) {
		return fmt.Errorf("name: must match regex %q", jobNameRegex.String())
	}
	// Ensure max_concurrency is non-negative.
	if v.MaxConcurrency < 0 {
		return fmt.Errorf("max_concurrency: %d must be a non-negative number", v.MaxConcurrency)
	}
	if err := validateAgent(v, podNamespace); err != nil {
		return err
	}
	if err := validatePodSpec(jobType, v.Spec); err != nil {
		return err
	}
	if err := validateLabels(v.Labels); err != nil {
		return err
	}
	if v.Spec == nil || len(v.Spec.Containers) == 0 {
		return nil // knative-build and jenkins jobs have no spec
	}
	return nil
}

// validateJobConfig validates if all the jobspecs/presets are valid
// if you are mutating the jobs, please add it to finalizeJobConfig above
func (c *Config) validateJobConfig() error {
	type orgRepoJobName struct {
		orgRepo, jobName string
	}

	// Validate presubmits.
	// Checking that no duplicate job in prow config exists on the same org / repo / branch.
	validPresubmits := map[orgRepoJobName][]Presubmit{}
	for repo, jobs := range c.Presubmits {
		for _, job := range jobs {
			repoJobName := orgRepoJobName{repo, job.Name}
			for _, existingJob := range validPresubmits[repoJobName] {
				if existingJob.Brancher.Intersects(job.Brancher) {
					return fmt.Errorf("duplicated presubmit job: %s", job.Name)
				}
			}
			validPresubmits[repoJobName] = append(validPresubmits[repoJobName], job)
		}
	}

	for _, v := range c.AllPresubmits(nil) {
		if err := validateJobBase(v.JobBase, PresubmitJob, c.PodNamespace); err != nil {
			return fmt.Errorf("invalid presubmit job %s: %v", v.Name, err)
		}
		if err := validateTriggering(v); err != nil {
			return err
		}
	}

	// Validate postsubmits.
	// Checking that no duplicate job in prow config exists on the same org / repo / branch.
	validPostsubmits := map[orgRepoJobName][]Postsubmit{}
	for repo, jobs := range c.Postsubmits {
		for _, job := range jobs {
			repoJobName := orgRepoJobName{repo, job.Name}
			for _, existingJob := range validPostsubmits[repoJobName] {
				if existingJob.Brancher.Intersects(job.Brancher) {
					return fmt.Errorf("duplicated postsubmit job: %s", job.Name)
				}
			}
			validPostsubmits[repoJobName] = append(validPostsubmits[repoJobName], job)
		}
	}

	for _, j := range c.AllPostsubmits(nil) {
		if err := validateJobBase(j.JobBase, PostsubmitJob, c.PodNamespace); err != nil {
			return fmt.Errorf("invalid postsubmit job %s: %v", j.Name, err)
		}
	}

	// validate no duplicated periodics
	validPeriodics := sets.NewString()
	// Ensure that the periodic durations are valid and specs exist.
	for _, p := range c.AllPeriodics() {
		if validPeriodics.Has(p.Name) {
			return fmt.Errorf("duplicated periodic job : %s", p.Name)
		}
		validPeriodics.Insert(p.Name)
		if err := validateJobBase(p.JobBase, PeriodicJob, c.PodNamespace); err != nil {
			return fmt.Errorf("invalid periodic job %s: %v", p.Name, err)
		}
	}
	// Set the interval on the periodic jobs. It doesn't make sense to do this
	// for child jobs.
	for j, p := range c.Periodics {
		if p.Cron != "" && p.Interval != "" {
			return fmt.Errorf("cron and interval cannot be both set in periodic %s", p.Name)
		} else if p.Cron == "" && p.Interval == "" {
			return fmt.Errorf("cron and interval cannot be both empty in periodic %s", p.Name)
		} else if p.Cron != "" {
			if _, err := cron.Parse(p.Cron); err != nil {
				return fmt.Errorf("invalid cron string %s in periodic %s: %v", p.Cron, p.Name, err)
			}
		} else {
			d, err := time.ParseDuration(c.Periodics[j].Interval)
			if err != nil {
				return fmt.Errorf("cannot parse duration for %s: %v", c.Periodics[j].Name, err)
			}
			c.Periodics[j].interval = d
		}
	}

	return nil
}

// GetPostsubmits lets return all the post submits
func (c *Config) GetPostsubmits(repository scm.Repository) []Postsubmit {
	fullNames := c.fullNames(repository)
	var answer []Postsubmit
	for _, fn := range fullNames {
		answer = append(answer, c.Postsubmits[fn]...)
	}
	return answer
}

// GetPresubmits lets return all the pre submits for the given repo
func (c *Config) GetPresubmits(repository scm.Repository) []Presubmit {
	fullNames := c.fullNames(repository)
	var answer []Presubmit
	for _, fn := range fullNames {
		answer = append(answer, c.Presubmits[fn]...)
	}
	return answer
}

func (c *Config) fullNames(repository scm.Repository) []string {
	owner := repository.Namespace
	name := repository.Name
	fullName := repository.FullName
	if fullName == "" {
		fullName = scm.Join(owner, name)
	}
	fullNames := []string{fullName}
	lowerOwner := strings.ToLower(owner)
	if lowerOwner != owner {
		fullNames = append(fullNames, scm.Join(lowerOwner, name))
	}
	return fullNames
}

// DefaultConfigPath will be used if a --config-path is unset
const DefaultConfigPath = "/etc/config/config.yaml"

// Path returns the value for the component's configPath if provided
// explicitly or default otherwise.
func Path(value string) string {
	if value != "" {
		return value
	}
	logrus.Warningf("defaulting to %s until 15 July 2019, please migrate", DefaultConfigPath)
	return DefaultConfigPath
}

func parseProwConfig(c *Config) error {
	reportTmpl, err := template.New("Report").Parse(c.Plank.ReportTemplateString)
	if err != nil {
		return fmt.Errorf("parsing template: %v", err)
	}
	c.Plank.ReportTemplate = reportTmpl

	if c.Gerrit.TickIntervalString == "" {
		c.Gerrit.TickInterval = time.Minute
	} else {
		tickInterval, err := time.ParseDuration(c.Gerrit.TickIntervalString)
		if err != nil {
			return fmt.Errorf("cannot parse duration for c.gerrit.tick_interval: %v", err)
		}
		c.Gerrit.TickInterval = tickInterval
	}

	if c.Gerrit.RateLimit == 0 {
		c.Gerrit.RateLimit = 5
	}

	for i := range c.JenkinsOperators {
		if err := ValidateController(&c.JenkinsOperators[i].Controller); err != nil {
			return fmt.Errorf("validating jenkins_operators config: %v", err)
		}
		sel, err := labels.Parse(c.JenkinsOperators[i].LabelSelectorString)
		if err != nil {
			return fmt.Errorf("invalid jenkins_operators.label_selector option: %v", err)
		}
		c.JenkinsOperators[i].LabelSelector = sel
		// TODO: Invalidate overlapping selectors more
		if len(c.JenkinsOperators) > 1 && c.JenkinsOperators[i].LabelSelectorString == "" {
			return errors.New("selector overlap: cannot use an empty label_selector with multiple selectors")
		}
		if len(c.JenkinsOperators) == 1 && c.JenkinsOperators[0].LabelSelectorString != "" {
			return errors.New("label_selector is invalid when used for a single jenkins-operator")
		}
	}

	if c.PushGateway.IntervalString == "" {
		c.PushGateway.Interval = time.Minute
	} else {
		interval, err := time.ParseDuration(c.PushGateway.IntervalString)
		if err != nil {
			return fmt.Errorf("cannot parse duration for push_gateway.interval: %v", err)
		}
		c.PushGateway.Interval = interval
	}

	if c.Keeper.SyncPeriodString == "" {
		c.Keeper.SyncPeriod = time.Minute
	} else {
		period, err := time.ParseDuration(c.Keeper.SyncPeriodString)
		if err != nil {
			return fmt.Errorf("cannot parse duration for tide.sync_period: %v", err)
		}
		c.Keeper.SyncPeriod = period
	}
	if c.Keeper.StatusUpdatePeriodString == "" {
		c.Keeper.StatusUpdatePeriod = c.Keeper.SyncPeriod
	} else {
		period, err := time.ParseDuration(c.Keeper.StatusUpdatePeriodString)
		if err != nil {
			return fmt.Errorf("cannot parse duration for tide.status_update_period: %v", err)
		}
		c.Keeper.StatusUpdatePeriod = period
	}

	if c.Keeper.MaxGoroutines == 0 {
		c.Keeper.MaxGoroutines = 20
	}
	if c.Keeper.MaxGoroutines <= 0 {
		return fmt.Errorf("keeper has invalid max_goroutines (%d), it needs to be a positive number", c.Keeper.MaxGoroutines)
	}

	for name, method := range c.Keeper.MergeType {
		if method != MergeMerge &&
			method != MergeRebase &&
			method != MergeSquash {
			return fmt.Errorf("merge type %q for %s is not a valid type", method, name)
		}
	}

	for i, tq := range c.Keeper.Queries {
		if err := tq.Validate(); err != nil {
			return fmt.Errorf("keeper query (index %d) is invalid: %v", i, err)
		}
	}

	if c.LighthouseJobNamespace == "" {
		c.LighthouseJobNamespace = "default"
	}
	if c.PodNamespace == "" {
		c.PodNamespace = "default"
	}

	if c.GitHubOptions.LinkURLFromConfig == "" {
		c.GitHubOptions.LinkURLFromConfig = "https://github.com"
	}
	linkURL, err := url.Parse(c.GitHubOptions.LinkURLFromConfig)
	if err != nil {
		return fmt.Errorf("unable to parse github.link_url, might not be a valid url: %v", err)
	}
	c.GitHubOptions.LinkURL = linkURL

	if c.LogLevel == "" {
		c.LogLevel = os.Getenv("LOG_LEVEL")
		if c.LogLevel == "" {
			c.LogLevel = "info"
		}
	}
	lvl, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		return err
	}
	logrus.WithField("level", lvl.String()).Infof("setting the log level")
	logrus.SetLevel(lvl)

	return nil
}

func (c *JobConfig) decorationRequested() bool {
	for _, vs := range c.Presubmits {
		for i := range vs {
			if vs[i].Decorate {
				return true
			}
		}
	}

	for _, js := range c.Postsubmits {
		for i := range js {
			if js[i].Decorate {
				return true
			}
		}
	}

	for i := range c.Periodics {
		if c.Periodics[i].Decorate {
			return true
		}
	}

	return false
}

func validateLabels(labels map[string]string) error {
	for label, value := range labels {
		for _, prowLabel := range Labels() {
			if label == prowLabel {
				return fmt.Errorf("label %s is reserved for decoration", label)
			}
		}
		if errs := validation.IsQualifiedName(label); len(errs) != 0 {
			return fmt.Errorf("invalid label %s: %v", label, errs)
		}
		if errs := validation.IsValidLabelValue(labels[label]); len(errs) != 0 {
			return fmt.Errorf("label %s has invalid value %s: %v", label, value, errs)
		}
	}
	return nil
}

func validateAgent(v JobBase, podNamespace string) error {
	agents := sets.NewString(AvailablePipelineAgentTypes()...)
	agent := v.Agent
	switch {
	case !agents.Has(agent):
		return fmt.Errorf("agent must be one of %s (found %q)", strings.Join(agents.List(), ", "), agent)
		/*	case v.Spec != nil && agent != k:
				return fmt.Errorf("job specs require agent: %s (found %q)", k, agent)
			case agent == k && v.Spec == nil:
				return errors.New("kubernetes jobs require a spec")
			case v.BuildSpec != nil && agent != b:
				return fmt.Errorf("job build_specs require agent: %s (found %q)", b, agent)
			case agent == b && v.BuildSpec == nil:
				return errors.New("knative-build jobs require a build_spec")
			case v.DecorationConfig != nil && agent != k && agent != b:
				// TODO(fejta): only source decoration supported...
				return fmt.Errorf("decoration requires agent: %s or %s (found %q)", k, b, agent)
			case v.ErrorOnEviction && agent != k:
				return fmt.Errorf("error_on_eviction only applies to agent: %s (found %q)", k, agent)
			case v.Namespace == nil || *v.Namespace == "":
				return fmt.Errorf("failed to default namespace")
			case *v.Namespace != podNamespace && agent != b:
				// TODO(fejta): update plank to allow this (depends on client change)
				return fmt.Errorf("namespace customization requires agent: %s (found %q)", b, agent)
		*/
	}
	return nil
}

func resolvePresets(name string, labels map[string]string, spec *v1.PodSpec, presets []Preset) error {
	for _, preset := range presets {
		if err := mergePreset(preset, labels, spec); err != nil {
			return fmt.Errorf("job %s failed to merge presets: %v", name, err)
		}
	}

	return nil
}

func validatePodSpec(jobType PipelineKind, spec *v1.PodSpec) error {
	if spec == nil {
		return nil
	}

	if len(spec.InitContainers) != 0 {
		return errors.New("pod spec may not use init containers")
	}

	if n := len(spec.Containers); n != 1 {
		return fmt.Errorf("pod spec must specify exactly 1 container, found: %d", n)
	}

	/*	for _, env := range spec.Containers[0].Env {
			for _, prowEnv := range downwardapi.EnvForType(jobType) {
				if env.Name == prowEnv {
					// TODO(fejta): consider allowing this
					return fmt.Errorf("env %s is reserved", env.Name)
				}
			}
		}
	*/

	for _, mount := range spec.Containers[0].VolumeMounts {
		for _, prowMount := range VolumeMounts() {
			if mount.Name == prowMount {
				return fmt.Errorf("volumeMount name %s is reserved for decoration", prowMount)
			}
		}
		for _, prowMountPath := range VolumeMountPaths() {
			if strings.HasPrefix(mount.MountPath, prowMountPath) || strings.HasPrefix(prowMountPath, mount.MountPath) {
				return fmt.Errorf("mount %s at %s conflicts with decoration mount at %s", mount.Name, mount.MountPath, prowMountPath)
			}
		}
	}

	for _, volume := range spec.Volumes {
		for _, prowVolume := range VolumeMounts() {
			if volume.Name == prowVolume {
				return fmt.Errorf("volume %s is a reserved for decoration", volume.Name)
			}
		}
	}

	return nil
}

func validateTriggering(job Presubmit) error {
	if job.AlwaysRun && job.RunIfChanged != "" {
		return fmt.Errorf("job %s is set to always run but also declares run_if_changed targets, which are mutually exclusive", job.Name)
	}

	if !job.SkipReport && job.Context == "" {
		return fmt.Errorf("job %s is set to report but has no context configured", job.Name)
	}

	return nil
}

// ValidateController validates the provided controller config.
func ValidateController(c *Controller) error {
	urlTmpl, err := template.New("JobURL").Parse(c.JobURLTemplateString)
	if err != nil {
		return fmt.Errorf("parsing template: %v", err)
	}
	c.JobURLTemplate = urlTmpl

	reportTmpl, err := template.New("Report").Parse(c.ReportTemplateString)
	if err != nil {
		return fmt.Errorf("parsing template: %v", err)
	}
	c.ReportTemplate = reportTmpl
	if c.MaxConcurrency < 0 {
		return fmt.Errorf("controller has invalid max_concurrency (%d), it needs to be a non-negative number", c.MaxConcurrency)
	}
	if c.MaxGoroutines == 0 {
		c.MaxGoroutines = 20
	}
	if c.MaxGoroutines <= 0 {
		return fmt.Errorf("controller has invalid max_goroutines (%d), it needs to be a positive number", c.MaxGoroutines)
	}
	return nil
}

// DefaultTriggerFor returns the default regexp string used to match comments
// that should trigger the job with this name.
func DefaultTriggerFor(name string) string {
	return fmt.Sprintf(`(?m)^/test( | .* )%s,?($|\s.*)`, name)
}

// DefaultRerunCommandFor returns the default rerun command for the job with
// this name.
func DefaultRerunCommandFor(name string) string {
	return fmt.Sprintf("/test %s", name)
}

// defaultJobBase configures common parameters, currently Agent and Namespace.
func (c *ProwConfig) defaultJobBase(base *JobBase) {
	if base.Agent == "" { // Use the Jenkins X type by default
		base.Agent = JenkinsXAgent
	}
	if base.Namespace == nil || *base.Namespace == "" {
		s := c.PodNamespace
		base.Namespace = &s
	}
	if base.Cluster == "" {
		base.Cluster = "default"
	}
}

func (c *ProwConfig) defaultPresubmitFields(js []Presubmit) {
	for i := range js {
		c.defaultJobBase(&js[i].JobBase)
		if js[i].Context == "" {
			js[i].Context = js[i].Name
		}
		// Default the values of Trigger and RerunCommand if both fields are
		// specified. Otherwise let validation fail as both or neither should have
		// been specified.
		if js[i].Trigger == "" && js[i].RerunCommand == "" {
			js[i].Trigger = DefaultTriggerFor(js[i].Name)
			js[i].RerunCommand = DefaultRerunCommandFor(js[i].Name)
		}
	}
}

func (c *ProwConfig) defaultPostsubmitFields(js []Postsubmit) {
	for i := range js {
		c.defaultJobBase(&js[i].JobBase)
		if js[i].Context == "" {
			js[i].Context = js[i].Name
		}
	}
}

func (c *ProwConfig) defaultPeriodicFields(js []Periodic) {
	for i := range js {
		c.defaultJobBase(&js[i].JobBase)
	}
}

// SetPresubmitRegexes compiles and validates all the regular expressions for
// the provided presubmits.
func SetPresubmitRegexes(js []Presubmit) error {
	for i, j := range js {
		if re, err := regexp.Compile(j.Trigger); err == nil {
			js[i].re = re
		} else {
			return fmt.Errorf("could not compile trigger regex for %s: %v", j.Name, err)
		}
		if !js[i].re.MatchString(j.RerunCommand) {
			return fmt.Errorf("for job %s, rerun command \"%s\" does not match trigger \"%s\"", j.Name, j.RerunCommand, j.Trigger)
		}
		b, err := setBrancherRegexes(j.Brancher)
		if err != nil {
			return fmt.Errorf("could not set branch regexes for %s: %v", j.Name, err)
		}
		js[i].Brancher = b

		c, err := setChangeRegexes(j.RegexpChangeMatcher)
		if err != nil {
			return fmt.Errorf("could not set change regexes for %s: %v", j.Name, err)
		}
		js[i].RegexpChangeMatcher = c
	}
	return nil
}

// setBrancherRegexes compiles and validates all the regular expressions for
// the provided branch specifiers.
func setBrancherRegexes(br Brancher) (Brancher, error) {
	if len(br.Branches) > 0 {
		if re, err := regexp.Compile(strings.Join(br.Branches, `|`)); err == nil {
			br.re = re
		} else {
			return br, fmt.Errorf("could not compile positive branch regex: %v", err)
		}
	}
	if len(br.SkipBranches) > 0 {
		if re, err := regexp.Compile(strings.Join(br.SkipBranches, `|`)); err == nil {
			br.reSkip = re
		} else {
			return br, fmt.Errorf("could not compile negative branch regex: %v", err)
		}
	}
	return br, nil
}

func setChangeRegexes(cm RegexpChangeMatcher) (RegexpChangeMatcher, error) {
	if cm.RunIfChanged != "" {
		re, err := regexp.Compile(cm.RunIfChanged)
		if err != nil {
			return cm, fmt.Errorf("could not compile run_if_changed regex: %v", err)
		}
		cm.reChanges = re
	}
	return cm, nil
}

// SetPostsubmitRegexes compiles and validates all the regular expressions for
// the provided postsubmits.
func SetPostsubmitRegexes(ps []Postsubmit) error {
	for i, j := range ps {
		b, err := setBrancherRegexes(j.Brancher)
		if err != nil {
			return fmt.Errorf("could not set branch regexes for %s: %v", j.Name, err)
		}
		ps[i].Brancher = b
		c, err := setChangeRegexes(j.RegexpChangeMatcher)
		if err != nil {
			return fmt.Errorf("could not set change regexes for %s: %v", j.Name, err)
		}
		ps[i].RegexpChangeMatcher = c
	}
	return nil
}

// Labels returns a string slice with label consts from kube.
func Labels() []string {
	return []string{LighthouseJobTypeLabel, CreatedByLighthouse, LighthouseJobIDLabel}
}

// VolumeMounts returns a string slice with *MountName consts in it.
func VolumeMounts() []string {
	return []string{logMountName, codeMountName, toolsMountName, gcsCredentialsMountName}
}

// VolumeMountPaths returns a string slice with *MountPath consts in it.
func VolumeMountPaths() []string {
	return []string{logMountPath, codeMountPath, toolsMountPath, gcsCredentialsMountPath}
}
