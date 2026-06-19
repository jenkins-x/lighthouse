# Package github.com/tektoncd/pipeline/pkg/apis/pipeline/v1

- [EmbeddedTask](#EmbeddedTask)
- [IncludeParamsList](#IncludeParamsList)
- [Matrix](#Matrix)
- [OnErrorType](#OnErrorType)
- [ParamSpecs](#ParamSpecs)
- [ParamType](#ParamType)
- [Params](#Params)
- [PipelineRef](#PipelineRef)
- [PipelineResult](#PipelineResult)
- [PipelineRunSpec](#PipelineRunSpec)
- [PipelineRunSpecStatus](#PipelineRunSpecStatus)
- [PipelineSpec](#PipelineSpec)
- [PipelineTask](#PipelineTask)
- [PipelineTaskMetadata](#PipelineTaskMetadata)
- [PipelineTaskOnErrorType](#PipelineTaskOnErrorType)
- [PipelineTaskRunSpec](#PipelineTaskRunSpec)
- [PipelineTaskRunTemplate](#PipelineTaskRunTemplate)
- [PipelineWorkspaceDeclaration](#PipelineWorkspaceDeclaration)
- [PropertySpec](#PropertySpec)
- [Ref](#Ref)
- [ResolverName](#ResolverName)
- [ResultValue](#ResultValue)
- [ResultsType](#ResultsType)
- [Sidecar](#Sidecar)
- [Step](#Step)
- [StepOutputConfig](#StepOutputConfig)
- [StepResult](#StepResult)
- [StepTemplate](#StepTemplate)
- [StepWhenExpressions](#StepWhenExpressions)
- [TaskBreakpoints](#TaskBreakpoints)
- [TaskKind](#TaskKind)
- [TaskRef](#TaskRef)
- [TaskResult](#TaskResult)
- [TaskRunDebug](#TaskRunDebug)
- [TaskRunSidecarSpec](#TaskRunSidecarSpec)
- [TaskRunStepSpec](#TaskRunStepSpec)
- [TimeoutFields](#TimeoutFields)
- [Volumes](#Volumes)
- [WhenExpressions](#WhenExpressions)
- [WorkspaceBinding](#WorkspaceBinding)
- [WorkspaceDeclaration](#WorkspaceDeclaration)
- [WorkspacePipelineTaskBinding](#WorkspacePipelineTaskBinding)
- [WorkspaceUsage](#WorkspaceUsage)


## EmbeddedTask

EmbeddedTask is used to define a Task inline within a Pipeline's PipelineTasks.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `apiVersion` | string | No | +optional |
| `kind` | string | No | +optional |
| `spec` | [RawExtension](./k8s-io-apimachinery-pkg-runtime.md#RawExtension) | No | Spec is a specification of a custom task<br />+optional |
| `metadata` | [PipelineTaskMetadata](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineTaskMetadata) | No | +optional |
| `params` | [ParamSpecs](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ParamSpecs) | No | Params is a list of input parameters required to run the task. Params<br />must be supplied as inputs in TaskRuns unless they declare a default<br />value.<br />+optional |
| `displayName` | string | No | DisplayName is a user-facing name of the task that may be<br />used to populate a UI.<br />+optional |
| `description` | string | No | Description is a user-facing description of the task that may be<br />used to populate a UI.<br />+optional |
| `steps` | [][Step](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Step) | No | Steps are the steps of the build; each step is run sequentially with the<br />source mounted into /workspace.<br />+listType=atomic |
| `volumes` | [Volumes](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Volumes) | No | Volumes is a collection of volumes that are available to mount into the<br />steps of the build.<br />See Pod.spec.volumes (API version: v1)<br />+kubebuilder:pruning:PreserveUnknownFields<br />+kubebuilder:validation:Schemaless |
| `stepTemplate` | *[StepTemplate](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#StepTemplate) | No | StepTemplate can be used as the basis for all step containers within the<br />Task, so that the steps inherit settings on the base container. |
| `sidecars` | [][Sidecar](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Sidecar) | No | Sidecars are run alongside the Task's step containers. They begin before<br />the steps start and end after the steps complete.<br />+listType=atomic |
| `workspaces` | [][WorkspaceDeclaration](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#WorkspaceDeclaration) | No | Workspaces are the volumes that this Task requires.<br />+listType=atomic |
| `results` | [][TaskResult](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#TaskResult) | No | Results are values that this Task can output<br />+listType=atomic |

## IncludeParamsList

IncludeParamsList is a list of IncludeParams which allows passing in specific combinations of Parameters into the Matrix.<br />+listType=atomic



## Matrix

Matrix is used to fan out Tasks in a Pipeline

| Stanza | Type | Required | Description |
|---|---|---|---|
| `params` | [Params](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Params) | No | Params is a list of parameters used to fan out the pipelineTask<br />Params takes only `Parameters` of type `"array"`<br />Each array element is supplied to the `PipelineTask` by substituting `params` of type `"string"` in the underlying `Task`.<br />The names of the `params` in the `Matrix` must match the names of the `params` in the underlying `Task` that they will be substituting. |
| `include` | [IncludeParamsList](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#IncludeParamsList) | No | Include is a list of IncludeParams which allows passing in specific combinations of Parameters into the Matrix.<br />+optional |

## OnErrorType

OnErrorType defines a list of supported exiting behavior of a container on error



## ParamSpecs

ParamSpecs is a list of ParamSpec<br />+listType=atomic



## ParamType

ParamType indicates the type of an input parameter;<br />Used to distinguish between a single string and an array of strings.



## Params

Params is a list of Param<br />+listType=atomic



## PipelineRef

PipelineRef can be used to refer to a specific instance of a Pipeline.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names |
| `apiVersion` | string | No | API version of the referent<br />+optional |
| `resolver` | [ResolverName](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ResolverName) | No | Resolver is the name of the resolver that should perform<br />resolution of the referenced Tekton resource, such as "git".<br />+optional |
| `params` | [Params](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Params) | No | Params contains the parameters used to identify the<br />referenced Tekton resource. Example entries might include<br />"repo" or "path" but the set of params ultimately depends on<br />the chosen resolver.<br />+optional |

## PipelineResult

PipelineResult used to describe the results of a pipeline

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name the given name |
| `type` | [ResultsType](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ResultsType) | No | Type is the user-specified type of the result.<br />The possible types are 'string', 'array', and 'object', with 'string' as the default.<br />'array' and 'object' types are alpha features. |
| `description` | string | Yes | Description is a human-readable description of the result<br />+optional |
| `value` | [ResultValue](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ResultValue) | Yes | Value the expression used to retrieve the value<br />+kubebuilder:pruning:PreserveUnknownFields<br />+kubebuilder:validation:Schemaless |

## PipelineRunSpec

PipelineRunSpec defines the desired state of PipelineRun

| Stanza | Type | Required | Description |
|---|---|---|---|
| `pipelineRef` | *[PipelineRef](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineRef) | No | +optional |
| `pipelineSpec` | *[PipelineSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineSpec) | No | Specifying PipelineSpec can be disabled by setting<br />`disable-inline-spec` feature flag.<br />See Pipeline.spec (API version: tekton.dev/v1)<br />+optional<br />+kubebuilder:pruning:PreserveUnknownFields<br />+kubebuilder:validation:Schemaless |
| `params` | [Params](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Params) | No | Params is a list of parameter names and values. |
| `status` | [PipelineRunSpecStatus](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineRunSpecStatus) | No | Used for cancelling a pipelinerun (and maybe more later on)<br />+optional |
| `timeouts` | *[TimeoutFields](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#TimeoutFields) | No | Time after which the Pipeline times out.<br />Currently three keys are accepted in the map<br />pipeline, tasks and finally<br />with Timeouts.pipeline >= Timeouts.tasks + Timeouts.finally<br />+optional |
| `taskRunTemplate` | [PipelineTaskRunTemplate](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineTaskRunTemplate) | No | TaskRunTemplate represent template of taskrun<br />+optional |
| `workspaces` | [][WorkspaceBinding](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#WorkspaceBinding) | No | Workspaces holds a set of workspace bindings that must match names<br />with those declared in the pipeline.<br />+optional<br />+listType=atomic |
| `taskRunSpecs` | [][PipelineTaskRunSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineTaskRunSpec) | No | TaskRunSpecs holds a set of runtime specs<br />+optional<br />+listType=atomic |
| `managedBy` | *string | No | ManagedBy indicates which controller is responsible for reconciling<br />this resource. If unset or set to "tekton.dev/pipeline", the default<br />Tekton controller will manage this resource.<br />This field is immutable.<br />+optional |

## PipelineRunSpecStatus

PipelineRunSpecStatus defines the pipelinerun spec status the user can provide



## PipelineSpec

PipelineSpec defines the desired state of Pipeline.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `displayName` | string | No | DisplayName is a user-facing name of the pipeline that may be<br />used to populate a UI.<br />+optional |
| `description` | string | No | Description is a user-facing description of the pipeline that may be<br />used to populate a UI.<br />+optional |
| `tasks` | [][PipelineTask](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineTask) | No | Tasks declares the graph of Tasks that execute when this Pipeline is run.<br />+listType=atomic |
| `params` | [ParamSpecs](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ParamSpecs) | No | Params declares a list of input parameters that must be supplied when<br />this Pipeline is run. |
| `workspaces` | [][PipelineWorkspaceDeclaration](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineWorkspaceDeclaration) | No | Workspaces declares a set of named workspaces that are expected to be<br />provided by a PipelineRun.<br />+optional<br />+listType=atomic |
| `results` | [][PipelineResult](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineResult) | No | Results are values that this pipeline can output once run<br />+optional<br />+listType=atomic |
| `finally` | [][PipelineTask](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineTask) | No | Finally declares the list of Tasks that execute just before leaving the Pipeline<br />i.e. either after all Tasks are finished executing successfully<br />or after a failure which would result in ending the Pipeline<br />+listType=atomic |

## PipelineTask

PipelineTask defines a task in a Pipeline, passing inputs from both<br />Params and from the output of previous tasks.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name is the name of this task within the context of a Pipeline. Name is<br />used as a coordinate with the `from` and `runAfter` fields to establish<br />the execution order of tasks relative to one another. |
| `displayName` | string | No | DisplayName is the display name of this task within the context of a Pipeline.<br />This display name may be used to populate a UI.<br />+optional |
| `description` | string | No | Description is the description of this task within the context of a Pipeline.<br />This description may be used to populate a UI.<br />+optional |
| `taskRef` | *[TaskRef](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#TaskRef) | No | TaskRef is a reference to a task definition.<br />+optional |
| `taskSpec` | *[EmbeddedTask](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#EmbeddedTask) | No | TaskSpec is a specification of a task<br />Specifying TaskSpec can be disabled by setting<br />`disable-inline-spec` feature flag.<br />See Task.spec (API version: tekton.dev/v1)<br />+optional<br />+kubebuilder:pruning:PreserveUnknownFields<br />+kubebuilder:validation:Schemaless |
| `when` | [WhenExpressions](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#WhenExpressions) | No | When is a list of when expressions that need to be true for the task to run<br />+optional |
| `retries` | int | No | Retries represents how many times this task should be retried in case of task failure: ConditionSucceeded set to False<br />+optional |
| `runAfter` | []string | No | RunAfter is the list of PipelineTask names that should be executed before<br />this Task executes. (Used to force a specific ordering in graph execution.)<br />+optional<br />+listType=atomic |
| `params` | [Params](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Params) | No | Parameters declares parameters passed to this task.<br />+optional |
| `matrix` | *[Matrix](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Matrix) | No | Matrix declares parameters used to fan out this task.<br />+optional |
| `workspaces` | [][WorkspacePipelineTaskBinding](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#WorkspacePipelineTaskBinding) | No | Workspaces maps workspaces from the pipeline spec to the workspaces<br />declared in the Task.<br />+optional<br />+listType=atomic |
| `timeout` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Duration after which the TaskRun times out. Defaults to 1 hour.<br />Refer Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration<br />+optional |
| `pipelineRef` | *[PipelineRef](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineRef) | No | PipelineRef is a reference to a pipeline definition<br />Note: PipelineRef is in preview mode and not yet supported<br />+optional |
| `pipelineSpec` | *[PipelineSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineSpec) | No | PipelineSpec is a specification of a pipeline<br />Note: PipelineSpec is in preview mode and not yet supported<br />Specifying PipelineSpec can be disabled by setting<br />`disable-inline-spec` feature flag.<br />See Pipeline.spec (API version: tekton.dev/v1)<br />+optional<br />+kubebuilder:pruning:PreserveUnknownFields<br />+kubebuilder:validation:Schemaless |
| `onError` | [PipelineTaskOnErrorType](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineTaskOnErrorType) | No | OnError defines the exiting behavior of a PipelineRun on error<br />can be set to [ continue | stopAndFail ]<br />+optional |

## PipelineTaskMetadata

PipelineTaskMetadata contains the labels or annotations for an EmbeddedTask

| Stanza | Type | Required | Description |
|---|---|---|---|
| `labels` | map[string]string | No | +optional |
| `annotations` | map[string]string | No | +optional |

## PipelineTaskOnErrorType

PipelineTaskOnErrorType defines a list of supported failure handling behaviors of a PipelineTask on error



## PipelineTaskRunSpec

PipelineTaskRunSpec  can be used to configure specific<br />specs for a concrete Task

| Stanza | Type | Required | Description |
|---|---|---|---|
| `pipelineTaskName` | string | No |  |
| `serviceAccountName` | string | No |  |
| `podTemplate` | *[PodTemplate](./github-com-tektoncd-pipeline-pkg-apis-pipeline-pod.md#PodTemplate) | No |  |
| `stepSpecs` | [][TaskRunStepSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#TaskRunStepSpec) | No | +listType=atomic |
| `sidecarSpecs` | [][TaskRunSidecarSpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#TaskRunSidecarSpec) | No | +listType=atomic |
| `metadata` | *[PipelineTaskMetadata](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PipelineTaskMetadata) | No | +optional |
| `computeResources` | *[ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | No | Compute resources to use for this TaskRun |
| `timeout` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Duration after which the TaskRun times out. Overrides the timeout specified<br />on the Task's spec if specified. Takes lower precedence to PipelineRun's<br />`spec.timeouts.tasks`<br />Refer Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration<br />+optional |

## PipelineTaskRunTemplate

PipelineTaskRunTemplate is used to specify run specifications for all Task in pipelinerun.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `podTemplate` | *[PodTemplate](./github-com-tektoncd-pipeline-pkg-apis-pipeline-pod.md#PodTemplate) | No | +optional |
| `serviceAccountName` | string | No | +optional |

## PipelineWorkspaceDeclaration

PipelineWorkspaceDeclaration creates a named slot in a Pipeline that a PipelineRun<br />is expected to populate with a workspace binding.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name of a workspace to be provided by a PipelineRun. |
| `description` | string | No | Description is a human readable string describing how the workspace will be<br />used in the Pipeline. It can be useful to include a bit of detail about which<br />tasks are intended to have access to the data on the workspace.<br />+optional |
| `optional` | bool | No | Optional marks a Workspace as not being required in PipelineRuns. By default<br />this field is false and so declared workspaces are required. |

## PropertySpec

PropertySpec defines the struct for object keys

| Stanza | Type | Required | Description |
|---|---|---|---|
| `type` | [ParamType](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ParamType) | No |  |

## Ref

Ref can be used to refer to a specific instance of a StepAction.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referenced step |
| `resolver` | [ResolverName](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ResolverName) | No | Resolver is the name of the resolver that should perform<br />resolution of the referenced Tekton resource, such as "git".<br />+optional |
| `params` | [Params](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Params) | No | Params contains the parameters used to identify the<br />referenced Tekton resource. Example entries might include<br />"repo" or "path" but the set of params ultimately depends on<br />the chosen resolver.<br />+optional |

## ResolverName

ResolverName is the name of a resolver from which a resource can be<br />requested.



## ResultValue

ResultValue is a type alias of ParamValue



## ResultsType

ResultsType indicates the type of a result;<br />Used to distinguish between a single string and an array of strings.<br />Note that there is ResultType used to find out whether a<br />RunResult is from a task result or not, which is different from<br />this ResultsType.



## Sidecar

Sidecar has nearly the same data structure as Step but does not have the ability to timeout.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name of the Sidecar specified as a DNS_LABEL.<br />Each Sidecar in a Task must have a unique name (DNS_LABEL).<br />Cannot be updated. |
| `image` | string | No | Image reference name.<br />More info: https://kubernetes.io/docs/concepts/containers/images<br />+optional |
| `command` | []string | No | Entrypoint array. Not executed within a shell.<br />The image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the Sidecar's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional<br />+listType=atomic |
| `args` | []string | No | Arguments to the entrypoint.<br />The image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the Sidecar's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional<br />+listType=atomic |
| `workingDir` | string | No | Sidecar's working directory.<br />If not specified, the container runtime's default will be used, which<br />might be configured in the container image.<br />Cannot be updated.<br />+optional |
| `ports` | [][ContainerPort](./k8s-io-api-core-v1.md#ContainerPort) | No | List of ports to expose from the Sidecar. Exposing a port here gives<br />the system additional information about the network connections a<br />container uses, but is primarily informational. Not specifying a port here<br />DOES NOT prevent that port from being exposed. Any port which is<br />listening on the default "0.0.0.0" address inside a container will be<br />accessible from the network.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=containerPort<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=containerPort<br />+listMapKey=protocol |
| `envFrom` | [][EnvFromSource](./k8s-io-api-core-v1.md#EnvFromSource) | No | List of sources to populate environment variables in the Sidecar.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the container is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated.<br />+optional<br />+listType=atomic |
| `env` | [][EnvVar](./k8s-io-api-core-v1.md#EnvVar) | No | List of environment variables to set in the Sidecar.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=name<br />+patchStrategy=merge<br />+listType=atomic |
| `computeResources` | [ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | No | ComputeResources required by this Sidecar.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br />+optional |
| `volumeMounts` | [][VolumeMount](./k8s-io-api-core-v1.md#VolumeMount) | No | Volumes to mount into the Sidecar's filesystem.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=mountPath<br />+patchStrategy=merge<br />+listType=atomic |
| `volumeDevices` | [][VolumeDevice](./k8s-io-api-core-v1.md#VolumeDevice) | No | volumeDevices is the list of block devices to be used by the Sidecar.<br />+patchMergeKey=devicePath<br />+patchStrategy=merge<br />+optional<br />+listType=atomic |
| `livenessProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | Periodic probe of Sidecar liveness.<br />Container will be restarted if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `readinessProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | Periodic probe of Sidecar service readiness.<br />Container will be removed from service endpoints if the probe fails.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `startupProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | StartupProbe indicates that the Pod the Sidecar is running in has successfully initialized.<br />If specified, no other probes are executed until this completes successfully.<br />If this probe fails, the Pod will be restarted, just as if the livenessProbe failed.<br />This can be used to provide different probe parameters at the beginning of a Pod's lifecycle,<br />when it might take a long time to load data or warm a cache, than during steady-state operation.<br />This cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `lifecycle` | *[Lifecycle](./k8s-io-api-core-v1.md#Lifecycle) | No | Actions that the management system should take in response to Sidecar lifecycle events.<br />Cannot be updated.<br />+optional |
| `terminationMessagePath` | string | No | Optional: Path at which the file to which the Sidecar's termination message<br />will be written is mounted into the Sidecar's filesystem.<br />Message written is intended to be brief final status, such as an assertion failure message.<br />Will be truncated by the node if greater than 4096 bytes. The total message length across<br />all containers will be limited to 12kb.<br />Defaults to /dev/termination-log.<br />Cannot be updated.<br />+optional |
| `terminationMessagePolicy` | [TerminationMessagePolicy](./k8s-io-api-core-v1.md#TerminationMessagePolicy) | No | Indicate how the termination message should be populated. File will use the contents of<br />terminationMessagePath to populate the Sidecar status message on both success and failure.<br />FallbackToLogsOnError will use the last chunk of Sidecar log output if the termination<br />message file is empty and the Sidecar exited with an error.<br />The log output is limited to 2048 bytes or 80 lines, whichever is smaller.<br />Defaults to File.<br />Cannot be updated.<br />+optional |
| `imagePullPolicy` | [PullPolicy](./k8s-io-api-core-v1.md#PullPolicy) | No | Image pull policy.<br />One of Always, Never, IfNotPresent.<br />Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images<br />+optional |
| `securityContext` | *[SecurityContext](./k8s-io-api-core-v1.md#SecurityContext) | No | SecurityContext defines the security options the Sidecar should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/<br />+optional |
| `stdin` | bool | No | Whether this Sidecar should allocate a buffer for stdin in the container runtime. If this<br />is not set, reads from stdin in the Sidecar will always result in EOF.<br />Default is false.<br />+optional |
| `stdinOnce` | bool | No | Whether the container runtime should close the stdin channel after it has been opened by<br />a single attach. When stdin is true the stdin stream will remain open across multiple attach<br />sessions. If stdinOnce is set to true, stdin is opened on Sidecar start, is empty until the<br />first client attaches to stdin, and then remains open and accepts data until the client disconnects,<br />at which time stdin is closed and remains closed until the Sidecar is restarted. If this<br />flag is false, a container processes that reads from stdin will never receive an EOF.<br />Default is false<br />+optional |
| `tty` | bool | No | Whether this Sidecar should allocate a TTY for itself, also requires 'stdin' to be true.<br />Default is false.<br />+optional |
| `script` | string | No | Script is the contents of an executable file to execute.<br /><br />If Script is not empty, the Step cannot have an Command or Args.<br />+optional |
| `workspaces` | [][WorkspaceUsage](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#WorkspaceUsage) | No | This is an alpha field. You must set the "enable-api-fields" feature flag to "alpha"<br />for this field to be supported.<br /><br />Workspaces is a list of workspaces from the Task that this Sidecar wants<br />exclusive access to. Adding a workspace to this list means that any<br />other Step or Sidecar that does not also request this Workspace will<br />not have access to it.<br />+optional<br />+listType=atomic |
| `restartPolicy` | *[ContainerRestartPolicy](./k8s-io-api-core-v1.md#ContainerRestartPolicy) | No | RestartPolicy refers to kubernetes RestartPolicy. It can only be set for an<br />initContainer and must have it's policy set to "Always". It is currently<br />left optional to help support Kubernetes versions prior to 1.29 when this feature<br />was introduced.<br />+optional |

## Step

Step runs a subcomponent of a Task

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name of the Step specified as a DNS_LABEL.<br />Each Step in a Task must have a unique name. |
| `displayName` | string | No | DisplayName is a user-facing name of the step that may be<br />used to populate a UI.<br />+optional |
| `image` | string | No | Docker image name.<br />More info: https://kubernetes.io/docs/concepts/containers/images<br />+optional |
| `command` | []string | No | Entrypoint array. Not executed within a shell.<br />The image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional<br />+listType=atomic |
| `args` | []string | No | Arguments to the entrypoint.<br />The image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional<br />+listType=atomic |
| `workingDir` | string | No | Step's working directory.<br />If not specified, the container runtime's default will be used, which<br />might be configured in the container image.<br />Cannot be updated.<br />+optional |
| `envFrom` | [][EnvFromSource](./k8s-io-api-core-v1.md#EnvFromSource) | No | List of sources to populate environment variables in the Step.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the Step is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated.<br />+optional<br />+listType=atomic |
| `env` | [][EnvVar](./k8s-io-api-core-v1.md#EnvVar) | No | List of environment variables to set in the Step.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=name<br />+patchStrategy=merge<br />+listType=atomic |
| `computeResources` | [ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | No | ComputeResources required by this Step.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br />+optional |
| `volumeMounts` | [][VolumeMount](./k8s-io-api-core-v1.md#VolumeMount) | No | Volumes to mount into the Step's filesystem.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=mountPath<br />+patchStrategy=merge<br />+listType=atomic |
| `volumeDevices` | [][VolumeDevice](./k8s-io-api-core-v1.md#VolumeDevice) | No | volumeDevices is the list of block devices to be used by the Step.<br />+patchMergeKey=devicePath<br />+patchStrategy=merge<br />+optional<br />+listType=atomic |
| `imagePullPolicy` | [PullPolicy](./k8s-io-api-core-v1.md#PullPolicy) | No | Image pull policy.<br />One of Always, Never, IfNotPresent.<br />Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images<br />+optional |
| `securityContext` | *[SecurityContext](./k8s-io-api-core-v1.md#SecurityContext) | No | SecurityContext defines the security options the Step should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/<br />+optional |
| `script` | string | No | Script is the contents of an executable file to execute.<br /><br />If Script is not empty, the Step cannot have an Command and the Args will be passed to the Script.<br />+optional |
| `timeout` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Timeout is the time after which the step times out. Defaults to never.<br />Refer to Go's ParseDuration documentation for expected format: https://golang.org/pkg/time/#ParseDuration<br />+optional |
| `workspaces` | [][WorkspaceUsage](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#WorkspaceUsage) | No | This is an alpha field. You must set the "enable-api-fields" feature flag to "alpha"<br />for this field to be supported.<br /><br />Workspaces is a list of workspaces from the Task that this Step wants<br />exclusive access to. Adding a workspace to this list means that any<br />other Step or Sidecar that does not also request this Workspace will<br />not have access to it.<br />+optional<br />+listType=atomic |
| `onError` | [OnErrorType](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#OnErrorType) | No | OnError defines the exiting behavior of a container on error<br />can be set to [ continue | stopAndFail ] |
| `stdoutConfig` | *[StepOutputConfig](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#StepOutputConfig) | No | Stores configuration for the stdout stream of the step.<br />+optional |
| `stderrConfig` | *[StepOutputConfig](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#StepOutputConfig) | No | Stores configuration for the stderr stream of the step.<br />+optional |
| `ref` | *[Ref](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Ref) | No | Contains the reference to an existing StepAction.<br />+optional |
| `params` | [Params](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Params) | No | Params declares parameters passed to this step action.<br />+optional |
| `results` | [][StepResult](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#StepResult) | No | Results declares StepResults produced by the Step.<br /><br />It can be used in an inlined Step when used to store Results to $(step.results.resultName.path).<br />It cannot be used when referencing StepActions using [v1.Step.Ref].<br />The Results declared by the StepActions will be stored here instead.<br />+optional<br />+listType=atomic |
| `when` | [StepWhenExpressions](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#StepWhenExpressions) | No | When is a list of when expressions that need to be true for the task to run<br />+optional |

## StepOutputConfig

StepOutputConfig stores configuration for a step output stream.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `path` | string | No | Path to duplicate stdout stream to on container's local filesystem.<br />+optional |

## StepResult

StepResult used to describe the Results of a Step.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name the given name |
| `type` | [ResultsType](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ResultsType) | No | The possible types are 'string', 'array', and 'object', with 'string' as the default.<br />+optional |
| `properties` | map[string][PropertySpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PropertySpec) | No | Properties is the JSON Schema properties to support key-value pairs results.<br />+optional |
| `description` | string | No | Description is a human-readable description of the result<br />+optional |

## StepTemplate

StepTemplate is a template for a Step

| Stanza | Type | Required | Description |
|---|---|---|---|
| `image` | string | No | Image reference name.<br />More info: https://kubernetes.io/docs/concepts/containers/images<br />+optional |
| `command` | []string | No | Entrypoint array. Not executed within a shell.<br />The image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the Step's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional<br />+listType=atomic |
| `args` | []string | No | Arguments to the entrypoint.<br />The image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the Step's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional<br />+listType=atomic |
| `workingDir` | string | No | Step's working directory.<br />If not specified, the container runtime's default will be used, which<br />might be configured in the container image.<br />Cannot be updated.<br />+optional |
| `envFrom` | [][EnvFromSource](./k8s-io-api-core-v1.md#EnvFromSource) | No | List of sources to populate environment variables in the Step.<br />The keys defined within a source must be a C_IDENTIFIER. All invalid keys<br />will be reported as an event when the Step is starting. When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated.<br />+optional<br />+listType=atomic |
| `env` | [][EnvVar](./k8s-io-api-core-v1.md#EnvVar) | No | List of environment variables to set in the Step.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=name<br />+patchStrategy=merge<br />+listType=atomic |
| `computeResources` | [ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | No | ComputeResources required by this Step.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br />+optional |
| `volumeMounts` | [][VolumeMount](./k8s-io-api-core-v1.md#VolumeMount) | No | Volumes to mount into the Step's filesystem.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=mountPath<br />+patchStrategy=merge<br />+listType=atomic |
| `volumeDevices` | [][VolumeDevice](./k8s-io-api-core-v1.md#VolumeDevice) | No | volumeDevices is the list of block devices to be used by the Step.<br />+patchMergeKey=devicePath<br />+patchStrategy=merge<br />+optional<br />+listType=atomic |
| `imagePullPolicy` | [PullPolicy](./k8s-io-api-core-v1.md#PullPolicy) | No | Image pull policy.<br />One of Always, Never, IfNotPresent.<br />Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images<br />+optional |
| `securityContext` | *[SecurityContext](./k8s-io-api-core-v1.md#SecurityContext) | No | SecurityContext defines the security options the Step should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/security-context/<br />+optional |

## StepWhenExpressions





## TaskBreakpoints

TaskBreakpoints defines the breakpoint config for a particular Task

| Stanza | Type | Required | Description |
|---|---|---|---|
| `onFailure` | string | No | if enabled, pause TaskRun on failure of a step<br />failed step will not exit<br />+optional |
| `beforeSteps` | []string | No | +optional<br />+listType=atomic |

## TaskKind

TaskKind defines the type of Task used by the pipeline.



## TaskRef

TaskRef can be used to refer to a specific instance of a task.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names |
| `kind` | [TaskKind](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#TaskKind) | No | TaskKind indicates the Kind of the Task:<br />1. Namespaced Task when Kind is set to "Task". If Kind is "", it defaults to "Task".<br />2. Custom Task when Kind is non-empty and APIVersion is non-empty |
| `apiVersion` | string | No | API version of the referent<br />Note: A Task with non-empty APIVersion and Kind is considered a Custom Task<br />+optional |
| `resolver` | [ResolverName](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ResolverName) | No | Resolver is the name of the resolver that should perform<br />resolution of the referenced Tekton resource, such as "git".<br />+optional |
| `params` | [Params](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#Params) | No | Params contains the parameters used to identify the<br />referenced Tekton resource. Example entries might include<br />"repo" or "path" but the set of params ultimately depends on<br />the chosen resolver.<br />+optional |

## TaskResult

TaskResult used to describe the results of a task

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name the given name |
| `type` | [ResultsType](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ResultsType) | No | Type is the user-specified type of the result. The possible type<br />is currently "string" and will support "array" in following work.<br />+optional |
| `properties` | map[string][PropertySpec](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#PropertySpec) | No | Properties is the JSON Schema properties to support key-value pairs results.<br />+optional |
| `description` | string | No | Description is a human-readable description of the result<br />+optional |
| `value` | *[ResultValue](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#ResultValue) | No | Value the expression used to retrieve the value of the result from an underlying Step.<br />+optional<br />+kubebuilder:pruning:PreserveUnknownFields<br />+kubebuilder:validation:Schemaless |

## TaskRunDebug

TaskRunDebug defines the breakpoint config for a particular TaskRun

| Stanza | Type | Required | Description |
|---|---|---|---|
| `breakpoints` | *[TaskBreakpoints](./github-com-tektoncd-pipeline-pkg-apis-pipeline-v1.md#TaskBreakpoints) | No | +optional |

## TaskRunSidecarSpec

TaskRunSidecarSpec is used to override the values of a Sidecar in the corresponding Task.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | The name of the Sidecar to override. |
| `computeResources` | [ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | Yes | The resource requirements to apply to the Sidecar. |

## TaskRunStepSpec

TaskRunStepSpec is used to override the values of a Step in the corresponding Task.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | The name of the Step to override. |
| `computeResources` | [ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | Yes | The resource requirements to apply to the Step. |

## TimeoutFields

TimeoutFields allows granular specification of pipeline, task, and finally timeouts

| Stanza | Type | Required | Description |
|---|---|---|---|
| `pipeline` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Pipeline sets the maximum allowed duration for execution of the entire pipeline. The sum of individual timeouts for tasks and finally must not exceed this value. |
| `tasks` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Tasks sets the maximum allowed duration of this pipeline's tasks |
| `finally` | *[Duration](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Duration) | No | Finally sets the maximum allowed duration of this pipeline's finally |

## Volumes

+listType=atomic



## WhenExpressions

WhenExpressions are used to specify whether a Task should be executed or skipped<br />All of them need to evaluate to True for a guarded Task to be executed.



## WorkspaceBinding

WorkspaceBinding maps a Task's declared workspace to a Volume.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name of the workspace populated by the volume. |
| `subPath` | string | No | SubPath is optionally a directory on the volume which should be used<br />for this binding (i.e. the volume will be mounted at this sub directory).<br />+optional |
| `volumeClaimTemplate` | *[PersistentVolumeClaim](./k8s-io-api-core-v1.md#PersistentVolumeClaim) | No | VolumeClaimTemplate is a template for a claim that will be created in the same namespace.<br />The PipelineRun controller is responsible for creating a unique claim for each instance of PipelineRun.<br />See PersistentVolumeClaim (API version: v1)<br />+optional<br />+kubebuilder:pruning:PreserveUnknownFields<br />+kubebuilder:validation:Schemaless |
| `persistentVolumeClaim` | *[PersistentVolumeClaimVolumeSource](./k8s-io-api-core-v1.md#PersistentVolumeClaimVolumeSource) | No | PersistentVolumeClaimVolumeSource represents a reference to a<br />PersistentVolumeClaim in the same namespace. Either this OR EmptyDir can be used.<br />+optional |
| `emptyDir` | *[EmptyDirVolumeSource](./k8s-io-api-core-v1.md#EmptyDirVolumeSource) | No | EmptyDir represents a temporary directory that shares a Task's lifetime.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir<br />Either this OR PersistentVolumeClaim can be used.<br />+optional |
| `configMap` | *[ConfigMapVolumeSource](./k8s-io-api-core-v1.md#ConfigMapVolumeSource) | No | ConfigMap represents a configMap that should populate this workspace.<br />+optional |
| `secret` | *[SecretVolumeSource](./k8s-io-api-core-v1.md#SecretVolumeSource) | No | Secret represents a secret that should populate this workspace.<br />+optional |
| `projected` | *[ProjectedVolumeSource](./k8s-io-api-core-v1.md#ProjectedVolumeSource) | No | Projected represents a projected volume that should populate this workspace.<br />+optional |
| `csi` | *[CSIVolumeSource](./k8s-io-api-core-v1.md#CSIVolumeSource) | No | CSI (Container Storage Interface) represents ephemeral storage that is handled by certain external CSI drivers.<br />+optional |

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
| `workspace` | string | No | Workspace is the name of the workspace declared by the pipeline<br />+optional |
| `subPath` | string | No | SubPath is optionally a directory on the volume which should be used<br />for this binding (i.e. the volume will be mounted at this sub directory).<br />+optional |

## WorkspaceUsage

WorkspaceUsage is used by a Step or Sidecar to declare that it wants isolated access<br />to a Workspace defined in a Task.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name is the name of the workspace this Step or Sidecar wants access to. |
| `mountPath` | string | Yes | MountPath is the path that the workspace should be mounted to inside the Step or Sidecar,<br />overriding any MountPath specified in the Task's WorkspaceDeclaration. |


