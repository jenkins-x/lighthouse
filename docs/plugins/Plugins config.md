# Plugins config

- [Approve](#Approve)
- [Blockade](#Blockade)
- [Cat](#Cat)
- [CherryPickUnapproved](#CherryPickUnapproved)
- [ConfigMapSpec](#ConfigMapSpec)
- [ConfigUpdater](#ConfigUpdater)
- [Configuration](#Configuration)
- [ExternalPlugin](#ExternalPlugin)
- [Label](#Label)
- [Lgtm](#Lgtm)
- [Milestone](#Milestone)
- [Owners](#Owners)
- [RequireMatchingLabel](#RequireMatchingLabel)
- [RequireSIG](#RequireSIG)
- [SigMention](#SigMention)
- [Size](#Size)
- [Trigger](#Trigger)
- [Welcome](#Welcome)


## Approve

Approve specifies a configuration for a single approve.<br /><br />The configuration for the approve plugin is defined as a list of these structures.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Repos | `repos` | []string | No | Repos is either of the form org/repos or just org. |
| IssueRequired | `issue_required` | bool | No | IssueRequired indicates if an associated issue is required for approval in<br />the specified repos. |
| RequireSelfApproval | `require_self_approval` | *bool | No | RequireSelfApproval requires PR authors to explicitly approve their PRs.<br />Otherwise the plugin assumes the author of the PR approves the changes in the PR. |
| LgtmActsAsApprove | `lgtm_acts_as_approve` | bool | No | LgtmActsAsApprove indicates that the lgtm command should be used to<br />indicate approval |
| IgnoreReviewState | `ignore_review_state` | *bool | No | IgnoreReviewState causes the approve plugin to ignore the GitHub review state. Otherwise:<br />* an APPROVE github review is equivalent to leaving an "/approve" message.<br />* A REQUEST_CHANGES github review is equivalent to leaving an /approve cancel" message. |

## Blockade

Blockade specifies a configuration for a single blockade.<br /><br />The configuration for the blockade plugin is defined as a list of these structures.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Repos | `repos` | []string | No | Repos are either of the form org/repos or just org. |
| BlockRegexps | `blockregexps` | []string | No | BlockRegexps are regular expressions matching the file paths to block. |
| ExceptionRegexps | `exceptionregexps` | []string | No | ExceptionRegexps are regular expressions matching the file paths that are exceptions to the BlockRegexps. |
| Explanation | `explanation` | string | No | Explanation is a string that will be included in the comment left when blocking a PR. This should<br />be an explanation of why the paths specified are blockaded. |

## Cat

Cat contains the configuration for the cat plugin.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| KeyPath | `key_path` | string | No | Path to file containing an api key for thecatapi.com |

## CherryPickUnapproved

CherryPickUnapproved is the config for the cherrypick-unapproved plugin.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| BranchRegexp | `branchregexp` | string | No | BranchRegexp is the regular expression for branch names such that<br />the plugin treats only PRs against these branch names as cherrypick PRs.<br />Compiles into BranchRe during config load. |
| BranchRe | `-` | *regexp.Regexp | No |  |
| Comment | `comment` | string | No | Comment is the comment added by the plugin while adding the<br />`do-not-merge/cherry-pick-not-approved` label. |

## ConfigMapSpec

ConfigMapSpec contains configuration options for the configMap being updated<br />by the config-updater plugin.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Name | `name` | string | Yes | Name of ConfigMap |
| Key | `key` | string | No | Key is the key in the ConfigMap to update with the file contents.<br />If no explicit key is given, the basename of the file will be used. |
| Namespace | `namespace` | string | No | Namespace in which the configMap needs to be deployed. If no namespace is specified<br />it will be deployed to the LighthouseJobNamespace. |
| AdditionalNamespaces | `additional_namespaces` | []string | No | Namespaces in which the configMap needs to be deployed, in addition to the above<br />namespace provided, or the default if it is not set. |
| GZIP | `gzip` | *bool | No | GZIP toggles whether the key's data should be GZIP'd before being stored<br />If set to false and the global GZIP option is enabled, this file will<br />will not be GZIP'd. |
| Namespaces | `-` | []string | No | Namespaces is the fully resolved list of Namespaces to deploy the ConfigMap in |

## ConfigUpdater

ConfigUpdater contains the configuration for the config-updater plugin.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Maps | `maps` | map[string][ConfigMapSpec](#ConfigMapSpec) | No | A map of filename => ConfigMapSpec.<br />Whenever a commit changes filename, prow will update the corresponding configmap.<br />map[string]ConfigMapSpec{ "/my/path.yaml": {Name: "foo", Namespace: "otherNamespace" }}<br />will result in replacing the foo configmap whenever path.yaml changes |
| GZIP | `gzip` | bool | Yes | If GZIP is true then files will be gzipped before insertion into<br />their corresponding configmap |

## Configuration

Configuration is the top-level serialization target for plugin Configuration.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Plugins | `plugins` | map[string][]string | No | Plugins is a map of repositories (eg "k/k") to lists of<br />plugin names.<br />TODO: Link to the list of supported plugins.<br />https://github.com/kubernetes/test-infra/issues/3476 |
| ExternalPlugins | `external_plugins` | map[string][][ExternalPlugin](#ExternalPlugin) | No | ExternalPlugins is a map of repositories (eg "k/k") to lists of<br />external plugins. |
| Owners | `owners` | [Owners](#Owners) | No | Owners contains configuration related to handling OWNERS files. |
| Approve | `approve` | [][Approve](#Approve) | No | Built-in plugins specific configuration. |
| Blockades | `blockades` | [][Blockade](#Blockade) | No |  |
| Cat | `cat` | [Cat](#Cat) | No |  |
| CherryPickUnapproved | `cherry_pick_unapproved` | [CherryPickUnapproved](#CherryPickUnapproved) | No |  |
| ConfigUpdater | `config_updater` | [ConfigUpdater](#ConfigUpdater) | No |  |
| Label | `label` | [Label](#Label) | No |  |
| Lgtm | `lgtm` | [][Lgtm](#Lgtm) | No |  |
| RepoMilestone | `repo_milestone` | map[string][Milestone](#Milestone) | No |  |
| RequireMatchingLabel | `require_matching_label` | [][RequireMatchingLabel](#RequireMatchingLabel) | No |  |
| RequireSIG | `requiresig` | [RequireSIG](#RequireSIG) | No |  |
| SigMention | `sigmention` | [SigMention](#SigMention) | No |  |
| Size | `size` | [Size](#Size) | No |  |
| Triggers | `triggers` | [][Trigger](#Trigger) | No |  |
| Welcome | `welcome` | [][Welcome](#Welcome) | No |  |

## ExternalPlugin

ExternalPlugin holds configuration for registering an external<br />plugin in prow.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Name | `name` | string | Yes | Name of the plugin. |
| Endpoint | `endpoint` | string | No | Endpoint is the location of the external plugin. Defaults to<br />the name of the plugin, ie. "http://{{name}}". |
| Events | `events` | []string | No | Events are the events that need to be demuxed by the hook<br />server to the external plugin. If no events are specified,<br />everything is sent. |

## Label

Label contains the configuration for the label plugin.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| AdditionalLabels | `additional_labels` | []string | No | AdditionalLabels is a set of additional labels enabled for use<br />on top of the existing "kind/*", "priority/*", and "area/*" labels. |

## Lgtm

Lgtm specifies a configuration for a single lgtm.<br />The configuration for the lgtm plugin is defined as a list of these structures.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Repos | `repos` | []string | No | Repos is either of the form org/repos or just org. |
| ReviewActsAsLgtm | `review_acts_as_lgtm` | bool | No | ReviewActsAsLgtm indicates that a Github review of "approve" or "request changes"<br />acts as adding or removing the lgtm label |
| StoreTreeHash | `store_tree_hash` | bool | No | StoreTreeHash indicates if tree_hash should be stored inside a comment to detect<br />squashed commits before removing lgtm labels |
| StickyLgtmTeam | `trusted_team_for_sticky_lgtm` | string | No | WARNING: This disables the security mechanism that prevents a malicious member (or<br />compromised GitHub account) from merging arbitrary code. Use with caution.<br /><br />StickyLgtmTeam specifies the Github team whose members are trusted with sticky LGTM,<br />which eliminates the need to re-lgtm minor fixes/updates. |

## Milestone

Milestone contains the configuration options for the milestone and<br />milestonestatus plugins.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| MaintainersID | `maintainers_id` | int | No | ID of the github team for the milestone maintainers (used for setting status labels)<br />You can curl the following endpoint in order to determine the gitprovider.ID of your team<br />responsible for maintaining the milestones:<br />curl -H "Authorization: token <token>" https://api.github.com/orgs/<org-name>/teams |
| MaintainersTeam | `maintainers_team` | string | No |  |
| MaintainersFriendlyName | `maintainers_friendly_name` | string | No |  |

## Owners

Owners contains configuration related to handling OWNERS files.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| MDYAMLRepos | `mdyamlrepos` | []string | No | MDYAMLRepos is a list of org and org/repo strings specifying the repos that support YAML<br />OWNERS config headers at the top of markdown (*.md) files. These headers function just like<br />the config in an OWNERS file, but only apply to the file itself instead of the entire<br />directory and all sub-directories.<br />The yaml header must be at the start of the file and be bracketed with "---" like so:<br /><br />		---<br />		approvers:<br />		- mikedanese<br />		- thockin<br />		--- |
| SkipCollaborators | `skip_collaborators` | []string | No | SkipCollaborators disables collaborator cross-checks and forces both<br />the approve and lgtm plugins to use solely OWNERS files for access<br />control in the provided repos. |
| LabelsExcludeList | `labels_excludes` | []string | No | LabelsExcludeList holds a list of labels that should not be present in any<br />OWNERS file, preventing their automatic addition by the owners-label plugin.<br />This check is performed by the verify-owners plugin. |

## RequireMatchingLabel

RequireMatchingLabel is the config for the require-matching-label plugin.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Org | `org` | string | No | Org is the GitHub organization that this config applies to. |
| Repo | `repo` | string | No | Repo is the GitHub repository within Org that this config applies to.<br />This fields may be omitted to apply this config across all repos in Org. |
| Branch | `branch` | string | No | Branch is the branch ref of PRs that this config applies to.<br />This field is only valid if `prs: true` and may be omitted to apply this<br />config across all branches in the repo or org. |
| PRs | `prs` | bool | No | PRs is a bool indicating if this config applies to PRs. |
| Issues | `issues` | bool | No | Issues is a bool indicating if this config applies to issues. |
| Regexp | `regexp` | string | No | Regexp is the string specifying the regular expression used to look for<br />matching labels. |
| Re | `-` | *regexp.Regexp | No | Re is the compiled version of Regexp. It should not be specified in config. |
| MissingLabel | `missing_label` | string | No | MissingLabel is the label to apply if an issue does not have any label<br />matching the Regexp. |
| MissingComment | `missing_comment` | string | No | MissingComment is the comment to post when we add the MissingLabel to an<br />issue. This is typically used to explain why MissingLabel was added and<br />how to move forward.<br />This field is optional. If unspecified, no comment is created when labeling. |
| GracePeriod | `grace_period` | string | No | GracePeriod is the amount of time to wait before processing newly opened<br />or reopened issues and PRs. This delay allows other automation to apply<br />labels before we look for matching labels.<br />Defaults to '5s'. |
| GracePeriodDuration | `-` | time.Duration | Yes |  |

## RequireSIG

RequireSIG specifies configuration for the require-sig plugin.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| GroupListURL | `group_list_url` | string | No | GroupListURL is the URL where a list of the available SIGs can be found. |

## SigMention

SigMention specifies configuration for the sigmention plugin.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Regexp | `regexp` | string | No | Regexp parses comments and should return matches to team mentions.<br />These mentions enable labeling issues or PRs with sig/team labels.<br />Furthermore, teams with the following suffixes will be mapped to<br />kind/* labels:<br /><br />* @org/team-bugs             --maps to--> kind/bug<br />* @org/team-feature-requests --maps to--> kind/feature<br />* @org/team-api-reviews      --maps to--> kind/api-change<br />* @org/team-proposals        --maps to--> kind/design<br /><br />Note that you need to make sure your regexp covers the above<br />mentions if you want to use the extra labeling. Defaults to:<br />(?m)@kubernetes/sig-([\w-]*)-(misc|test-failures|bugs|feature-requests|proposals|pr-reviews|api-reviews)<br /><br />Compiles into Re during config load. |
| Re | `-` | *regexp.Regexp | No |  |

## Size

Size specifies configuration for the size plugin, defining lower bounds (in # lines changed) for each size label.<br />XS is assumed to be zero.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| S | `s` | int | Yes |  |
| M | `m` | int | Yes |  |
| L | `l` | int | Yes |  |
| Xl | `xl` | int | Yes |  |
| Xxl | `xxl` | int | Yes |  |

## Trigger

Trigger specifies a configuration for a single trigger.<br /><br />The configuration for the trigger plugin is defined as a list of these structures.

| Variable Name                | Stanza                            | Type     | Required | Description                                                                                                                                                                                                                           |
|------------------------------|-----------------------------------|----------|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Repos                        | `repos`                           | []string | No       | Repos is either of the form org/repos or just org.                                                                                                                                                                                    |
| TrustedOrg                   | `trusted_org`                     | string   | No       | TrustedOrg is the org whose members' PRs will be automatically built<br />for PRs to the above repos. The default is the PR's org.                                                                                                    |
| TrustedApps                  | `trusted_apps`                    | []string | No       | TrustedApps is the explicit list of GitHub apps whose PRs will be automatically<br />considered as trusted. The list should contain usernames of each GitHub App without [bot] suffix.<br/>By default, trigger will ignore this list. |
| JoinOrgURL                   | `join_org_url`                    | string   | No       | JoinOrgURL is a link that redirects users to a location where they<br />should be able to read more about joining the organization in order<br />to become trusted members. Defaults to the Github link of TrustedOrg.                |
| OnlyOrgMembers               | `only_org_members`                | bool     | No       | OnlyOrgMembers requires PRs and/or /ok-to-test comments to come from org members.<br />By default, trigger also include repo collaborators.                                                                                           |
| IgnoreOkToTest               | `ignore_ok_to_test`               | bool     | No       | IgnoreOkToTest makes trigger ignore /ok-to-test comments.<br />This is a security mitigation to only allow testing from trusted users.                                                                                                |
| ElideSkippedContexts         | `elide_skipped_contexts`          | bool     | No       | ElideSkippedContexts makes trigger not post "Skipped" contexts for jobs<br />that could run but do not run.                                                                                                                           |
| SkipDraftPR                  | `skip_draft_pr`                   | bool     | No       | SkipDraftPR when enabled, skips triggering pipelines for draft PRs<br />unless /ok-to-test is added.                                                                                                                                  |
| SkipReportComment            | `skip_report_comment`             | bool     | No       | SkipReportComment when enabled, skips report comments in the SCM provider based on the state of<br />the LighthouseJobs.                                                                                                              |
| SkipReportRunningStatus      | `skip_report_running_status`      | bool     | No       | SkipReportRunningStatus when enabled, skips report status in the SCM provider based on the current and last state of<br />the LighthouseJobs.                                                                                         |
| ShowReportCompletionDuration | `show_report_completion_duration` | bool     | No       | when enabled, show completion duration in report status in the SCM provider based on StartTime and CompletionTime of the PipelineActivity.                                                                                            |


## Welcome

Welcome is config for the welcome plugin.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Repos | `repos` | []string | No | Repos is either of the form org/repos or just org. |
| MessageTemplate | `message_template` | string | No | MessageTemplate is the welcome message template to post on new-contributor PRs<br />For the info struct see prow/plugins/welcome/welcome.go's PRInfo |


