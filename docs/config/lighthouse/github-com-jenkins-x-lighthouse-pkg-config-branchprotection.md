# Package github.com/jenkins-x/lighthouse/pkg/config/branchprotection

- [Branch](#Branch)
- [Config](#Config)
- [ContextPolicy](#ContextPolicy)
- [Org](#Org)
- [Repo](#Repo)
- [Restrictions](#Restrictions)
- [ReviewPolicy](#ReviewPolicy)


## Branch

Branch holds protection policy overrides for a particular branch.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `protect` | *bool | No | Protect overrides whether branch protection is enabled if set. |
| `required_status_checks` | *[ContextPolicy](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#ContextPolicy) | No | RequiredStatusChecks configures github contexts |
| `enforce_admins` | *bool | No | Admins overrides whether protections apply to admins if set. |
| `restrictions` | *[Restrictions](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#Restrictions) | No | Restrictions limits who can merge |
| `required_pull_request_reviews` | *[ReviewPolicy](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#ReviewPolicy) | No | RequiredPullRequestReviews specifies github approval/review criteria. |
| `exclude` | []string | No | Exclude specifies a set of regular expressions which identify branches<br />that should be excluded from the protection policy |

## Config

Config specifies the global branch protection policy

| Stanza | Type | Required | Description |
|---|---|---|---|
| `protect` | *bool | No | Protect overrides whether branch protection is enabled if set. |
| `required_status_checks` | *[ContextPolicy](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#ContextPolicy) | No | RequiredStatusChecks configures github contexts |
| `enforce_admins` | *bool | No | Admins overrides whether protections apply to admins if set. |
| `restrictions` | *[Restrictions](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#Restrictions) | No | Restrictions limits who can merge |
| `required_pull_request_reviews` | *[ReviewPolicy](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#ReviewPolicy) | No | RequiredPullRequestReviews specifies github approval/review criteria. |
| `exclude` | []string | No | Exclude specifies a set of regular expressions which identify branches<br />that should be excluded from the protection policy |
| `protect-tested-repos` | bool | No | ProtectTested determines if branch protection rules are set for all repos<br />that Prow has registered jobs for, regardless of if those repos are in the<br />branch protection config. |
| `orgs` | map[string][Org](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#Org) | No | Orgs holds branch protection options for orgs by name |
| `allow_disabled_policies` | bool | No | AllowDisabledPolicies allows a child to disable all protection even if the<br />branch has inherited protection options from a parent. |
| `allow_disabled_job_policies` | bool | No | AllowDisabledJobPolicies allows a branch to choose to opt out of branch protection<br />even if Prow has registered required jobs for that branch. |

## ContextPolicy

ContextPolicy configures required github contexts.<br />When merging policies, contexts are appended to context list from parent.<br />Strict determines whether merging to the branch invalidates existing contexts.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `contexts` | []string | No | Contexts appends required contexts that must be green to merge |
| `strict` | *bool | No | Strict overrides whether new commits in the base branch require updating the PR if set |

## Org

Org holds the default protection policy for an entire org, as well as any repo overrides.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `protect` | *bool | No | Protect overrides whether branch protection is enabled if set. |
| `required_status_checks` | *[ContextPolicy](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#ContextPolicy) | No | RequiredStatusChecks configures github contexts |
| `enforce_admins` | *bool | No | Admins overrides whether protections apply to admins if set. |
| `restrictions` | *[Restrictions](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#Restrictions) | No | Restrictions limits who can merge |
| `required_pull_request_reviews` | *[ReviewPolicy](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#ReviewPolicy) | No | RequiredPullRequestReviews specifies github approval/review criteria. |
| `exclude` | []string | No | Exclude specifies a set of regular expressions which identify branches<br />that should be excluded from the protection policy |
| `repos` | map[string][Repo](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#Repo) | No |  |

## Repo

Repo holds protection policy overrides for all branches in a repo, as well as specific branch overrides.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `protect` | *bool | No | Protect overrides whether branch protection is enabled if set. |
| `required_status_checks` | *[ContextPolicy](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#ContextPolicy) | No | RequiredStatusChecks configures github contexts |
| `enforce_admins` | *bool | No | Admins overrides whether protections apply to admins if set. |
| `restrictions` | *[Restrictions](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#Restrictions) | No | Restrictions limits who can merge |
| `required_pull_request_reviews` | *[ReviewPolicy](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#ReviewPolicy) | No | RequiredPullRequestReviews specifies github approval/review criteria. |
| `exclude` | []string | No | Exclude specifies a set of regular expressions which identify branches<br />that should be excluded from the protection policy |
| `branches` | map[string][Branch](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#Branch) | No |  |

## Restrictions

Restrictions limits who can merge<br />Users and Teams items are appended to parent lists.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `users` | []string | No |  |
| `teams` | []string | No |  |

## ReviewPolicy

ReviewPolicy specifies github approval/review criteria.<br />Any nil values inherit the policy from the parent, otherwise bool/ints are overridden.<br />Non-empty lists are appended to parent lists.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `dismissal_restrictions` | *[Restrictions](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#Restrictions) | No | Restrictions appends users/teams that are allowed to merge |
| `dismiss_stale_reviews` | *bool | No | DismissStale overrides whether new commits automatically dismiss old reviews if set |
| `require_code_owner_reviews` | *bool | No | RequireOwners overrides whether CODEOWNERS must approve PRs if set |
| `required_approving_review_count` | *int | No | Approvals overrides the number of approvals required if set (set to 0 to disable) |


