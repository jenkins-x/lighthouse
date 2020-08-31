# Package github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1

- [ActivityRecord](#ActivityRecord)
- [ActivityStageOrStep](#ActivityStageOrStep)
- [LighthouseJob](#LighthouseJob)
- [LighthouseJobSpec](#LighthouseJobSpec)
- [LighthouseJobStatus](#LighthouseJobStatus)
- [PipelineState](#PipelineState)
- [Pull](#Pull)
- [Refs](#Refs)


## ActivityRecord

ActivityRecord is a struct for reporting information on a pipeline, build, or other activity triggered by Lighthouse

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes |  |
| `jobId` | string | No |  |
| `owner` | string | No |  |
| `repo` | string | No |  |
| `branch` | string | No |  |
| `buildId` | string | No |  |
| `context` | string | No |  |
| `gitURL` | string | No |  |
| `logURL` | string | No |  |
| `linkURL` | string | No |  |
| `status` | [PipelineState](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#PipelineState) | No |  |
| `baseSHA` | string | No |  |
| `lastCommitSHA` | string | No |  |
| `startTime` | *[Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No |  |
| `completionTime` | *[Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No |  |
| `stages` | []*[ActivityStageOrStep](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#ActivityStageOrStep) | No |  |
| `steps` | []*[ActivityStageOrStep](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#ActivityStageOrStep) | No |  |

## ActivityStageOrStep

ActivityStageOrStep represents a stage of an activity

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes |  |
| `status` | [PipelineState](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#PipelineState) | Yes |  |
| `startTime` | *[Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No |  |
| `completionTime` | *[Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No |  |
| `stages` | []*[ActivityStageOrStep](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#ActivityStageOrStep) | No |  |
| `steps` | []*[ActivityStageOrStep](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#ActivityStageOrStep) | No |  |

## LighthouseJob

LighthouseJob contains the arguments to create a Jenkins X Pipeline and to report on it

| Stanza | Type | Required | Description |
|---|---|---|---|
| `kind` | string | No | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds<br />+optional |
| `apiVersion` | string | No | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources<br />+optional |
| `name` | string | No | Name must be unique within a namespace. Is required when creating resources, although<br />some resources may allow a client to request the generation of an appropriate name<br />automatically. Name is primarily intended for creation idempotence and configuration<br />definition.<br />Cannot be updated.<br />More info: http://kubernetes.io/docs/user-guide/identifiers#names<br />+optional |
| `generateName` | string | No | GenerateName is an optional prefix, used by the server, to generate a unique<br />name ONLY IF the Name field has not been provided.<br />If this field is used, the name returned to the client will be different<br />than the name passed. This value will also be combined with a unique suffix.<br />The provided value has the same validation rules as the Name field,<br />and may be truncated by the length of the suffix required to make the value<br />unique on the server.<br /><br />If this field is specified and the generated name exists, the server will<br />NOT return a 409 - instead, it will either return 201 Created or 500 with Reason<br />ServerTimeout indicating a unique name could not be found in the time allotted, and the client<br />should retry (optionally after the time indicated in the Retry-After header).<br /><br />Applied only if Name is not specified.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency<br />+optional |
| `namespace` | string | No | Namespace defines the space within each name must be unique. An empty namespace is<br />equivalent to the "default" namespace, but "default" is the canonical representation.<br />Not all objects are required to be scoped to a namespace - the value of this field for<br />those objects will be empty.<br /><br />Must be a DNS_LABEL.<br />Cannot be updated.<br />More info: http://kubernetes.io/docs/user-guide/namespaces<br />+optional |
| `selfLink` | string | No | SelfLink is a URL representing this object.<br />Populated by the system.<br />Read-only.<br /><br />DEPRECATED<br />Kubernetes will stop propagating this field in 1.20 release and the field is planned<br />to be removed in 1.21 release.<br />+optional |
| `uid` | [UID](./k8s-io-apimachinery-pkg-types.md#UID) | No | UID is the unique in time and space value for this object. It is typically generated by<br />the server on successful creation of a resource and is not allowed to change on PUT<br />operations.<br /><br />Populated by the system.<br />Read-only.<br />More info: http://kubernetes.io/docs/user-guide/identifiers#uids<br />+optional |
| `resourceVersion` | string | No | An opaque value that represents the internal version of this object that can<br />be used by clients to determine when objects have changed. May be used for optimistic<br />concurrency, change detection, and the watch operation on a resource or set of resources.<br />Clients must treat these values as opaque and passed unmodified back to the server.<br />They may only be valid for a particular resource or set of resources.<br /><br />Populated by the system.<br />Read-only.<br />Value must be treated as opaque by clients and .<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency<br />+optional |
| `generation` | int64 | No | A sequence number representing a specific generation of the desired state.<br />Populated by the system. Read-only.<br />+optional |
| `creationTimestamp` | [Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No | CreationTimestamp is a timestamp representing the server time when this object was<br />created. It is not guaranteed to be set in happens-before order across separate operations.<br />Clients may not set this value. It is represented in RFC3339 form and is in UTC.<br /><br />Populated by the system.<br />Read-only.<br />Null for lists.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata<br />+optional |
| `deletionTimestamp` | *[Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No | DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This<br />field is set by the server when a graceful deletion is requested by the user, and is not<br />directly settable by a client. The resource is expected to be deleted (no longer visible<br />from resource lists, and not reachable by name) after the time in this field, once the<br />finalizers list is empty. As long as the finalizers list contains items, deletion is blocked.<br />Once the deletionTimestamp is set, this value may not be unset or be set further into the<br />future, although it may be shortened or the resource may be deleted prior to this time.<br />For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react<br />by sending a graceful termination signal to the containers in the pod. After that 30 seconds,<br />the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup,<br />remove the pod from the API. In the presence of network partitions, this object may still<br />exist after this timestamp, until an administrator or automated process can determine the<br />resource is fully terminated.<br />If not set, graceful deletion of the object has not been requested.<br /><br />Populated by the system when a graceful deletion is requested.<br />Read-only.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata<br />+optional |
| `deletionGracePeriodSeconds` | *int64 | No | Number of seconds allowed for this object to gracefully terminate before<br />it will be removed from the system. Only set when deletionTimestamp is also set.<br />May only be shortened.<br />Read-only.<br />+optional |
| `labels` | map[string]string | No | Map of string keys and values that can be used to organize and categorize<br />(scope and select) objects. May match selectors of replication controllers<br />and services.<br />More info: http://kubernetes.io/docs/user-guide/labels<br />+optional |
| `annotations` | map[string]string | No | Annotations is an unstructured key value map stored with a resource that may be<br />set by external tools to store and retrieve arbitrary metadata. They are not<br />queryable and should be preserved when modifying objects.<br />More info: http://kubernetes.io/docs/user-guide/annotations<br />+optional |
| `ownerReferences` | [][OwnerReference](./k8s-io-apimachinery-pkg-apis-meta-v1.md#OwnerReference) | No | List of objects depended by this object. If ALL objects in the list have<br />been deleted, this object will be garbage collected. If this object is managed by a controller,<br />then an entry in this list will point to this controller, with the controller field set to true.<br />There cannot be more than one managing controller.<br />+optional<br />+patchMergeKey=uid<br />+patchStrategy=merge |
| `finalizers` | []string | No | Must be empty before the object is deleted from the registry. Each entry<br />is an identifier for the responsible component that will remove the entry<br />from the list. If the deletionTimestamp of the object is non-nil, entries<br />in this list can only be removed.<br />Finalizers may be processed and removed in any order.  Order is NOT enforced<br />because it introduces significant risk of stuck finalizers.<br />finalizers is a shared field, any actor with permission can reorder it.<br />If the finalizer list is processed in order, then this can lead to a situation<br />in which the component responsible for the first finalizer in the list is<br />waiting for a signal (field value, external system, or other) produced by a<br />component responsible for a finalizer later in the list, resulting in a deadlock.<br />Without enforced ordering finalizers are free to order amongst themselves and<br />are not vulnerable to ordering changes in the list.<br />+optional<br />+patchStrategy=merge |
| `clusterName` | string | No | The name of the cluster which the object belongs to.<br />This is used to distinguish resources with same name and namespace in different clusters.<br />This field is not set anywhere right now and apiserver is going to ignore it if set in create or update request.<br />+optional |
| `managedFields` | [][ManagedFieldsEntry](./k8s-io-apimachinery-pkg-apis-meta-v1.md#ManagedFieldsEntry) | No | ManagedFields maps workflow-id and version to the set of fields<br />that are managed by that workflow. This is mostly for internal<br />housekeeping, and users typically shouldn't need to set or<br />understand this field. A workflow can be the user's name, a<br />controller's name, or the name of a specific apply path like<br />"ci-cd". The set of fields is always in the version that the<br />workflow used when modifying the object.<br /><br />+optional |
| `spec` | [LighthouseJobSpec](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#LighthouseJobSpec) | No |  |
| `status` | [LighthouseJobStatus](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#LighthouseJobStatus) | No |  |

## LighthouseJobSpec

LighthouseJobSpec the spec of a pipeline request

| Stanza | Type | Required | Description |
|---|---|---|---|
| `type` | [PipelineKind](./github-com-jenkins-x-lighthouse-pkg-config-job.md#PipelineKind) | No | Type is the type of job and informs how<br />the jobs is triggered |
| `agent` | string | No | Agent is what should run this job, if anything. |
| `namespace` | string | No | Namespace defines where to create pods/resources. |
| `job` | string | No | Job is the name of the job |
| `refs` | *[Refs](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#Refs) | No | Refs is the code under test, determined at<br />runtime by Prow itself |
| `extra_refs` | [][Refs](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#Refs) | No | ExtraRefs are auxiliary repositories that<br />need to be cloned, determined from config |
| `context` | string | No | Context is the name of the status context used to<br />report back to GitHub |
| `rerun_command` | string | No | RerunCommand is the command a user would write to<br />trigger this job on their pull request |
| `max_concurrency` | int | No | MaxConcurrency restricts the total number of instances<br />of this job that can run in parallel at once |
| `pipeline_run_spec` | *[PipelineRunSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineRunSpec) | No | PipelineRunSpec provides the basis for running the test as a Tekton Pipeline<br />https://github.com/tektoncd/pipeline |
| `pipeline_run_params` | [][PipelineRunParam](./github-com-jenkins-x-lighthouse-pkg-config-job.md#PipelineRunParam) | No | PipelineRunParams are the params used by the pipeline run |
| `pod_spec` | *[PodSpec](./k8s-io-api-core-v1.md#PodSpec) | No | PodSpec provides the basis for running the test under a Kubernetes agent |

## LighthouseJobStatus

LighthouseJobStatus represents the status of a pipeline

| Stanza | Type | Required | Description |
|---|---|---|---|
| `state` | [PipelineState](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#PipelineState) | No | State is the full state of the job |
| `activityName` | string | No | ActivityName is the name of the PipelineActivity, PipelineRun, etc associated with this job, if any. |
| `description` | string | No | Description is used for the description of the commit status we report. |
| `reportURL` | string | No | ReportURL is the link that will be used in the commit status. |
| `startTime` | [Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No | StartTime is when the job was created. |
| `completionTime` | *[Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No | CompletionTime is when the job finished reconciling and entered a terminal state. |
| `lastReportState` | string | No | LastReportState is the state from the last time we reported commit status for this job. |
| `lastCommitSHA` | string | No | LastCommitSHA is the commit that will be/has been reported to on the SCM provider |
| `activity` | *[ActivityRecord](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#ActivityRecord) | No | Activity is the most recent activity recorded for the pipeline associated with this job. |

## PipelineState

PipelineState specifies the current pipelne status



## Pull

Pull describes a pull request at a particular point in time.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `number` | int | Yes |  |
| `author` | string | Yes |  |
| `sha` | string | Yes |  |
| `title` | string | No |  |
| `ref` | string | No | Ref is git ref can be checked out for a change<br />for example,<br />github: pull/123/head<br />gerrit: refs/changes/00/123/1 |
| `link` | string | No | Link links to the pull request itself. |
| `commit_link` | string | No | CommitLink links to the commit identified by the SHA. |
| `author_link` | string | No | AuthorLink links to the author of the pull request. |

## Refs

Refs describes how the repo was constructed.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `org` | string | Yes | Org is something like kubernetes or k8s.io |
| `repo` | string | Yes | Repo is something like test-infra |
| `repo_link` | string | No | RepoLink links to the source for Repo. |
| `base_ref` | string | No |  |
| `base_sha` | string | No |  |
| `base_link` | string | No | BaseLink is a link to the commit identified by BaseSHA. |
| `pulls` | [][Pull](./github-com-jenkins-x-lighthouse-pkg-apis-lighthouse-v1alpha1.md#Pull) | No |  |
| `path_alias` | string | No | PathAlias is the location under <root-dir>/src<br />where this repository is cloned. If this is not<br />set, <root-dir>/src/github.com/org/repo will be<br />used as the default. |
| `clone_uri` | string | No | CloneURI is the URI that is used to clone the<br />repository. If unset, will default to<br />`https://github.com/org/repo.git`. |
| `skip_submodules` | bool | No | SkipSubmodules determines if submodules should be<br />cloned when the job is run. Defaults to true. |
| `clone_depth` | int | No | CloneDepth is the depth of the clone that will be used.<br />A depth of zero will do a full clone. |


