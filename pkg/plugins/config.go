/*
Copyright 2018 The Kubernetes Authors.

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

package plugins

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/labels"
	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	failOnMissingPlugin = false
)

// Configuration is the top-level serialization target for plugin Configuration.
type Configuration struct {
	// Plugins is a map of repositories (eg "k/k") to lists of
	// plugin names.
	// TODO: Link to the list of supported plugins.
	// https://github.com/kubernetes/test-infra/issues/3476
	Plugins map[string][]string `json:"plugins,omitempty"`

	// ExternalPlugins is a map of repositories (eg "k/k") to lists of
	// external plugins.
	ExternalPlugins map[string][]ExternalPlugin `json:"external_plugins,omitempty"`

	// Owners contains configuration related to handling OWNERS files.
	Owners Owners `json:"owners,omitempty"`

	// Built-in plugins specific configuration.
	Approve              []Approve              `json:"approve,omitempty"`
	Blockades            []Blockade             `json:"blockades,omitempty"`
	Cat                  Cat                    `json:"cat,omitempty"`
	CherryPickUnapproved CherryPickUnapproved   `json:"cherry_pick_unapproved,omitempty"`
	ConfigUpdater        ConfigUpdater          `json:"config_updater,omitempty"`
	Label                Label                  `json:"label,omitempty"`
	Lgtm                 []Lgtm                 `json:"lgtm,omitempty"`
	RepoMilestone        map[string]Milestone   `json:"repo_milestone,omitempty"`
	RequireMatchingLabel []RequireMatchingLabel `json:"require_matching_label,omitempty"`
	RequireSIG           RequireSIG             `json:"requiresig,omitempty"`
	SigMention           SigMention             `json:"sigmention,omitempty"`
	Size                 Size                   `json:"size,omitempty"`
	Triggers             []Trigger              `json:"triggers,omitempty"`
	Welcome              []Welcome              `json:"welcome,omitempty"`
}

// ExternalPlugin holds configuration for registering an external
// plugin in prow.
type ExternalPlugin struct {
	// Name of the plugin.
	Name string `json:"name"`
	// Endpoint is the location of the external plugin. Defaults to
	// the name of the plugin, ie. "http://{{name}}".
	Endpoint string `json:"endpoint,omitempty"`
	// Events are the events that need to be demuxed by the hook
	// server to the external plugin. If no events are specified,
	// everything is sent.
	Events []string `json:"events,omitempty"`
}

// Owners contains configuration related to handling OWNERS files.
type Owners struct {
	// MDYAMLRepos is a list of org and org/repo strings specifying the repos that support YAML
	// OWNERS config headers at the top of markdown (*.md) files. These headers function just like
	// the config in an OWNERS file, but only apply to the file itself instead of the entire
	// directory and all sub-directories.
	// The yaml header must be at the start of the file and be bracketed with "---" like so:
	/*
		---
		approvers:
		- mikedanese
		- thockin
		---
	*/
	MDYAMLRepos []string `json:"mdyamlrepos,omitempty"`
	// SkipCollaborators disables collaborator cross-checks and forces both
	// the approve and lgtm plugins to use solely OWNERS files for access
	// control in the provided repos.
	SkipCollaborators []string `json:"skip_collaborators,omitempty"`
	// LabelsExcludeList holds a list of labels that should not be present in any
	// OWNERS file, preventing their automatic addition by the owners-label plugin.
	// This check is performed by the verify-owners plugin.
	LabelsExcludeList []string `json:"labels_excludes,omitempty"`
}

// MDYAMLEnabled returns a boolean denoting if the passed repo supports YAML OWNERS config headers
// at the top of markdown (*.md) files. These function like OWNERS files but only apply to the file
// itself.
func (c *Configuration) MDYAMLEnabled(org, repo string) bool {
	full := fmt.Sprintf("%s/%s", org, repo)
	for _, elem := range c.Owners.MDYAMLRepos {
		if elem == org || elem == full {
			return true
		}
	}
	return false
}

// SkipCollaborators returns a boolean denoting if collaborator cross-checks are enabled for
// the passed repo. If it's true, approve and lgtm plugins rely solely on OWNERS files.
func (c *Configuration) SkipCollaborators(org, repo string) bool {
	full := fmt.Sprintf("%s/%s", org, repo)
	for _, elem := range c.Owners.SkipCollaborators {
		if elem == org || elem == full {
			return true
		}
	}
	return false
}

// RequireSIG specifies configuration for the require-sig plugin.
type RequireSIG struct {
	// GroupListURL is the URL where a list of the available SIGs can be found.
	GroupListURL string `json:"group_list_url,omitempty"`
}

// SigMention specifies configuration for the sigmention plugin.
type SigMention struct {
	// Regexp parses comments and should return matches to team mentions.
	// These mentions enable labeling issues or PRs with sig/team labels.
	// Furthermore, teams with the following suffixes will be mapped to
	// kind/* labels:
	//
	// * @org/team-bugs             --maps to--> kind/bug
	// * @org/team-feature-requests --maps to--> kind/feature
	// * @org/team-api-reviews      --maps to--> kind/api-change
	// * @org/team-proposals        --maps to--> kind/design
	//
	// Note that you need to make sure your regexp covers the above
	// mentions if you want to use the extra labeling. Defaults to:
	// (?m)@kubernetes/sig-([\w-]*)-(misc|test-failures|bugs|feature-requests|proposals|pr-reviews|api-reviews)
	//
	// Compiles into Re during config load.
	Regexp string         `json:"regexp,omitempty"`
	Re     *regexp.Regexp `json:"-"`
}

// Size specifies configuration for the size plugin, defining lower bounds (in # lines changed) for each size label.
// XS is assumed to be zero.
type Size struct {
	S   int `json:"s"`
	M   int `json:"m"`
	L   int `json:"l"`
	Xl  int `json:"xl"`
	Xxl int `json:"xxl"`
}

// Blockade specifies a configuration for a single blockade.
//
// The configuration for the blockade plugin is defined as a list of these structures.
type Blockade struct {
	// Repos are either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// BlockRegexps are regular expressions matching the file paths to block.
	BlockRegexps []string `json:"blockregexps,omitempty"`
	// ExceptionRegexps are regular expressions matching the file paths that are exceptions to the BlockRegexps.
	ExceptionRegexps []string `json:"exceptionregexps,omitempty"`
	// Explanation is a string that will be included in the comment left when blocking a PR. This should
	// be an explanation of why the paths specified are blockaded.
	Explanation string `json:"explanation,omitempty"`
}

// Approve specifies a configuration for a single approve.
//
// The configuration for the approve plugin is defined as a list of these structures.
type Approve struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// IssueRequired indicates if an associated issue is required for approval in
	// the specified repos.
	IssueRequired bool `json:"issue_required,omitempty"`
	// RequireSelfApproval requires PR authors to explicitly approve their PRs.
	// Otherwise the plugin assumes the author of the PR approves the changes in the PR.
	RequireSelfApproval *bool `json:"require_self_approval,omitempty"`
	// LgtmActsAsApprove indicates that the lgtm command should be used to
	// indicate approval
	LgtmActsAsApprove bool `json:"lgtm_acts_as_approve,omitempty"`
	// IgnoreReviewState causes the approve plugin to ignore the GitHub review state. Otherwise:
	// * an APPROVE github review is equivalent to leaving an "/approve" message.
	// * A REQUEST_CHANGES github review is equivalent to leaving an /approve cancel" message.
	IgnoreReviewState *bool `json:"ignore_review_state,omitempty"`
	// IgnoreUpdateBot makes the approve plugin ignore PRs with the label updatebot
	IgnoreUpdateBot *bool `json:"ignore_updatebot,omitempty"`
}

// HasSelfApproval checks if it has self-approval
func (a Approve) HasSelfApproval() bool {
	if a.RequireSelfApproval != nil {
		return !*a.RequireSelfApproval
	}
	return true
}

// ConsiderReviewState checks if the rewview state is active
func (a Approve) ConsiderReviewState() bool {
	if a.IgnoreReviewState != nil {
		return !*a.IgnoreReviewState
	}
	return true
}

// ConsiderReviewState checks if the rewview state is active
func (a Approve) IgnoreUpdateBotLabel() bool {
	if a.IgnoreUpdateBot != nil {
		return *a.IgnoreUpdateBot
	}
	return true
}

// Lgtm specifies a configuration for a single lgtm.
// The configuration for the lgtm plugin is defined as a list of these structures.
type Lgtm struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// ReviewActsAsLgtm indicates that a Github review of "approve" or "request changes"
	// acts as adding or removing the lgtm label
	ReviewActsAsLgtm bool `json:"review_acts_as_lgtm,omitempty"`
	// StoreTreeHash indicates if tree_hash should be stored inside a comment to detect
	// squashed commits before removing lgtm labels
	StoreTreeHash bool `json:"store_tree_hash,omitempty"`
	// WARNING: This disables the security mechanism that prevents a malicious member (or
	// compromised GitHub account) from merging arbitrary code. Use with caution.
	//
	// StickyLgtmTeam specifies the Github team whose members are trusted with sticky LGTM,
	// which eliminates the need to re-lgtm minor fixes/updates.
	StickyLgtmTeam string `json:"trusted_team_for_sticky_lgtm,omitempty"`
}

// Cat contains the configuration for the cat plugin.
type Cat struct {
	// Path to file containing an api key for thecatapi.com
	KeyPath string `json:"key_path,omitempty"`
}

// Label contains the configuration for the label plugin.
type Label struct {
	// AdditionalLabels is a set of additional labels enabled for use
	// on top of the existing "kind/*", "priority/*", and "area/*" labels.
	AdditionalLabels []string `json:"additional_labels"`
}

// Trigger specifies a configuration for a single trigger.
//
// The configuration for the trigger plugin is defined as a list of these structures.
type Trigger struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// TrustedOrg is the org whose members' PRs will be automatically built
	// for PRs to the above repos. The default is the PR's org.
	TrustedOrg string `json:"trusted_org,omitempty"`
	// TrustedApps is the explicit list of GitHub apps whose PRs will be automatically
	// considered as trusted. The list should contain usernames of each GitHub App without [bot] suffix.
	// By default, trigger will ignore this list.
	TrustedApps []string `json:"trusted_apps,omitempty"`
	// JoinOrgURL is a link that redirects users to a location where they
	// should be able to read more about joining the organization in order
	// to become trusted members. Defaults to the Github link of TrustedOrg.
	JoinOrgURL string `json:"join_org_url,omitempty"`
	// OnlyOrgMembers requires PRs and/or /ok-to-test comments to come from org members.
	// By default, trigger also include repo collaborators.
	OnlyOrgMembers bool `json:"only_org_members,omitempty"`
	// IgnoreOkToTest makes trigger ignore /ok-to-test comments.
	// This is a security mitigation to only allow testing from trusted users.
	IgnoreOkToTest bool `json:"ignore_ok_to_test,omitempty"`
	// ElideSkippedContexts makes trigger not post "Skipped" contexts for jobs
	// that could run but do not run.
	ElideSkippedContexts bool `json:"elide_skipped_contexts,omitempty"`
	// SkipDraftPR when enabled, skips triggering pipelines for draft PRs, unless /ok-to-test is added.
	SkipDraftPR bool `json:"skip_draft_pr,omitempty"`
}

// Milestone contains the configuration options for the milestone and
// milestonestatus plugins.
type Milestone struct {
	// ID of the github team for the milestone maintainers (used for setting status labels)
	// You can curl the following endpoint in order to determine the gitprovider.ID of your team
	// responsible for maintaining the milestones:
	// curl -H "Authorization: token <token>" https://api.github.com/orgs/<org-name>/teams
	MaintainersID           int    `json:"maintainers_id,omitempty"`
	MaintainersTeam         string `json:"maintainers_team,omitempty"`
	MaintainersFriendlyName string `json:"maintainers_friendly_name,omitempty"`
}

// ConfigMapSpec contains configuration options for the configMap being updated
// by the config-updater plugin.
type ConfigMapSpec struct {
	// Name of ConfigMap
	Name string `json:"name"`
	// Key is the key in the ConfigMap to update with the file contents.
	// If no explicit key is given, the basename of the file will be used.
	Key string `json:"key,omitempty"`
	// Namespace in which the configMap needs to be deployed. If no namespace is specified
	// it will be deployed to the LighthouseJobNamespace.
	Namespace string `json:"namespace,omitempty"`
	// Namespaces in which the configMap needs to be deployed, in addition to the above
	// namespace provided, or the default if it is not set.
	AdditionalNamespaces []string `json:"additional_namespaces,omitempty"`
	// GZIP toggles whether the key's data should be GZIP'd before being stored
	// If set to false and the global GZIP option is enabled, this file will
	// will not be GZIP'd.
	GZIP *bool `json:"gzip,omitempty"`
	// Namespaces is the fully resolved list of Namespaces to deploy the ConfigMap in
	Namespaces []string `json:"-"`
}

// ConfigUpdater contains the configuration for the config-updater plugin.
type ConfigUpdater struct {
	// A map of filename => ConfigMapSpec.
	// Whenever a commit changes filename, prow will update the corresponding configmap.
	// map[string]ConfigMapSpec{ "/my/path.yaml": {Name: "foo", Namespace: "otherNamespace" }}
	// will result in replacing the foo configmap whenever path.yaml changes
	Maps map[string]ConfigMapSpec `json:"maps,omitempty"`
	// If GZIP is true then files will be gzipped before insertion into
	// their corresponding configmap
	GZIP bool `json:"gzip"`
}

// Welcome is config for the welcome plugin.
type Welcome struct {
	// Repos is either of the form org/repos or just org.
	Repos []string `json:"repos,omitempty"`
	// MessageTemplate is the welcome message template to post on new-contributor PRs
	// For the info struct see prow/plugins/welcome/welcome.go's PRInfo
	MessageTemplate string `json:"message_template,omitempty"`
}

// CherryPickUnapproved is the config for the cherrypick-unapproved plugin.
type CherryPickUnapproved struct {
	// BranchRegexp is the regular expression for branch names such that
	// the plugin treats only PRs against these branch names as cherrypick PRs.
	// Compiles into BranchRe during config load.
	BranchRegexp string         `json:"branchregexp,omitempty"`
	BranchRe     *regexp.Regexp `json:"-"`
	// Comment is the comment added by the plugin while adding the
	// `do-not-merge/cherry-pick-not-approved` label.
	Comment string `json:"comment,omitempty"`
}

// RequireMatchingLabel is the config for the require-matching-label plugin.
type RequireMatchingLabel struct {
	// Org is the GitHub organization that this config applies to.
	Org string `json:"org,omitempty"`
	// Repo is the GitHub repository within Org that this config applies to.
	// This fields may be omitted to apply this config across all repos in Org.
	Repo string `json:"repo,omitempty"`
	// Branch is the branch ref of PRs that this config applies to.
	// This field is only valid if `prs: true` and may be omitted to apply this
	// config across all branches in the repo or org.
	Branch string `json:"branch,omitempty"`
	// PRs is a bool indicating if this config applies to PRs.
	PRs bool `json:"prs,omitempty"`
	// Issues is a bool indicating if this config applies to issues.
	Issues bool `json:"issues,omitempty"`

	// Regexp is the string specifying the regular expression used to look for
	// matching labels.
	Regexp string `json:"regexp,omitempty"`
	// Re is the compiled version of Regexp. It should not be specified in config.
	Re *regexp.Regexp `json:"-"`

	// MissingLabel is the label to apply if an issue does not have any label
	// matching the Regexp.
	MissingLabel string `json:"missing_label,omitempty"`
	// MissingComment is the comment to post when we add the MissingLabel to an
	// issue. This is typically used to explain why MissingLabel was added and
	// how to move forward.
	// This field is optional. If unspecified, no comment is created when labeling.
	MissingComment string `json:"missing_comment,omitempty"`

	// GracePeriod is the amount of time to wait before processing newly opened
	// or reopened issues and PRs. This delay allows other automation to apply
	// labels before we look for matching labels.
	// Defaults to '5s'.
	GracePeriod         string        `json:"grace_period,omitempty"`
	GracePeriodDuration time.Duration `json:"-"`
}

// validate checks the following properties:
// - Org, Regexp, MissingLabel, and GracePeriod must be non-empty.
// - Repo does not contain a '/' (should use Org+Repo).
// - At least one of PRs or Issues must be true.
// - Branch only specified if 'prs: true'
// - MissingLabel must not match Regexp.
func (r RequireMatchingLabel) validate() error {
	if r.Org == "" {
		return errors.New("must specify 'org'")
	}
	if strings.Contains(r.Repo, "/") {
		return errors.New("'repo' may not contain '/'; specify the organization with 'org'")
	}
	if r.Regexp == "" {
		return errors.New("must specify 'regexp'")
	}
	if r.MissingLabel == "" {
		return errors.New("must specify 'missing_label'")
	}
	if r.GracePeriod == "" {
		return errors.New("must specify 'grace_period'")
	}
	if !r.PRs && !r.Issues {
		return errors.New("must specify 'prs: true' and/or 'issues: true'")
	}
	if !r.PRs && r.Branch != "" {
		return errors.New("branch cannot be specified without `prs: true'")
	}
	if r.Re.MatchString(r.MissingLabel) {
		return errors.New("'regexp' must not match 'missing_label'")
	}
	return nil
}

// Describe generates a human readable description of the behavior that this
// configuration specifies.
func (r RequireMatchingLabel) Describe() string {
	str := &strings.Builder{}
	fmt.Fprintf(str, "Applies the '%s' label ", r.MissingLabel)
	if r.MissingComment == "" {
		fmt.Fprint(str, "to ")
	} else {
		fmt.Fprint(str, "and comments on ")
	}

	if r.Issues {
		fmt.Fprint(str, "Issues ")
		if r.PRs {
			fmt.Fprint(str, "and ")
		}
	}
	if r.PRs {
		if r.Branch != "" {
			fmt.Fprintf(str, "'%s' branch ", r.Branch)
		}
		fmt.Fprint(str, "PRs ")
	}

	if r.Repo == "" {
		fmt.Fprintf(str, "in the '%s' GitHub org ", r.Org)
	} else {
		fmt.Fprintf(str, "in the '%s/%s' GitHub repo ", r.Org, r.Repo)
	}
	fmt.Fprintf(str, "that have no labels matching the regular expression '%s'.", r.Regexp)
	return str.String()
}

// TriggerFor finds the Trigger for a repo, if one exists
// a trigger can be listed for the repo itself or for the
// owning organization
func (c *Configuration) TriggerFor(org, repo string) *Trigger {
	for _, tr := range c.Triggers {
		for _, r := range tr.Repos {
			if r == org || r == fmt.Sprintf("%s/%s", org, repo) {
				return &tr
			}
		}
	}
	return &Trigger{}
}

// EnabledReposForPlugin returns the orgs and repos that have enabled the passed plugin.
func (c *Configuration) EnabledReposForPlugin(plugin string) (orgs, repos []string) {
	for repo, plugins := range c.Plugins {
		found := false
		for _, candidate := range plugins {
			if candidate == plugin {
				found = true
				break
			}
		}
		if found {
			if strings.Contains(repo, "/") {
				repos = append(repos, repo)
			} else {
				orgs = append(orgs, repo)
			}
		}
	}
	return
}

// EnabledReposForExternalPlugin returns the orgs and repos that have enabled the passed
// external plugin.
func (c *Configuration) EnabledReposForExternalPlugin(plugin string) (orgs, repos []string) {
	for repo, plugins := range c.ExternalPlugins {
		found := false
		for _, candidate := range plugins {
			if candidate.Name == plugin {
				found = true
				break
			}
		}
		if found {
			if strings.Contains(repo, "/") {
				repos = append(repos, repo)
			} else {
				orgs = append(orgs, repo)
			}
		}
	}
	return
}

// SetDefaults sets default options for config updating
func (c *ConfigUpdater) SetDefaults() {
	if len(c.Maps) == 0 {
		cf := "prow/config.yaml"
		pf := "prow/plugins.yaml"
		c.Maps = map[string]ConfigMapSpec{
			cf: {
				Name: "config",
			},
			pf: {
				Name: "plugins",
			},
		}
	}

	for name, spec := range c.Maps {
		spec.Namespaces = append([]string{spec.Namespace}, spec.AdditionalNamespaces...)
		c.Maps[name] = spec
	}
}

func (c *Configuration) setDefaults() {
	c.ConfigUpdater.SetDefaults()

	for repo, plugins := range c.ExternalPlugins {
		for i, p := range plugins {
			if p.Endpoint != "" {
				continue
			}
			c.ExternalPlugins[repo][i].Endpoint = fmt.Sprintf("http://%s", p.Name)
		}
	}
	for i, trigger := range c.Triggers {
		if trigger.TrustedOrg == "" || trigger.JoinOrgURL != "" {
			continue
		}
		c.Triggers[i].JoinOrgURL = fmt.Sprintf("https://github.com/orgs/%s/people", trigger.TrustedOrg)
	}
	if c.SigMention.Regexp == "" {
		c.SigMention.Regexp = `(?m)@kubernetes/sig-([\w-]*)-(misc|test-failures|bugs|feature-requests|proposals|pr-reviews|api-reviews)`
	}
	if c.Owners.LabelsExcludeList == nil {
		c.Owners.LabelsExcludeList = []string{labels.Approved, labels.LGTM}
	}
	for _, milestone := range c.RepoMilestone {
		if milestone.MaintainersFriendlyName == "" {
			milestone.MaintainersFriendlyName = "SIG Chairs/TLs"
		}
	}
	if c.CherryPickUnapproved.BranchRegexp == "" {
		c.CherryPickUnapproved.BranchRegexp = `^release-.*$`
	}
	if c.CherryPickUnapproved.Comment == "" {
		c.CherryPickUnapproved.Comment = `This PR is not for the master branch but does not have the ` + "`cherry-pick-approved`" + `  label. Adding the ` + "`do-not-merge/cherry-pick-not-approved`" + `  label.

To approve the cherry-pick, please assign the patch release manager for the release branch by writing ` + "`/assign @username`" + ` in a comment when ready.

The list of patch release managers for each release can be found [here](https://git.k8s.io/sig-release/release-managers.md).`
	}

	for i, rml := range c.RequireMatchingLabel {
		if rml.GracePeriod == "" {
			c.RequireMatchingLabel[i].GracePeriod = "5s"
		}
	}
}

// ValidatePluginsArePresent takes a map with plugin names as keys and errors or logs for each configured plugin that can't be found.
func (c *Configuration) ValidatePluginsArePresent(presentPlugins map[string]interface{}) error {
	var errList []string

	for _, configuration := range c.Plugins {
		for _, plugin := range configuration {
			if _, ok := presentPlugins[plugin]; !ok {
				if failOnMissingPlugin {
					errList = append(errList, fmt.Sprintf("unknown plugin: %s", plugin))
				} else {
					logrus.WithField("plugin", plugin).Warn("unknown plugin")
				}
			}
		}
	}
	if len(errList) > 0 {
		return fmt.Errorf("invalid plugin configuration:\n\t%v", strings.Join(errList, "\n\t"))
	}
	return nil
}

// validatePlugins will return error if
// there are unknown or duplicated plugins.
func validatePlugins(plugins map[string][]string) error {
	var errList []string
	for repo, repoConfig := range plugins {
		if strings.Contains(repo, "/") {
			org := strings.Split(repo, "/")[0]
			if dupes := findDuplicatedPluginConfig(repoConfig, plugins[org]); len(dupes) > 0 {
				errList = append(errList, fmt.Sprintf("plugins %v are duplicated for %s and %s", dupes, repo, org))
			}
		}
	}

	if len(errList) > 0 {
		return fmt.Errorf("invalid plugin configuration:\n\t%v", strings.Join(errList, "\n\t"))
	}
	return nil
}

func validateSizes(size Size) error {
	if size.S > size.M || size.M > size.L || size.L > size.Xl || size.Xl > size.Xxl {
		return errors.New("invalid size plugin configuration - one of the smaller sizes is bigger than a larger one")
	}

	return nil
}

func findDuplicatedPluginConfig(repoConfig, orgConfig []string) []string {
	var dupes []string
	for _, repoPlugin := range repoConfig {
		for _, orgPlugin := range orgConfig {
			if repoPlugin == orgPlugin {
				dupes = append(dupes, repoPlugin)
			}
		}
	}

	return dupes
}

func validateExternalPlugins(pluginMap map[string][]ExternalPlugin) error {
	var errors []string

	for repo, plugins := range pluginMap {
		if !strings.Contains(repo, "/") {
			continue
		}
		org := strings.Split(repo, "/")[0]

		var orgConfig []string
		for _, p := range pluginMap[org] {
			orgConfig = append(orgConfig, p.Name)
		}

		var repoConfig []string
		for _, p := range plugins {
			repoConfig = append(repoConfig, p.Name)
		}

		if dupes := findDuplicatedPluginConfig(repoConfig, orgConfig); len(dupes) > 0 {
			errors = append(errors, fmt.Sprintf("external plugins %v are duplicated for %s and %s", dupes, repo, org))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("invalid plugin configuration:\n\t%v", strings.Join(errors, "\n\t"))
	}
	return nil
}

func validateConfigUpdater(updater *ConfigUpdater) error {
	files := sets.NewString()
	configMapKeys := map[string]sets.String{}
	for file, config := range updater.Maps {
		if files.Has(file) {
			return fmt.Errorf("file %s listed more than once in config updater config", file)
		}
		files.Insert(file)

		key := config.Key
		if key == "" {
			key = path.Base(file)
		}

		if _, ok := configMapKeys[config.Name]; ok {
			if configMapKeys[config.Name].Has(key) {
				return fmt.Errorf("key %s in configmap %s updated with more than one file", key, config.Name)
			}
			configMapKeys[config.Name].Insert(key)
		} else {
			configMapKeys[config.Name] = sets.NewString(key)
		}
	}
	return nil
}

func validateRequireMatchingLabel(rs []RequireMatchingLabel) error {
	for i, r := range rs {
		if err := r.validate(); err != nil {
			return fmt.Errorf("error validating require_matching_label config #%d: %v", i, err)
		}
	}
	return nil
}

func compileRegexpsAndDurations(pc *Configuration) error {
	cRe, err := regexp.Compile(pc.SigMention.Regexp)
	if err != nil {
		return err
	}
	pc.SigMention.Re = cRe

	branchRe, err := regexp.Compile(pc.CherryPickUnapproved.BranchRegexp)
	if err != nil {
		return err
	}
	pc.CherryPickUnapproved.BranchRe = branchRe

	rs := pc.RequireMatchingLabel
	for i := range rs {
		re, err := regexp.Compile(rs[i].Regexp)
		if err != nil {
			return fmt.Errorf("failed to compile label regexp: %q, error: %v", rs[i].Regexp, err)
		}
		rs[i].Re = re

		var dur time.Duration
		dur, err = time.ParseDuration(rs[i].GracePeriod)
		if err != nil {
			return fmt.Errorf("failed to compile grace period duration: %q, error: %v", rs[i].GracePeriod, err)
		}
		rs[i].GracePeriodDuration = dur
	}
	return nil
}

// Validate validates the plugin configuration
func (c *Configuration) Validate() error {
	if len(c.Plugins) == 0 {
		logrus.Warn("no plugins specified-- check syntax?")
	}

	// Defaulting should run before validation.
	c.setDefaults()
	// Regexp compilation should run after defaulting, but before validation.
	if err := compileRegexpsAndDurations(c); err != nil {
		return err
	}

	if err := validatePlugins(c.Plugins); err != nil {
		return err
	}
	if err := validateExternalPlugins(c.ExternalPlugins); err != nil {
		return err
	}
	if err := validateConfigUpdater(&c.ConfigUpdater); err != nil {
		return err
	}
	if err := validateSizes(c.Size); err != nil {
		return err
	}
	if err := validateRequireMatchingLabel(c.RequireMatchingLabel); err != nil {
		return err
	}

	return nil
}
