# Package github.com/jenkins-x/lighthouse/pkg/config/lighthouse

- [Config](#Config)
- [GitHubOptions](#GitHubOptions)
- [JenkinsOperator](#JenkinsOperator)
- [OwnersDirExcludes](#OwnersDirExcludes)
- [Plank](#Plank)
- [ProviderConfig](#ProviderConfig)
- [PubsubSubscriptions](#PubsubSubscriptions)
- [PushGateway](#PushGateway)


## Config

Config is config for all lighthouse controllers

| Stanza | Type | Required | Description |
|---|---|---|---|
| `tide` | [Config](./github-com-jenkins-x-lighthouse-pkg-config-keeper.md#Config) | No |  |
| `plank` | [Plank](./github-com-jenkins-x-lighthouse-pkg-config-lighthouse.md#Plank) | No |  |
| `branch-protection` | [Config](./github-com-jenkins-x-lighthouse-pkg-config-branchprotection.md#Config) | No |  |
| `orgs` | map[string][Config](./github-com-jenkins-x-lighthouse-pkg-config-org.md#Config) | No |  |
| `jenkins_operators` | [][JenkinsOperator](./github-com-jenkins-x-lighthouse-pkg-config-lighthouse.md#JenkinsOperator) | No | TODO: Move this out of the main config. |
| `prowjob_namespace` | string | No | LighthouseJobNamespace is the namespace in the cluster that prow<br />components will use for looking up LighthouseJobs. The namespace<br />needs to exist and will not be created by prow.<br />Defaults to "default". |
| `pod_namespace` | string | No | PodNamespace is the namespace in the cluster that prow<br />components will use for looking up Pods owned by LighthouseJobs.<br />The namespace needs to exist and will not be created by prow.<br />Defaults to "default". |
| `log_level` | string | No | LogLevel enables dynamically updating the log level of the<br />standard logger that is used by all prow components.<br /><br />Valid values:<br /><br />"debug", "info", "warn", "warning", "error", "fatal", "panic"<br /><br />Defaults to "info". |
| `push_gateway` | [PushGateway](./github-com-jenkins-x-lighthouse-pkg-config-lighthouse.md#PushGateway) | No | PushGateway is a prometheus push gateway. |
| `owners_dir_excludes` | *[OwnersDirExcludes](./github-com-jenkins-x-lighthouse-pkg-config-lighthouse.md#OwnersDirExcludes) | No | OwnersDirExcludes is used to configure which directories to ignore when<br />searching for OWNERS{,_ALIAS} files in a repo. |
| `pubsub_subscriptions` | [PubsubSubscriptions](./github-com-jenkins-x-lighthouse-pkg-config-lighthouse.md#PubsubSubscriptions) | No | Pub/Sub Subscriptions that we want to listen to |
| `github` | [GitHubOptions](./github-com-jenkins-x-lighthouse-pkg-config-lighthouse.md#GitHubOptions) | No | GitHubOptions allows users to control how prow applications display GitHub website links. |
| `providerConfig` | *[ProviderConfig](./github-com-jenkins-x-lighthouse-pkg-config-lighthouse.md#ProviderConfig) | No | ProviderConfig contains optional SCM provider information |

## GitHubOptions

GitHubOptions allows users to control how prow applications display GitHub website links.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `link_url` | string | No | LinkURLFromConfig is the string representation of the link_url config parameter.<br />This config parameter allows users to override the default GitHub link url for all plugins.<br />If this option is not set, we assume "https://github.com". |

## JenkinsOperator

JenkinsOperator is config for the jenkins-operator controller.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `job_url_template` | string | No | JobURLTemplateString compiles into JobURLTemplate at load time. |
| `report_template` | string | No | ReportTemplateString compiles into ReportTemplate at load time. |
| `max_concurrency` | int | No | MaxConcurrency is the maximum number of tests running concurrently that<br />will be allowed by the controller. 0 implies no limit. |
| `max_goroutines` | int | No | MaxGoroutines is the maximum number of goroutines spawned inside the<br />controller to handle tests. Defaults to 20. Needs to be a positive<br />number. |
| `allow_cancellations` | bool | No | AllowCancellations enables aborting presubmit jobs for commits that<br />have been superseded by newer commits in Github pull requests. |
| `label_selector` | string | No | LabelSelectorString compiles into LabelSelector at load time.<br />If set, this option needs to match --label-selector used by<br />the desired jenkins-operator. This option is considered<br />invalid when provided with a single jenkins-operator config.<br /><br />For label selector syntax, see below:<br />https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors |

## OwnersDirExcludes

OwnersDirExcludes is used to configure which directories to ignore when<br />searching for OWNERS{,_ALIAS} files in a repo.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `repos` | map[string][]string | No | Repos configures a directory blacklist per repo (or org) |
| `default` | []string | No | Default configures a default blacklist for repos (or orgs) not<br />specifically configured |

## Plank

Plank is config for the plank controller.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `report_template` | string | No | ReportTemplateString compiles into ReportTemplate at load time. |

## ProviderConfig

ProviderConfig is optionally used to configure information about the SCM provider being used. These values will be<br />used as fallbacks if environment variables aren't set.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `kind` | string | No | Kind is the go-scm driver name |
| `server` | string | No | Server is the base URL for the provider, like https://github.com |
| `botUser` | string | No | BotUser is the username on the provider the bot will use |

## PubsubSubscriptions

PubsubSubscriptions maps GCP projects to a list of Topics.



## PushGateway

PushGateway is a prometheus push gateway.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `endpoint` | string | No | Endpoint is the location of the prometheus pushgateway<br />where prow will push metrics to. |
| `interval` | string | No | IntervalString compiles into Interval at load time. |
| `serve_metrics` | bool | Yes | ServeMetrics tells if or not the components serve metrics |


