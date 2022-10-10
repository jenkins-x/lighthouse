# Package github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1

- [ArrayOrString](#ArrayOrString)
- [EmbeddedTask](#EmbeddedTask)
- [Param](#Param)
- [ParamSpec](#ParamSpec)
- [ParamType](#ParamType)
- [PipelineDeclaredResource](#PipelineDeclaredResource)
- [PipelineRef](#PipelineRef)
- [PipelineResourceBinding](#PipelineResourceBinding)
- [PipelineResourceRef](#PipelineResourceRef)
- [PipelineResourceType](#PipelineResourceType)
- [PipelineResult](#PipelineResult)
- [PipelineRunSpec](#PipelineRunSpec)
- [PipelineRunSpecServiceAccountName](#PipelineRunSpecServiceAccountName)
- [PipelineRunSpecStatus](#PipelineRunSpecStatus)
- [PipelineSpec](#PipelineSpec)
- [PipelineTask](#PipelineTask)
- [PipelineTaskCondition](#PipelineTaskCondition)
- [PipelineTaskInputResource](#PipelineTaskInputResource)
- [PipelineTaskMetadata](#PipelineTaskMetadata)
- [PipelineTaskOutputResource](#PipelineTaskOutputResource)
- [PipelineTaskResources](#PipelineTaskResources)
- [PipelineTaskRunSpec](#PipelineTaskRunSpec)
- [PipelineWorkspaceDeclaration](#PipelineWorkspaceDeclaration)
- [PodTemplate](#PodTemplate)
- [Sidecar](#Sidecar)
- [Step](#Step)
- [TaskKind](#TaskKind)
- [TaskRef](#TaskRef)
- [TaskResource](#TaskResource)
- [TaskResources](#TaskResources)
- [TaskResult](#TaskResult)
- [TimeoutFields](#TimeoutFields)
- [WhenExpressions](#WhenExpressions)
- [WorkspaceBinding](#WorkspaceBinding)
- [WorkspaceDeclaration](#WorkspaceDeclaration)
- [WorkspacePipelineTaskBinding](#WorkspacePipelineTaskBinding)
- [WorkspaceUsage](#WorkspaceUsage)


## ArrayOrString

ArrayOrString is a type that can hold a single string or string array.<br />Used in JSON unmarshalling so that a single JSON field can accept<br />either an individual string or an array of strings.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `type` | [ParamType](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#ParamType) | Yes |  |
| `stringVal` | string | Yes |  |
| `arrayVal` | []string | No |  |

## EmbeddedTask

EmbeddedTask is used to define a Task inline within a Pipeline's PipelineTasks.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `apiVersion` | string | No | +optional |
| `kind` | string | No | +optional |
| `spec` | [RawExtension](./k8s-io-apimachinery-pkg-runtime.md#RawExtension) | No | Spec is a specification of a custom task<br />+optional |
| `metadata` | [PipelineTaskMetadata](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineTaskMetadata) | No | +optional |
| `resources` | *[TaskResources](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#TaskResources) | No | Resources is a list input and output resource to run the task<br />Resources are represented in TaskRuns as bindings to instances of<br />PipelineResources.<br />+optional |
| `params` | [][ParamSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#ParamSpec) | No | Params is a list of input parameters required to run the task. Params<br />must be supplied as inputs in TaskRuns unless they declare a default<br />value.<br />+optional |
| `description` | string | No | Description is a user-facing description of the task that may be<br />used to populate a UI.<br />+optional |
| `steps` | [][Step](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#Step) | No | Steps are the steps of the build; each step is run sequentially with the<br />source mounted into /workspace. |
| `volumes` | [][Volume](./k8s-io-api-core-v1.md#Volume) | No | Volumes is a collection of volumes that are available to mount into the<br />steps of the build. |
| `stepTemplate` | *[Container](./k8s-io-api-core-v1.md#Container) | No | StepTemplate can be used as the basis for all step containers within the<br />Task, so that the steps inherit settings on the base container. |
| `sidecars` | [][Sidecar](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#Sidecar) | No | Sidecars are run alongside the Task's step containers. They begin before<br />the steps start and end after the steps complete. |
| `workspaces` | [][WorkspaceDeclaration](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#WorkspaceDeclaration) | No | Workspaces are the volumes that this Task requires. |
| `results` | [][TaskResult](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#TaskResult) | No | Results are values that this Task can output |

## Param

Param declares an ArrayOrString to use for the parameter called name.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes |  |
| `value` | [ArrayOrString](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#ArrayOrString) | Yes |  |

## ParamSpec

ParamSpec defines arbitrary parameters needed beyond typed inputs (such as<br />resources). Parameter values are provided by users as inputs on a TaskRun<br />or PipelineRun.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name declares the name by which a parameter is referenced. |
| `type` | [ParamType](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#ParamType) | No | Type is the user-specified type of the parameter. The possible types<br />are currently "string" and "array", and "string" is the default.<br />+optional |
| `description` | string | No | Description is a user-facing description of the parameter that may be<br />used to populate a UI.<br />+optional |
| `default` | *[ArrayOrString](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#ArrayOrString) | No | Default is the value a parameter takes if no input value is supplied. If<br />default is set, a Task may be executed without a supplied value for the<br />parameter.<br />+optional |

## ParamType

ParamType indicates the type of an input parameter;<br />Used to distinguish between a single string and an array of strings.



## PipelineDeclaredResource

PipelineDeclaredResource is used by a Pipeline to declare the types of the<br />PipelineResources that it will required to run and names which can be used to<br />refer to these PipelineResources in PipelineTaskResourceBindings.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name that will be used by the Pipeline to refer to this resource.<br />It does not directly correspond to the name of any PipelineResources Task<br />inputs or outputs, and it does not correspond to the actual names of the<br />PipelineResources that will be bound in the PipelineRun. |
| `type` | [PipelineResourceType](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineResourceType) | Yes | Type is the type of the PipelineResource. |
| `optional` | bool | No | Optional declares the resource as optional.<br />optional: true - the resource is considered optional<br />optional: false - the resource is considered required (default/equivalent of not specifying it) |

## PipelineRef

PipelineRef can be used to refer to a specific instance of a Pipeline.<br />Copied from CrossVersionObjectReference: https://github.com/kubernetes/kubernetes/blob/169df7434155cbbc22f1532cba8e0a9588e29ad8/pkg/apis/autoscaling/types.go#L64

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names |
| `apiVersion` | string | No | API version of the referent<br />+optional |
| `bundle` | string | No | Bundle url reference to a Tekton Bundle.<br />+optional |

## PipelineResourceBinding

PipelineResourceBinding connects a reference to an instance of a PipelineResource<br />with a PipelineResource dependency that the Pipeline has declared

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name is the name of the PipelineResource in the Pipeline's declaration |
| `resourceRef` | *[PipelineResourceRef](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineResourceRef) | No | ResourceRef is a reference to the instance of the actual PipelineResource<br />that should be used<br />+optional |
| `resourceSpec` | *[PipelineResourceSpec](./github-com-tektoncd-pipeline-pkg-apis-resource-v1alpha1.md#PipelineResourceSpec) | No | ResourceSpec is specification of a resource that should be created and<br />consumed by the task<br />+optional |

## PipelineResourceRef

PipelineResourceRef can be used to refer to a specific instance of a Resource

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names |
| `apiVersion` | string | No | API version of the referent<br />+optional |

## PipelineResourceType

PipelineResourceType represents the type of endpoint the pipelineResource is, so that the<br />controller will know this pipelineResource should be fetched and optionally what<br />additional metatdata should be provided for it.



## PipelineResult

PipelineResult used to describe the results of a pipeline

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name the given name |
| `description` | string | Yes | Description is a human-readable description of the result<br />+optional |
| `value` | string | Yes | Value the expression used to retrieve the value |

## PipelineRunSpec

PipelineRunSpec defines the desired state of PipelineRun

| Stanza | Type | Required | Description |
|---|---|---|---|
| `pipelineRef` | *[PipelineRef](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineRef) | No | +optional |
| `pipelineSpec` | *[PipelineSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineSpec) | No | +optional |
| `resources` | [][PipelineResourceBinding](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineResourceBinding) | No | Resources is a list of bindings specifying which actual instances of<br />PipelineResources to use for the resources the Pipeline has declared<br />it needs. |
| `params` | [][Param](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#Param) | No | Params is a list of parameter names and values. |
| `serviceAccountName` | string | No | +optional |
| `serviceAccountNames` | [][PipelineRunSpecServiceAccountName](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineRunSpecServiceAccountName) | No | Deprecated: use taskRunSpecs.ServiceAccountName instead<br />+optional |
| `status` | [PipelineRunSpecStatus](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineRunSpecStatus) | No | Used for cancelling a pipelinerun (and maybe more later on)<br />+optional |
| `timeouts` | *[TimeoutFields](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#TimeoutFields) | No | This is an alpha field. You must set the "enable-api-fields" feature flag to "alpha"<br />for this field to be supported.<br /><br />Time after which the Pipeline times out.<br />Currently three keys are accepted in the map<br />pipeline, tasks and finally<br />with Timeouts.pipeline >= Timeouts.tasks + Timeouts.finally<br />+optional |
| `timeout` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Time after which the Pipeline times out. Defaults to never.<br />Refer to Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration<br />+optional |
| `podTemplate` | *[PodTemplate](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PodTemplate) | No | PodTemplate holds pod specific configuration |
| `workspaces` | [][WorkspaceBinding](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#WorkspaceBinding) | No | Workspaces holds a set of workspace bindings that must match names<br />with those declared in the pipeline.<br />+optional |
| `taskRunSpecs` | [][PipelineTaskRunSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineTaskRunSpec) | No | TaskRunSpecs holds a set of runtime specs<br />+optional |

## PipelineRunSpecServiceAccountName

PipelineRunSpecServiceAccountName can be used to configure specific<br />ServiceAccountName for a concrete Task

| Stanza | Type | Required | Description |
|---|---|---|---|
| `taskName` | string | No |  |
| `serviceAccountName` | string | No |  |

## PipelineRunSpecStatus

PipelineRunSpecStatus defines the pipelinerun spec status the user can provide



## PipelineSpec

PipelineSpec defines the desired state of Pipeline.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `description` | string | No | Description is a user-facing description of the pipeline that may be<br />used to populate a UI.<br />+optional |
| `resources` | [][PipelineDeclaredResource](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineDeclaredResource) | No | Resources declares the names and types of the resources given to the<br />Pipeline's tasks as inputs and outputs. |
| `tasks` | [][PipelineTask](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineTask) | No | Tasks declares the graph of Tasks that execute when this Pipeline is run. |
| `params` | [][ParamSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#ParamSpec) | No | Params declares a list of input parameters that must be supplied when<br />this Pipeline is run. |
| `workspaces` | [][PipelineWorkspaceDeclaration](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineWorkspaceDeclaration) | No | Workspaces declares a set of named workspaces that are expected to be<br />provided by a PipelineRun.<br />+optional |
| `results` | [][PipelineResult](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineResult) | No | Results are values that this pipeline can output once run<br />+optional |
| `finally` | [][PipelineTask](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineTask) | No | Finally declares the list of Tasks that execute just before leaving the Pipeline<br />i.e. either after all Tasks are finished executing successfully<br />or after a failure which would result in ending the Pipeline |

## PipelineTask

PipelineTask defines a task in a Pipeline, passing inputs from both<br />Params and from the output of previous tasks.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name is the name of this task within the context of a Pipeline. Name is<br />used as a coordinate with the `from` and `runAfter` fields to establish<br />the execution order of tasks relative to one another. |
| `taskRef` | *[TaskRef](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#TaskRef) | No | TaskRef is a reference to a task definition.<br />+optional |
| `taskSpec` | *[EmbeddedTask](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#EmbeddedTask) | No | TaskSpec is a specification of a task<br />+optional |
| `conditions` | [][PipelineTaskCondition](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineTaskCondition) | No | Conditions is a list of conditions that need to be true for the task to run<br />Conditions are deprecated, use WhenExpressions instead<br />+optional |
| `when` | [WhenExpressions](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#WhenExpressions) | No | WhenExpressions is a list of when expressions that need to be true for the task to run<br />+optional |
| `retries` | int | No | Retries represents how many times this task should be retried in case of task failure: ConditionSucceeded set to False<br />+optional |
| `runAfter` | []string | No | RunAfter is the list of PipelineTask names that should be executed before<br />this Task executes. (Used to force a specific ordering in graph execution.)<br />+optional |
| `resources` | *[PipelineTaskResources](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineTaskResources) | No | Resources declares the resources given to this task as inputs and<br />outputs.<br />+optional |
| `params` | [][Param](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#Param) | No | Parameters declares parameters passed to this task.<br />+optional |
| `workspaces` | [][WorkspacePipelineTaskBinding](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#WorkspacePipelineTaskBinding) | No | Workspaces maps workspaces from the pipeline spec to the workspaces<br />declared in the Task.<br />+optional |
| `timeout` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Time after which the TaskRun times out. Defaults to 1 hour.<br />Specified TaskRun timeout should be less than 24h.<br />Refer Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration<br />+optional |

## PipelineTaskCondition

PipelineTaskCondition allows a PipelineTask to declare a Condition to be evaluated before<br />the Task is run.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `conditionRef` | string | Yes | ConditionRef is the name of the Condition to use for the conditionCheck |
| `params` | [][Param](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#Param) | No | Params declare parameters passed to this Condition<br />+optional |
| `resources` | [][PipelineTaskInputResource](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineTaskInputResource) | No | Resources declare the resources provided to this Condition as input |

## PipelineTaskInputResource

PipelineTaskInputResource maps the name of a declared PipelineResource input<br />dependency in a Task to the resource in the Pipeline's DeclaredPipelineResources<br />that should be used. This input may come from a previous task.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name of the PipelineResource as declared by the Task. |
| `resource` | string | Yes | Resource is the name of the DeclaredPipelineResource to use. |
| `from` | []string | No | From is the list of PipelineTask names that the resource has to come from.<br />(Implies an ordering in the execution graph.)<br />+optional |

## PipelineTaskMetadata

PipelineTaskMetadata contains the labels or annotations for an EmbeddedTask

| Stanza | Type | Required | Description |
|---|---|---|---|
| `labels` | map[string]string | No | +optional |
| `annotations` | map[string]string | No | +optional |

## PipelineTaskOutputResource

PipelineTaskOutputResource maps the name of a declared PipelineResource output<br />dependency in a Task to the resource in the Pipeline's DeclaredPipelineResources<br />that should be used.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name of the PipelineResource as declared by the Task. |
| `resource` | string | Yes | Resource is the name of the DeclaredPipelineResource to use. |

## PipelineTaskResources

PipelineTaskResources allows a Pipeline to declare how its DeclaredPipelineResources<br />should be provided to a Task as its inputs and outputs.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `inputs` | [][PipelineTaskInputResource](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineTaskInputResource) | No | Inputs holds the mapping from the PipelineResources declared in<br />DeclaredPipelineResources to the input PipelineResources required by the Task. |
| `outputs` | [][PipelineTaskOutputResource](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PipelineTaskOutputResource) | No | Outputs holds the mapping from the PipelineResources declared in<br />DeclaredPipelineResources to the input PipelineResources required by the Task. |

## PipelineTaskRunSpec

PipelineTaskRunSpec  can be used to configure specific<br />specs for a concrete Task

| Stanza | Type | Required | Description |
|---|---|---|---|
| `pipelineTaskName` | string | No |  |
| `taskServiceAccountName` | string | No |  |
| `taskPodTemplate` | *[PodTemplate](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#PodTemplate) | No |  |

## PipelineWorkspaceDeclaration

PipelineWorkspaceDeclaration creates a named slot in a Pipeline that a PipelineRun<br />is expected to populate with a workspace binding.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name of a workspace to be provided by a PipelineRun. |
| `description` | string | No | Description is a human readable string describing how the workspace will be<br />used in the Pipeline. It can be useful to include a bit of detail about which<br />tasks are intended to have access to the data on the workspace.<br />+optional |
| `optional` | bool | No | Optional marks a Workspace as not being required in PipelineRuns. By default<br />this field is false and so declared workspaces are required. |

## PodTemplate

PodTemplate holds pod specific configuration



## Sidecar

Sidecar has nearly the same data structure as Step, consisting of a Container and an optional Script, but does not have the ability to timeout.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name of the container specified as a DNS_LABEL.<br />Each container in a pod must have a unique name (DNS_LABEL).<br />Cannot be updated. |
| `image` | string | No | Docker image name.<br />More info: https://kubernetes.io/docs/concepts/containers/images<br />This field is optional to allow higher level config management to default or override<br />container images in workload controllers like Deployments and StatefulSets.<br />+optional |
| `command` | []string | No | Entrypoint array. Not executed within a shell.<br />The docker image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional |
| `args` | []string | No | Arguments to the entrypoint.<br />The docker image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional |
| `workingDir` | string | No | Container's working directory.<br />If not specified, the container runtime's default will be used, which<br />might be configured in the container image.<br />Cannot be updated.<br />+optional |
| `ports` | [][ContainerPort](./k8s-io-api-core-v1.md#ContainerPort) | No | List of ports to expose from the container. Exposing a port here gives<br />the system additional information about the network connections a<br />container uses, but is primarily informational. Not specifying a port here<br />DOES NOT prevent that port from being exposed. Any port which is<br />listening on the default "0.0.0.0" address inside a container will be<br />accessible from the network.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=containerPort<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=containerPort<br />+listMapKey=protocol |
| `envFrom` | [][EnvFromSource](./k8s-io-api-core-v1.md#EnvFromSource) | No | List of sources to populate environment variables in the container.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the container is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated.<br />+optional |
| `env` | [][EnvVar](./k8s-io-api-core-v1.md#EnvVar) | No | List of environment variables to set in the container.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=name<br />+patchStrategy=merge |
| `resources` | [ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | No | Compute Resources required by this container.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br />+optional |
| `volumeMounts` | [][VolumeMount](./k8s-io-api-core-v1.md#VolumeMount) | No | Pod volumes to mount into the container's filesystem.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=mountPath<br />+patchStrategy=merge |
| `volumeDevices` | [][VolumeDevice](./k8s-io-api-core-v1.md#VolumeDevice) | No | volumeDevices is the list of block devices to be used by the container.<br />+patchMergeKey=devicePath<br />+patchStrategy=merge<br />+optional |
| `livenessProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | Periodic probe of container liveness.<br />Container will be restarted if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `readinessProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | Periodic probe of container service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `startupProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | StartupProbe indicates that the Pod has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />This cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `lifecycle` | *[Lifecycle](./k8s-io-api-core-v1.md#Lifecycle) | No | Actions that the management system should take in response to container lifecycle events.<br />Cannot be updated.<br />+optional |
| `terminationMessagePath` | string | No | Optional: Path at which the file to which the container's termination message<br />will be written is mounted into the container's filesystem.<br />Message written is intended to be brief final status, such as an assertion failure message.<br />Will be truncated by the node if greater than 4096 bytes. The total message length across<br />all containers will be limited to 12kb.<br />Defaults to /dev/termination-log.<br />Cannot be updated.<br />+optional |
| `terminationMessagePolicy` | [TerminationMessagePolicy](./k8s-io-api-core-v1.md#TerminationMessagePolicy) | No | Indicate how the termination message should be populated. File will use the contents of<br />terminationMessagePath to populate the container status message on both success and failure.<br />FallbackToLogsOnError will use the last chunk of container log output if the termination<br />message file is empty and the container exited with an error.<br />The log output is limited to 2048 bytes or 80 lines, whichever is smaller.<br />Defaults to File.<br />Cannot be updated.<br />+optional |
| `imagePullPolicy` | [PullPolicy](./k8s-io-api-core-v1.md#PullPolicy) | No | Image pull policy.<br />One of Always, Never, IfNotPresent.<br />Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images<br />+optional |
| `securityContext` | *[SecurityContext](./k8s-io-api-core-v1.md#SecurityContext) | No | SecurityContext defines the security options the container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/<br />+optional |
| `stdin` | bool | No | Whether this container should allocate a buffer for stdin in the container runtime. If this<br />is not set, reads from stdin in the container will always result in EOF.<br />Default is false.<br />+optional |
| `stdinOnce` | bool | No | Whether the container runtime should close the stdin channel after it has been opened by<br />a single attach. When stdin is true the stdin stream will remain open across multiple attach<br />sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the<br />first client attaches to stdin, and then remains open and accepts data until the client disconnects,<br />at which time stdin is closed and remains closed until the container is restarted. If this<br />flag is false, a container processes that reads from stdin will never receive an EOF.<br />Default is false<br />+optional |
| `tty` | bool | No | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true.<br />Default is false.<br />+optional |
| `script` | string | No | Script is the contents of an executable file to execute.<br /><br />If Script is not empty, the Step cannot have an Command or Args.<br />+optional |
| `workspaces` | [][WorkspaceUsage](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#WorkspaceUsage) | No | This is an alpha field. You must set the "enable-api-fields" feature flag to "alpha"<br />for this field to be supported.<br /><br />Workspaces is a list of workspaces from the Task that this Sidecar wants<br />exclusive access to. Adding a workspace to this list means that any<br />other Step or Sidecar that does not also request this Workspace will<br />not have access to it.<br />+optional |

## Step

Step embeds the Container type, which allows it to include fields not<br />provided by Container.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name of the container specified as a DNS_LABEL.<br />Each container in a pod must have a unique name (DNS_LABEL).<br />Cannot be updated. |
| `image` | string | No | Docker image name.<br />More info: https://kubernetes.io/docs/concepts/containers/images<br />This field is optional to allow higher level config management to default or override<br />container images in workload controllers like Deployments and StatefulSets.<br />+optional |
| `command` | []string | No | Entrypoint array. Not executed within a shell.<br />The docker image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional |
| `args` | []string | No | Arguments to the entrypoint.<br />The docker image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional |
| `workingDir` | string | No | Container's working directory.<br />If not specified, the container runtime's default will be used, which<br />might be configured in the container image.<br />Cannot be updated.<br />+optional |
| `ports` | [][ContainerPort](./k8s-io-api-core-v1.md#ContainerPort) | No | List of ports to expose from the container. Exposing a port here gives<br />the system additional information about the network connections a<br />container uses, but is primarily informational. Not specifying a port here<br />DOES NOT prevent that port from being exposed. Any port which is<br />listening on the default "0.0.0.0" address inside a container will be<br />accessible from the network.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=containerPort<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=containerPort<br />+listMapKey=protocol |
| `envFrom` | [][EnvFromSource](./k8s-io-api-core-v1.md#EnvFromSource) | No | List of sources to populate environment variables in the container.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the container is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated.<br />+optional |
| `env` | [][EnvVar](./k8s-io-api-core-v1.md#EnvVar) | No | List of environment variables to set in the container.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=name<br />+patchStrategy=merge |
| `resources` | [ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | No | Compute Resources required by this container.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br />+optional |
| `volumeMounts` | [][VolumeMount](./k8s-io-api-core-v1.md#VolumeMount) | No | Pod volumes to mount into the container's filesystem.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=mountPath<br />+patchStrategy=merge |
| `volumeDevices` | [][VolumeDevice](./k8s-io-api-core-v1.md#VolumeDevice) | No | volumeDevices is the list of block devices to be used by the container.<br />+patchMergeKey=devicePath<br />+patchStrategy=merge<br />+optional |
| `livenessProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | Periodic probe of container liveness.<br />Container will be restarted if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `readinessProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | Periodic probe of container service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `startupProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | StartupProbe indicates that the Pod has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />This cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `lifecycle` | *[Lifecycle](./k8s-io-api-core-v1.md#Lifecycle) | No | Actions that the management system should take in response to container lifecycle events.<br />Cannot be updated.<br />+optional |
| `terminationMessagePath` | string | No | Optional: Path at which the file to which the container's termination message<br />will be written is mounted into the container's filesystem.<br />Message written is intended to be brief final status, such as an assertion failure message.<br />Will be truncated by the node if greater than 4096 bytes. The total message length across<br />all containers will be limited to 12kb.<br />Defaults to /dev/termination-log.<br />Cannot be updated.<br />+optional |
| `terminationMessagePolicy` | [TerminationMessagePolicy](./k8s-io-api-core-v1.md#TerminationMessagePolicy) | No | Indicate how the termination message should be populated. File will use the contents of<br />terminationMessagePath to populate the container status message on both success and failure.<br />FallbackToLogsOnError will use the last chunk of container log output if the termination<br />message file is empty and the container exited with an error.<br />The log output is limited to 2048 bytes or 80 lines, whichever is smaller.<br />Defaults to File.<br />Cannot be updated.<br />+optional |
| `imagePullPolicy` | [PullPolicy](./k8s-io-api-core-v1.md#PullPolicy) | No | Image pull policy.<br />One of Always, Never, IfNotPresent.<br />Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images<br />+optional |
| `securityContext` | *[SecurityContext](./k8s-io-api-core-v1.md#SecurityContext) | No | SecurityContext defines the security options the container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/<br />+optional |
| `stdin` | bool | No | Whether this container should allocate a buffer for stdin in the container runtime. If this<br />is not set, reads from stdin in the container will always result in EOF.<br />Default is false.<br />+optional |
| `stdinOnce` | bool | No | Whether the container runtime should close the stdin channel after it has been opened by<br />a single attach. When stdin is true the stdin stream will remain open across multiple attach<br />sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the<br />first client attaches to stdin, and then remains open and accepts data until the client disconnects,<br />at which time stdin is closed and remains closed until the container is restarted. If this<br />flag is false, a container processes that reads from stdin will never receive an EOF.<br />Default is false<br />+optional |
| `tty` | bool | No | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true.<br />Default is false.<br />+optional |
| `script` | string | No | Script is the contents of an executable file to execute.<br /><br />If Script is not empty, the Step cannot have an Command and the Args will be passed to the Script.<br />+optional |
| `timeout` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Timeout is the time after which the step times out. Defaults to never.<br />Refer to Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration<br />+optional |
| `workspaces` | [][WorkspaceUsage](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#WorkspaceUsage) | No | This is an alpha field. You must set the "enable-api-fields" feature flag to "alpha"<br />for this field to be supported.<br /><br />Workspaces is a list of workspaces from the Task that this Step wants<br />exclusive access to. Adding a workspace to this list means that any<br />other Step or Sidecar that does not also request this Workspace will<br />not have access to it.<br />+optional |
| `onError` | string | No | OnError defines the exiting behavior of a container on error<br />can be set to [ continue | stopAndFail ]<br />stopAndFail indicates exit the taskRun if the container exits with non-zero exit code<br />continue indicates continue executing the rest of the steps irrespective of the container exit code |

## TaskKind

TaskKind defines the type of Task used by the pipeline.



## TaskRef

TaskRef can be used to refer to a specific instance of a task.<br />Copied from CrossVersionObjectReference: https://github.com/kubernetes/kubernetes/blob/169df7434155cbbc22f1532cba8e0a9588e29ad8/pkg/apis/autoscaling/types.go#L64

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names |
| `kind` | [TaskKind](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#TaskKind) | No | TaskKind indicates the kind of the task, namespaced or cluster scoped. |
| `apiVersion` | string | No | API version of the referent<br />+optional |
| `bundle` | string | No | Bundle url reference to a Tekton Bundle.<br />+optional |

## TaskResource

TaskResource defines an input or output Resource declared as a requirement<br />by a Task. The Name field will be used to refer to these Resources within<br />the Task definition, and when provided as an Input, the Name will be the<br />path to the volume mounted containing this Resource as an input (e.g.<br />an input Resource named `workspace` will be mounted at `/workspace`).



## TaskResources

TaskResources allows a Pipeline to declare how its DeclaredPipelineResources<br />should be provided to a Task as its inputs and outputs.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `inputs` | [][TaskResource](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#TaskResource) | No | Inputs holds the mapping from the PipelineResources declared in<br />DeclaredPipelineResources to the input PipelineResources required by the Task. |
| `outputs` | [][TaskResource](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1beta1.md#TaskResource) | No | Outputs holds the mapping from the PipelineResources declared in<br />DeclaredPipelineResources to the input PipelineResources required by the Task. |

## TaskResult

TaskResult used to describe the results of a task

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name the given name |
| `description` | string | Yes | Description is a human-readable description of the result<br />+optional |

## TimeoutFields

TimeoutFields allows granular specification of pipeline, task, and finally timeouts

| Stanza | Type | Required | Description |
|---|---|---|---|
| `pipeline` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Pipeline sets the maximum allowed duration for execution of the entire pipeline. The sum of individual timeouts for tasks and finally must not exceed this value. |
| `tasks` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Tasks sets the maximum allowed duration of this pipeline's tasks |
| `finally` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Finally sets the maximum allowed duration of this pipeline's finally |

## WhenExpressions

WhenExpressions are used to specify whether a Task should be executed or skipped<br />All of them need to evaluate to True for a guarded Task to be executed.



## WorkspaceBinding

WorkspaceBinding maps a Task's declared workspace to a Volume.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name of the workspace populated by the volume. |
| `subPath` | string | No | SubPath is optionally a directory on the volume which should be used<br />for this binding (i.e. the volume will be mounted at this sub directory).<br />+optional |
| `volumeClaimTemplate` | *[PersistentVolumeClaim](./k8s-io-api-core-v1.md#PersistentVolumeClaim) | No | VolumeClaimTemplate is a template for a claim that will be created in the same namespace.<br />The PipelineRun controller is responsible for creating a unique claim for each instance of PipelineRun.<br />+optional |
| `persistentVolumeClaim` | *[PersistentVolumeClaimVolumeSource](./k8s-io-api-core-v1.md#PersistentVolumeClaimVolumeSource) | No | PersistentVolumeClaimVolumeSource represents a reference to a<br />PersistentVolumeClaim in the same namespace. Either this OR EmptyDir can be used.<br />+optional |
| `emptyDir` | *[EmptyDirVolumeSource](./k8s-io-api-core-v1.md#EmptyDirVolumeSource) | No | EmptyDir represents a temporary directory that shares a Task's lifetime.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir<br />Either this OR PersistentVolumeClaim can be used.<br />+optional |
| `configMap` | *[ConfigMapVolumeSource](./k8s-io-api-core-v1.md#ConfigMapVolumeSource) | No | ConfigMap represents a configMap that should populate this workspace.<br />+optional |
| `secret` | *[SecretVolumeSource](./k8s-io-api-core-v1.md#SecretVolumeSource) | No | Secret represents a secret that should populate this workspace.<br />+optional |

## WorkspaceDeclaration

WorkspaceDeclaration is a declaration of a volume that a Task requires.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name by which you can bind the volume at runtime. |
| `description` | string | No | Description is an optional human readable description of this volume.<br />+optional |
| `mountPath` | string | No | MountPath overrides the directory that the volume will be made available at.<br />+optional |
| `readOnly` | bool | No | ReadOnly dictates whether a mounted volume is writable. By default this<br />field is false and so mounted volumes are writable. |
| `optional` | bool | No | Optional marks a Workspace as not being required in TaskRuns. By default<br />this field is false and so declared workspaces are required. |

## WorkspacePipelineTaskBinding

WorkspacePipelineTaskBinding describes how a workspace passed into the pipeline should be<br />mapped to a task's declared workspace.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name of the workspace as declared by the task |
| `workspace` | string | Yes | Workspace is the name of the workspace declared by the pipeline |
| `subPath` | string | No | SubPath is optionally a directory on the volume which should be used<br />for this binding (i.e. the volume will be mounted at this sub directory).<br />+optional |

## WorkspaceUsage

WorkspaceUsage is used by a Step or Sidecar to declare that it wants isolated access<br />to a Workspace defined in a Task.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name of the workspace this Step or Sidecar wants access to. |
| `mountPath` | string | Yes | MountPath is the path that the workspace should be mounted to inside the Step or Sidecar,<br />overriding any MountPath specified in the Task's WorkspaceDeclaration. |


