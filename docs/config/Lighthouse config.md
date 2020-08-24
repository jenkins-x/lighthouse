# Lighthouse config

- [Branch](#Branch)
- [BranchProtection](#BranchProtection)
- [Config](#Config)
- [ContextPolicy](#ContextPolicy)
- [Controller](#Controller)
- [Cookie](#Cookie)
- [GitHubOptions](#GitHubOptions)
- [GithubOAuthConfig](#GithubOAuthConfig)
- [JenkinsOperator](#JenkinsOperator)
- [JobConfig](#JobConfig)
- [Keeper](#Keeper)
- [KeeperContextPolicy](#KeeperContextPolicy)
- [KeeperContextPolicyOptions](#KeeperContextPolicyOptions)
- [KeeperMergeCommitTemplate](#KeeperMergeCommitTemplate)
- [KeeperOrgContextPolicy](#KeeperOrgContextPolicy)
- [KeeperQueries](#KeeperQueries)
- [KeeperQuery](#KeeperQuery)
- [KeeperRepoContextPolicy](#KeeperRepoContextPolicy)
- [Org](#Org)
- [OwnersDirExcludes](#OwnersDirExcludes)
- [PipelineKind](#PipelineKind)
- [Plank](#Plank)
- [Policy](#Policy)
- [ProviderConfig](#ProviderConfig)
- [ProwConfig](#ProwConfig)
- [PubsubSubscriptions](#PubsubSubscriptions)
- [PullRequestMergeType](#PullRequestMergeType)
- [PushGateway](#PushGateway)
- [QueryMap](#QueryMap)
- [Repo](#Repo)
- [Restrictions](#Restrictions)
- [ReviewPolicy](#ReviewPolicy)


## Branch

Branch holds protection policy overrides for a particular branch.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | [Policy](#Policy) | Yes |  |

## BranchProtection

BranchProtection specifies the global branch protection policy

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | [Policy](#Policy) | Yes |  |
| ProtectTested | `protect-tested-repos` | bool | No | ProtectTested determines if branch protection rules are set for all repos<br />that Prow has registered jobs for, regardless of if those repos are in the<br />branch protection config. |
| Orgs | `orgs` | map[string][Org](#Org) | No | Orgs holds branch protection options for orgs by name |
| AllowDisabledPolicies | `allow_disabled_policies` | bool | No | AllowDisabledPolicies allows a child to disable all protection even if the<br />branch has inherited protection options from a parent. |
| AllowDisabledJobPolicies | `allow_disabled_job_policies` | bool | No | AllowDisabledJobPolicies allows a branch to choose to opt out of branch protection<br />even if Prow has registered required jobs for that branch. |

## Config

Config is a read-only snapshot of the config.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | [JobConfig](#JobConfig) | Yes |  |
|  |  | [ProwConfig](#ProwConfig) | Yes |  |

## ContextPolicy

ContextPolicy configures required github contexts.<br />When merging policies, contexts are appended to context list from parent.<br />Strict determines whether merging to the branch invalidates existing contexts.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Contexts | `contexts` | []string | No | Contexts appends required contexts that must be green to merge |
| Strict | `strict` | *bool | No | Strict overrides whether new commits in the base branch require updating the PR if set |

## Controller

Controller holds configuration applicable to all agent-specific<br />prow controllers.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| JobURLTemplateString | `job_url_template` | string | No | JobURLTemplateString compiles into JobURLTemplate at load time. |
| JobURLTemplate | `-` | *template.Template | No | JobURLTemplate is compiled at load time from JobURLTemplateString. It<br />will be passed a builder.PipelineOptions and is used to set the URL for the<br />"Details" link on GitHub as well as the link from deck. |
| ReportTemplateString | `report_template` | string | No | ReportTemplateString compiles into ReportTemplate at load time. |
| ReportTemplate | `-` | *template.Template | No | ReportTemplate is compiled at load time from ReportTemplateString. It<br />will be passed a builder.PipelineOptions and can provide an optional blurb below<br />the test failures comment. |
| MaxConcurrency | `max_concurrency` | int | No | MaxConcurrency is the maximum number of tests running concurrently that<br />will be allowed by the controller. 0 implies no limit. |
| MaxGoroutines | `max_goroutines` | int | No | MaxGoroutines is the maximum number of goroutines spawned inside the<br />controller to handle tests. Defaults to 20. Needs to be a positive<br />number. |
| AllowCancellations | `allow_cancellations` | bool | No | AllowCancellations enables aborting presubmit jobs for commits that<br />have been superseded by newer commits in Github pull requests. |

## Cookie

Cookie holds the secret returned from github that authenticates the user who authorized this app.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Secret | `secret` | string | No |  |

## GitHubOptions

GitHubOptions allows users to control how prow applications display GitHub website links.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| LinkURLFromConfig | `link_url` | string | No | LinkURLFromConfig is the string representation of the link_url config parameter.<br />This config parameter allows users to override the default GitHub link url for all plugins.<br />If this option is not set, we assume "https://github.com". |
| LinkURL |  | *url.URL | No | LinkURL is the url representation of LinkURLFromConfig. This variable should be used<br />in all places internally. |

## GithubOAuthConfig

GithubOAuthConfig is a config for requesting users access tokens from Github API. It also has<br />a Cookie Store that retains user credentials deriving from Github API.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| ClientID | `client_id` | string | Yes |  |
| ClientSecret | `client_secret` | string | Yes |  |
| RedirectURL | `redirect_url` | string | Yes |  |
| Scopes | `scopes` | []string | No |  |
| FinalRedirectURL | `final_redirect_url` | string | Yes |  |
| CookieStore | `-` | *sessions.CookieStore | No |  |

## JenkinsOperator

JenkinsOperator is config for the jenkins-operator controller.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | [Controller](#Controller) | Yes |  |
| LabelSelectorString | `label_selector` | string | No | LabelSelectorString compiles into LabelSelector at load time.<br />If set, this option needs to match --label-selector used by<br />the desired jenkins-operator. This option is considered<br />invalid when provided with a single jenkins-operator config.<br /><br />For label selector syntax, see below:<br />https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors |
| LabelSelector | `-` | labels.Selector | Yes | LabelSelector is used so different jenkins-operator replicas<br />can use their own configuration. |

## JobConfig

JobConfig is a type alias for job.Config



## Keeper

Keeper is config for the keeper pool.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| SyncPeriodString | `sync_period` | string | No | SyncPeriodString compiles into SyncPeriod at load time. |
| SyncPeriod | `-` | time.Duration | Yes | SyncPeriod specifies how often Keeper will sync jobs with Github. Defaults to 1m. |
| StatusUpdatePeriodString | `status_update_period` | string | No | StatusUpdatePeriodString compiles into StatusUpdatePeriod at load time. |
| StatusUpdatePeriod | `-` | time.Duration | Yes | StatusUpdatePeriod specifies how often Keeper will update Github status contexts.<br />Defaults to the value of SyncPeriod. |
| Queries | `queries` | [KeeperQueries](#KeeperQueries) | No | Queries represents a list of GitHub search queries that collectively<br />specify the set of PRs that meet merge requirements. |
| MergeType | `merge_method` | map[string][PullRequestMergeType](#PullRequestMergeType) | No | A key/value pair of an org/repo as the key and merge method to override<br />the default method of merge. Valid options are squash, rebase, and merge. |
| MergeTemplate | `merge_commit_template` | map[string][KeeperMergeCommitTemplate](#KeeperMergeCommitTemplate) | No | A key/value pair of an org/repo as the key and Go template to override<br />the default merge commit title and/or message. Template is passed the<br />PullRequest struct (prow/github/types.go#PullRequest) |
| TargetURL | `target_url` | string | No | URL for keeper status contexts.<br />We can consider allowing this to be set separately for separate repos, or<br />allowing it to be a template. |
| PRStatusBaseURL | `pr_status_base_url` | string | No | PRStatusBaseURL is the base URL for the PR status page.<br />This is used to link to a merge requirements overview<br />in the keeper status context. |
| BlockerLabel | `blocker_label` | string | No | BlockerLabel is an optional label that is used to identify merge blocking<br />Github issues.<br />Leave this blank to disable this feature and save 1 API token per sync loop. |
| SquashLabel | `squash_label` | string | No | SquashLabel is an optional label that is used to identify PRs that should<br />always be squash merged.<br />Leave this blank to disable this feature. |
| RebaseLabel | `rebase_label` | string | No | RebaseLabel is an optional label that is used to identify PRs that should<br />always be rebased and merged.<br />Leave this blank to disable this feature. |
| MergeLabel | `merge_label` | string | No | MergeLabel is an optional label that is used to identify PRs that should<br />always be merged with all individual commits from the PR.<br />Leave this blank to disable this feature. |
| MaxGoroutines | `max_goroutines` | int | No | MaxGoroutines is the maximum number of goroutines spawned inside the<br />controller to handle org/repo:branch pools. Defaults to 20. Needs to be a<br />positive number. |
| ContextOptions | `context_options` | [KeeperContextPolicyOptions](#KeeperContextPolicyOptions) | No | KeeperContextPolicyOptions defines merge options for context. If not set it will infer<br />the required and optional contexts from the prow jobs configured and use the github<br />combined status; otherwise it may apply the branch protection setting or let user<br />define their own options in case branch protection is not used. |
| BatchSizeLimitMap | `batch_size_limit` | map[string]int | No | BatchSizeLimitMap is a key/value pair of an org or org/repo as the key and<br />integer batch size limit as the value. The empty string key can be used as<br />a global default.<br />Special values:<br /> 0 => unlimited batch size<br />-1 => batch merging disabled :( |

## KeeperContextPolicy

KeeperContextPolicy configures options about how to handle various contexts.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| SkipUnknownContexts | `skip-unknown-contexts` | *bool | No | whether to consider unknown contexts optional (skip) or required. |
| RequiredContexts | `required-contexts` | []string | No |  |
| RequiredIfPresentContexts | `required-if-present-contexts` | []string | No |  |
| OptionalContexts | `optional-contexts` | []string | No |  |
| FromBranchProtection | `from-branch-protection` | *bool | No | Infer required and optional jobs from Branch Protection configuration |

## KeeperContextPolicyOptions

KeeperContextPolicyOptions holds the default policy, and any org overrides.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | [KeeperContextPolicy](#KeeperContextPolicy) | Yes |  |
| Orgs | `orgs` | map[string][KeeperOrgContextPolicy](#KeeperOrgContextPolicy) | No | Github Orgs |

## KeeperMergeCommitTemplate

KeeperMergeCommitTemplate holds templates to use for merge commits.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| TitleTemplate | `title` | string | No |  |
| BodyTemplate | `body` | string | No |  |
| Title | `-` | *template.Template | No |  |
| Body | `-` | *template.Template | No |  |

## KeeperOrgContextPolicy

KeeperOrgContextPolicy overrides the policy for an org, and any repo overrides.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | [KeeperContextPolicy](#KeeperContextPolicy) | Yes |  |
| Repos | `repos` | map[string][KeeperRepoContextPolicy](#KeeperRepoContextPolicy) | No |  |

## KeeperQueries

KeeperQueries is a KeeperQuery slice.



## KeeperQuery

KeeperQuery is turned into a GitHub search query. See the docs for details:<br />https://help.github.com/articles/searching-issues-and-pull-requests/

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Orgs | `orgs` | []string | No |  |
| Repos | `repos` | []string | No |  |
| ExcludedRepos | `excludedRepos` | []string | No |  |
| ExcludedBranches | `excludedBranches` | []string | No |  |
| IncludedBranches | `includedBranches` | []string | No |  |
| Labels | `labels` | []string | No |  |
| MissingLabels | `missingLabels` | []string | No |  |
| Milestone | `milestone` | string | No |  |
| ReviewApprovedRequired | `reviewApprovedRequired` | bool | No |  |

## KeeperRepoContextPolicy

KeeperRepoContextPolicy overrides the policy for repo, and any branch overrides.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | [KeeperContextPolicy](#KeeperContextPolicy) | Yes |  |
| Branches | `branches` | map[string][KeeperContextPolicy](#KeeperContextPolicy) | No |  |

## Org

Org holds the default protection policy for an entire org, as well as any repo overrides.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | [Policy](#Policy) | Yes |  |
| Repos | `repos` | map[string][Repo](#Repo) | No |  |

## OwnersDirExcludes

OwnersDirExcludes is used to configure which directories to ignore when<br />searching for OWNERS{,_ALIAS} files in a repo.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Repos | `repos` | map[string][]string | No | Repos configures a directory blacklist per repo (or org) |
| Default | `default` | []string | No | Default configures a default blacklist for repos (or orgs) not<br />specifically configured |

## PipelineKind

PipelineKind specifies how the job is triggered.



## Plank

Plank is config for the plank controller.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| ReportTemplateString | `report_template` | string | No | ReportTemplateString compiles into ReportTemplate at load time. |
| ReportTemplate | `-` | *template.Template | No | ReportTemplate is compiled at load time from ReportTemplateString. It<br />will be passed a builder.PipelineOptions and can provide an optional blurb below<br />the test failures comment. |

## Policy

Policy for the config/org/repo/branch.<br />When merging policies, a nil value results in inheriting the parent policy.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Protect | `protect` | *bool | No | Protect overrides whether branch protection is enabled if set. |
| RequiredStatusChecks | `required_status_checks` | *[ContextPolicy](#ContextPolicy) | No | RequiredStatusChecks configures github contexts |
| Admins | `enforce_admins` | *bool | No | Admins overrides whether protections apply to admins if set. |
| Restrictions | `restrictions` | *[Restrictions](#Restrictions) | No | Restrictions limits who can merge |
| RequiredPullRequestReviews | `required_pull_request_reviews` | *[ReviewPolicy](#ReviewPolicy) | No | RequiredPullRequestReviews specifies github approval/review criteria. |
| Exclude | `exclude` | []string | No | Exclude specifies a set of regular expressions which identify branches<br />that should be excluded from the protection policy |

## ProviderConfig

ProviderConfig is optionally used to configure information about the SCM provider being used. These values will be<br />used as fallbacks if environment variables aren't set.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Kind | `kind` | string | No | Kind is the go-scm driver name |
| Server | `server` | string | No | Server is the base URL for the provider, like https://github.com |
| BotUser | `botUser` | string | No | BotUser is the username on the provider the bot will use |

## ProwConfig

ProwConfig is config for all prow controllers

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Keeper | `tide` | [Keeper](#Keeper) | No |  |
| Plank | `plank` | [Plank](#Plank) | No |  |
| BranchProtection | `branch-protection` | [BranchProtection](#BranchProtection) | No |  |
| Orgs | `orgs` | map[string]org.Config | No |  |
| JenkinsOperators | `jenkins_operators` | [][JenkinsOperator](#JenkinsOperator) | No | TODO: Move this out of the main config. |
| LighthouseJobNamespace | `prowjob_namespace` | string | No | LighthouseJobNamespace is the namespace in the cluster that prow<br />components will use for looking up LighthouseJobs. The namespace<br />needs to exist and will not be created by prow.<br />Defaults to "default". |
| PodNamespace | `pod_namespace` | string | No | PodNamespace is the namespace in the cluster that prow<br />components will use for looking up Pods owned by LighthouseJobs.<br />The namespace needs to exist and will not be created by prow.<br />Defaults to "default". |
| LogLevel | `log_level` | string | No | LogLevel enables dynamically updating the log level of the<br />standard logger that is used by all prow components.<br /><br />Valid values:<br /><br />"debug", "info", "warn", "warning", "error", "fatal", "panic"<br /><br />Defaults to "info". |
| PushGateway | `push_gateway` | [PushGateway](#PushGateway) | No | PushGateway is a prometheus push gateway. |
| OwnersDirExcludes | `owners_dir_excludes` | *[OwnersDirExcludes](#OwnersDirExcludes) | No | OwnersDirExcludes is used to configure which directories to ignore when<br />searching for OWNERS{,_ALIAS} files in a repo. |
| OwnersDirBlacklist | `owners_dir_blacklist` | *[OwnersDirExcludes](#OwnersDirExcludes) | No | OwnersDirExcludes is DEPRECATED in favor of OwnersDirExcludes |
| PubSubSubscriptions | `pubsub_subscriptions` | [PubsubSubscriptions](#PubsubSubscriptions) | No | Pub/Sub Subscriptions that we want to listen to |
| GitHubOptions | `github` | [GitHubOptions](#GitHubOptions) | No | GitHubOptions allows users to control how prow applications display GitHub website links. |
| ProviderConfig | `providerConfig` | *[ProviderConfig](#ProviderConfig) | No | ProviderConfig contains optional SCM provider information |

## PubsubSubscriptions

PubsubSubscriptions maps GCP projects to a list of Topics.



## PullRequestMergeType

PullRequestMergeType inidicates the type of the pull request



## PushGateway

PushGateway is a prometheus push gateway.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Endpoint | `endpoint` | string | No | Endpoint is the location of the prometheus pushgateway<br />where prow will push metrics to. |
| IntervalString | `interval` | string | No | IntervalString compiles into Interval at load time. |
| Interval | `-` | time.Duration | Yes | Interval specifies how often prow will push metrics<br />to the pushgateway. Defaults to 1m. |
| ServeMetrics | `serve_metrics` | bool | Yes | ServeMetrics tells if or not the components serve metrics |

## QueryMap

QueryMap is a struct mapping from "org/repo" -> KeeperQueries that<br />apply to that org or repo. It is lazily populated, but threadsafe.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| queries |  | [KeeperQueries](#KeeperQueries) | Yes |  |
| cache |  | map[string][KeeperQueries](#KeeperQueries) | No |  |
|  |  | sync.Mutex | Yes |  |

## Repo

Repo holds protection policy overrides for all branches in a repo, as well as specific branch overrides.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | [Policy](#Policy) | Yes |  |
| Branches | `branches` | map[string][Branch](#Branch) | No |  |

## Restrictions

Restrictions limits who can merge<br />Users and Teams items are appended to parent lists.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Users | `users` | []string | No |  |
| Teams | `teams` | []string | No |  |

## ReviewPolicy

ReviewPolicy specifies github approval/review criteria.<br />Any nil values inherit the policy from the parent, otherwise bool/ints are overridden.<br />Non-empty lists are appended to parent lists.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| DismissalRestrictions | `dismissal_restrictions` | *[Restrictions](#Restrictions) | No | Restrictions appends users/teams that are allowed to merge |
| DismissStale | `dismiss_stale_reviews` | *bool | No | DismissStale overrides whether new commits automatically dismiss old reviews if set |
| RequireOwners | `require_code_owner_reviews` | *bool | No | RequireOwners overrides whether CODEOWNERS must approve PRs if set |
| Approvals | `required_approving_review_count` | *int | No | Approvals overrides the number of approvals required if set (set to 0 to disable) |


