# Lighthouse (v1alpha1)

- [ActivityRecord](#ActivityRecord)
- [ActivityStageOrStep](#ActivityStageOrStep)
- [ByNum](#ByNum)
- [DecorationConfig](#DecorationConfig)
- [Duration](#Duration)
- [LighthouseJob](#LighthouseJob)
- [LighthouseJobList](#LighthouseJobList)
- [LighthouseJobSpec](#LighthouseJobSpec)
- [LighthouseJobStatus](#LighthouseJobStatus)
- [PipelineState](#PipelineState)
- [Pull](#Pull)
- [Refs](#Refs)


## ActivityRecord

ActivityRecord is a struct for reporting information on a pipeline, build, or other activity triggered by Lighthouse

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Name | `name` | string | Yes |  |
| JobID | `jobId` | string | No |  |
| Owner | `owner` | string | No |  |
| Repo | `repo` | string | No |  |
| Branch | `branch` | string | No |  |
| BuildIdentifier | `buildId` | string | No |  |
| Context | `context` | string | No |  |
| GitURL | `gitURL` | string | No |  |
| LogURL | `logURL` | string | No |  |
| LinkURL | `linkURL` | string | No |  |
| Status | `status` | [PipelineState](#PipelineState) | No |  |
| BaseSHA | `baseSHA` | string | No |  |
| LastCommitSHA | `lastCommitSHA` | string | No |  |
| StartTime | `startTime` | *metav1.Time | No |  |
| CompletionTime | `completionTime` | *metav1.Time | No |  |
| Stages | `stages` | []*[ActivityStageOrStep](#ActivityStageOrStep) | No |  |
| Steps | `steps` | []*[ActivityStageOrStep](#ActivityStageOrStep) | No |  |

## ActivityStageOrStep

ActivityStageOrStep represents a stage of an activity

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Name | `name` | string | Yes |  |
| Status | `status` | [PipelineState](#PipelineState) | Yes |  |
| StartTime | `startTime` | *metav1.Time | No |  |
| CompletionTime | `completionTime` | *metav1.Time | No |  |
| Stages | `stages` | []*[ActivityStageOrStep](#ActivityStageOrStep) | No |  |
| Steps | `steps` | []*[ActivityStageOrStep](#ActivityStageOrStep) | No |  |

## ByNum

ByNum implements sort.Interface for []Pull to sort by ascending PR number.



## DecorationConfig

DecorationConfig specifies how to augment pods.<br /><br />This is primarily used to provide automatic integration with gubernator<br />and testgrid.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Timeout | `timeout` | *[Duration](#Duration) | No | Timeout is how long the pod utilities will wait<br />before aborting a job with SIGINT. |
| GracePeriod | `grace_period` | *[Duration](#Duration) | No | GracePeriod is how long the pod utilities will wait<br />after sending SIGINT to send SIGKILL when aborting<br />a job. Only applicable if decorating the PodSpec. |
| GCSCredentialsSecret | `gcs_credentials_secret` | string | No | GCSCredentialsSecret is the name of the Kubernetes secret<br />that holds GCS push credentials. |
| SSHKeySecrets | `ssh_key_secrets` | []string | No | SSHKeySecrets are the names of Kubernetes secrets that contain<br />SSK keys which should be used during the cloning process. |
| SSHHostFingerprints | `ssh_host_fingerprints` | []string | No | SSHHostFingerprints are the fingerprints of known SSH hosts<br />that the cloning process can trust.<br />Launch with ssh-keyscan [-t rsa] host |
| SkipCloning | `skip_cloning` | *bool | No | SkipCloning determines if we should clone source code in the<br />initcontainers for jobs that specify refs |
| CookiefileSecret | `cookiefile_secret` | string | No | CookieFileSecret is the name of a kubernetes secret that contains<br />a git http.cookiefile, which should be used during the cloning process. |

## Duration

Duration is a wrapper around time.Duration that parses times in either<br />'integer number of nanoseconds' or 'duration string' formats and serializes<br />to 'duration string' format.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Duration |  | time.Duration | Yes |  |

## LighthouseJob

LighthouseJob contains the arguments to create a Jenkins X Pipeline and to report on it

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | metav1.TypeMeta | Yes |  |
|  | `metadata` | metav1.ObjectMeta | No |  |
| Spec | `spec` | [LighthouseJobSpec](#LighthouseJobSpec) | No |  |
| Status | `status` | [LighthouseJobStatus](#LighthouseJobStatus) | No |  |

## LighthouseJobList

LighthouseJobList represents a list of pipeline options

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
|  |  | metav1.TypeMeta | Yes |  |
|  | `metadata` | metav1.ListMeta | No | +optional |
| Items | `items` | [][LighthouseJob](#LighthouseJob) | No |  |

## LighthouseJobSpec

LighthouseJobSpec the spec of a pipeline request

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Type | `type` | config.PipelineKind | No | Type is the type of job and informs how<br />the jobs is triggered |
| Agent | `agent` | string | No | Agent is what should run this job, if anything. |
| Namespace | `namespace` | string | No | Namespace defines where to create pods/resources. |
| Job | `job` | string | No | Job is the name of the job |
| Refs | `refs` | *[Refs](#Refs) | No | Refs is the code under test, determined at<br />runtime by Prow itself |
| ExtraRefs | `extra_refs` | [][Refs](#Refs) | No | ExtraRefs are auxiliary repositories that<br />need to be cloned, determined from config |
| Context | `context` | string | No | Context is the name of the status context used to<br />report back to GitHub |
| RerunCommand | `rerun_command` | string | No | RerunCommand is the command a user would write to<br />trigger this job on their pull request |
| MaxConcurrency | `max_concurrency` | int | No | MaxConcurrency restricts the total number of instances<br />of this job that can run in parallel at once |
| PipelineRunSpec | `pipeline_run_spec` | *tektonv1beta1.PipelineRunSpec | No | PipelineRunSpec provides the basis for running the test as a Tekton Pipeline<br />https://github.com/tektoncd/pipeline |
| PipelineRunParams | `pipeline_run_params` | []job.PipelineRunParam | No | PipelineRunParams are the params used by the pipeline run |
| PodSpec | `pod_spec` | *corev1.PodSpec | No | PodSpec provides the basis for running the test under a Kubernetes agent |

## LighthouseJobStatus

LighthouseJobStatus represents the status of a pipeline

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| State | `state` | [PipelineState](#PipelineState) | No | State is the full state of the job |
| ActivityName | `activityName` | string | No | ActivityName is the name of the PipelineActivity, PipelineRun, etc associated with this job, if any. |
| Description | `description` | string | No | Description is used for the description of the commit status we report. |
| ReportURL | `reportURL` | string | No | ReportURL is the link that will be used in the commit status. |
| StartTime | `startTime` | metav1.Time | No | StartTime is when the job was created. |
| CompletionTime | `completionTime` | *metav1.Time | No | CompletionTime is when the job finished reconciling and entered a terminal state. |
| LastReportState | `lastReportState` | string | No | LastReportState is the state from the last time we reported commit status for this job. |
| LastCommitSHA | `lastCommitSHA` | string | No | LastCommitSHA is the commit that will be/has been reported to on the SCM provider |
| Activity | `activity` | *[ActivityRecord](#ActivityRecord) | No | Activity is the most recent activity recorded for the pipeline associated with this job. |

## PipelineState

PipelineState specifies the current pipelne status



## Pull

Pull describes a pull request at a particular point in time.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Number | `number` | int | Yes |  |
| Author | `author` | string | Yes |  |
| SHA | `sha` | string | Yes |  |
| Title | `title` | string | No |  |
| Ref | `ref` | string | No | Ref is git ref can be checked out for a change<br />for example,<br />github: pull/123/head<br />gerrit: refs/changes/00/123/1 |
| Link | `link` | string | No | Link links to the pull request itself. |
| CommitLink | `commit_link` | string | No | CommitLink links to the commit identified by the SHA. |
| AuthorLink | `author_link` | string | No | AuthorLink links to the author of the pull request. |

## Refs

Refs describes how the repo was constructed.

| Variable Name | Stanza | Type | Required | Description |
|---|---|---|---|---|
| Org | `org` | string | Yes | Org is something like kubernetes or k8s.io |
| Repo | `repo` | string | Yes | Repo is something like test-infra |
| RepoLink | `repo_link` | string | No | RepoLink links to the source for Repo. |
| BaseRef | `base_ref` | string | No |  |
| BaseSHA | `base_sha` | string | No |  |
| BaseLink | `base_link` | string | No | BaseLink is a link to the commit identified by BaseSHA. |
| Pulls | `pulls` | [][Pull](#Pull) | No |  |
| PathAlias | `path_alias` | string | No | PathAlias is the location under <root-dir>/src<br />where this repository is cloned. If this is not<br />set, <root-dir>/src/github.com/org/repo will be<br />used as the default. |
| CloneURI | `clone_uri` | string | No | CloneURI is the URI that is used to clone the<br />repository. If unset, will default to<br />`https://github.com/org/repo.git`. |
| SkipSubmodules | `skip_submodules` | bool | No | SkipSubmodules determines if submodules should be<br />cloned when the job is run. Defaults to true. |
| CloneDepth | `clone_depth` | int | No | CloneDepth is the depth of the clone that will be used.<br />A depth of zero will do a full clone. |


