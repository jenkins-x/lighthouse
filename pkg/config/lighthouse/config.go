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
	// Pub/Sub Subscriptions that we want to listen to
	PubSubSubscriptions PubsubSubscriptions `json:"pubsub_subscriptions,omitempty"`
	// GitHubOptions allows users to control how prow applications display GitHub website links.
	GitHubOptions GitHubOptions `json:"github,omitempty"`
	// ProviderConfig contains optional SCM provider information
	ProviderConfig *ProviderConfig `json:"providerConfig,omitempty"`
}

// Parse initializes and validates the Config
func (c *Config) Parse() error {
	if err := c.Plank.Parse(); err != nil {
		return err
	}
	for i := range c.JenkinsOperators {
		if err := c.JenkinsOperators[i].Parse(); err != nil {
			return err
		}
		// TODO: Invalidate overlapping selectors more
		if len(c.JenkinsOperators) > 1 && c.JenkinsOperators[i].LabelSelectorString == "" {
			return errors.New("selector overlap: cannot use an empty label_selector with multiple selectors")
		}
		if len(c.JenkinsOperators) == 1 && c.JenkinsOperators[0].LabelSelectorString != "" {
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
