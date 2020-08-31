# Package github.com/jenkins-x/lighthouse/pkg/config/org

- [Config](#Config)
- [Privacy](#Privacy)
- [RepoPermissionLevel](#RepoPermissionLevel)
- [Team](#Team)


## Config

Config declares org metadata as well as its people and teams.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `billing_email` | *string | No |  |
| `company` | *string | No |  |
| `email` | *string | No |  |
| `name` | *string | No |  |
| `description` | *string | No |  |
| `location` | *string | No |  |
| `has_organization_projects` | *bool | No |  |
| `has_repository_projects` | *bool | No |  |
| `default_repository_permission` | *[RepoPermissionLevel](./github-com-jenkins-x-lighthouse-pkg-config-org.md#RepoPermissionLevel) | No |  |
| `members_can_create_repositories` | *bool | No |  |
| `teams` | map[string][Team](./github-com-jenkins-x-lighthouse-pkg-config-org.md#Team) | No |  |
| `members` | []string | No |  |
| `admins` | []string | No |  |

## Privacy

Privacy is secret or closed.<br /><br />See https://developer.github.com/v3/teams/#edit-team



## RepoPermissionLevel

RepoPermissionLevel is admin, write, read or none.<br /><br />See https://developer.github.com/v3/repos/collaborators/#review-a-users-permission-level



## Team

Team declares metadata as well as its poeple.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `description` | *string | No |  |
| `privacy` | *[Privacy](./github-com-jenkins-x-lighthouse-pkg-config-org.md#Privacy) | No |  |
| `members` | []string | No |  |
| `maintainers` | []string | No |  |
| `teams` | map[string][Team](./github-com-jenkins-x-lighthouse-pkg-config-org.md#Team) | No |  |
| `previously` | []string | No |  |


