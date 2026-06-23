# Package k8s.io/api/core/v1

- [AppArmorProfile](#AppArmorProfile)
- [AppArmorProfileType](#AppArmorProfileType)
- [CSIVolumeSource](#CSIVolumeSource)
- [Capabilities](#Capabilities)
- [Capability](#Capability)
- [ClaimResourceStatus](#ClaimResourceStatus)
- [ClusterTrustBundleProjection](#ClusterTrustBundleProjection)
- [ConditionStatus](#ConditionStatus)
- [ConfigMapEnvSource](#ConfigMapEnvSource)
- [ConfigMapKeySelector](#ConfigMapKeySelector)
- [ConfigMapProjection](#ConfigMapProjection)
- [ConfigMapVolumeSource](#ConfigMapVolumeSource)
- [ContainerPort](#ContainerPort)
- [ContainerRestartPolicy](#ContainerRestartPolicy)
- [DownwardAPIProjection](#DownwardAPIProjection)
- [DownwardAPIVolumeFile](#DownwardAPIVolumeFile)
- [EmptyDirVolumeSource](#EmptyDirVolumeSource)
- [EnvFromSource](#EnvFromSource)
- [EnvVar](#EnvVar)
- [EnvVarSource](#EnvVarSource)
- [ExecAction](#ExecAction)
- [FileKeySelector](#FileKeySelector)
- [GRPCAction](#GRPCAction)
- [HTTPGetAction](#HTTPGetAction)
- [HTTPHeader](#HTTPHeader)
- [KeyToPath](#KeyToPath)
- [Lifecycle](#Lifecycle)
- [LifecycleHandler](#LifecycleHandler)
- [LocalObjectReference](#LocalObjectReference)
- [ModifyVolumeStatus](#ModifyVolumeStatus)
- [MountPropagationMode](#MountPropagationMode)
- [ObjectFieldSelector](#ObjectFieldSelector)
- [PersistentVolumeAccessMode](#PersistentVolumeAccessMode)
- [PersistentVolumeClaim](#PersistentVolumeClaim)
- [PersistentVolumeClaimCondition](#PersistentVolumeClaimCondition)
- [PersistentVolumeClaimConditionType](#PersistentVolumeClaimConditionType)
- [PersistentVolumeClaimModifyVolumeStatus](#PersistentVolumeClaimModifyVolumeStatus)
- [PersistentVolumeClaimPhase](#PersistentVolumeClaimPhase)
- [PersistentVolumeClaimSpec](#PersistentVolumeClaimSpec)
- [PersistentVolumeClaimStatus](#PersistentVolumeClaimStatus)
- [PersistentVolumeClaimVolumeSource](#PersistentVolumeClaimVolumeSource)
- [PersistentVolumeMode](#PersistentVolumeMode)
- [PodCertificateProjection](#PodCertificateProjection)
- [Probe](#Probe)
- [ProcMountType](#ProcMountType)
- [ProjectedVolumeSource](#ProjectedVolumeSource)
- [Protocol](#Protocol)
- [PullPolicy](#PullPolicy)
- [RecursiveReadOnlyMode](#RecursiveReadOnlyMode)
- [ResourceClaim](#ResourceClaim)
- [ResourceFieldSelector](#ResourceFieldSelector)
- [ResourceList](#ResourceList)
- [ResourceName](#ResourceName)
- [ResourceRequirements](#ResourceRequirements)
- [SELinuxOptions](#SELinuxOptions)
- [SeccompProfile](#SeccompProfile)
- [SeccompProfileType](#SeccompProfileType)
- [SecretEnvSource](#SecretEnvSource)
- [SecretKeySelector](#SecretKeySelector)
- [SecretProjection](#SecretProjection)
- [SecretVolumeSource](#SecretVolumeSource)
- [SecurityContext](#SecurityContext)
- [ServiceAccountTokenProjection](#ServiceAccountTokenProjection)
- [Signal](#Signal)
- [SleepAction](#SleepAction)
- [StorageMedium](#StorageMedium)
- [TCPSocketAction](#TCPSocketAction)
- [TerminationMessagePolicy](#TerminationMessagePolicy)
- [TypedLocalObjectReference](#TypedLocalObjectReference)
- [TypedObjectReference](#TypedObjectReference)
- [URIScheme](#URIScheme)
- [VolumeDevice](#VolumeDevice)
- [VolumeMount](#VolumeMount)
- [VolumeProjection](#VolumeProjection)
- [VolumeResourceRequirements](#VolumeResourceRequirements)
- [WindowsSecurityContextOptions](#WindowsSecurityContextOptions)


## AppArmorProfile

AppArmorProfile defines a pod or container's AppArmor settings.<br />+union

| Stanza | Type | Required | Description |
|---|---|---|---|
| `type` | [AppArmorProfileType](./k8s-io-api-core-v1.md#AppArmorProfileType) | Yes | type indicates which kind of AppArmor profile will be applied.<br />Valid options are:<br />  Localhost - a profile pre-loaded on the node.<br />  RuntimeDefault - the container runtime's default profile.<br />  Unconfined - no AppArmor enforcement.<br />+unionDiscriminator |
| `localhostProfile` | *string | No | localhostProfile indicates a profile loaded on the node that should be used.<br />The profile must be preconfigured on the node to work.<br />Must match the loaded name of the profile.<br />Must be set if and only if type is "Localhost".<br />+optional |

## AppArmorProfileType

+enum



## CSIVolumeSource

Represents a source location of a volume to mount, managed by an external CSI driver

| Stanza | Type | Required | Description |
|---|---|---|---|
| `driver` | string | Yes | driver is the name of the CSI driver that handles this volume.<br />Consult with your admin for the correct name as registered in the cluster. |
| `readOnly` | *bool | No | readOnly specifies a read-only configuration for the volume.<br />Defaults to false (read/write).<br />+optional |
| `fsType` | *string | No | fsType to mount. Ex. "ext4", "xfs", "ntfs".<br />If not provided, the empty value is passed to the associated CSI driver<br />which will determine the default filesystem to apply.<br />+optional |
| `volumeAttributes` | map[string]string | No | volumeAttributes stores driver-specific properties that are passed to the CSI<br />driver. Consult your driver's documentation for supported values.<br />+optional |
| `nodePublishSecretRef` | *[LocalObjectReference](./k8s-io-api-core-v1.md#LocalObjectReference) | No | nodePublishSecretRef is a reference to the secret object containing<br />sensitive information to pass to the CSI driver to complete the CSI<br />NodePublishVolume and NodeUnpublishVolume calls.<br />This field is optional, and  may be empty if no secret is required. If the<br />secret object contains more than one secret, all secret references are passed.<br />+optional |

## Capabilities

Adds and removes POSIX capabilities from running containers.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `add` | [][Capability](./k8s-io-api-core-v1.md#Capability) | No | Added capabilities<br />+optional<br />+listType=atomic |
| `drop` | [][Capability](./k8s-io-api-core-v1.md#Capability) | No | Removed capabilities<br />+optional<br />+listType=atomic |

## Capability

Capability represent POSIX capabilities type



## ClaimResourceStatus

+enum<br />When a controller receives persistentvolume claim update with ClaimResourceStatus for a resource<br />that it does not recognizes, then it should ignore that update and let other controllers<br />handle it.



## ClusterTrustBundleProjection

ClusterTrustBundleProjection describes how to select a set of<br />ClusterTrustBundle objects and project their contents into the pod<br />filesystem.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | *string | No | Select a single ClusterTrustBundle by object name.  Mutually-exclusive<br />with signerName and labelSelector.<br />+optional |
| `signerName` | *string | No | Select all ClusterTrustBundles that match this signer name.<br />Mutually-exclusive with name.  The contents of all selected<br />ClusterTrustBundles will be unified and deduplicated.<br />+optional |
| `labelSelector` | *[LabelSelector](./k8s-io-apimachinery-pkg-apis-meta-v1.md#LabelSelector) | No | Select all ClusterTrustBundles that match this label selector.  Only has<br />effect if signerName is set.  Mutually-exclusive with name.  If unset,<br />interpreted as "match nothing".  If set but empty, interpreted as "match<br />everything".<br />+optional |
| `optional` | *bool | No | If true, don't block pod startup if the referenced ClusterTrustBundle(s)<br />aren't available.  If using name, then the named ClusterTrustBundle is<br />allowed not to exist.  If using signerName, then the combination of<br />signerName and labelSelector is allowed to match zero<br />ClusterTrustBundles.<br />+optional |
| `path` | string | Yes | Relative path from the volume root to write the bundle. |

## ConditionStatus





## ConfigMapEnvSource

ConfigMapEnvSource selects a ConfigMap to populate the environment<br />variables with.<br /><br />The contents of the target ConfigMap's Data field will represent the<br />key-value pairs as environment variables.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent.<br />This field is effectively required, but due to backwards compatibility is<br />allowed to be empty. Instances of this type with an empty value here are<br />almost certainly wrong.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br />+optional<br />+default=""<br />+kubebuilder:default=""<br />TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |
| `optional` | *bool | No | Specify whether the ConfigMap must be defined<br />+optional |

## ConfigMapKeySelector

Selects a key from a ConfigMap.<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent.<br />This field is effectively required, but due to backwards compatibility is<br />allowed to be empty. Instances of this type with an empty value here are<br />almost certainly wrong.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br />+optional<br />+default=""<br />+kubebuilder:default=""<br />TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |
| `key` | string | Yes | The key to select. |
| `optional` | *bool | No | Specify whether the ConfigMap or its key must be defined<br />+optional |

## ConfigMapProjection

Adapts a ConfigMap into a projected volume.<br /><br />The contents of the target ConfigMap's Data field will be presented in a<br />projected volume as files using the keys in the Data field as the file names,<br />unless the items element is populated with specific mappings of keys to paths.<br />Note that this is identical to a configmap volume source without the default<br />mode.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent.<br />This field is effectively required, but due to backwards compatibility is<br />allowed to be empty. Instances of this type with an empty value here are<br />almost certainly wrong.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br />+optional<br />+default=""<br />+kubebuilder:default=""<br />TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |
| `items` | [][KeyToPath](./k8s-io-api-core-v1.md#KeyToPath) | No | items if unspecified, each key-value pair in the Data field of the referenced<br />ConfigMap will be projected into the volume as a file whose name is the<br />key and content is the value. If specified, the listed keys will be<br />projected into the specified paths, and unlisted keys will not be<br />present. If a key is specified which is not present in the ConfigMap,<br />the volume setup will error unless it is marked optional. Paths must be<br />relative and may not contain the '..' path or start with '..'.<br />+optional<br />+listType=atomic |
| `optional` | *bool | No | optional specify whether the ConfigMap or its keys must be defined<br />+optional |

## ConfigMapVolumeSource

Adapts a ConfigMap into a volume.<br /><br />The contents of the target ConfigMap's Data field will be presented in a<br />volume as files using the keys in the Data field as the file names, unless<br />the items element is populated with specific mappings of keys to paths.<br />ConfigMap volumes support ownership management and SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent.<br />This field is effectively required, but due to backwards compatibility is<br />allowed to be empty. Instances of this type with an empty value here are<br />almost certainly wrong.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br />+optional<br />+default=""<br />+kubebuilder:default=""<br />TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |
| `items` | [][KeyToPath](./k8s-io-api-core-v1.md#KeyToPath) | No | items if unspecified, each key-value pair in the Data field of the referenced<br />ConfigMap will be projected into the volume as a file whose name is the<br />key and content is the value. If specified, the listed keys will be<br />projected into the specified paths, and unlisted keys will not be<br />present. If a key is specified which is not present in the ConfigMap,<br />the volume setup will error unless it is marked optional. Paths must be<br />relative and may not contain the '..' path or start with '..'.<br />+optional<br />+listType=atomic |
| `defaultMode` | *int32 | No | defaultMode is optional: mode bits used to set permissions on created files by default.<br />Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br />YAML accepts both octal and decimal values, JSON requires decimal values for mode bits.<br />Defaults to 0644.<br />Directories within the path are not affected by this setting.<br />This might be in conflict with other options that affect the file<br />mode, like fsGroup, and the result can be other mode bits set.<br />+optional |
| `optional` | *bool | No | optional specify whether the ConfigMap or its keys must be defined<br />+optional |

## ContainerPort

ContainerPort represents a network port in a single container.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each<br />named port in a pod must have a unique name. Name for the port that can be<br />referred to by services.<br />+optional |
| `hostPort` | int32 | No | Number of port to expose on the host.<br />If specified, this must be a valid port number, 0 < x < 65536.<br />If HostNetwork is specified, this must match ContainerPort.<br />Most containers do not need this.<br />+optional |
| `containerPort` | int32 | Yes | Number of port to expose on the pod's IP address.<br />This must be a valid port number, 0 < x < 65536. |
| `protocol` | [Protocol](./k8s-io-api-core-v1.md#Protocol) | No | Protocol for port. Must be UDP, TCP, or SCTP.<br />Defaults to "TCP".<br />+optional<br />+default="TCP" |
| `hostIP` | string | No | What host IP to bind the external port to.<br />+optional |

## ContainerRestartPolicy

ContainerRestartPolicy is the restart policy for a single container.<br />The only allowed values are "Always", "Never", and "OnFailure".



## DownwardAPIProjection

Represents downward API info for projecting into a projected volume.<br />Note that this is identical to a downwardAPI volume source without the default<br />mode.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `items` | [][DownwardAPIVolumeFile](./k8s-io-api-core-v1.md#DownwardAPIVolumeFile) | No | Items is a list of DownwardAPIVolume file<br />+optional<br />+listType=atomic |

## DownwardAPIVolumeFile

DownwardAPIVolumeFile represents information to create the file containing the pod field

| Stanza | Type | Required | Description |
|---|---|---|---|
| `path` | string | Yes | Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..' |
| `fieldRef` | *[ObjectFieldSelector](./k8s-io-api-core-v1.md#ObjectFieldSelector) | No | Required: Selects a field of the pod: only annotations, labels, name, namespace and uid are supported.<br />+optional |
| `resourceFieldRef` | *[ResourceFieldSelector](./k8s-io-api-core-v1.md#ResourceFieldSelector) | No | Selects a resource of the container: only resources limits and requests<br />(limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.<br />+optional |
| `mode` | *int32 | No | Optional: mode bits used to set permissions on this file, must be an octal value<br />between 0000 and 0777 or a decimal value between 0 and 511.<br />YAML accepts both octal and decimal values, JSON requires decimal values for mode bits.<br />If not specified, the volume defaultMode will be used.<br />This might be in conflict with other options that affect the file<br />mode, like fsGroup, and the result can be other mode bits set.<br />+optional |

## EmptyDirVolumeSource

Represents an empty directory for a pod.<br />Empty directory volumes support ownership management and SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `medium` | [StorageMedium](./k8s-io-api-core-v1.md#StorageMedium) | No | medium represents what type of storage medium should back this directory.<br />The default is "" which means to use the node's default medium.<br />Must be an empty string (default) or Memory.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir<br />+optional |
| `sizeLimit` | *[Quantity](./k8s-io-apimachinery-pkg-api-resource.md#Quantity) | No | sizeLimit is the total amount of local storage required for this EmptyDir volume.<br />The size limit is also applicable for memory medium.<br />The maximum usage on memory medium EmptyDir would be the minimum value between<br />the SizeLimit specified here and the sum of memory limits of all containers in a pod.<br />The default is nil which means that the limit is undefined.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir<br />+optional |

## EnvFromSource

EnvFromSource represents the source of a set of ConfigMaps or Secrets

| Stanza | Type | Required | Description |
|---|---|---|---|
| `prefix` | string | No | Optional text to prepend to the name of each environment variable.<br />May consist of any printable ASCII characters except '='.<br />+optional |
| `configMapRef` | *[ConfigMapEnvSource](./k8s-io-api-core-v1.md#ConfigMapEnvSource) | No | The ConfigMap to select from<br />+optional |
| `secretRef` | *[SecretEnvSource](./k8s-io-api-core-v1.md#SecretEnvSource) | No | The Secret to select from<br />+optional |

## EnvVar

EnvVar represents an environment variable present in a Container.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name of the environment variable.<br />May consist of any printable ASCII characters except '='. |
| `value` | string | No | Variable references $(VAR_NAME) are expanded<br />using the previously defined environment variables in the container and<br />any service environment variables. If a variable cannot be resolved,<br />the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e.<br />"$$(VAR_NAME)" will produce the string literal "$(VAR_NAME)".<br />Escaped references will never be expanded, regardless of whether the variable<br />exists or not.<br />Defaults to "".<br />+optional |
| `valueFrom` | *[EnvVarSource](./k8s-io-api-core-v1.md#EnvVarSource) | No | Source for the environment variable's value. Cannot be used if value is not empty.<br />+optional |

## EnvVarSource

EnvVarSource represents a source for the value of an EnvVar.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `fieldRef` | *[ObjectFieldSelector](./k8s-io-api-core-v1.md#ObjectFieldSelector) | No | Selects a field of the pod: supports metadata.name, metadata.namespace, `metadata.labels['<KEY>']`, `metadata.annotations['<KEY>']`,<br />spec.nodeName, spec.serviceAccountName, status.hostIP, status.podIP, status.podIPs.<br />+optional |
| `resourceFieldRef` | *[ResourceFieldSelector](./k8s-io-api-core-v1.md#ResourceFieldSelector) | No | Selects a resource of the container: only resources limits and requests<br />(limits.cpu, limits.memory, limits.ephemeral-storage, requests.cpu, requests.memory and requests.ephemeral-storage) are currently supported.<br />+optional |
| `configMapKeyRef` | *[ConfigMapKeySelector](./k8s-io-api-core-v1.md#ConfigMapKeySelector) | No | Selects a key of a ConfigMap.<br />+optional |
| `secretKeyRef` | *[SecretKeySelector](./k8s-io-api-core-v1.md#SecretKeySelector) | No | Selects a key of a secret in the pod's namespace<br />+optional |
| `fileKeyRef` | *[FileKeySelector](./k8s-io-api-core-v1.md#FileKeySelector) | No | FileKeyRef selects a key of the env file.<br />Requires the EnvFiles feature gate to be enabled.<br /><br />+featureGate=EnvFiles<br />+optional |

## ExecAction

ExecAction describes a "run in container" action.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `command` | []string | No | Command is the command line to execute inside the container, the working directory for the<br />command  is root ('/') in the container's filesystem. The command is simply exec'd, it is<br />not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use<br />a shell, you need to explicitly call out to that shell.<br />Exit status of 0 is treated as live/healthy and non-zero is unhealthy.<br />+optional<br />+listType=atomic |

## FileKeySelector

FileKeySelector selects a key of the env file.<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `volumeName` | string | Yes | The name of the volume mount containing the env file.<br />+required |
| `path` | string | Yes | The path within the volume from which to select the file.<br />Must be relative and may not contain the '..' path or start with '..'.<br />+required |
| `key` | string | Yes | The key within the env file. An invalid key will prevent the pod from starting.<br />The keys defined within a source may consist of any printable ASCII characters except '='.<br />During Alpha stage of the EnvFiles feature gate, the key size is limited to 128 characters.<br />+required |
| `optional` | *bool | No | Specify whether the file or its key must be defined. If the file or key<br />does not exist, then the env var is not published.<br />If optional is set to true and the specified key does not exist,<br />the environment variable will not be set in the Pod's containers.<br /><br />If optional is set to false and the specified key does not exist,<br />an error will be returned during Pod creation.<br />+optional<br />+default=false |

## GRPCAction

GRPCAction specifies an action involving a GRPC service.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `port` | int32 | Yes | Port number of the gRPC service. Number must be in the range 1 to 65535. |
| `service` | *string | No | Service is the name of the service to place in the gRPC HealthCheckRequest<br />(see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).<br /><br />If this is not specified, the default behavior is defined by gRPC.<br />+optional<br />+default="" |

## HTTPGetAction

HTTPGetAction describes an action based on HTTP Get requests.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `path` | string | No | Path to access on the HTTP server.<br />+optional |
| `port` | [IntOrString](./k8s-io-apimachinery-pkg-util-intstr.md#IntOrString) | Yes | Name or number of the port to access on the container.<br />Number must be in the range 1 to 65535.<br />Name must be an IANA_SVC_NAME. |
| `host` | string | No | Host name to connect to, defaults to the pod IP. You probably want to set<br />"Host" in httpHeaders instead.<br />+optional |
| `scheme` | [URIScheme](./k8s-io-api-core-v1.md#URIScheme) | No | Scheme to use for connecting to the host.<br />Defaults to HTTP.<br />+optional |
| `httpHeaders` | [][HTTPHeader](./k8s-io-api-core-v1.md#HTTPHeader) | No | Custom headers to set in the request. HTTP allows repeated headers.<br />+optional<br />+listType=atomic |

## HTTPHeader

HTTPHeader describes a custom header to be used in HTTP probes

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | The header field name.<br />This will be canonicalized upon output, so case-variant names will be understood as the same header. |
| `value` | string | Yes | The header field value |

## KeyToPath

Maps a string key to a path within a volume.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `key` | string | Yes | key is the key to project. |
| `path` | string | Yes | path is the relative path of the file to map the key to.<br />May not be an absolute path.<br />May not contain the path element '..'.<br />May not start with the string '..'. |
| `mode` | *int32 | No | mode is Optional: mode bits used to set permissions on this file.<br />Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br />YAML accepts both octal and decimal values, JSON requires decimal values for mode bits.<br />If not specified, the volume defaultMode will be used.<br />This might be in conflict with other options that affect the file<br />mode, like fsGroup, and the result can be other mode bits set.<br />+optional |

## Lifecycle

Lifecycle describes actions that the management system should take in response to container lifecycle<br />events. For the PostStart and PreStop lifecycle handlers, management of the container blocks<br />until the action is complete, unless the container process fails, in which case the handler is aborted.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `postStart` | *[LifecycleHandler](./k8s-io-api-core-v1.md#LifecycleHandler) | No | PostStart is called immediately after a container is created. If the handler fails,<br />the container is terminated and restarted according to its restart policy.<br />Other management of the container blocks until the hook completes.<br />More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks<br />+optional |
| `preStop` | *[LifecycleHandler](./k8s-io-api-core-v1.md#LifecycleHandler) | No | PreStop is called immediately before a container is terminated due to an<br />API request or management event such as liveness/startup probe failure,<br />preemption, resource contention, etc. The handler is not called if the<br />container crashes or exits. The Pod's termination grace period countdown begins before the<br />PreStop hook is executed. Regardless of the outcome of the handler, the<br />container will eventually terminate within the Pod's termination grace<br />period (unless delayed by finalizers). Other management of the container blocks until the hook completes<br />or until the termination grace period is reached.<br />More info: https://kubernetes.io/docs/concepts/containers/container-lifecycle-hooks/#container-hooks<br />+optional |
| `stopSignal` | *[Signal](./k8s-io-api-core-v1.md#Signal) | No | StopSignal defines which signal will be sent to a container when it is being stopped.<br />If not specified, the default is defined by the container runtime in use.<br />StopSignal can only be set for Pods with a non-empty .spec.os.name<br />+optional |

## LifecycleHandler

LifecycleHandler defines a specific action that should be taken in a lifecycle<br />hook. One and only one of the fields, except TCPSocket must be specified.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `exec` | *[ExecAction](./k8s-io-api-core-v1.md#ExecAction) | No | Exec specifies a command to execute in the container.<br />+optional |
| `httpGet` | *[HTTPGetAction](./k8s-io-api-core-v1.md#HTTPGetAction) | No | HTTPGet specifies an HTTP GET request to perform.<br />+optional |
| `tcpSocket` | *[TCPSocketAction](./k8s-io-api-core-v1.md#TCPSocketAction) | No | Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept<br />for backward compatibility. There is no validation of this field and<br />lifecycle hooks will fail at runtime when it is specified.<br />+optional |
| `sleep` | *[SleepAction](./k8s-io-api-core-v1.md#SleepAction) | No | Sleep represents a duration that the container should sleep.<br />+optional |

## LocalObjectReference

LocalObjectReference contains enough information to let you locate the<br />referenced object inside the same namespace.<br />---<br />New uses of this type are discouraged because of difficulty describing its usage when embedded in APIs.<br /> 1. Invalid usage help.  It is impossible to add specific help for individual usage.  In most embedded usages, there are particular<br />    restrictions like, "must refer only to types A and B" or "UID not honored" or "name must be restricted".<br />    Those cannot be well described when embedded.<br /> 2. Inconsistent validation.  Because the usages are different, the validation rules are different by usage, which makes it hard for users to predict what will happen.<br /> 3. We cannot easily change it.  Because this type is embedded in many locations, updates to this type<br />    will affect numerous schemas.  Don't make new APIs embed an underspecified API type they do not control.<br /><br />Instead of using this type, create a locally provided and used type that is well-focused on your reference.<br />For example, ServiceReferences for admission registration: https://github.com/kubernetes/api/blob/release-1.17/admissionregistration/v1/types.go#L533 .<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent.<br />This field is effectively required, but due to backwards compatibility is<br />allowed to be empty. Instances of this type with an empty value here are<br />almost certainly wrong.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br />+optional<br />+default=""<br />+kubebuilder:default=""<br />TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |

## ModifyVolumeStatus

ModifyVolumeStatus represents the status object of ControllerModifyVolume operation

| Stanza | Type | Required | Description |
|---|---|---|---|
| `targetVolumeAttributesClassName` | string | No | targetVolumeAttributesClassName is the name of the VolumeAttributesClass the PVC currently being reconciled |
| `status` | [PersistentVolumeClaimModifyVolumeStatus](./k8s-io-api-core-v1.md#PersistentVolumeClaimModifyVolumeStatus) | Yes | status is the status of the ControllerModifyVolume operation. It can be in any of following states:<br /> - Pending<br />   Pending indicates that the PersistentVolumeClaim cannot be modified due to unmet requirements, such as<br />   the specified VolumeAttributesClass not existing.<br /> - InProgress<br />   InProgress indicates that the volume is being modified.<br /> - Infeasible<br />  Infeasible indicates that the request has been rejected as invalid by the CSI driver. To<br />	  resolve the error, a valid VolumeAttributesClass needs to be specified.<br />Note: New statuses can be added in the future. Consumers should check for unknown statuses and fail appropriately. |

## MountPropagationMode

MountPropagationMode describes mount propagation.<br />+enum



## ObjectFieldSelector

ObjectFieldSelector selects an APIVersioned field of an object.<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `apiVersion` | string | No | Version of the schema the FieldPath is written in terms of, defaults to "v1".<br />+optional |
| `fieldPath` | string | Yes | Path of the field to select in the specified API version. |

## PersistentVolumeAccessMode

+enum



## PersistentVolumeClaim

PersistentVolumeClaim is a user's request for and claim to a persistent volume

| Stanza | Type | Required | Description |
|---|---|---|---|
| `kind` | string | No | Kind is a string value representing the REST resource this object represents.<br />Servers may infer this from the endpoint the client submits requests to.<br />Cannot be updated.<br />In CamelCase.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds<br />+optional |
| `apiVersion` | string | No | APIVersion defines the versioned schema of this representation of an object.<br />Servers should convert recognized schemas to the latest internal value, and<br />may reject unrecognized values.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources<br />+optional |
| `name` | string | No | Name must be unique within a namespace. Is required when creating resources, although<br />some resources may allow a client to request the generation of an appropriate name<br />automatically. Name is primarily intended for creation idempotence and configuration<br />definition.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#names<br />+optional |
| `generateName` | string | No | GenerateName is an optional prefix, used by the server, to generate a unique<br />name ONLY IF the Name field has not been provided.<br />If this field is used, the name returned to the client will be different<br />than the name passed. This value will also be combined with a unique suffix.<br />The provided value has the same validation rules as the Name field,<br />and may be truncated by the length of the suffix required to make the value<br />unique on the server.<br /><br />If this field is specified and the generated name exists, the server will return a 409.<br /><br />Applied only if Name is not specified.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency<br />+optional |
| `namespace` | string | No | Namespace defines the space within which each name must be unique. An empty namespace is<br />equivalent to the "default" namespace, but "default" is the canonical representation.<br />Not all objects are required to be scoped to a namespace - the value of this field for<br />those objects will be empty.<br /><br />Must be a DNS_LABEL.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces<br />+optional |
| `selfLink` | string | No | Deprecated: selfLink is a legacy read-only field that is no longer populated by the system.<br />+optional |
| `uid` | [UID](./k8s-io-apimachinery-pkg-types.md#UID) | No | UID is the unique in time and space value for this object. It is typically generated by<br />the server on successful creation of a resource and is not allowed to change on PUT<br />operations.<br /><br />Populated by the system.<br />Read-only.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names#uids<br />+optional |
| `resourceVersion` | string | No | An opaque value that represents the internal version of this object that can<br />be used by clients to determine when objects have changed. May be used for optimistic<br />concurrency, change detection, and the watch operation on a resource or set of resources.<br />Clients must treat these values as opaque and passed unmodified back to the server.<br />They may only be valid for a particular resource or set of resources.<br /><br />Populated by the system.<br />Read-only.<br />Value must be treated as opaque by clients and .<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency<br />+optional |
| `generation` | int64 | No | A sequence number representing a specific generation of the desired state.<br />Populated by the system. Read-only.<br />+optional |
| `creationTimestamp` | [Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No | CreationTimestamp is a timestamp representing the server time when this object was<br />created. It is not guaranteed to be set in happens-before order across separate operations.<br />Clients may not set this value. It is represented in RFC3339 form and is in UTC.<br /><br />Populated by the system.<br />Read-only.<br />Null for lists.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata<br />+optional |
| `deletionTimestamp` | *[Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No | DeletionTimestamp is RFC 3339 date and time at which this resource will be deleted. This<br />field is set by the server when a graceful deletion is requested by the user, and is not<br />directly settable by a client. The resource is expected to be deleted (no longer visible<br />from resource lists, and not reachable by name) after the time in this field, once the<br />finalizers list is empty. As long as the finalizers list contains items, deletion is blocked.<br />Once the deletionTimestamp is set, this value may not be unset or be set further into the<br />future, although it may be shortened or the resource may be deleted prior to this time.<br />For example, a user may request that a pod is deleted in 30 seconds. The Kubelet will react<br />by sending a graceful termination signal to the containers in the pod. After that 30 seconds,<br />the Kubelet will send a hard termination signal (SIGKILL) to the container and after cleanup,<br />remove the pod from the API. In the presence of network partitions, this object may still<br />exist after this timestamp, until an administrator or automated process can determine the<br />resource is fully terminated.<br />If not set, graceful deletion of the object has not been requested.<br /><br />Populated by the system when a graceful deletion is requested.<br />Read-only.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata<br />+optional |
| `deletionGracePeriodSeconds` | *int64 | No | Number of seconds allowed for this object to gracefully terminate before<br />it will be removed from the system. Only set when deletionTimestamp is also set.<br />May only be shortened.<br />Read-only.<br />+optional |
| `labels` | map[string]string | No | Map of string keys and values that can be used to organize and categorize<br />(scope and select) objects. May match selectors of replication controllers<br />and services.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels<br />+optional |
| `annotations` | map[string]string | No | Annotations is an unstructured key value map stored with a resource that may be<br />set by external tools to store and retrieve arbitrary metadata. They are not<br />queryable and should be preserved when modifying objects.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations<br />+optional |
| `ownerReferences` | [][OwnerReference](./k8s-io-apimachinery-pkg-apis-meta-v1.md#OwnerReference) | No | List of objects depended by this object. If ALL objects in the list have<br />been deleted, this object will be garbage collected. If this object is managed by a controller,<br />then an entry in this list will point to this controller, with the controller field set to true.<br />There cannot be more than one managing controller.<br />+optional<br />+patchMergeKey=uid<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=uid |
| `finalizers` | []string | No | Must be empty before the object is deleted from the registry. Each entry<br />is an identifier for the responsible component that will remove the entry<br />from the list. If the deletionTimestamp of the object is non-nil, entries<br />in this list can only be removed.<br />Finalizers may be processed and removed in any order.  Order is NOT enforced<br />because it introduces significant risk of stuck finalizers.<br />finalizers is a shared field, any actor with permission can reorder it.<br />If the finalizer list is processed in order, then this can lead to a situation<br />in which the component responsible for the first finalizer in the list is<br />waiting for a signal (field value, external system, or other) produced by a<br />component responsible for a finalizer later in the list, resulting in a deadlock.<br />Without enforced ordering finalizers are free to order amongst themselves and<br />are not vulnerable to ordering changes in the list.<br />+optional<br />+patchStrategy=merge<br />+listType=set |
| `managedFields` | [][ManagedFieldsEntry](./k8s-io-apimachinery-pkg-apis-meta-v1.md#ManagedFieldsEntry) | No | ManagedFields maps workflow-id and version to the set of fields<br />that are managed by that workflow. This is mostly for internal<br />housekeeping, and users typically shouldn't need to set or<br />understand this field. A workflow can be the user's name, a<br />controller's name, or the name of a specific apply path like<br />"ci-cd". The set of fields is always in the version that the<br />workflow used when modifying the object.<br /><br />+optional<br />+listType=atomic |
| `spec` | [PersistentVolumeClaimSpec](./k8s-io-api-core-v1.md#PersistentVolumeClaimSpec) | No | spec defines the desired characteristics of a volume requested by a pod author.<br />More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims<br />+optional |
| `status` | [PersistentVolumeClaimStatus](./k8s-io-api-core-v1.md#PersistentVolumeClaimStatus) | No | status represents the current information/status of a persistent volume claim.<br />Read-only.<br />More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims<br />+optional |

## PersistentVolumeClaimCondition

PersistentVolumeClaimCondition contains details about state of pvc

| Stanza | Type | Required | Description |
|---|---|---|---|
| `type` | [PersistentVolumeClaimConditionType](./k8s-io-api-core-v1.md#PersistentVolumeClaimConditionType) | Yes | Type is the type of the condition.<br />More info: https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#:~:text=set%20to%20%27ResizeStarted%27.-,PersistentVolumeClaimCondition,-contains%20details%20about |
| `status` | [ConditionStatus](./k8s-io-api-core-v1.md#ConditionStatus) | Yes | Status is the status of the condition.<br />Can be True, False, Unknown.<br />More info: https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#:~:text=state%20of%20pvc-,conditions.status,-(string)%2C%20required |
| `lastProbeTime` | [Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No | lastProbeTime is the time we probed the condition.<br />+optional |
| `lastTransitionTime` | [Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No | lastTransitionTime is the time the condition transitioned from one status to another.<br />+optional |
| `reason` | string | No | reason is a unique, this should be a short, machine understandable string that gives the reason<br />for condition's last transition. If it reports "Resizing" that means the underlying<br />persistent volume is being resized.<br />+optional |
| `message` | string | No | message is the human-readable message indicating details about last transition.<br />+optional |

## PersistentVolumeClaimConditionType

PersistentVolumeClaimConditionType defines the condition of PV claim.<br />Valid values are:<br />  - "Resizing", "FileSystemResizePending"<br /><br />The following additional values can be expected:<br />  - "ControllerResizeError", "NodeResizeError"<br /><br />If VolumeAttributesClass feature gate is enabled, then following additional values can be expected:<br />  - "ModifyVolumeError", "ModifyingVolume"



## PersistentVolumeClaimModifyVolumeStatus

+enum<br />New statuses can be added in the future. Consumers should check for unknown statuses and fail appropriately



## PersistentVolumeClaimPhase

+enum



## PersistentVolumeClaimSpec

PersistentVolumeClaimSpec describes the common attributes of storage devices<br />and allows a Source for provider-specific attributes

| Stanza | Type | Required | Description |
|---|---|---|---|
| `accessModes` | [][PersistentVolumeAccessMode](./k8s-io-api-core-v1.md#PersistentVolumeAccessMode) | No | accessModes contains the desired access modes the volume should have.<br />More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1<br />+optional<br />+listType=atomic |
| `selector` | *[LabelSelector](./k8s-io-apimachinery-pkg-apis-meta-v1.md#LabelSelector) | No | selector is a label query over volumes to consider for binding.<br />+optional |
| `resources` | [VolumeResourceRequirements](./k8s-io-api-core-v1.md#VolumeResourceRequirements) | No | resources represents the minimum resources the volume should have.<br />Users are allowed to specify resource requirements<br />that are lower than previous value but must still be higher than capacity recorded in the<br />status field of the claim.<br />More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources<br />+optional |
| `volumeName` | string | No | volumeName is the binding reference to the PersistentVolume backing this claim.<br />+optional |
| `storageClassName` | *string | No | storageClassName is the name of the StorageClass required by the claim.<br />More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1<br />+optional |
| `volumeMode` | *[PersistentVolumeMode](./k8s-io-api-core-v1.md#PersistentVolumeMode) | No | volumeMode defines what type of volume is required by the claim.<br />Value of Filesystem is implied when not included in claim spec.<br />+optional |
| `dataSource` | *[TypedLocalObjectReference](./k8s-io-api-core-v1.md#TypedLocalObjectReference) | No | dataSource field can be used to specify either:<br />* An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot)<br />* An existing PVC (PersistentVolumeClaim)<br />If the provisioner or an external controller can support the specified data source,<br />it will create a new volume based on the contents of the specified data source.<br />When the AnyVolumeDataSource feature gate is enabled, dataSource contents will be copied to dataSourceRef,<br />and dataSourceRef contents will be copied to dataSource when dataSourceRef.namespace is not specified.<br />If the namespace is specified, then dataSourceRef will not be copied to dataSource.<br />+optional |
| `dataSourceRef` | *[TypedObjectReference](./k8s-io-api-core-v1.md#TypedObjectReference) | No | dataSourceRef specifies the object from which to populate the volume with data, if a non-empty<br />volume is desired. This may be any object from a non-empty API group (non<br />core object) or a PersistentVolumeClaim object.<br />When this field is specified, volume binding will only succeed if the type of<br />the specified object matches some installed volume populator or dynamic<br />provisioner.<br />This field will replace the functionality of the dataSource field and as such<br />if both fields are non-empty, they must have the same value. For backwards<br />compatibility, when namespace isn't specified in dataSourceRef,<br />both fields (dataSource and dataSourceRef) will be set to the same<br />value automatically if one of them is empty and the other is non-empty.<br />When namespace is specified in dataSourceRef,<br />dataSource isn't set to the same value and must be empty.<br />There are three important differences between dataSource and dataSourceRef:<br />* While dataSource only allows two specific types of objects, dataSourceRef<br />  allows any non-core object, as well as PersistentVolumeClaim objects.<br />* While dataSource ignores disallowed values (dropping them), dataSourceRef<br />  preserves all values, and generates an error if a disallowed value is<br />  specified.<br />* While dataSource only allows local objects, dataSourceRef allows objects<br />  in any namespaces.<br />(Beta) Using this field requires the AnyVolumeDataSource feature gate to be enabled.<br />(Alpha) Using the namespace field of dataSourceRef requires the CrossNamespaceVolumeDataSource feature gate to be enabled.<br />+optional |
| `volumeAttributesClassName` | *string | No | volumeAttributesClassName may be used to set the VolumeAttributesClass used by this claim.<br />If specified, the CSI driver will create or update the volume with the attributes defined<br />in the corresponding VolumeAttributesClass. This has a different purpose than storageClassName,<br />it can be changed after the claim is created. An empty string or nil value indicates that no<br />VolumeAttributesClass will be applied to the claim. If the claim enters an Infeasible error state,<br />this field can be reset to its previous value (including nil) to cancel the modification.<br />If the resource referred to by volumeAttributesClass does not exist, this PersistentVolumeClaim will be<br />set to a Pending state, as reflected by the modifyVolumeStatus field, until such as a resource<br />exists.<br />More info: https://kubernetes.io/docs/concepts/storage/volume-attributes-classes/<br />+featureGate=VolumeAttributesClass<br />+optional |

## PersistentVolumeClaimStatus

PersistentVolumeClaimStatus is the current status of a persistent volume claim.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `phase` | [PersistentVolumeClaimPhase](./k8s-io-api-core-v1.md#PersistentVolumeClaimPhase) | No | phase represents the current phase of PersistentVolumeClaim.<br />+optional |
| `accessModes` | [][PersistentVolumeAccessMode](./k8s-io-api-core-v1.md#PersistentVolumeAccessMode) | No | accessModes contains the actual access modes the volume backing the PVC has.<br />More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1<br />+optional<br />+listType=atomic |
| `capacity` | [ResourceList](./k8s-io-api-core-v1.md#ResourceList) | No | capacity represents the actual resources of the underlying volume.<br />+optional |
| `conditions` | [][PersistentVolumeClaimCondition](./k8s-io-api-core-v1.md#PersistentVolumeClaimCondition) | No | conditions is the current Condition of persistent volume claim. If underlying persistent volume is being<br />resized then the Condition will be set to 'Resizing'.<br />+optional<br />+patchMergeKey=type<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=type |
| `allocatedResources` | [ResourceList](./k8s-io-api-core-v1.md#ResourceList) | No | allocatedResources tracks the resources allocated to a PVC including its capacity.<br />Key names follow standard Kubernetes label syntax. Valid values are either:<br />	* Un-prefixed keys:<br />		- storage - the capacity of the volume.<br />	* Custom resources must use implementation-defined prefixed names such as "example.com/my-custom-resource"<br />Apart from above values - keys that are unprefixed or have kubernetes.io prefix are considered<br />reserved and hence may not be used.<br /><br />Capacity reported here may be larger than the actual capacity when a volume expansion operation<br />is requested.<br />For storage quota, the larger value from allocatedResources and PVC.spec.resources is used.<br />If allocatedResources is not set, PVC.spec.resources alone is used for quota calculation.<br />If a volume expansion capacity request is lowered, allocatedResources is only<br />lowered if there are no expansion operations in progress and if the actual volume capacity<br />is equal or lower than the requested capacity.<br /><br />A controller that receives PVC update with previously unknown resourceName<br />should ignore the update for the purpose it was designed. For example - a controller that<br />only is responsible for resizing capacity of the volume, should ignore PVC updates that change other valid<br />resources associated with PVC.<br />+optional |
| `allocatedResourceStatuses` | map[[ResourceName](./k8s-io-api-core-v1.md#ResourceName)][ClaimResourceStatus](./k8s-io-api-core-v1.md#ClaimResourceStatus) | No | allocatedResourceStatuses stores status of resource being resized for the given PVC.<br />Key names follow standard Kubernetes label syntax. Valid values are either:<br />	* Un-prefixed keys:<br />		- storage - the capacity of the volume.<br />	* Custom resources must use implementation-defined prefixed names such as "example.com/my-custom-resource"<br />Apart from above values - keys that are unprefixed or have kubernetes.io prefix are considered<br />reserved and hence may not be used.<br /><br />ClaimResourceStatus can be in any of following states:<br />	- ControllerResizeInProgress:<br />		State set when resize controller starts resizing the volume in control-plane.<br />	- ControllerResizeFailed:<br />		State set when resize has failed in resize controller with a terminal error.<br />	- NodeResizePending:<br />		State set when resize controller has finished resizing the volume but further resizing of<br />		volume is needed on the node.<br />	- NodeResizeInProgress:<br />		State set when kubelet starts resizing the volume.<br />	- NodeResizeFailed:<br />		State set when resizing has failed in kubelet with a terminal error. Transient errors don't set<br />		NodeResizeFailed.<br />For example: if expanding a PVC for more capacity - this field can be one of the following states:<br />	- pvc.status.allocatedResourceStatus['storage'] = "ControllerResizeInProgress"<br />     - pvc.status.allocatedResourceStatus['storage'] = "ControllerResizeFailed"<br />     - pvc.status.allocatedResourceStatus['storage'] = "NodeResizePending"<br />     - pvc.status.allocatedResourceStatus['storage'] = "NodeResizeInProgress"<br />     - pvc.status.allocatedResourceStatus['storage'] = "NodeResizeFailed"<br />When this field is not set, it means that no resize operation is in progress for the given PVC.<br /><br />A controller that receives PVC update with previously unknown resourceName or ClaimResourceStatus<br />should ignore the update for the purpose it was designed. For example - a controller that<br />only is responsible for resizing capacity of the volume, should ignore PVC updates that change other valid<br />resources associated with PVC.<br />+mapType=granular<br />+optional |
| `currentVolumeAttributesClassName` | *string | No | currentVolumeAttributesClassName is the current name of the VolumeAttributesClass the PVC is using.<br />When unset, there is no VolumeAttributeClass applied to this PersistentVolumeClaim<br />+featureGate=VolumeAttributesClass<br />+optional |
| `modifyVolumeStatus` | *[ModifyVolumeStatus](./k8s-io-api-core-v1.md#ModifyVolumeStatus) | No | ModifyVolumeStatus represents the status object of ControllerModifyVolume operation.<br />When this is unset, there is no ModifyVolume operation being attempted.<br />+featureGate=VolumeAttributesClass<br />+optional |

## PersistentVolumeClaimVolumeSource

PersistentVolumeClaimVolumeSource references the user's PVC in the same namespace.<br />This volume finds the bound PV and mounts that volume for the pod. A<br />PersistentVolumeClaimVolumeSource is, essentially, a wrapper around another<br />type of volume that is owned by someone else (the system).

| Stanza | Type | Required | Description |
|---|---|---|---|
| `claimName` | string | Yes | claimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume.<br />More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims |
| `readOnly` | bool | No | readOnly Will force the ReadOnly setting in VolumeMounts.<br />Default false.<br />+optional |

## PersistentVolumeMode

PersistentVolumeMode describes how a volume is intended to be consumed, either Block or Filesystem.<br />+enum



## PodCertificateProjection

PodCertificateProjection provides a private key and X.509 certificate in the<br />pod filesystem.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `signerName` | string | No | Kubelet's generated CSRs will be addressed to this signer.<br /><br />+required |
| `keyType` | string | No | The type of keypair Kubelet will generate for the pod.<br /><br />Valid values are "RSA3072", "RSA4096", "ECDSAP256", "ECDSAP384",<br />"ECDSAP521", and "ED25519".<br /><br />+required |
| `maxExpirationSeconds` | *int32 | No | maxExpirationSeconds is the maximum lifetime permitted for the<br />certificate.<br /><br />Kubelet copies this value verbatim into the PodCertificateRequests it<br />generates for this projection.<br /><br />If omitted, kube-apiserver will set it to 86400(24 hours). kube-apiserver<br />will reject values shorter than 3600 (1 hour).  The maximum allowable<br />value is 7862400 (91 days).<br /><br />The signer implementation is then free to issue a certificate with any<br />lifetime *shorter* than MaxExpirationSeconds, but no shorter than 3600<br />seconds (1 hour).  This constraint is enforced by kube-apiserver.<br />`kubernetes.io` signers will never issue certificates with a lifetime<br />longer than 24 hours.<br /><br />+optional |
| `credentialBundlePath` | string | No | Write the credential bundle at this path in the projected volume.<br /><br />The credential bundle is a single file that contains multiple PEM blocks.<br />The first PEM block is a PRIVATE KEY block, containing a PKCS#8 private<br />key.<br /><br />The remaining blocks are CERTIFICATE blocks, containing the issued<br />certificate chain from the signer (leaf and any intermediates).<br /><br />Using credentialBundlePath lets your Pod's application code make a single<br />atomic read that retrieves a consistent key and certificate chain.  If you<br />project them to separate files, your application code will need to<br />additionally check that the leaf certificate was issued to the key.<br /><br />+optional |
| `keyPath` | string | No | Write the key at this path in the projected volume.<br /><br />Most applications should use credentialBundlePath.  When using keyPath<br />and certificateChainPath, your application needs to check that the key<br />and leaf certificate are consistent, because it is possible to read the<br />files mid-rotation.<br /><br />+optional |
| `certificateChainPath` | string | No | Write the certificate chain at this path in the projected volume.<br /><br />Most applications should use credentialBundlePath.  When using keyPath<br />and certificateChainPath, your application needs to check that the key<br />and leaf certificate are consistent, because it is possible to read the<br />files mid-rotation.<br /><br />+optional |
| `userAnnotations` | map[string]string | No | userAnnotations allow pod authors to pass additional information to<br />the signer implementation.  Kubernetes does not restrict or validate this<br />metadata in any way.<br /><br />These values are copied verbatim into the `spec.unverifiedUserAnnotations` field of<br />the PodCertificateRequest objects that Kubelet creates.<br /><br />Entries are subject to the same validation as object metadata annotations,<br />with the addition that all keys must be domain-prefixed. No restrictions<br />are placed on values, except an overall size limitation on the entire field.<br /><br />Signers should document the keys and values they support. Signers should<br />deny requests that contain keys they do not recognize. |

## Probe

Probe describes a health check to be performed against a container to determine whether it is<br />alive or ready to receive traffic.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `exec` | *[ExecAction](./k8s-io-api-core-v1.md#ExecAction) | No | Exec specifies a command to execute in the container.<br />+optional |
| `httpGet` | *[HTTPGetAction](./k8s-io-api-core-v1.md#HTTPGetAction) | No | HTTPGet specifies an HTTP GET request to perform.<br />+optional |
| `tcpSocket` | *[TCPSocketAction](./k8s-io-api-core-v1.md#TCPSocketAction) | No | TCPSocket specifies a connection to a TCP port.<br />+optional |
| `grpc` | *[GRPCAction](./k8s-io-api-core-v1.md#GRPCAction) | No | GRPC specifies a GRPC HealthCheckRequest.<br />+optional |
| `initialDelaySeconds` | int32 | No | Number of seconds after the container has started before liveness probes are initiated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `timeoutSeconds` | int32 | No | Number of seconds after which the probe times out.<br />Defaults to 1 second. Minimum value is 1.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle#container-probes<br />+optional |
| `periodSeconds` | int32 | No | How often (in seconds) to perform the probe.<br />Default to 10 seconds. Minimum value is 1.<br />+optional |
| `successThreshold` | int32 | No | Minimum consecutive successes for the probe to be considered successful after having failed.<br />Defaults to 1. Must be 1 for liveness and startup. Minimum value is 1.<br />+optional |
| `failureThreshold` | int32 | No | Minimum consecutive failures for the probe to be considered failed after having succeeded.<br />Defaults to 3. Minimum value is 1.<br />+optional |
| `terminationGracePeriodSeconds` | *int64 | No | Optional duration in seconds the pod needs to terminate gracefully upon probe failure.<br />The grace period is the duration in seconds after the processes running in the pod are sent<br />a termination signal and the time when the processes are forcibly halted with a kill signal.<br />Set this value longer than the expected cleanup time for your process.<br />If this value is nil, the pod's terminationGracePeriodSeconds will be used. Otherwise, this<br />value overrides the value provided by the pod spec.<br />Value must be non-negative integer. The value zero indicates stop immediately via<br />the kill signal (no opportunity to shut down).<br />This is a beta field and requires enabling ProbeTerminationGracePeriod feature gate.<br />Minimum value is 1. spec.terminationGracePeriodSeconds is used if unset.<br />+optional |

## ProcMountType

+enum



## ProjectedVolumeSource

Represents a projected volume source

| Stanza | Type | Required | Description |
|---|---|---|---|
| `sources` | [][VolumeProjection](./k8s-io-api-core-v1.md#VolumeProjection) | No | sources is the list of volume projections. Each entry in this list<br />handles one source.<br />+optional<br />+listType=atomic |
| `defaultMode` | *int32 | No | defaultMode are the mode bits used to set permissions on created files by default.<br />Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br />YAML accepts both octal and decimal values, JSON requires decimal values for mode bits.<br />Directories within the path are not affected by this setting.<br />This might be in conflict with other options that affect the file<br />mode, like fsGroup, and the result can be other mode bits set.<br />+optional |

## Protocol

Protocol defines network protocols supported for things like container ports.<br />+enum



## PullPolicy

PullPolicy describes a policy for if/when to pull a container image<br />+enum



## RecursiveReadOnlyMode

RecursiveReadOnlyMode describes recursive-readonly mode.



## ResourceClaim

ResourceClaim references one entry in PodSpec.ResourceClaims.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name must match the name of one entry in pod.spec.resourceClaims of<br />the Pod where this field is used. It makes that resource available<br />inside a container. |
| `request` | string | No | Request is the name chosen for a request in the referenced claim.<br />If empty, everything from the claim is made available, otherwise<br />only the result of this request.<br /><br />+optional |

## ResourceFieldSelector

ResourceFieldSelector represents container resources (cpu, memory) and their output format<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `containerName` | string | No | Container name: required for volumes, optional for env vars<br />+optional |
| `resource` | string | Yes | Required: resource to select |
| `divisor` | [Quantity](./k8s-io-apimachinery-pkg-api-resource.md#Quantity) | No | Specifies the output format of the exposed resources, defaults to "1"<br />+optional |

## ResourceList

ResourceList is a set of (resource name, quantity) pairs.



## ResourceName

ResourceName is the name identifying various resources in a ResourceList.



## ResourceRequirements

ResourceRequirements describes the compute resource requirements.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `limits` | [ResourceList](./k8s-io-api-core-v1.md#ResourceList) | No | Limits describes the maximum amount of compute resources allowed.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br />+optional |
| `requests` | [ResourceList](./k8s-io-api-core-v1.md#ResourceList) | No | Requests describes the minimum amount of compute resources required.<br />If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,<br />otherwise to an implementation-defined value. Requests cannot exceed Limits.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br />+optional |
| `claims` | [][ResourceClaim](./k8s-io-api-core-v1.md#ResourceClaim) | No | Claims lists the names of resources, defined in spec.resourceClaims,<br />that are used by this container.<br /><br />This field depends on the<br />DynamicResourceAllocation feature gate.<br /><br />This field is immutable. It can only be set for containers.<br /><br />+listType=map<br />+listMapKey=name<br />+featureGate=DynamicResourceAllocation<br />+optional |

## SELinuxOptions

SELinuxOptions are the labels to be applied to the container

| Stanza | Type | Required | Description |
|---|---|---|---|
| `user` | string | No | User is a SELinux user label that applies to the container.<br />+optional |
| `role` | string | No | Role is a SELinux role label that applies to the container.<br />+optional |
| `type` | string | No | Type is a SELinux type label that applies to the container.<br />+optional |
| `level` | string | No | Level is SELinux level label that applies to the container.<br />+optional |

## SeccompProfile

SeccompProfile defines a pod/container's seccomp profile settings.<br />Only one profile source may be set.<br />+union

| Stanza | Type | Required | Description |
|---|---|---|---|
| `type` | [SeccompProfileType](./k8s-io-api-core-v1.md#SeccompProfileType) | Yes | type indicates which kind of seccomp profile will be applied.<br />Valid options are:<br /><br />Localhost - a profile defined in a file on the node should be used.<br />RuntimeDefault - the container runtime default profile should be used.<br />Unconfined - no profile should be applied.<br />+unionDiscriminator |
| `localhostProfile` | *string | No | localhostProfile indicates a profile defined in a file on the node should be used.<br />The profile must be preconfigured on the node to work.<br />Must be a descending path, relative to the kubelet's configured seccomp profile location.<br />Must be set if type is "Localhost". Must NOT be set for any other type.<br />+optional |

## SeccompProfileType

SeccompProfileType defines the supported seccomp profile types.<br />+enum



## SecretEnvSource

SecretEnvSource selects a Secret to populate the environment<br />variables with.<br /><br />The contents of the target Secret's Data field will represent the<br />key-value pairs as environment variables.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent.<br />This field is effectively required, but due to backwards compatibility is<br />allowed to be empty. Instances of this type with an empty value here are<br />almost certainly wrong.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br />+optional<br />+default=""<br />+kubebuilder:default=""<br />TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |
| `optional` | *bool | No | Specify whether the Secret must be defined<br />+optional |

## SecretKeySelector

SecretKeySelector selects a key of a Secret.<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent.<br />This field is effectively required, but due to backwards compatibility is<br />allowed to be empty. Instances of this type with an empty value here are<br />almost certainly wrong.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br />+optional<br />+default=""<br />+kubebuilder:default=""<br />TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |
| `key` | string | Yes | The key of the secret to select from.  Must be a valid secret key. |
| `optional` | *bool | No | Specify whether the Secret or its key must be defined<br />+optional |

## SecretProjection

Adapts a secret into a projected volume.<br /><br />The contents of the target Secret's Data field will be presented in a<br />projected volume as files using the keys in the Data field as the file names.<br />Note that this is identical to a secret volume source without the default<br />mode.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name of the referent.<br />This field is effectively required, but due to backwards compatibility is<br />allowed to be empty. Instances of this type with an empty value here are<br />almost certainly wrong.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br />+optional<br />+default=""<br />+kubebuilder:default=""<br />TODO: Drop `kubebuilder:default` when controller-gen doesn't need it https://github.com/kubernetes-sigs/kubebuilder/issues/3896. |
| `items` | [][KeyToPath](./k8s-io-api-core-v1.md#KeyToPath) | No | items if unspecified, each key-value pair in the Data field of the referenced<br />Secret will be projected into the volume as a file whose name is the<br />key and content is the value. If specified, the listed keys will be<br />projected into the specified paths, and unlisted keys will not be<br />present. If a key is specified which is not present in the Secret,<br />the volume setup will error unless it is marked optional. Paths must be<br />relative and may not contain the '..' path or start with '..'.<br />+optional<br />+listType=atomic |
| `optional` | *bool | No | optional field specify whether the Secret or its key must be defined<br />+optional |

## SecretVolumeSource

Adapts a Secret into a volume.<br /><br />The contents of the target Secret's Data field will be presented in a volume<br />as files using the keys in the Data field as the file names.<br />Secret volumes support ownership management and SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `secretName` | string | No | secretName is the name of the secret in the pod's namespace to use.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#secret<br />+optional |
| `items` | [][KeyToPath](./k8s-io-api-core-v1.md#KeyToPath) | No | items If unspecified, each key-value pair in the Data field of the referenced<br />Secret will be projected into the volume as a file whose name is the<br />key and content is the value. If specified, the listed keys will be<br />projected into the specified paths, and unlisted keys will not be<br />present. If a key is specified which is not present in the Secret,<br />the volume setup will error unless it is marked optional. Paths must be<br />relative and may not contain the '..' path or start with '..'.<br />+optional<br />+listType=atomic |
| `defaultMode` | *int32 | No | defaultMode is Optional: mode bits used to set permissions on created files by default.<br />Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br />YAML accepts both octal and decimal values, JSON requires decimal values<br />for mode bits. Defaults to 0644.<br />Directories within the path are not affected by this setting.<br />This might be in conflict with other options that affect the file<br />mode, like fsGroup, and the result can be other mode bits set.<br />+optional |
| `optional` | *bool | No | optional field specify whether the Secret or its keys must be defined<br />+optional |

## SecurityContext

SecurityContext holds security configuration that will be applied to a container.<br />Some fields are present in both SecurityContext and PodSecurityContext.  When both<br />are set, the values in SecurityContext take precedence.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `capabilities` | *[Capabilities](./k8s-io-api-core-v1.md#Capabilities) | No | The capabilities to add/drop when running containers.<br />Defaults to the default set of capabilities granted by the container runtime.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `privileged` | *bool | No | Run container in privileged mode.<br />Processes in privileged containers are essentially equivalent to root on the host.<br />Defaults to false.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `seLinuxOptions` | *[SELinuxOptions](./k8s-io-api-core-v1.md#SELinuxOptions) | No | The SELinux context to be applied to the container.<br />If unspecified, the container runtime will allocate a random SELinux context for each<br />container.  May also be set in PodSecurityContext.  If set in both SecurityContext and<br />PodSecurityContext, the value specified in SecurityContext takes precedence.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `windowsOptions` | *[WindowsSecurityContextOptions](./k8s-io-api-core-v1.md#WindowsSecurityContextOptions) | No | The Windows specific settings applied to all containers.<br />If unspecified, the options from the PodSecurityContext will be used.<br />If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.<br />Note that this field cannot be set when spec.os.name is linux.<br />+optional |
| `runAsUser` | *int64 | No | The UID to run the entrypoint of the container process.<br />Defaults to user specified in image metadata if unspecified.<br />May also be set in PodSecurityContext.  If set in both SecurityContext and<br />PodSecurityContext, the value specified in SecurityContext takes precedence.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `runAsGroup` | *int64 | No | The GID to run the entrypoint of the container process.<br />Uses runtime default if unset.<br />May also be set in PodSecurityContext.  If set in both SecurityContext and<br />PodSecurityContext, the value specified in SecurityContext takes precedence.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `runAsNonRoot` | *bool | No | Indicates that the container must run as a non-root user.<br />If true, the Kubelet will validate the image at runtime to ensure that it<br />does not run as UID 0 (root) and fail to start the container if it does.<br />If unset or false, no such validation will be performed.<br />May also be set in PodSecurityContext.  If set in both SecurityContext and<br />PodSecurityContext, the value specified in SecurityContext takes precedence.<br />+optional |
| `readOnlyRootFilesystem` | *bool | No | Whether this container has a read-only root filesystem.<br />Default is false.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `allowPrivilegeEscalation` | *bool | No | AllowPrivilegeEscalation controls whether a process can gain more<br />privileges than its parent process. This bool directly controls if<br />the no_new_privs flag will be set on the container process.<br />AllowPrivilegeEscalation is true always when the container is:<br />1) run as Privileged<br />2) has CAP_SYS_ADMIN<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `procMount` | *[ProcMountType](./k8s-io-api-core-v1.md#ProcMountType) | No | procMount denotes the type of proc mount to use for the containers.<br />The default value is Default which uses the container runtime defaults for<br />readonly paths and masked paths.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `seccompProfile` | *[SeccompProfile](./k8s-io-api-core-v1.md#SeccompProfile) | No | The seccomp options to use by this container. If seccomp options are<br />provided at both the pod & container level, the container options<br />override the pod options.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `appArmorProfile` | *[AppArmorProfile](./k8s-io-api-core-v1.md#AppArmorProfile) | No | appArmorProfile is the AppArmor options to use by this container. If set, this profile<br />overrides the pod's appArmorProfile.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |

## ServiceAccountTokenProjection

ServiceAccountTokenProjection represents a projected service account token<br />volume. This projection can be used to insert a service account token into<br />the pods runtime filesystem for use against APIs (Kubernetes API Server or<br />otherwise).

| Stanza | Type | Required | Description |
|---|---|---|---|
| `audience` | string | No | audience is the intended audience of the token. A recipient of a token<br />must identify itself with an identifier specified in the audience of the<br />token, and otherwise should reject the token. The audience defaults to the<br />identifier of the apiserver.<br />+optional |
| `expirationSeconds` | *int64 | No | expirationSeconds is the requested duration of validity of the service<br />account token. As the token approaches expiration, the kubelet volume<br />plugin will proactively rotate the service account token. The kubelet will<br />start trying to rotate the token if the token is older than 80 percent of<br />its time to live or if the token is older than 24 hours.Defaults to 1 hour<br />and must be at least 10 minutes.<br />+optional |
| `path` | string | Yes | path is the path relative to the mount point of the file to project the<br />token into. |

## Signal

Signal defines the stop signal of containers<br />+enum



## SleepAction

SleepAction describes a "sleep" action.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `seconds` | int64 | Yes | Seconds is the number of seconds to sleep. |

## StorageMedium

StorageMedium defines ways that storage can be allocated to a volume.



## TCPSocketAction

TCPSocketAction describes an action based on opening a socket

| Stanza | Type | Required | Description |
|---|---|---|---|
| `port` | [IntOrString](./k8s-io-apimachinery-pkg-util-intstr.md#IntOrString) | Yes | Number or name of the port to access on the container.<br />Number must be in the range 1 to 65535.<br />Name must be an IANA_SVC_NAME. |
| `host` | string | No | Optional: Host name to connect to, defaults to the pod IP.<br />+optional |

## TerminationMessagePolicy

TerminationMessagePolicy describes how termination messages are retrieved from a container.<br />+enum



## TypedLocalObjectReference

TypedLocalObjectReference contains enough information to let you locate the<br />typed referenced object inside the same namespace.<br />---<br />New uses of this type are discouraged because of difficulty describing its usage when embedded in APIs.<br /> 1. Invalid usage help.  It is impossible to add specific help for individual usage.  In most embedded usages, there are particular<br />    restrictions like, "must refer only to types A and B" or "UID not honored" or "name must be restricted".<br />    Those cannot be well described when embedded.<br /> 2. Inconsistent validation.  Because the usages are different, the validation rules are different by usage, which makes it hard for users to predict what will happen.<br /> 3. The fields are both imprecise and overly precise.  Kind is not a precise mapping to a URL. This can produce ambiguity<br />    during interpretation and require a REST mapping.  In most cases, the dependency is on the group,resource tuple<br />    and the version of the actual struct is irrelevant.<br /> 4. We cannot easily change it.  Because this type is embedded in many locations, updates to this type<br />    will affect numerous schemas.  Don't make new APIs embed an underspecified API type they do not control.<br /><br />Instead of using this type, create a locally provided and used type that is well-focused on your reference.<br />For example, ServiceReferences for admission registration: https://github.com/kubernetes/api/blob/release-1.17/admissionregistration/v1/types.go#L533 .<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `apiGroup` | *string | No | APIGroup is the group for the resource being referenced.<br />If APIGroup is not specified, the specified Kind must be in the core API group.<br />For any other third-party types, APIGroup is required.<br />+optional |
| `kind` | string | Yes | Kind is the type of resource being referenced |
| `name` | string | Yes | Name is the name of resource being referenced |

## TypedObjectReference

TypedObjectReference contains enough information to let you locate the typed referenced object

| Stanza | Type | Required | Description |
|---|---|---|---|
| `apiGroup` | *string | No | APIGroup is the group for the resource being referenced.<br />If APIGroup is not specified, the specified Kind must be in the core API group.<br />For any other third-party types, APIGroup is required.<br />+optional |
| `kind` | string | Yes | Kind is the type of resource being referenced |
| `name` | string | Yes | Name is the name of resource being referenced |
| `namespace` | *string | No | Namespace is the namespace of resource being referenced<br />Note that when a namespace is specified, a gateway.networking.k8s.io/ReferenceGrant object is required in the referent namespace to allow that namespace's owner to accept the reference. See the ReferenceGrant documentation for details.<br />(Alpha) This field requires the CrossNamespaceVolumeDataSource feature gate to be enabled.<br />+featureGate=CrossNamespaceVolumeDataSource<br />+optional |

## URIScheme

URIScheme identifies the scheme used for connection to a host for Get actions<br />+enum



## VolumeDevice

volumeDevice describes a mapping of a raw block device within a container.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | name must match the name of a persistentVolumeClaim in the pod |
| `devicePath` | string | Yes | devicePath is the path inside of the container that the device will be mapped to. |

## VolumeMount

VolumeMount describes a mounting of a Volume within a container.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | This must match the Name of a Volume. |
| `readOnly` | bool | No | Mounted read-only if true, read-write otherwise (false or unspecified).<br />Defaults to false.<br />+optional |
| `recursiveReadOnly` | *[RecursiveReadOnlyMode](./k8s-io-api-core-v1.md#RecursiveReadOnlyMode) | No | RecursiveReadOnly specifies whether read-only mounts should be handled<br />recursively.<br /><br />If ReadOnly is false, this field has no meaning and must be unspecified.<br /><br />If ReadOnly is true, and this field is set to Disabled, the mount is not made<br />recursively read-only.  If this field is set to IfPossible, the mount is made<br />recursively read-only, if it is supported by the container runtime.  If this<br />field is set to Enabled, the mount is made recursively read-only if it is<br />supported by the container runtime, otherwise the pod will not be started and<br />an error will be generated to indicate the reason.<br /><br />If this field is set to IfPossible or Enabled, MountPropagation must be set to<br />None (or be unspecified, which defaults to None).<br /><br />If this field is not specified, it is treated as an equivalent of Disabled.<br />+optional |
| `mountPath` | string | Yes | Path within the container at which the volume should be mounted.  Must<br />not contain ':'. |
| `subPath` | string | No | Path within the volume from which the container's volume should be mounted.<br />Defaults to "" (volume's root).<br />+optional |
| `mountPropagation` | *[MountPropagationMode](./k8s-io-api-core-v1.md#MountPropagationMode) | No | mountPropagation determines how mounts are propagated from the host<br />to container and the other way around.<br />When not set, MountPropagationNone is used.<br />This field is beta in 1.10.<br />When RecursiveReadOnly is set to IfPossible or to Enabled, MountPropagation must be None or unspecified<br />(which defaults to None).<br />+optional |
| `subPathExpr` | string | No | Expanded path within the volume from which the container's volume should be mounted.<br />Behaves similarly to SubPath but environment variable references $(VAR_NAME) are expanded using the container's environment.<br />Defaults to "" (volume's root).<br />SubPathExpr and SubPath are mutually exclusive.<br />+optional |

## VolumeProjection

Projection that may be projected along with other supported volume types.<br />Exactly one of these fields must be set.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `secret` | *[SecretProjection](./k8s-io-api-core-v1.md#SecretProjection) | No | secret information about the secret data to project<br />+optional |
| `downwardAPI` | *[DownwardAPIProjection](./k8s-io-api-core-v1.md#DownwardAPIProjection) | No | downwardAPI information about the downwardAPI data to project<br />+optional |
| `configMap` | *[ConfigMapProjection](./k8s-io-api-core-v1.md#ConfigMapProjection) | No | configMap information about the configMap data to project<br />+optional |
| `serviceAccountToken` | *[ServiceAccountTokenProjection](./k8s-io-api-core-v1.md#ServiceAccountTokenProjection) | No | serviceAccountToken is information about the serviceAccountToken data to project<br />+optional |
| `clusterTrustBundle` | *[ClusterTrustBundleProjection](./k8s-io-api-core-v1.md#ClusterTrustBundleProjection) | No | ClusterTrustBundle allows a pod to access the `.spec.trustBundle` field<br />of ClusterTrustBundle objects in an auto-updating file.<br /><br />Alpha, gated by the ClusterTrustBundleProjection feature gate.<br /><br />ClusterTrustBundle objects can either be selected by name, or by the<br />combination of signer name and a label selector.<br /><br />Kubelet performs aggressive normalization of the PEM contents written<br />into the pod filesystem.  Esoteric PEM features such as inter-block<br />comments and block headers are stripped.  Certificates are deduplicated.<br />The ordering of certificates within the file is arbitrary, and Kubelet<br />may change the order over time.<br /><br />+featureGate=ClusterTrustBundleProjection<br />+optional |
| `podCertificate` | *[PodCertificateProjection](./k8s-io-api-core-v1.md#PodCertificateProjection) | No | Projects an auto-rotating credential bundle (private key and certificate<br />chain) that the pod can use either as a TLS client or server.<br /><br />Kubelet generates a private key and uses it to send a<br />PodCertificateRequest to the named signer.  Once the signer approves the<br />request and issues a certificate chain, Kubelet writes the key and<br />certificate chain to the pod filesystem.  The pod does not start until<br />certificates have been issued for each podCertificate projected volume<br />source in its spec.<br /><br />Kubelet will begin trying to rotate the certificate at the time indicated<br />by the signer using the PodCertificateRequest.Status.BeginRefreshAt<br />timestamp.<br /><br />Kubelet can write a single file, indicated by the credentialBundlePath<br />field, or separate files, indicated by the keyPath and<br />certificateChainPath fields.<br /><br />The credential bundle is a single file in PEM format.  The first PEM<br />entry is the private key (in PKCS#8 format), and the remaining PEM<br />entries are the certificate chain issued by the signer (typically,<br />signers will return their certificate chain in leaf-to-root order).<br /><br />Prefer using the credential bundle format, since your application code<br />can read it atomically.  If you use keyPath and certificateChainPath,<br />your application must make two separate file reads. If these coincide<br />with a certificate rotation, it is possible that the private key and leaf<br />certificate you read may not correspond to each other.  Your application<br />will need to check for this condition, and re-read until they are<br />consistent.<br /><br />The named signer controls chooses the format of the certificate it<br />issues; consult the signer implementation's documentation to learn how to<br />use the certificates it issues.<br /><br />+featureGate=PodCertificateProjection<br />+optional |

## VolumeResourceRequirements

VolumeResourceRequirements describes the storage resource requirements for a volume.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `limits` | [ResourceList](./k8s-io-api-core-v1.md#ResourceList) | No | Limits describes the maximum amount of compute resources allowed.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br />+optional |
| `requests` | [ResourceList](./k8s-io-api-core-v1.md#ResourceList) | No | Requests describes the minimum amount of compute resources required.<br />If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,<br />otherwise to an implementation-defined value. Requests cannot exceed Limits.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br />+optional |

## WindowsSecurityContextOptions

WindowsSecurityContextOptions contain Windows-specific options and credentials.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `gmsaCredentialSpecName` | *string | No | GMSACredentialSpecName is the name of the GMSA credential spec to use.<br />+optional |
| `gmsaCredentialSpec` | *string | No | GMSACredentialSpec is where the GMSA admission webhook<br />(https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the<br />GMSA credential spec named by the GMSACredentialSpecName field.<br />+optional |
| `runAsUserName` | *string | No | The UserName in Windows to run the entrypoint of the container process.<br />Defaults to the user specified in image metadata if unspecified.<br />May also be set in PodSecurityContext. If set in both SecurityContext and<br />PodSecurityContext, the value specified in SecurityContext takes precedence.<br />+optional |
| `hostProcess` | *bool | No | HostProcess determines if a container should be run as a 'Host Process' container.<br />All of a Pod's containers must have the same effective HostProcess value<br />(it is not allowed to have a mix of HostProcess containers and non-HostProcess containers).<br />In addition, if HostProcess is true then HostNetwork must also be set to true.<br />+optional |


