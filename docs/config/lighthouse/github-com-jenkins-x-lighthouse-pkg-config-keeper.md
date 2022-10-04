# Package github.com/jenkins-x/lighthouse/pkg/config/keeper

- [Config](#Config)
- [ContextPolicy](#ContextPolicy)
- [ContextPolicyOptions](#ContextPolicyOptions)
- [MergeCommitTemplate](#MergeCommitTemplate)
- [OrgContextPolicy](#OrgContextPolicy)
- [PullRequestMergeType](#PullRequestMergeType)
- [Queries](#Queries)
- [RepoContextPolicy](#RepoContextPolicy)


## Config

Config is the config for the keeper pool.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `sync_period` | string | No | SyncPeriodString compiles into SyncPeriod at load time. |
| `status_update_period` | string | No | StatusUpdatePeriodString compiles into StatusUpdatePeriod at load time. |
| `queries` | [Queries](./github-com-jenkins-x-lighthouse-pkg-config-keeper.md#Queries) | No | Queries represents a list of GitHub search queries that collectively<br />specify the set of PRs that meet merge requirements. |
| `default_merge_method` | [PullRequestMergeType](./github-com-jenkins-x-lighthouse-pkg-config-keeper.md#PullRequestMergeType) | No | The default merge type for lighthouse to use, and the merge_method list will override this. Defaults to "merge" |
| `merge_method` | map[string][PullRequestMergeType](./github-com-jenkins-x-lighthouse-pkg-config-keeper.md#PullRequestMergeType) | No | A key/value pair of an org/repo as the key and merge method to override<br />the default method of merge. Valid options are squash, rebase, and merge. |
| `merge_commit_template` | map[string][MergeCommitTemplate](./github-com-jenkins-x-lighthouse-pkg-config-keeper.md#MergeCommitTemplate) | No | A key/value pair of an org/repo as the key and Go template to override<br />the default merge commit title and/or message. Template is passed the<br />PullRequest struct (prow/github/types.go#PullRequest) |
| `target_url` | string | No | URL for keeper status contexts.<br />We can consider allowing this to be set separately for separate repos, or<br />allowing it to be a template. |
| `pr_status_base_url` | string | No | PRStatusBaseURL is the base URL for the PR status page.<br />This is used to link to a merge requirements overview<br />in the keeper status context. |
| `blocker_label` | string | No | BlockerLabel is an optional label that is used to identify merge blocking<br />Github issues.<br />Leave this blank to disable this feature and save 1 API token per sync loop. |
| `squash_label` | string | No | SquashLabel is an optional label that is used to identify PRs that should<br />always be squash merged.<br />Leave this blank to disable this feature. |
| `rebase_label` | string | No | RebaseLabel is an optional label that is used to identify PRs that should<br />always be rebased and merged.<br />Leave this blank to disable this feature. |
| `merge_label` | string | No | MergeLabel is an optional label that is used to identify PRs that should<br />always be merged with all individual commits from the PR.<br />Leave this blank to disable this feature. |
| `max_goroutines` | int | No | MaxGoroutines is the maximum number of goroutines spawned inside the<br />controller to handle org/repo:branch pools. Defaults to 20. Needs to be a<br />positive number. |
| `context_options` | [ContextPolicyOptions](./github-com-jenkins-x-lighthouse-pkg-config-keeper.md#ContextPolicyOptions) | No | KeeperContextPolicyOptions defines merge options for context. If not set it will infer<br />the required and optional contexts from the prow jobs configured and use the github<br />combined status; otherwise it may apply the branch protection setting or let user<br />define their own options in case branch protection is not used. |
| `batch_size_limit` | map[string]int | No | BatchSizeLimitMap is a key/value pair of an org or org/repo as the key and<br />integer batch size limit as the value. The empty string key can be used as<br />a global default.<br />Special values:<br /> 0 => unlimited batch size<br />-1 => batch merging disabled :( |

## ContextPolicy

ContextPolicy configures options about how to handle various contexts.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `skip-unknown-contexts` | *bool | No | whether to consider unknown contexts optional (skip) or required. |
| `required-contexts` | []string | No |  |
| `required-if-present-contexts` | []string | No |  |
| `optional-contexts` | []string | No |  |
| `from-branch-protection` | *bool | No | Infer required and optional jobs from Branch Protection configuration |

## ContextPolicyOptions

ContextPolicyOptions holds the default policy, and any org overrides.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `skip-unknown-contexts` | *bool | No | whether to consider unknown contexts optional (skip) or required. |
| `required-contexts` | []string | No |  |
| `required-if-present-contexts` | []string | No |  |
| `optional-contexts` | []string | No |  |
| `from-branch-protection` | *bool | No | Infer required and optional jobs from Branch Protection configuration |
| `orgs` | map[string][OrgContextPolicy](./github-com-jenkins-x-lighthouse-pkg-config-keeper.md#OrgContextPolicy) | No | Github Orgs |

## MergeCommitTemplate

MergeCommitTemplate holds templates to use for merge commits.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `title` | string | No |  |
| `body` | string | No |  |

## OrgContextPolicy

OrgContextPolicy overrides the policy for an org, and any repo overrides.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `skip-unknown-contexts` | *bool | No | whether to consider unknown contexts optional (skip) or required. |
| `required-contexts` | []string | No |  |
| `required-if-present-contexts` | []string | No |  |
| `optional-contexts` | []string | No |  |
| `from-branch-protection` | *bool | No | Infer required and optional jobs from Branch Protection configuration |
| `repos` | map[string][RepoContextPolicy](./github-com-jenkins-x-lighthouse-pkg-config-keeper.md#RepoContextPolicy) | No |  |

## PullRequestMergeType

PullRequestMergeType inidicates the type of the pull request



## Queries

Queries is a Query slice.



## RepoContextPolicy

RepoContextPolicy overrides the policy for repo, and any branch overrides.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `skip-unknown-contexts` | *bool | No | whether to consider unknown contexts optional (skip) or required. |
| `required-contexts` | []string | No |  |
| `required-if-present-contexts` | []string | No |  |
| `optional-contexts` | []string | No |  |
| `from-branch-protection` | *bool | No | Infer required and optional jobs from Branch Protection configuration |
| `branches` | map[string][ContextPolicy](./github-com-jenkins-x-lighthouse-pkg-config-keeper.md#ContextPolicy) | No |  |


