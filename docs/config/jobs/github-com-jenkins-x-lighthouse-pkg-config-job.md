# Package github.com/jenkins-x/lighthouse/pkg/config/job

- [Config](#Config)
- [JenkinsSpec](#JenkinsSpec)
- [Periodic](#Periodic)
- [PipelineRunParam](#PipelineRunParam)
- [Postsubmit](#Postsubmit)
- [Preset](#Preset)
- [Presubmit](#Presubmit)


## Config

Config is config for all prow jobs

| Stanza | Type | Required | Description |
|---|---|---|---|
| `presets` | [][Preset](./github-com-jenkins-x-lighthouse-pkg-config-job.md#Preset) | No | Presets apply to all job types. |
| `presubmits` | map[string][][Presubmit](./github-com-jenkins-x-lighthouse-pkg-config-job.md#Presubmit) | No | Full repo name (such as "kubernetes/kubernetes") -> list of jobs. |
| `postsubmits` | map[string][][Postsubmit](./github-com-jenkins-x-lighthouse-pkg-config-job.md#Postsubmit) | No |  |
| `periodics` | [][Periodic](./github-com-jenkins-x-lighthouse-pkg-config-job.md#Periodic) | No | Periodics are not associated with any repo. |

## JenkinsSpec

JenkinsSpec holds optional Jenkins job config

| Stanza | Type | Required | Description |
|---|---|---|---|
| `branch_source_job` | bool | No | Job is managed by the GH branch source plugin<br />and requires a specific path |

## Periodic

Periodic runs on a timer.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `decorate` | bool | No | Decorate determines if we decorate the PodSpec or not |
| `path_alias` | string | No | PathAlias is the location under <root-dir>/src<br />where the repository under test is cloned. If this<br />is not set, <root-dir>/src/github.com/org/repo will<br />be used as the default. |
| `clone_uri` | string | No | CloneURI is the URI that is used to clone the<br />repository. If unset, will default to<br />`https://github.com/org/repo.git`. |
| `skip_submodules` | bool | No | SkipSubmodules determines if submodules should be<br />cloned when the job is run. Defaults to true. |
| `clone_depth` | int | No | CloneDepth is the depth of the clone that will be used.<br />A depth of zero will do a full clone. |
| `name` | string | Yes | The name of the job. Must match regex [A-Za-z0-9-._]+<br />e.g. pull-test-infra-bazel-build |
| `labels` | map[string]string | No | Labels are added to LighthouseJobs and pods created for this job. |
| `annotations` | map[string]string | No | Annotations are unused by prow itself, but provide a space to configure other automation. |
| `max_concurrency` | int | No | MaximumConcurrency of this job, 0 implies no limit. |
| `agent` | string | Yes | Agent that will take care of running this job. |
| `cluster` | string | No | Cluster is the alias of the cluster to run this job in.<br />(Default: kube.DefaultClusterAlias) |
| `namespace` | *string | No | Namespace is the namespace in which pods schedule.<br />  nil: results in config.PodNamespace (aka pod default)<br />  empty: results in config.LighthouseJobNamespace (aka same as LighthouseJob) |
| `error_on_eviction` | bool | No | ErrorOnEviction indicates that the LighthouseJob should be completed and given<br />the ErrorState status if the pod that is executing the job is evicted.<br />If this field is unspecified or false, a new pod will be created to replace<br />the evicted one. |
| `source` | string | No | SourcePath contains the path where the tekton pipeline run is defined |
| `spec` | *[PodSpec](./k8s-io-api-core-v1.md#PodSpec) | No | Spec is the Kubernetes pod spec used if Agent is kubernetes. |
| `pipeline_run_spec` | *[PipelineRunSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineRunSpec) | No | PipelineRunSpec is the Tekton PipelineRun spec used if agent is tekton-pipeline |
| `pipeline_run_params` | [][PipelineRunParam](./github-com-jenkins-x-lighthouse-pkg-config-job.md#PipelineRunParam) | No | PipelineRunParams are the params used by the pipeline run |
| `cron` | string | Yes | Cron representation of job trigger time |
| `tags` | []string | No | Tags for config entries |

## PipelineRunParam

PipelineRunParam represents a param used by the pipeline run

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name is the name of the param |
| `value_template` | string | No | ValueTemplate is the template used to build the value from well know variables |

## Postsubmit

Postsubmit runs on push events.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `decorate` | bool | No | Decorate determines if we decorate the PodSpec or not |
| `path_alias` | string | No | PathAlias is the location under <root-dir>/src<br />where the repository under test is cloned. If this<br />is not set, <root-dir>/src/github.com/org/repo will<br />be used as the default. |
| `clone_uri` | string | No | CloneURI is the URI that is used to clone the<br />repository. If unset, will default to<br />`https://github.com/org/repo.git`. |
| `skip_submodules` | bool | No | SkipSubmodules determines if submodules should be<br />cloned when the job is run. Defaults to true. |
| `clone_depth` | int | No | CloneDepth is the depth of the clone that will be used.<br />A depth of zero will do a full clone. |
| `name` | string | Yes | The name of the job. Must match regex [A-Za-z0-9-._]+<br />e.g. pull-test-infra-bazel-build |
| `labels` | map[string]string | No | Labels are added to LighthouseJobs and pods created for this job. |
| `annotations` | map[string]string | No | Annotations are unused by prow itself, but provide a space to configure other automation. |
| `max_concurrency` | int | No | MaximumConcurrency of this job, 0 implies no limit. |
| `agent` | string | Yes | Agent that will take care of running this job. |
| `cluster` | string | No | Cluster is the alias of the cluster to run this job in.<br />(Default: kube.DefaultClusterAlias) |
| `namespace` | *string | No | Namespace is the namespace in which pods schedule.<br />  nil: results in config.PodNamespace (aka pod default)<br />  empty: results in config.LighthouseJobNamespace (aka same as LighthouseJob) |
| `error_on_eviction` | bool | No | ErrorOnEviction indicates that the LighthouseJob should be completed and given<br />the ErrorState status if the pod that is executing the job is evicted.<br />If this field is unspecified or false, a new pod will be created to replace<br />the evicted one. |
| `source` | string | No | SourcePath contains the path where the tekton pipeline run is defined |
| `spec` | *[PodSpec](./k8s-io-api-core-v1.md#PodSpec) | No | Spec is the Kubernetes pod spec used if Agent is kubernetes. |
| `pipeline_run_spec` | *[PipelineRunSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineRunSpec) | No | PipelineRunSpec is the Tekton PipelineRun spec used if agent is tekton-pipeline |
| `pipeline_run_params` | [][PipelineRunParam](./github-com-jenkins-x-lighthouse-pkg-config-job.md#PipelineRunParam) | No | PipelineRunParams are the params used by the pipeline run |
| `run_if_changed` | string | No | RunIfChanged defines a regex used to select which subset of file changes should trigger this job.<br />If any file in the changeset matches this regex, the job will be triggered |
| `skip_branches` | []string | No | Do not run against these branches. Default is no branches. |
| `branches` | []string | No | Only run against these branches. Default is all branches. |
| `context` | string | No | Context is the name of the GitHub status context for the job.<br />Defaults: the same as the name of the job. |
| `skip_report` | bool | No | SkipReport skips commenting and setting status on GitHub. |
| `jenkins_spec` | *[JenkinsSpec](./github-com-jenkins-x-lighthouse-pkg-config-job.md#JenkinsSpec) | No |  |

## Preset

Preset is intended to match the k8s' PodPreset feature, and may be removed<br />if that feature goes beta.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `labels` | map[string]string | No |  |
| `env` | [][EnvVar](./k8s-io-api-core-v1.md#EnvVar) | No |  |
| `volumes` | [][Volume](./k8s-io-api-core-v1.md#Volume) | No |  |
| `volumeMounts` | [][VolumeMount](./k8s-io-api-core-v1.md#VolumeMount) | No |  |

## Presubmit

Presubmit runs on PRs.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `decorate` | bool | No | Decorate determines if we decorate the PodSpec or not |
| `path_alias` | string | No | PathAlias is the location under <root-dir>/src<br />where the repository under test is cloned. If this<br />is not set, <root-dir>/src/github.com/org/repo will<br />be used as the default. |
| `clone_uri` | string | No | CloneURI is the URI that is used to clone the<br />repository. If unset, will default to<br />`https://github.com/org/repo.git`. |
| `skip_submodules` | bool | No | SkipSubmodules determines if submodules should be<br />cloned when the job is run. Defaults to true. |
| `clone_depth` | int | No | CloneDepth is the depth of the clone that will be used.<br />A depth of zero will do a full clone. |
| `name` | string | Yes | The name of the job. Must match regex [A-Za-z0-9-._]+<br />e.g. pull-test-infra-bazel-build |
| `labels` | map[string]string | No | Labels are added to LighthouseJobs and pods created for this job. |
| `annotations` | map[string]string | No | Annotations are unused by prow itself, but provide a space to configure other automation. |
| `max_concurrency` | int | No | MaximumConcurrency of this job, 0 implies no limit. |
| `agent` | string | Yes | Agent that will take care of running this job. |
| `cluster` | string | No | Cluster is the alias of the cluster to run this job in.<br />(Default: kube.DefaultClusterAlias) |
| `namespace` | *string | No | Namespace is the namespace in which pods schedule.<br />  nil: results in config.PodNamespace (aka pod default)<br />  empty: results in config.LighthouseJobNamespace (aka same as LighthouseJob) |
| `error_on_eviction` | bool | No | ErrorOnEviction indicates that the LighthouseJob should be completed and given<br />the ErrorState status if the pod that is executing the job is evicted.<br />If this field is unspecified or false, a new pod will be created to replace<br />the evicted one. |
| `source` | string | No | SourcePath contains the path where the tekton pipeline run is defined |
| `spec` | *[PodSpec](./k8s-io-api-core-v1.md#PodSpec) | No | Spec is the Kubernetes pod spec used if Agent is kubernetes. |
| `pipeline_run_spec` | *[PipelineRunSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineRunSpec) | No | PipelineRunSpec is the Tekton PipelineRun spec used if agent is tekton-pipeline |
| `pipeline_run_params` | [][PipelineRunParam](./github-com-jenkins-x-lighthouse-pkg-config-job.md#PipelineRunParam) | No | PipelineRunParams are the params used by the pipeline run |
| `skip_branches` | []string | No | Do not run against these branches. Default is no branches. |
| `branches` | []string | No | Only run against these branches. Default is all branches. |
| `run_if_changed` | string | No | RunIfChanged defines a regex used to select which subset of file changes should trigger this job.<br />If any file in the changeset matches this regex, the job will be triggered |
| `context` | string | No | Context is the name of the GitHub status context for the job.<br />Defaults: the same as the name of the job. |
| `skip_report` | bool | No | SkipReport skips commenting and setting status on GitHub. |
| `always_run` | bool | Yes | AlwaysRun automatically for every PR, or only when a comment triggers it. |
| `optional` | bool | No | Optional indicates that the job's status context should not be required for merge. |
| `trigger` | string | No | Trigger is the regular expression to trigger the job.<br />e.g. `@k8s-bot e2e test this`<br />RerunCommand must also be specified if this field is specified.<br />(Default: `(?m)^/test (?:.*? )?<job name>(?: .*?)?$`) |
| `rerun_command` | string | No | The RerunCommand to give users. Must match Trigger.<br />Trigger must also be specified if this field is specified.<br />(Default: `/test <job name>`) |
| `jenkins_spec` | *[JenkinsSpec](./github-com-jenkins-x-lighthouse-pkg-config-job.md#JenkinsSpec) | No |  |


