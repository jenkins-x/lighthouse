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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config/branchprotection"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/config/keeper"
	"github.com/jenkins-x/lighthouse/pkg/config/lighthouse"
	"github.com/jenkins-x/lighthouse/pkg/config/util"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
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

// JobConfig is a type alias for job.Config
type JobConfig = job.Config

// ProwConfig is a type alias for lighthouse.Config
type ProwConfig = lighthouse.Config

// Config is a read-only snapshot of the config.
type Config struct {
	JobConfig
	ProwConfig
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
		var jc job.Config
		if err := yamlToConfig(jobConfig, &jc); err != nil {
			return nil, err
		}
		if err := nc.Merge(jc); err != nil {
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

		var subConfig job.Config
		if err := yamlToConfig(path, &subConfig); err != nil {
			return err
		}
		return nc.Merge(subConfig)
	})

	if err != nil {
		return nil, err
	}

	return &nc, nil
}

// yamlToConfig converts a yaml file into a Config object
func yamlToConfig(path string, nc interface{}) error {
	b, err := os.ReadFile(path) // #nosec
	if err != nil {
		return fmt.Errorf("error reading %s: %v", path, err)
	}
	if err := yaml.Unmarshal(b, nc); err != nil {
		return fmt.Errorf("error unmarshaling %s: %v", path, err)
	}
	var jc *job.Config
	switch v := nc.(type) {
	case *job.Config:
		jc = v
	case *Config:
		jc = &v.JobConfig
	}
	for rep := range jc.Presubmits {
		fix := func(job *job.Presubmit) {
			job.SourcePath = path
		}
		for i := range jc.Presubmits[rep] {
			fix(&jc.Presubmits[rep][i])
		}
	}
	for rep := range jc.Postsubmits {
		fix := func(job *job.Postsubmit) {
			job.SourcePath = path
		}
		for i := range jc.Postsubmits[rep] {
			fix(&jc.Postsubmits[rep][i])
		}
	}

	fix := func(job *job.Periodic) {
		job.SourcePath = path
	}
	for i := range jc.Periodics {
		fix(&jc.Periodics[i])
	}
	return nil
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

func parseProwConfig(c *Config) error {
	if err := c.ProwConfig.Parse(); err != nil {
		return err
	}
	lvl, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		return err
	}
	logrus.WithField("level", lvl.String()).Infof("setting the log level")
	logrus.SetLevel(lvl)
	return nil
}

// finalizeAndValidate sets default configurations, validates the configuration, etc
func (c *Config) finalizeAndValidate() (*Config, error) {
	if err := c.JobConfig.Init(c.ProwConfig); err != nil {
		return nil, err
	}
	if err := c.JobConfig.Validate(c.ProwConfig); err != nil {
		return nil, err
	}
	return c, nil
}

// GetPostsubmits lets return all the post submits
func (c *Config) GetPostsubmits(repository scm.Repository) []job.Postsubmit {
	fullNames := util.FullNames(repository)
	var answer []job.Postsubmit
	for _, fn := range fullNames {
		answer = append(answer, c.Postsubmits[fn]...)
	}
	return answer
}

// GetDeployments lets return all the deployments
func (c *Config) GetDeployments(repository scm.Repository) []job.Deployment {
	fullNames := util.FullNames(repository)
	var answer []job.Deployment
	for _, fn := range fullNames {
		answer = append(answer, c.Deployments[fn]...)
	}
	return answer
}

// GetPresubmits lets return all the pre submits for the given repo
func (c *Config) GetPresubmits(repository scm.Repository) []job.Presubmit {
	fullNames := util.FullNames(repository)
	var answer []job.Presubmit
	for _, fn := range fullNames {
		answer = append(answer, c.Presubmits[fn]...)
	}
	return answer
}

// BranchRequirements partitions status contexts for a given org, repo branch into three buckets:
//   - contexts that are always required to be present
//   - contexts that are required, _if_ present
//   - contexts that are always optional
func BranchRequirements(org, repo, branch string, presubmits map[string][]job.Presubmit) ([]string, []string, []string) {
	jobs, ok := presubmits[org+"/"+repo]
	if !ok {
		return nil, nil, nil
	}
	var required, requiredIfPresent, optional []string
	for _, j := range jobs {
		if !j.CouldRun(branch) {
			continue
		}

		if j.ContextRequired() {
			if j.TriggersConditionally() {
				// jobs that trigger conditionally cannot be
				// required as their status may not exist on PRs
				requiredIfPresent = append(requiredIfPresent, j.Context)
			} else {
				// jobs that produce required contexts and will
				// always run should be required at all times
				required = append(required, j.Context)
			}
		} else {
			optional = append(optional, j.Context)
		}
	}
	return required, requiredIfPresent, optional
}

// GetBranchProtection returns the policy for a given branch.
//
// Handles merging any policies defined at repo/org/global levels into the branch policy.
func (c *Config) GetBranchProtection(org, repo, branch string) (*branchprotection.Policy, error) {
	if _, present := c.BranchProtection.Orgs[org]; !present {
		return nil, nil // only consider branches in configured orgs
	}
	b, err := c.BranchProtection.GetOrg(org).GetRepo(repo).GetBranch(branch)
	if err != nil {
		return nil, err
	}

	return c.GetPolicy(org, repo, branch, *b)
}

// GetPolicy returns the protection policy for the branch, after merging in presubmits.
func (c *Config) GetPolicy(org, repo, branch string, b branchprotection.Branch) (*branchprotection.Policy, error) {
	policy := b.Policy

	// Automatically require contexts from prow which must always be present
	if prowContexts, _, _ := BranchRequirements(org, repo, branch, c.Presubmits); len(prowContexts) > 0 {
		// Error if protection is disabled
		if policy.Protect != nil && !*policy.Protect {
			if c.BranchProtection.AllowDisabledJobPolicies {
				logrus.Warnf("%s/%s=%s has required jobs but has protect: false", org, repo, branch)
				return nil, nil
			}
			return nil, fmt.Errorf("required prow jobs require branch protection")

		}
		ps := branchprotection.Policy{
			RequiredStatusChecks: &branchprotection.ContextPolicy{
				Contexts: prowContexts,
			},
		}
		// Require protection by default if ProtectTested is true
		if c.BranchProtection.ProtectTested {
			yes := true
			ps.Protect = &yes
		}
		policy = policy.Apply(ps)
	}

	if policy.Protect != nil && !*policy.Protect {
		// Ensure that protection is false => no protection settings
		var old *bool
		old, policy.Protect = policy.Protect, old
		switch {
		case policy.IsDefined() && c.BranchProtection.AllowDisabledPolicies:
			logrus.Warnf("%s/%s=%s defines a policy but has protect: false", org, repo, branch)
			policy = branchprotection.Policy{
				Protect: policy.Protect,
			}
		case policy.IsDefined():
			return nil, fmt.Errorf("%s/%s=%s defines a policy, which requires protect: true", org, repo, branch)
		}
		policy.Protect = old
	}

	if !policy.IsDefined() {
		return nil, nil
	}
	return &policy, nil
}

// GetKeeperContextPolicy parses the prow config to find context merge options.
// If none are set, it will use the prow jobs configured and use the default github combined status.
// Otherwise if set it will use the branch protection setting, or the listed jobs.
func (c Config) GetKeeperContextPolicy(org, repo, branch string) (*keeper.ContextPolicy, error) {
	options := c.Keeper.ContextOptions.Parse(org, repo, branch)
	// Adding required and optional contexts from options
	required := sets.NewString(options.RequiredContexts...)
	requiredIfPresent := sets.NewString(options.RequiredIfPresentContexts...)
	optional := sets.NewString(options.OptionalContexts...)

	// automatically generate required and optional entries for Prow Pipelines
	prowRequired, prowRequiredIfPresent, prowOptional := BranchRequirements(org, repo, branch, c.Presubmits)
	required.Insert(prowRequired...)
	requiredIfPresent.Insert(prowRequiredIfPresent...)
	optional.Insert(prowOptional...)

	// Using Branch protection configuration
	if options.FromBranchProtection != nil && *options.FromBranchProtection {
		bp, err := c.GetBranchProtection(org, repo, branch)
		if err != nil {
			logrus.WithError(err).Warningf("Error getting branch protection for %s/%s+%s", org, repo, branch)
		} else if bp != nil && bp.Protect != nil && *bp.Protect && bp.RequiredStatusChecks != nil {
			required.Insert(bp.RequiredStatusChecks.Contexts...)
		}
	}

	// Remove anything from the required list that's also in the required if present list, since that may have been
	// duplicated by branch protection.
	required.Delete(requiredIfPresent.List()...)

	t := &keeper.ContextPolicy{
		RequiredContexts:          required.List(),
		RequiredIfPresentContexts: requiredIfPresent.List(),
		OptionalContexts:          optional.List(),
		SkipUnknownContexts:       options.SkipUnknownContexts,
	}
	if err := t.Validate(); err != nil {
		return t, err
	}
	return t, nil
}

// VolumeMounts returns a string slice with *MountName consts in it.
func VolumeMounts() []string {
	return []string{logMountName, codeMountName, toolsMountName, gcsCredentialsMountName}
}

// VolumeMountPaths returns a string slice with *MountPath consts in it.
func VolumeMountPaths() []string {
	return []string{logMountPath, codeMountPath, toolsMountPath, gcsCredentialsMountPath}
}
