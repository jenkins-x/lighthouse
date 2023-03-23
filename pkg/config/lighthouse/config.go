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

package lighthouse

import (
	"errors"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/config/branchprotection"
	"github.com/jenkins-x/lighthouse/pkg/config/keeper"
	"github.com/jenkins-x/lighthouse/pkg/config/org"
)

// Config is config for all lighthouse controllers
type Config struct {
	Keeper           keeper.Config           `json:"tide,omitempty"`
	Plank            Plank                   `json:"plank,omitempty"`
	BranchProtection branchprotection.Config `json:"branch-protection,omitempty"`
	Orgs             map[string]org.Config   `json:"orgs,omitempty"`
	InRepoConfig     InRepoConfig            `json:"in_repo_config"`

	// TODO: Move this out of the main config.
	Jenkinses []JenkinsConfig `json:"jenkinses,omitempty"`
	// LighthouseJobNamespace is the namespace in the cluster that lighthouse
	// components will use for looking up LighthouseJobs. The namespace
	// needs to exist and will not be created by lighthouse.
	// Defaults to "default".
	LighthouseJobNamespace string `json:"prowjob_namespace,omitempty"`
	// PodNamespace is the namespace in the cluster that lighthouse
	// components will use for looking up Pods owned by LighthouseJobs.
	// The namespace needs to exist and will not be created by lighthouse.
	// Defaults to "default".
	PodNamespace string `json:"pod_namespace,omitempty"`
	// LogLevel enables dynamically updating the log level of the
	// standard logger that is used by all lighthouse components.
	//
	// Valid values:
	//
	// "trace", "debug", "info", "warn", "warning", "error", "fatal", "panic"
	//
	// Defaults to "info".
	LogLevel string `json:"log_level,omitempty"`
	// PushGateway is a prometheus push gateway.
	PushGateway PushGateway `json:"push_gateway,omitempty"`
	// OwnersDirExcludes is used to configure which directories to ignore when
	// searching for OWNERS{,_ALIAS} files in a repo.
	OwnersDirExcludes *OwnersDirExcludes `json:"owners_dir_excludes,omitempty"`
	// Pub/Sub Subscriptions that we want to listen to
	PubSubSubscriptions PubsubSubscriptions `json:"pubsub_subscriptions,omitempty"`
	// GitHubOptions allows users to control how lighthouse applications display GitHub website links.
	GitHubOptions GitHubOptions `json:"github,omitempty"`
	// ProviderConfig contains optional SCM provider information
	ProviderConfig *ProviderConfig `json:"providerConfig,omitempty"`
}

// Parse initializes and validates the Config
func (c *Config) Parse() error {
	if err := c.Plank.Parse(); err != nil {
		return err
	}
	for i := range c.Jenkinses {
		if err := c.Jenkinses[i].Parse(); err != nil {
			return err
		}
		// TODO: Invalidate overlapping selectors more
		if len(c.Jenkinses) > 1 && c.Jenkinses[i].LabelSelectorString == "" {
			return errors.New("selector overlap: cannot use an empty label_selector with multiple selectors")
		}
		if len(c.Jenkinses) == 1 && c.Jenkinses[0].LabelSelectorString != "" {
			return errors.New("label_selector is invalid when used for a single jenkins-operator")
		}
	}
	if err := c.PushGateway.Parse(); err != nil {
		return err
	}
	if err := c.Keeper.Parse(); err != nil {
		return err
	}
	if c.LighthouseJobNamespace == "" {
		c.LighthouseJobNamespace = "default"
	}
	if c.PodNamespace == "" {
		c.PodNamespace = "default"
	}
	if err := c.GitHubOptions.Parse(); err != nil {
		return err
	}
	if c.LogLevel == "" {
		c.LogLevel = os.Getenv("LOG_LEVEL")
		if c.LogLevel == "" {
			c.LogLevel = "info"
		}
	}
	return nil
}

// InRepoConfig to enable configuration inside the source code of a repository
//
// this struct mirrors the similar struct inside lighthouse
type InRepoConfig struct {
	// Enabled describes whether InRepoConfig is enabled for a given repository. This can
	// be set globally, per org or per repo using '*', 'org' or 'org/repo' as key. The
	// narrowest match always takes precedence.
	Enabled map[string]*bool `json:"enabled,omitempty"`
}

// InRepoConfigEnabled returns whether InRepoConfig is enabled for a given repository.
func (c *Config) InRepoConfigEnabled(identifier string) bool {
	if c.InRepoConfig.Enabled[identifier] != nil {
		return *c.InRepoConfig.Enabled[identifier]
	}
	identifierSlashSplit := strings.Split(identifier, "/")
	if len(identifierSlashSplit) == 2 && c.InRepoConfig.Enabled[identifierSlashSplit[0]] != nil {
		return *c.InRepoConfig.Enabled[identifierSlashSplit[0]]
	}
	if c.InRepoConfig.Enabled["*"] != nil {
		return *c.InRepoConfig.Enabled["*"]
	}
	logrus.Infof("in-repo configuration not enabled for %s", identifier)
	return false
}
