# Package k8s.io/api/core/v1

- [AWSElasticBlockStoreVolumeSource](#AWSElasticBlockStoreVolumeSource)
- [Affinity](#Affinity)
- [AppArmorProfile](#AppArmorProfile)
- [AppArmorProfileType](#AppArmorProfileType)
- [AzureDataDiskCachingMode](#AzureDataDiskCachingMode)
- [AzureDataDiskKind](#AzureDataDiskKind)
- [AzureDiskVolumeSource](#AzureDiskVolumeSource)
- [AzureFileVolumeSource](#AzureFileVolumeSource)
- [CSIVolumeSource](#CSIVolumeSource)
- [Capabilities](#Capabilities)
- [Capability](#Capability)
- [CephFSVolumeSource](#CephFSVolumeSource)
- [CinderVolumeSource](#CinderVolumeSource)
- [ClaimResourceStatus](#ClaimResourceStatus)
- [ClusterTrustBundleProjection](#ClusterTrustBundleProjection)
- [ConditionStatus](#ConditionStatus)
- [ConfigMapEnvSource](#ConfigMapEnvSource)
- [ConfigMapKeySelector](#ConfigMapKeySelector)
- [ConfigMapProjection](#ConfigMapProjection)
- [ConfigMapVolumeSource](#ConfigMapVolumeSource)
- [Container](#Container)
- [ContainerPort](#ContainerPort)
- [ContainerResizePolicy](#ContainerResizePolicy)
- [ContainerRestartPolicy](#ContainerRestartPolicy)
- [ContainerRestartRule](#ContainerRestartRule)
- [ContainerRestartRuleAction](#ContainerRestartRuleAction)
- [ContainerRestartRuleOnExitCodes](#ContainerRestartRuleOnExitCodes)
- [ContainerRestartRuleOnExitCodesOperator](#ContainerRestartRuleOnExitCodesOperator)
- [DNSPolicy](#DNSPolicy)
- [DownwardAPIProjection](#DownwardAPIProjection)
- [DownwardAPIVolumeFile](#DownwardAPIVolumeFile)
- [DownwardAPIVolumeSource](#DownwardAPIVolumeSource)
- [EmptyDirVolumeSource](#EmptyDirVolumeSource)
- [EnvFromSource](#EnvFromSource)
- [EnvVar](#EnvVar)
- [EnvVarSource](#EnvVarSource)
- [EphemeralContainer](#EphemeralContainer)
- [EphemeralVolumeSource](#EphemeralVolumeSource)
- [ExecAction](#ExecAction)
- [FCVolumeSource](#FCVolumeSource)
- [FileKeySelector](#FileKeySelector)
- [FlexVolumeSource](#FlexVolumeSource)
- [FlockerVolumeSource](#FlockerVolumeSource)
- [GCEPersistentDiskVolumeSource](#GCEPersistentDiskVolumeSource)
- [GRPCAction](#GRPCAction)
- [GitRepoVolumeSource](#GitRepoVolumeSource)
- [GlusterfsVolumeSource](#GlusterfsVolumeSource)
- [HTTPGetAction](#HTTPGetAction)
- [HTTPHeader](#HTTPHeader)
- [HostAlias](#HostAlias)
- [HostPathType](#HostPathType)
- [HostPathVolumeSource](#HostPathVolumeSource)
- [ISCSIVolumeSource](#ISCSIVolumeSource)
- [ImageVolumeSource](#ImageVolumeSource)
- [KeyToPath](#KeyToPath)
- [Lifecycle](#Lifecycle)
- [LifecycleHandler](#LifecycleHandler)
- [LocalObjectReference](#LocalObjectReference)
- [ModifyVolumeStatus](#ModifyVolumeStatus)
- [MountPropagationMode](#MountPropagationMode)
- [NFSVolumeSource](#NFSVolumeSource)
- [NodeAffinity](#NodeAffinity)
- [NodeInclusionPolicy](#NodeInclusionPolicy)
- [NodeSelector](#NodeSelector)
- [NodeSelectorOperator](#NodeSelectorOperator)
- [NodeSelectorRequirement](#NodeSelectorRequirement)
- [NodeSelectorTerm](#NodeSelectorTerm)
- [OSName](#OSName)
- [ObjectFieldSelector](#ObjectFieldSelector)
- [PersistentVolumeAccessMode](#PersistentVolumeAccessMode)
- [PersistentVolumeClaim](#PersistentVolumeClaim)
- [PersistentVolumeClaimCondition](#PersistentVolumeClaimCondition)
- [PersistentVolumeClaimConditionType](#PersistentVolumeClaimConditionType)
- [PersistentVolumeClaimModifyVolumeStatus](#PersistentVolumeClaimModifyVolumeStatus)
- [PersistentVolumeClaimPhase](#PersistentVolumeClaimPhase)
- [PersistentVolumeClaimSpec](#PersistentVolumeClaimSpec)
- [PersistentVolumeClaimStatus](#PersistentVolumeClaimStatus)
- [PersistentVolumeClaimTemplate](#PersistentVolumeClaimTemplate)
- [PersistentVolumeClaimVolumeSource](#PersistentVolumeClaimVolumeSource)
- [PersistentVolumeMode](#PersistentVolumeMode)
- [PhotonPersistentDiskVolumeSource](#PhotonPersistentDiskVolumeSource)
- [PodAffinity](#PodAffinity)
- [PodAffinityTerm](#PodAffinityTerm)
- [PodAntiAffinity](#PodAntiAffinity)
- [PodCertificateProjection](#PodCertificateProjection)
- [PodConditionType](#PodConditionType)
- [PodDNSConfig](#PodDNSConfig)
- [PodDNSConfigOption](#PodDNSConfigOption)
- [PodFSGroupChangePolicy](#PodFSGroupChangePolicy)
- [PodOS](#PodOS)
- [PodReadinessGate](#PodReadinessGate)
- [PodResourceClaim](#PodResourceClaim)
- [PodSELinuxChangePolicy](#PodSELinuxChangePolicy)
- [PodSchedulingGate](#PodSchedulingGate)
- [PodSchedulingGroup](#PodSchedulingGroup)
- [PodSecurityContext](#PodSecurityContext)
- [PodSpec](#PodSpec)
- [PortworxVolumeSource](#PortworxVolumeSource)
- [PreemptionPolicy](#PreemptionPolicy)
- [PreferredSchedulingTerm](#PreferredSchedulingTerm)
- [Probe](#Probe)
- [ProcMountType](#ProcMountType)
- [ProjectedVolumeSource](#ProjectedVolumeSource)
- [Protocol](#Protocol)
- [PullPolicy](#PullPolicy)
- [QuobyteVolumeSource](#QuobyteVolumeSource)
- [RBDVolumeSource](#RBDVolumeSource)
- [RecursiveReadOnlyMode](#RecursiveReadOnlyMode)
- [ResourceClaim](#ResourceClaim)
- [ResourceFieldSelector](#ResourceFieldSelector)
- [ResourceList](#ResourceList)
- [ResourceName](#ResourceName)
- [ResourceRequirements](#ResourceRequirements)
- [ResourceResizeRestartPolicy](#ResourceResizeRestartPolicy)
- [RestartPolicy](#RestartPolicy)
- [SELinuxOptions](#SELinuxOptions)
- [ScaleIOVolumeSource](#ScaleIOVolumeSource)
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
- [StorageOSVolumeSource](#StorageOSVolumeSource)
- [SupplementalGroupsPolicy](#SupplementalGroupsPolicy)
- [Sysctl](#Sysctl)
- [TCPSocketAction](#TCPSocketAction)
- [TaintEffect](#TaintEffect)
- [TerminationMessagePolicy](#TerminationMessagePolicy)
- [Toleration](#Toleration)
- [TolerationOperator](#TolerationOperator)
- [TopologySpreadConstraint](#TopologySpreadConstraint)
- [TypedLocalObjectReference](#TypedLocalObjectReference)
- [TypedObjectReference](#TypedObjectReference)
- [URIScheme](#URIScheme)
- [UnsatisfiableConstraintAction](#UnsatisfiableConstraintAction)
- [Volume](#Volume)
- [VolumeDevice](#VolumeDevice)
- [VolumeMount](#VolumeMount)
- [VolumeProjection](#VolumeProjection)
- [VolumeResourceRequirements](#VolumeResourceRequirements)
- [VsphereVirtualDiskVolumeSource](#VsphereVirtualDiskVolumeSource)
- [WeightedPodAffinityTerm](#WeightedPodAffinityTerm)
- [WindowsSecurityContextOptions](#WindowsSecurityContextOptions)


## AWSElasticBlockStoreVolumeSource

Represents a Persistent Disk resource in AWS.<br /><br />An AWS EBS disk must exist before mounting to a container. The disk<br />must also be in the same AWS zone as the kubelet. An AWS EBS disk<br />can only be mounted as read/write once. AWS EBS volumes support<br />ownership management and SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `volumeID` | string | Yes | volumeID is unique ID of the persistent disk resource in AWS (Amazon EBS volume).<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore |
| `fsType` | string | No | fsType is the filesystem type of the volume that you want to mount.<br />Tip: Ensure that the filesystem type is supported by the host operating system.<br />Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore<br />TODO: how do we prevent errors in the filesystem from compromising the machine<br />+optional |
| `partition` | int32 | No | partition is the partition in the volume that you want to mount.<br />If omitted, the default is to mount by volume name.<br />Examples: For volume /dev/sda1, you specify the partition as "1".<br />Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty).<br />+optional |
| `readOnly` | bool | No | readOnly value true will force the readOnly setting in VolumeMounts.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore<br />+optional |

## Affinity

Affinity is a group of affinity scheduling rules.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `nodeAffinity` | *[NodeAffinity](./k8s-io-api-core-v1.md#NodeAffinity) | No | Describes node affinity scheduling rules for the pod.<br />+optional |
| `podAffinity` | *[PodAffinity](./k8s-io-api-core-v1.md#PodAffinity) | No | Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).<br />+optional |
| `podAntiAffinity` | *[PodAntiAffinity](./k8s-io-api-core-v1.md#PodAntiAffinity) | No | Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).<br />+optional |

## AppArmorProfile

AppArmorProfile defines a pod or container's AppArmor settings.<br />+union

| Stanza | Type | Required | Description |
|---|---|---|---|
| `type` | [AppArmorProfileType](./k8s-io-api-core-v1.md#AppArmorProfileType) | Yes | type indicates which kind of AppArmor profile will be applied.<br />Valid options are:<br />  Localhost - a profile pre-loaded on the node.<br />  RuntimeDefault - the container runtime's default profile.<br />  Unconfined - no AppArmor enforcement.<br />+unionDiscriminator |
| `localhostProfile` | *string | No | localhostProfile indicates a profile loaded on the node that should be used.<br />The profile must be preconfigured on the node to work.<br />Must match the loaded name of the profile.<br />Must be set if and only if type is "Localhost".<br />+optional |

## AppArmorProfileType

+enum



## AzureDataDiskCachingMode

+enum



## AzureDataDiskKind

+enum



## AzureDiskVolumeSource

AzureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `diskName` | string | Yes | diskName is the Name of the data disk in the blob storage |
| `diskURI` | string | Yes | diskURI is the URI of data disk in the blob storage |
| `cachingMode` | *[AzureDataDiskCachingMode](./k8s-io-api-core-v1.md#AzureDataDiskCachingMode) | No | cachingMode is the Host Caching mode: None, Read Only, Read Write.<br />+optional<br />+default=ref(AzureDataDiskCachingReadWrite) |
| `fsType` | *string | No | fsType is Filesystem type to mount.<br />Must be a filesystem type supported by the host operating system.<br />Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br />+optional<br />+default="ext4" |
| `readOnly` | *bool | No | readOnly Defaults to false (read/write). ReadOnly here will force<br />the ReadOnly setting in VolumeMounts.<br />+optional<br />+default=false |
| `kind` | *[AzureDataDiskKind](./k8s-io-api-core-v1.md#AzureDataDiskKind) | No | kind expected values are Shared: multiple blob disks per storage account  Dedicated: single blob disk per storage account  Managed: azure managed data disk (only in managed availability set). defaults to shared<br />+default=ref(AzureSharedBlobDisk) |

## AzureFileVolumeSource

AzureFile represents an Azure File Service mount on the host and bind mount to the pod.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `secretName` | string | Yes | secretName is the  name of secret that contains Azure Storage Account Name and Key |
| `shareName` | string | Yes | shareName is the azure share Name |
| `readOnly` | bool | No | readOnly defaults to false (read/write). ReadOnly here will force<br />the ReadOnly setting in VolumeMounts.<br />+optional |

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



## CephFSVolumeSource

Represents a Ceph Filesystem mount that lasts the lifetime of a pod<br />Cephfs volumes do not support ownership management or SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `monitors` | []string | No | monitors is Required: Monitors is a collection of Ceph monitors<br />More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br />+listType=atomic |
| `path` | string | No | path is Optional: Used as the mounted root, rather than the full Ceph tree, default is /<br />+optional |
| `user` | string | No | user is optional: User is the rados user name, default is admin<br />More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br />+optional |
| `secretFile` | string | No | secretFile is Optional: SecretFile is the path to key ring for User, default is /etc/ceph/user.secret<br />More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br />+optional |
| `secretRef` | *[LocalObjectReference](./k8s-io-api-core-v1.md#LocalObjectReference) | No | secretRef is Optional: SecretRef is reference to the authentication secret for User, default is empty.<br />More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br />+optional |
| `readOnly` | bool | No | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force<br />the ReadOnly setting in VolumeMounts.<br />More info: https://examples.k8s.io/volumes/cephfs/README.md#how-to-use-it<br />+optional |

## CinderVolumeSource

Represents a cinder volume resource in Openstack.<br />A Cinder volume must exist before mounting to a container.<br />The volume must also be in the same region as the kubelet.<br />Cinder volumes support ownership management and SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `volumeID` | string | Yes | volumeID used to identify the volume in cinder.<br />More info: https://examples.k8s.io/mysql-cinder-pd/README.md |
| `fsType` | string | No | fsType is the filesystem type to mount.<br />Must be a filesystem type supported by the host operating system.<br />Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br />More info: https://examples.k8s.io/mysql-cinder-pd/README.md<br />+optional |
| `readOnly` | bool | No | readOnly defaults to false (read/write). ReadOnly here will force<br />the ReadOnly setting in VolumeMounts.<br />More info: https://examples.k8s.io/mysql-cinder-pd/README.md<br />+optional |
| `secretRef` | *[LocalObjectReference](./k8s-io-api-core-v1.md#LocalObjectReference) | No | secretRef is optional: points to a secret object containing parameters used to connect<br />to OpenStack.<br />+optional |

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

## Container

A single application container that you want to run within a pod.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name of the container specified as a DNS_LABEL.<br />Each container in a pod must have a unique name (DNS_LABEL).<br />Cannot be updated. |
| `image` | string | No | Container image name.<br />More info: https://kubernetes.io/docs/concepts/containers/images<br />This field is optional to allow higher level config management to default or override<br />container images in workload controllers like Deployments and StatefulSets.<br />+optional |
| `command` | []string | No | Entrypoint array. Not executed within a shell.<br />The container image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional<br />+listType=atomic |
| `args` | []string | No | Arguments to the entrypoint.<br />The container image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional<br />+listType=atomic |
| `workingDir` | string | No | Container's working directory.<br />If not specified, the container runtime's default will be used, which<br />might be configured in the container image.<br />Cannot be updated.<br />+optional |
| `ports` | [][ContainerPort](./k8s-io-api-core-v1.md#ContainerPort) | No | List of ports to expose from the container. Not specifying a port here<br />DOES NOT prevent that port from being exposed. Any port which is<br />listening on the default "0.0.0.0" address inside a container will be<br />accessible from the network.<br />Modifying this array with strategic merge patch may corrupt the data.<br />For more information See https://github.com/kubernetes/kubernetes/issues/108255.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=containerPort<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=containerPort<br />+listMapKey=protocol |
| `envFrom` | [][EnvFromSource](./k8s-io-api-core-v1.md#EnvFromSource) | No | List of sources to populate environment variables in the container.<br />The keys defined within a source may consist of any printable ASCII characters except '='.<br />When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated.<br />+optional<br />+listType=atomic |
| `env` | [][EnvVar](./k8s-io-api-core-v1.md#EnvVar) | No | List of environment variables to set in the container.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=name<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=name |
| `resources` | [ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | No | Compute Resources required by this container.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br />+optional |
| `resizePolicy` | [][ContainerResizePolicy](./k8s-io-api-core-v1.md#ContainerResizePolicy) | No | Resources resize policy for the container.<br />This field cannot be set on ephemeral containers.<br />+featureGate=InPlacePodVerticalScaling<br />+optional<br />+listType=atomic |
| `restartPolicy` | *[ContainerRestartPolicy](./k8s-io-api-core-v1.md#ContainerRestartPolicy) | No | RestartPolicy defines the restart behavior of individual containers in a pod.<br />This overrides the pod-level restart policy. When this field is not specified,<br />the restart behavior is defined by the Pod's restart policy and the container type.<br />Additionally, setting the RestartPolicy as "Always" for the init container will<br />have the following effect:<br />this init container will be continually restarted on<br />exit until all regular containers have terminated. Once all regular<br />containers have completed, all init containers with restartPolicy "Always"<br />will be shut down. This lifecycle differs from normal init containers and<br />is often referred to as a "sidecar" container. Although this init<br />container still starts in the init container sequence, it does not wait<br />for the container to complete before proceeding to the next init<br />container. Instead, the next init container starts immediately after this<br />init container is started, or after any startupProbe has successfully<br />completed.<br />+optional |
| `restartPolicyRules` | [][ContainerRestartRule](./k8s-io-api-core-v1.md#ContainerRestartRule) | No | Represents a list of rules to be checked to determine if the<br />container should be restarted on exit. The rules are evaluated in<br />order. Once a rule matches a container exit condition, the remaining<br />rules are ignored. If no rule matches the container exit condition,<br />the Container-level restart policy determines the whether the container<br />is restarted or not. Constraints on the rules:<br />- At most 20 rules are allowed.<br />- Rules can have the same action.<br />- Identical rules are not forbidden in validations.<br />When rules are specified, container MUST set RestartPolicy explicitly<br />even it if matches the Pod's RestartPolicy.<br />+featureGate=ContainerRestartRules<br />+optional<br />+listType=atomic |
| `volumeMounts` | [][VolumeMount](./k8s-io-api-core-v1.md#VolumeMount) | No | Pod volumes to mount into the container's filesystem.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=mountPath<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=mountPath |
| `volumeDevices` | [][VolumeDevice](./k8s-io-api-core-v1.md#VolumeDevice) | No | volumeDevices is the list of block devices to be used by the container.<br />+patchMergeKey=devicePath<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=devicePath<br />+optional |
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

## ContainerPort

ContainerPort represents a network port in a single container.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | If specified, this must be an IANA_SVC_NAME and unique within the pod. Each<br />named port in a pod must have a unique name. Name for the port that can be<br />referred to by services.<br />+optional |
| `hostPort` | int32 | No | Number of port to expose on the host.<br />If specified, this must be a valid port number, 0 < x < 65536.<br />If HostNetwork is specified, this must match ContainerPort.<br />Most containers do not need this.<br />+optional |
| `containerPort` | int32 | Yes | Number of port to expose on the pod's IP address.<br />This must be a valid port number, 0 < x < 65536. |
| `protocol` | [Protocol](./k8s-io-api-core-v1.md#Protocol) | No | Protocol for port. Must be UDP, TCP, or SCTP.<br />Defaults to "TCP".<br />+optional<br />+default="TCP" |
| `hostIP` | string | No | What host IP to bind the external port to.<br />+optional |

## ContainerResizePolicy

ContainerResizePolicy represents resource resize policy for the container.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `resourceName` | [ResourceName](./k8s-io-api-core-v1.md#ResourceName) | Yes | Name of the resource to which this resource resize policy applies.<br />Supported values: cpu, memory. |
| `restartPolicy` | [ResourceResizeRestartPolicy](./k8s-io-api-core-v1.md#ResourceResizeRestartPolicy) | Yes | Restart policy to apply when specified resource is resized.<br />If not specified, it defaults to NotRequired. |

## ContainerRestartPolicy

ContainerRestartPolicy is the restart policy for a single container.<br />The only allowed values are "Always", "Never", and "OnFailure".



## ContainerRestartRule

ContainerRestartRule describes how a container exit is handled.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `action` | [ContainerRestartRuleAction](./k8s-io-api-core-v1.md#ContainerRestartRuleAction) | No | Specifies the action taken on a container exit if the requirements<br />are satisfied. The only possible value is "Restart" to restart the<br />container.<br />+required |
| `exitCodes` | *[ContainerRestartRuleOnExitCodes](./k8s-io-api-core-v1.md#ContainerRestartRuleOnExitCodes) | No | Represents the exit codes to check on container exits.<br />+optional<br />+oneOf=when |

## ContainerRestartRuleAction

ContainerRestartRuleAction describes the action to take when the<br />container exits.



## ContainerRestartRuleOnExitCodes

ContainerRestartRuleOnExitCodes describes the condition<br />for handling an exited container based on its exit codes.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `operator` | [ContainerRestartRuleOnExitCodesOperator](./k8s-io-api-core-v1.md#ContainerRestartRuleOnExitCodesOperator) | No | Represents the relationship between the container exit code(s) and the<br />specified values. Possible values are:<br />- In: the requirement is satisfied if the container exit code is in the<br />  set of specified values.<br />- NotIn: the requirement is satisfied if the container exit code is<br />  not in the set of specified values.<br />+required |
| `values` | []int32 | No | Specifies the set of values to check for container exit codes.<br />At most 255 elements are allowed.<br />+optional<br />+listType=set |

## ContainerRestartRuleOnExitCodesOperator

ContainerRestartRuleOnExitCodesOperator describes the operator<br />to take for the exit codes.



## DNSPolicy

DNSPolicy defines how a pod's DNS will be configured.<br />+enum



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

## DownwardAPIVolumeSource

DownwardAPIVolumeSource represents a volume containing downward API info.<br />Downward API volumes support ownership management and SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `items` | [][DownwardAPIVolumeFile](./k8s-io-api-core-v1.md#DownwardAPIVolumeFile) | No | Items is a list of downward API volume file<br />+optional<br />+listType=atomic |
| `defaultMode` | *int32 | No | Optional: mode bits to use on created files by default. Must be a<br />Optional: mode bits used to set permissions on created files by default.<br />Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511.<br />YAML accepts both octal and decimal values, JSON requires decimal values for mode bits.<br />Defaults to 0644.<br />Directories within the path are not affected by this setting.<br />This might be in conflict with other options that affect the file<br />mode, like fsGroup, and the result can be other mode bits set.<br />+optional |

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

## EphemeralContainer

An EphemeralContainer is a temporary container that you may add to an existing Pod for<br />user-initiated activities such as debugging. Ephemeral containers have no resource or<br />scheduling guarantees, and they will not be restarted when they exit or when a Pod is<br />removed or restarted. The kubelet may evict a Pod if an ephemeral container causes the<br />Pod to exceed its resource allocation.<br /><br />To add an ephemeral container, use the ephemeralcontainers subresource of an existing<br />Pod. Ephemeral containers may not be removed or restarted.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name of the ephemeral container specified as a DNS_LABEL.<br />This name must be unique among all containers, init containers and ephemeral containers. |
| `image` | string | No | Container image name.<br />More info: https://kubernetes.io/docs/concepts/containers/images |
| `command` | []string | No | Entrypoint array. Not executed within a shell.<br />The image's ENTRYPOINT is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional<br />+listType=atomic |
| `args` | []string | No | Arguments to the entrypoint.<br />The image's CMD is used if this is not provided.<br />Variable references $(VAR_NAME) are expanded using the container's environment. If a variable<br />cannot be resolved, the reference in the input string will be unchanged. Double $$ are reduced<br />to a single $, which allows for escaping the $(VAR_NAME) syntax: i.e. "$$(VAR_NAME)" will<br />produce the string literal "$(VAR_NAME)". Escaped references will never be expanded, regardless<br />of whether the variable exists or not. Cannot be updated.<br />More info: https://kubernetes.io/docs/tasks/inject-data-application/define-command-argument-container/#running-a-command-in-a-shell<br />+optional<br />+listType=atomic |
| `workingDir` | string | No | Container's working directory.<br />If not specified, the container runtime's default will be used, which<br />might be configured in the container image.<br />Cannot be updated.<br />+optional |
| `ports` | [][ContainerPort](./k8s-io-api-core-v1.md#ContainerPort) | No | Ports are not allowed for ephemeral containers.<br />+optional<br />+patchMergeKey=containerPort<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=containerPort<br />+listMapKey=protocol |
| `envFrom` | [][EnvFromSource](./k8s-io-api-core-v1.md#EnvFromSource) | No | List of sources to populate environment variables in the container.<br />The keys defined within a source may consist of any printable ASCII characters except '='.<br />When a key exists in multiple<br />sources, the value associated with the last source will take precedence.<br />Values defined by an Env with a duplicate key will take precedence.<br />Cannot be updated.<br />+optional<br />+listType=atomic |
| `env` | [][EnvVar](./k8s-io-api-core-v1.md#EnvVar) | No | List of environment variables to set in the container.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=name<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=name |
| `resources` | [ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | No | Resources are not allowed for ephemeral containers. Ephemeral containers use spare resources<br />already allocated to the pod.<br />+optional |
| `resizePolicy` | [][ContainerResizePolicy](./k8s-io-api-core-v1.md#ContainerResizePolicy) | No | Resources resize policy for the container.<br />+featureGate=InPlacePodVerticalScaling<br />+optional<br />+listType=atomic |
| `restartPolicy` | *[ContainerRestartPolicy](./k8s-io-api-core-v1.md#ContainerRestartPolicy) | No | Restart policy for the container to manage the restart behavior of each<br />container within a pod.<br />You cannot set this field on ephemeral containers.<br />+optional |
| `restartPolicyRules` | [][ContainerRestartRule](./k8s-io-api-core-v1.md#ContainerRestartRule) | No | Represents a list of rules to be checked to determine if the<br />container should be restarted on exit. You cannot set this field on<br />ephemeral containers.<br />+featureGate=ContainerRestartRules<br />+optional<br />+listType=atomic |
| `volumeMounts` | [][VolumeMount](./k8s-io-api-core-v1.md#VolumeMount) | No | Pod volumes to mount into the container's filesystem. Subpath mounts are not allowed for ephemeral containers.<br />Cannot be updated.<br />+optional<br />+patchMergeKey=mountPath<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=mountPath |
| `volumeDevices` | [][VolumeDevice](./k8s-io-api-core-v1.md#VolumeDevice) | No | volumeDevices is the list of block devices to be used by the container.<br />+patchMergeKey=devicePath<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=devicePath<br />+optional |
| `livenessProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | Probes are not allowed for ephemeral containers.<br />+optional |
| `readinessProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | Probes are not allowed for ephemeral containers.<br />+optional |
| `startupProbe` | *[Probe](./k8s-io-api-core-v1.md#Probe) | No | Probes are not allowed for ephemeral containers.<br />+optional |
| `lifecycle` | *[Lifecycle](./k8s-io-api-core-v1.md#Lifecycle) | No | Lifecycle is not allowed for ephemeral containers.<br />+optional |
| `terminationMessagePath` | string | No | Optional: Path at which the file to which the container's termination message<br />will be written is mounted into the container's filesystem.<br />Message written is intended to be brief final status, such as an assertion failure message.<br />Will be truncated by the node if greater than 4096 bytes. The total message length across<br />all containers will be limited to 12kb.<br />Defaults to /dev/termination-log.<br />Cannot be updated.<br />+optional |
| `terminationMessagePolicy` | [TerminationMessagePolicy](./k8s-io-api-core-v1.md#TerminationMessagePolicy) | No | Indicate how the termination message should be populated. File will use the contents of<br />terminationMessagePath to populate the container status message on both success and failure.<br />FallbackToLogsOnError will use the last chunk of container log output if the termination<br />message file is empty and the container exited with an error.<br />The log output is limited to 2048 bytes or 80 lines, whichever is smaller.<br />Defaults to File.<br />Cannot be updated.<br />+optional |
| `imagePullPolicy` | [PullPolicy](./k8s-io-api-core-v1.md#PullPolicy) | No | Image pull policy.<br />One of Always, Never, IfNotPresent.<br />Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/containers/images#updating-images<br />+optional |
| `securityContext` | *[SecurityContext](./k8s-io-api-core-v1.md#SecurityContext) | No | Optional: SecurityContext defines the security options the ephemeral container should be run with.<br />If set, the fields of SecurityContext override the equivalent fields of PodSecurityContext.<br />+optional |
| `stdin` | bool | No | Whether this container should allocate a buffer for stdin in the container runtime. If this<br />is not set, reads from stdin in the container will always result in EOF.<br />Default is false.<br />+optional |
| `stdinOnce` | bool | No | Whether the container runtime should close the stdin channel after it has been opened by<br />a single attach. When stdin is true the stdin stream will remain open across multiple attach<br />sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the<br />first client attaches to stdin, and then remains open and accepts data until the client disconnects,<br />at which time stdin is closed and remains closed until the container is restarted. If this<br />flag is false, a container processes that reads from stdin will never receive an EOF.<br />Default is false<br />+optional |
| `tty` | bool | No | Whether this container should allocate a TTY for itself, also requires 'stdin' to be true.<br />Default is false.<br />+optional |
| `targetContainerName` | string | No | If set, the name of the container from PodSpec that this ephemeral container targets.<br />The ephemeral container will be run in the namespaces (IPC, PID, etc) of this container.<br />If not set then the ephemeral container uses the namespaces configured in the Pod spec.<br /><br />The container runtime must implement support for this feature. If the runtime does not<br />support namespace targeting then the result of setting this field is undefined.<br />+optional |

## EphemeralVolumeSource

Represents an ephemeral volume that is handled by a normal storage driver.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `volumeClaimTemplate` | *[PersistentVolumeClaimTemplate](./k8s-io-api-core-v1.md#PersistentVolumeClaimTemplate) | No | Will be used to create a stand-alone PVC to provision the volume.<br />The pod in which this EphemeralVolumeSource is embedded will be the<br />owner of the PVC, i.e. the PVC will be deleted together with the<br />pod.  The name of the PVC will be `<pod name>-<volume name>` where<br />`<volume name>` is the name from the `PodSpec.Volumes` array<br />entry. Pod validation will reject the pod if the concatenated name<br />is not valid for a PVC (for example, too long).<br /><br />An existing PVC with that name that is not owned by the pod<br />will *not* be used for the pod to avoid using an unrelated<br />volume by mistake. Starting the pod is then blocked until<br />the unrelated PVC is removed. If such a pre-created PVC is<br />meant to be used by the pod, the PVC has to updated with an<br />owner reference to the pod once the pod exists. Normally<br />this should not be necessary, but it may be useful when<br />manually reconstructing a broken cluster.<br /><br />This field is read-only and no changes will be made by Kubernetes<br />to the PVC after it has been created.<br /><br />Required, must not be nil. |

## ExecAction

ExecAction describes a "run in container" action.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `command` | []string | No | Command is the command line to execute inside the container, the working directory for the<br />command  is root ('/') in the container's filesystem. The command is simply exec'd, it is<br />not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use<br />a shell, you need to explicitly call out to that shell.<br />Exit status of 0 is treated as live/healthy and non-zero is unhealthy.<br />+optional<br />+listType=atomic |

## FCVolumeSource

Represents a Fibre Channel volume.<br />Fibre Channel volumes can only be mounted as read/write once.<br />Fibre Channel volumes support ownership management and SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `targetWWNs` | []string | No | targetWWNs is Optional: FC target worldwide names (WWNs)<br />+optional<br />+listType=atomic |
| `lun` | *int32 | No | lun is Optional: FC target lun number<br />+optional |
| `fsType` | string | No | fsType is the filesystem type to mount.<br />Must be a filesystem type supported by the host operating system.<br />Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br />TODO: how do we prevent errors in the filesystem from compromising the machine<br />+optional |
| `readOnly` | bool | No | readOnly is Optional: Defaults to false (read/write). ReadOnly here will force<br />the ReadOnly setting in VolumeMounts.<br />+optional |
| `wwids` | []string | No | wwids Optional: FC volume world wide identifiers (wwids)<br />Either wwids or combination of targetWWNs and lun must be set, but not both simultaneously.<br />+optional<br />+listType=atomic |

## FileKeySelector

FileKeySelector selects a key of the env file.<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `volumeName` | string | Yes | The name of the volume mount containing the env file.<br />+required |
| `path` | string | Yes | The path within the volume from which to select the file.<br />Must be relative and may not contain the '..' path or start with '..'.<br />+required |
| `key` | string | Yes | The key within the env file. An invalid key will prevent the pod from starting.<br />The keys defined within a source may consist of any printable ASCII characters except '='.<br />During Alpha stage of the EnvFiles feature gate, the key size is limited to 128 characters.<br />+required |
| `optional` | *bool | No | Specify whether the file or its key must be defined. If the file or key<br />does not exist, then the env var is not published.<br />If optional is set to true and the specified key does not exist,<br />the environment variable will not be set in the Pod's containers.<br /><br />If optional is set to false and the specified key does not exist,<br />an error will be returned during Pod creation.<br />+optional<br />+default=false |

## FlexVolumeSource

FlexVolume represents a generic volume resource that is<br />provisioned/attached using an exec based plugin.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `driver` | string | Yes | driver is the name of the driver to use for this volume. |
| `fsType` | string | No | fsType is the filesystem type to mount.<br />Must be a filesystem type supported by the host operating system.<br />Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script.<br />+optional |
| `secretRef` | *[LocalObjectReference](./k8s-io-api-core-v1.md#LocalObjectReference) | No | secretRef is Optional: secretRef is reference to the secret object containing<br />sensitive information to pass to the plugin scripts. This may be<br />empty if no secret object is specified. If the secret object<br />contains more than one secret, all secrets are passed to the plugin<br />scripts.<br />+optional |
| `readOnly` | bool | No | readOnly is Optional: defaults to false (read/write). ReadOnly here will force<br />the ReadOnly setting in VolumeMounts.<br />+optional |
| `options` | map[string]string | No | options is Optional: this field holds extra command options if any.<br />+optional |

## FlockerVolumeSource

Represents a Flocker volume mounted by the Flocker agent.<br />One and only one of datasetName and datasetUUID should be set.<br />Flocker volumes do not support ownership management or SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `datasetName` | string | No | datasetName is Name of the dataset stored as metadata -> name on the dataset for Flocker<br />should be considered as deprecated<br />+optional |
| `datasetUUID` | string | No | datasetUUID is the UUID of the dataset. This is unique identifier of a Flocker dataset<br />+optional |

## GCEPersistentDiskVolumeSource

Represents a Persistent Disk resource in Google Compute Engine.<br /><br />A GCE PD must exist before mounting to a container. The disk must<br />also be in the same GCE project and zone as the kubelet. A GCE PD<br />can only be mounted as read/write once or read-only many times. GCE<br />PDs support ownership management and SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `pdName` | string | Yes | pdName is unique name of the PD resource in GCE. Used to identify the disk in GCE.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk |
| `fsType` | string | No | fsType is filesystem type of the volume that you want to mount.<br />Tip: Ensure that the filesystem type is supported by the host operating system.<br />Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk<br />TODO: how do we prevent errors in the filesystem from compromising the machine<br />+optional |
| `partition` | int32 | No | partition is the partition in the volume that you want to mount.<br />If omitted, the default is to mount by volume name.<br />Examples: For volume /dev/sda1, you specify the partition as "1".<br />Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty).<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk<br />+optional |
| `readOnly` | bool | No | readOnly here will force the ReadOnly setting in VolumeMounts.<br />Defaults to false.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk<br />+optional |

## GRPCAction

GRPCAction specifies an action involving a GRPC service.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `port` | int32 | Yes | Port number of the gRPC service. Number must be in the range 1 to 65535. |
| `service` | *string | No | Service is the name of the service to place in the gRPC HealthCheckRequest<br />(see https://github.com/grpc/grpc/blob/master/doc/health-checking.md).<br /><br />If this is not specified, the default behavior is defined by gRPC.<br />+optional<br />+default="" |

## GitRepoVolumeSource

Represents a volume that is populated with the contents of a git repository.<br />Git repo volumes do not support ownership management.<br />Git repo volumes support SELinux relabeling.<br /><br />DEPRECATED: GitRepo is deprecated. To provision a container with a git repo, mount an<br />EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir<br />into the Pod's container.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `repository` | string | Yes | repository is the URL |
| `revision` | string | No | revision is the commit hash for the specified revision.<br />+optional |
| `directory` | string | No | directory is the target directory name.<br />Must not contain or start with '..'.  If '.' is supplied, the volume directory will be the<br />git repository.  Otherwise, if specified, the volume will contain the git repository in<br />the subdirectory with the given name.<br />+optional |

## GlusterfsVolumeSource

Represents a Glusterfs mount that lasts the lifetime of a pod.<br />Glusterfs volumes do not support ownership management or SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `endpoints` | string | Yes | endpoints is the endpoint name that details Glusterfs topology. |
| `path` | string | Yes | path is the Glusterfs volume path.<br />More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod |
| `readOnly` | bool | No | readOnly here will force the Glusterfs volume to be mounted with read-only permissions.<br />Defaults to false.<br />More info: https://examples.k8s.io/volumes/glusterfs/README.md#create-a-pod<br />+optional |

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

## HostAlias

HostAlias holds the mapping between IP and hostnames that will be injected as an entry in the<br />pod's hosts file.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `ip` | string | Yes | IP address of the host file entry.<br />+required |
| `hostnames` | []string | No | Hostnames for the above IP address.<br />+listType=atomic |

## HostPathType

+enum



## HostPathVolumeSource

Represents a host path mapped into a pod.<br />Host path volumes do not support ownership management or SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `path` | string | Yes | path of the directory on the host.<br />If the path is a symlink, it will follow the link to the real path.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath |
| `type` | *[HostPathType](./k8s-io-api-core-v1.md#HostPathType) | No | type for HostPath Volume<br />Defaults to ""<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath<br />+optional |

## ISCSIVolumeSource

Represents an ISCSI disk.<br />ISCSI volumes can only be mounted as read/write once.<br />ISCSI volumes support ownership management and SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `targetPortal` | string | Yes | targetPortal is iSCSI Target Portal. The Portal is either an IP or ip_addr:port if the port<br />is other than default (typically TCP ports 860 and 3260). |
| `iqn` | string | Yes | iqn is the target iSCSI Qualified Name. |
| `lun` | int32 | Yes | lun represents iSCSI Target Lun number. |
| `iscsiInterface` | string | No | iscsiInterface is the interface Name that uses an iSCSI transport.<br />Defaults to 'default' (tcp).<br />+optional<br />+default="default" |
| `fsType` | string | No | fsType is the filesystem type of the volume that you want to mount.<br />Tip: Ensure that the filesystem type is supported by the host operating system.<br />Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#iscsi<br />TODO: how do we prevent errors in the filesystem from compromising the machine<br />+optional |
| `readOnly` | bool | No | readOnly here will force the ReadOnly setting in VolumeMounts.<br />Defaults to false.<br />+optional |
| `portals` | []string | No | portals is the iSCSI Target Portal List. The portal is either an IP or ip_addr:port if the port<br />is other than default (typically TCP ports 860 and 3260).<br />+optional<br />+listType=atomic |
| `chapAuthDiscovery` | bool | No | chapAuthDiscovery defines whether support iSCSI Discovery CHAP authentication<br />+optional |
| `chapAuthSession` | bool | No | chapAuthSession defines whether support iSCSI Session CHAP authentication<br />+optional |
| `secretRef` | *[LocalObjectReference](./k8s-io-api-core-v1.md#LocalObjectReference) | No | secretRef is the CHAP Secret for iSCSI target and initiator authentication<br />+optional |
| `initiatorName` | *string | No | initiatorName is the custom iSCSI Initiator Name.<br />If initiatorName is specified with iscsiInterface simultaneously, new iSCSI interface<br /><target portal>:<volume name> will be created for the connection.<br />+optional |

## ImageVolumeSource

ImageVolumeSource represents a image volume resource.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `reference` | string | No | Required: Image or artifact reference to be used.<br />Behaves in the same way as pod.spec.containers[*].image.<br />Pull secrets will be assembled in the same way as for the container image by looking up node credentials, SA image pull secrets, and pod spec image pull secrets.<br />More info: https://kubernetes.io/docs/concepts/containers/images<br />This field is optional to allow higher level config management to default or override<br />container images in workload controllers like Deployments and StatefulSets.<br />+optional |
| `pullPolicy` | [PullPolicy](./k8s-io-api-core-v1.md#PullPolicy) | No | Policy for pulling OCI objects. Possible values are:<br />Always: the kubelet always attempts to pull the reference. Container creation will fail If the pull fails.<br />Never: the kubelet never pulls the reference and only uses a local image or artifact. Container creation will fail if the reference isn't present.<br />IfNotPresent: the kubelet pulls if the reference isn't already present on disk. Container creation will fail if the reference isn't present and the pull fails.<br />Defaults to Always if :latest tag is specified, or IfNotPresent otherwise.<br />+optional |

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



## NFSVolumeSource

Represents an NFS mount that lasts the lifetime of a pod.<br />NFS volumes do not support ownership management or SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `server` | string | Yes | server is the hostname or IP address of the NFS server.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs |
| `path` | string | Yes | path that is exported by the NFS server.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs |
| `readOnly` | bool | No | readOnly here will force the NFS export to be mounted with read-only permissions.<br />Defaults to false.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs<br />+optional |

## NodeAffinity

Node affinity is a group of node affinity scheduling rules.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `requiredDuringSchedulingIgnoredDuringExecution` | *[NodeSelector](./k8s-io-api-core-v1.md#NodeSelector) | No | If the affinity requirements specified by this field are not met at<br />scheduling time, the pod will not be scheduled onto the node.<br />If the affinity requirements specified by this field cease to be met<br />at some point during pod execution (e.g. due to an update), the system<br />may or may not try to eventually evict the pod from its node.<br />+optional |
| `preferredDuringSchedulingIgnoredDuringExecution` | [][PreferredSchedulingTerm](./k8s-io-api-core-v1.md#PreferredSchedulingTerm) | No | The scheduler will prefer to schedule pods to nodes that satisfy<br />the affinity expressions specified by this field, but it may choose<br />a node that violates one or more of the expressions. The node that is<br />most preferred is the one with the greatest sum of weights, i.e.<br />for each node that meets all of the scheduling requirements (resource<br />request, requiredDuringScheduling affinity expressions, etc.),<br />compute a sum by iterating through the elements of this field and adding<br />"weight" to the sum if the node matches the corresponding matchExpressions; the<br />node(s) with the highest sum are the most preferred.<br />+optional<br />+listType=atomic |

## NodeInclusionPolicy

NodeInclusionPolicy defines the type of node inclusion policy<br />+enum



## NodeSelector

A node selector represents the union of the results of one or more label queries<br />over a set of nodes; that is, it represents the OR of the selectors represented<br />by the node selector terms.<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `nodeSelectorTerms` | [][NodeSelectorTerm](./k8s-io-api-core-v1.md#NodeSelectorTerm) | No | Required. A list of node selector terms. The terms are ORed.<br />+listType=atomic |

## NodeSelectorOperator

A node selector operator is the set of operators that can be used in<br />a node selector requirement.<br />+enum



## NodeSelectorRequirement

A node selector requirement is a selector that contains values, a key, and an operator<br />that relates the key and values.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `key` | string | Yes | The label key that the selector applies to. |
| `operator` | [NodeSelectorOperator](./k8s-io-api-core-v1.md#NodeSelectorOperator) | Yes | Represents a key's relationship to a set of values.<br />Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt. |
| `values` | []string | No | An array of string values. If the operator is In or NotIn,<br />the values array must be non-empty. If the operator is Exists or DoesNotExist,<br />the values array must be empty. If the operator is Gt or Lt, the values<br />array must have a single element, which will be interpreted as an integer.<br />This array is replaced during a strategic merge patch.<br />+optional<br />+listType=atomic |

## NodeSelectorTerm

A null or empty node selector term matches no objects. The requirements of<br />them are ANDed.<br />The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `matchExpressions` | [][NodeSelectorRequirement](./k8s-io-api-core-v1.md#NodeSelectorRequirement) | No | A list of node selector requirements by node's labels.<br />+optional<br />+listType=atomic |
| `matchFields` | [][NodeSelectorRequirement](./k8s-io-api-core-v1.md#NodeSelectorRequirement) | No | A list of node selector requirements by node's fields.<br />+optional<br />+listType=atomic |

## OSName

OSName is the set of OS'es that can be used in OS.



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

## PersistentVolumeClaimTemplate

PersistentVolumeClaimTemplate is used to produce<br />PersistentVolumeClaim objects as part of an EphemeralVolumeSource.

| Stanza | Type | Required | Description |
|---|---|---|---|
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
| `spec` | [PersistentVolumeClaimSpec](./k8s-io-api-core-v1.md#PersistentVolumeClaimSpec) | Yes | The specification for the PersistentVolumeClaim. The entire content is<br />copied unchanged into the PVC that gets created from this<br />template. The same fields as in a PersistentVolumeClaim<br />are also valid here. |

## PersistentVolumeClaimVolumeSource

PersistentVolumeClaimVolumeSource references the user's PVC in the same namespace.<br />This volume finds the bound PV and mounts that volume for the pod. A<br />PersistentVolumeClaimVolumeSource is, essentially, a wrapper around another<br />type of volume that is owned by someone else (the system).

| Stanza | Type | Required | Description |
|---|---|---|---|
| `claimName` | string | Yes | claimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume.<br />More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims |
| `readOnly` | bool | No | readOnly Will force the ReadOnly setting in VolumeMounts.<br />Default false.<br />+optional |

## PersistentVolumeMode

PersistentVolumeMode describes how a volume is intended to be consumed, either Block or Filesystem.<br />+enum



## PhotonPersistentDiskVolumeSource

Represents a Photon Controller persistent disk resource.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `pdID` | string | Yes | pdID is the ID that identifies Photon Controller persistent disk |
| `fsType` | string | No | fsType is the filesystem type to mount.<br />Must be a filesystem type supported by the host operating system.<br />Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. |

## PodAffinity

Pod affinity is a group of inter pod affinity scheduling rules.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `requiredDuringSchedulingIgnoredDuringExecution` | [][PodAffinityTerm](./k8s-io-api-core-v1.md#PodAffinityTerm) | No | If the affinity requirements specified by this field are not met at<br />scheduling time, the pod will not be scheduled onto the node.<br />If the affinity requirements specified by this field cease to be met<br />at some point during pod execution (e.g. due to a pod label update), the<br />system may or may not try to eventually evict the pod from its node.<br />When there are multiple elements, the lists of nodes corresponding to each<br />podAffinityTerm are intersected, i.e. all terms must be satisfied.<br />+optional<br />+listType=atomic |
| `preferredDuringSchedulingIgnoredDuringExecution` | [][WeightedPodAffinityTerm](./k8s-io-api-core-v1.md#WeightedPodAffinityTerm) | No | The scheduler will prefer to schedule pods to nodes that satisfy<br />the affinity expressions specified by this field, but it may choose<br />a node that violates one or more of the expressions. The node that is<br />most preferred is the one with the greatest sum of weights, i.e.<br />for each node that meets all of the scheduling requirements (resource<br />request, requiredDuringScheduling affinity expressions, etc.),<br />compute a sum by iterating through the elements of this field and adding<br />"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the<br />node(s) with the highest sum are the most preferred.<br />+optional<br />+listType=atomic |

## PodAffinityTerm

Defines a set of pods (namely those matching the labelSelector<br />relative to the given namespace(s)) that this pod should be<br />co-located (affinity) or not co-located (anti-affinity) with,<br />where co-located is defined as running on a node whose value of<br />the label with key <topologyKey> matches that of any node on which<br />a pod of the set of pods is running

| Stanza | Type | Required | Description |
|---|---|---|---|
| `labelSelector` | *[LabelSelector](./k8s-io-apimachinery-pkg-apis-meta-v1.md#LabelSelector) | No | A label query over a set of resources, in this case pods.<br />If it's null, this PodAffinityTerm matches with no Pods.<br />+optional |
| `namespaces` | []string | No | namespaces specifies a static list of namespace names that the term applies to.<br />The term is applied to the union of the namespaces listed in this field<br />and the ones selected by namespaceSelector.<br />null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br />+optional<br />+listType=atomic |
| `topologyKey` | string | Yes | This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching<br />the labelSelector in the specified namespaces, where co-located is defined as running on a node<br />whose value of the label with key topologyKey matches that of any node on which any of the<br />selected pods is running.<br />Empty topologyKey is not allowed. |
| `namespaceSelector` | *[LabelSelector](./k8s-io-apimachinery-pkg-apis-meta-v1.md#LabelSelector) | No | A label query over the set of namespaces that the term applies to.<br />The term is applied to the union of the namespaces selected by this field<br />and the ones listed in the namespaces field.<br />null selector and null or empty namespaces list means "this pod's namespace".<br />An empty selector ({}) matches all namespaces.<br />+optional |
| `matchLabelKeys` | []string | No | MatchLabelKeys is a set of pod label keys to select which pods will<br />be taken into consideration. The keys are used to lookup values from the<br />incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`<br />to select the group of existing pods which pods will be taken into consideration<br />for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming<br />pod labels will be ignored. The default value is empty.<br />The same key is forbidden to exist in both matchLabelKeys and labelSelector.<br />Also, matchLabelKeys cannot be set when labelSelector isn't set.<br /><br />+listType=atomic<br />+optional |
| `mismatchLabelKeys` | []string | No | MismatchLabelKeys is a set of pod label keys to select which pods will<br />be taken into consideration. The keys are used to lookup values from the<br />incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`<br />to select the group of existing pods which pods will be taken into consideration<br />for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming<br />pod labels will be ignored. The default value is empty.<br />The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.<br />Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br /><br />+listType=atomic<br />+optional |

## PodAntiAffinity

Pod anti affinity is a group of inter pod anti affinity scheduling rules.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `requiredDuringSchedulingIgnoredDuringExecution` | [][PodAffinityTerm](./k8s-io-api-core-v1.md#PodAffinityTerm) | No | If the anti-affinity requirements specified by this field are not met at<br />scheduling time, the pod will not be scheduled onto the node.<br />If the anti-affinity requirements specified by this field cease to be met<br />at some point during pod execution (e.g. due to a pod label update), the<br />system may or may not try to eventually evict the pod from its node.<br />When there are multiple elements, the lists of nodes corresponding to each<br />podAffinityTerm are intersected, i.e. all terms must be satisfied.<br />+optional<br />+listType=atomic |
| `preferredDuringSchedulingIgnoredDuringExecution` | [][WeightedPodAffinityTerm](./k8s-io-api-core-v1.md#WeightedPodAffinityTerm) | No | The scheduler will prefer to schedule pods to nodes that satisfy<br />the anti-affinity expressions specified by this field, but it may choose<br />a node that violates one or more of the expressions. The node that is<br />most preferred is the one with the greatest sum of weights, i.e.<br />for each node that meets all of the scheduling requirements (resource<br />request, requiredDuringScheduling anti-affinity expressions, etc.),<br />compute a sum by iterating through the elements of this field and subtracting<br />"weight" from the sum if the node has pods which matches the corresponding podAffinityTerm; the<br />node(s) with the highest sum are the most preferred.<br />+optional<br />+listType=atomic |

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

## PodConditionType

PodConditionType is a valid value for PodCondition.Type



## PodDNSConfig

PodDNSConfig defines the DNS parameters of a pod in addition to<br />those generated from DNSPolicy.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `nameservers` | []string | No | A list of DNS name server IP addresses.<br />This will be appended to the base nameservers generated from DNSPolicy.<br />Duplicated nameservers will be removed.<br />+optional<br />+listType=atomic |
| `searches` | []string | No | A list of DNS search domains for host-name lookup.<br />This will be appended to the base search paths generated from DNSPolicy.<br />Duplicated search paths will be removed.<br />+optional<br />+listType=atomic |
| `options` | [][PodDNSConfigOption](./k8s-io-api-core-v1.md#PodDNSConfigOption) | No | A list of DNS resolver options.<br />This will be merged with the base options generated from DNSPolicy.<br />Duplicated entries will be removed. Resolution options given in Options<br />will override those that appear in the base DNSPolicy.<br />+optional<br />+listType=atomic |

## PodDNSConfigOption

PodDNSConfigOption defines DNS resolver options of a pod.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name is this DNS resolver option's name.<br />Required. |
| `value` | *string | No | Value is this DNS resolver option's value.<br />+optional |

## PodFSGroupChangePolicy

PodFSGroupChangePolicy holds policies that will be used for applying fsGroup to a volume<br />when volume is mounted.<br />+enum



## PodOS

PodOS defines the OS parameters of a pod.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | [OSName](./k8s-io-api-core-v1.md#OSName) | Yes | Name is the name of the operating system. The currently supported values are linux and windows.<br />Additional value may be defined in future and can be one of:<br />https://github.com/opencontainers/runtime-spec/blob/master/config.md#platform-specific-configuration<br />Clients should expect to handle additional values and treat unrecognized values in this field as os: null |

## PodReadinessGate

PodReadinessGate contains the reference to a pod condition

| Stanza | Type | Required | Description |
|---|---|---|---|
| `conditionType` | [PodConditionType](./k8s-io-api-core-v1.md#PodConditionType) | Yes | ConditionType refers to a condition in the pod's condition list with matching type. |

## PodResourceClaim

PodResourceClaim references exactly one ResourceClaim, either directly<br />or by naming a ResourceClaimTemplate which is then turned into a ResourceClaim<br />for the pod.<br /><br />It adds a name to it that uniquely identifies the ResourceClaim inside the Pod.<br />Containers that need access to the ResourceClaim reference it with this name.<br /><br />When the DRAWorkloadResourceClaims feature gate is enabled and this Pod<br />belongs to a PodGroup, a PodResourceClaim is matched to a<br />PodGroupResourceClaim if all of their fields are equal (Name,<br />ResourceClaimName, and ResourceClaimTemplateName). A matched claim references<br />a single ResourceClaim shared across all Pods in the PodGroup, reserved for<br />the PodGroup in ResourceClaimStatus.ReservedFor rather than for individual<br />Pods.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name uniquely identifies this resource claim inside the pod.<br />This must be a DNS_LABEL. |
| `resourceClaimName` | *string | No | ResourceClaimName is the name of a ResourceClaim object in the same<br />namespace as this pod.<br /><br />Exactly one of ResourceClaimName and ResourceClaimTemplateName must<br />be set. |
| `resourceClaimTemplateName` | *string | No | ResourceClaimTemplateName is the name of a ResourceClaimTemplate<br />object in the same namespace as this pod.<br /><br />The template will be used to create a new ResourceClaim, which will<br />be bound to this pod. When this pod is deleted, the ResourceClaim<br />will also be deleted. The pod name and resource name, along with a<br />generated component, will be used to form a unique name for the<br />ResourceClaim, which will be recorded in pod.status.resourceClaimStatuses.<br /><br />When the DRAWorkloadResourceClaims feature gate is enabled and the pod<br />belongs to a PodGroup that defines a PodGroupResourceClaim with the same<br />Name and ResourceClaimTemplateName, this PodResourceClaim resolves to the<br />ResourceClaim generated for the PodGroup. All pods in the group that<br />define an equivalent PodResourceClaim matching the<br />PodGroupResourceClaim's Name and ResourceClaimTemplateName share the same<br />generated ResourceClaim. ResourceClaims generated for a PodGroup are<br />owned by the PodGroup and their lifecycles are tied to the PodGroup<br />instead of any individual pod.<br /><br />This field is immutable and no changes will be made to the<br />corresponding ResourceClaim by the control plane after creating the<br />ResourceClaim.<br /><br />Exactly one of ResourceClaimName and ResourceClaimTemplateName must<br />be set. |

## PodSELinuxChangePolicy

PodSELinuxChangePolicy defines how the container's SELinux label is applied to all volumes used by the Pod.



## PodSchedulingGate

PodSchedulingGate is associated to a Pod to guard its scheduling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name of the scheduling gate.<br />Each scheduling gate must have a unique name field. |

## PodSchedulingGroup

PodSchedulingGroup identifies the runtime scheduling group instance that a Pod belongs to.<br />The scheduler uses this information to apply workload-aware scheduling semantics.<br />Exactly one field must be specified.<br />+union

| Stanza | Type | Required | Description |
|---|---|---|---|
| `podGroupName` | *string | No | PodGroupName specifies the name of the standalone PodGroup object<br />that represents the runtime instance of this group.<br />Must be a DNS subdomain.<br /><br />+optional<br />+oneOf=GroupSelection |

## PodSecurityContext

PodSecurityContext holds pod-level security attributes and common container settings.<br />Some fields are also present in container.securityContext.  Field values of<br />container.securityContext take precedence over field values of PodSecurityContext.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `seLinuxOptions` | *[SELinuxOptions](./k8s-io-api-core-v1.md#SELinuxOptions) | No | The SELinux context to be applied to all containers.<br />If unspecified, the container runtime will allocate a random SELinux context for each<br />container.  May also be set in SecurityContext.  If set in<br />both SecurityContext and PodSecurityContext, the value specified in SecurityContext<br />takes precedence for that container.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `windowsOptions` | *[WindowsSecurityContextOptions](./k8s-io-api-core-v1.md#WindowsSecurityContextOptions) | No | The Windows specific settings applied to all containers.<br />If unspecified, the options within a container's SecurityContext will be used.<br />If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence.<br />Note that this field cannot be set when spec.os.name is linux.<br />+optional |
| `runAsUser` | *int64 | No | The UID to run the entrypoint of the container process.<br />Defaults to user specified in image metadata if unspecified.<br />May also be set in SecurityContext.  If set in both SecurityContext and<br />PodSecurityContext, the value specified in SecurityContext takes precedence<br />for that container.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `runAsGroup` | *int64 | No | The GID to run the entrypoint of the container process.<br />Uses runtime default if unset.<br />May also be set in SecurityContext.  If set in both SecurityContext and<br />PodSecurityContext, the value specified in SecurityContext takes precedence<br />for that container.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `runAsNonRoot` | *bool | No | Indicates that the container must run as a non-root user.<br />If true, the Kubelet will validate the image at runtime to ensure that it<br />does not run as UID 0 (root) and fail to start the container if it does.<br />If unset or false, no such validation will be performed.<br />May also be set in SecurityContext.  If set in both SecurityContext and<br />PodSecurityContext, the value specified in SecurityContext takes precedence.<br />+optional |
| `supplementalGroups` | []int64 | No | A list of groups applied to the first process run in each container, in<br />addition to the container's primary GID and fsGroup (if specified).  If<br />the SupplementalGroupsPolicy feature is enabled, the<br />supplementalGroupsPolicy field determines whether these are in addition<br />to or instead of any group memberships defined in the container image.<br />If unspecified, no additional groups are added, though group memberships<br />defined in the container image may still be used, depending on the<br />supplementalGroupsPolicy field.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional<br />+listType=atomic |
| `supplementalGroupsPolicy` | *[SupplementalGroupsPolicy](./k8s-io-api-core-v1.md#SupplementalGroupsPolicy) | No | Defines how supplemental groups of the first container processes are calculated.<br />Valid values are "Merge" and "Strict". If not specified, "Merge" is used.<br />(Alpha) Using the field requires the SupplementalGroupsPolicy feature gate to be enabled<br />and the container runtime must implement support for this feature.<br />Note that this field cannot be set when spec.os.name is windows.<br />TODO: update the default value to "Merge" when spec.os.name is not windows in v1.34<br />+featureGate=SupplementalGroupsPolicy<br />+optional |
| `fsGroup` | *int64 | No | A special supplemental group that applies to all containers in a pod.<br />Some volume types allow the Kubelet to change the ownership of that volume<br />to be owned by the pod:<br /><br />1. The owning GID will be the FSGroup<br />2. The setgid bit is set (new files created in the volume will be owned by FSGroup)<br />3. The permission bits are OR'd with rw-rw----<br /><br />If unset, the Kubelet will not modify the ownership and permissions of any volume.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `sysctls` | [][Sysctl](./k8s-io-api-core-v1.md#Sysctl) | No | Sysctls hold a list of namespaced sysctls used for the pod. Pods with unsupported<br />sysctls (by the container runtime) might fail to launch.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional<br />+listType=atomic |
| `fsGroupChangePolicy` | *[PodFSGroupChangePolicy](./k8s-io-api-core-v1.md#PodFSGroupChangePolicy) | No | fsGroupChangePolicy defines behavior of changing ownership and permission of the volume<br />before being exposed inside Pod. This field will only apply to<br />volume types which support fsGroup based ownership(and permissions).<br />It will have no effect on ephemeral volume types such as: secret, configmaps<br />and emptydir.<br />Valid values are "OnRootMismatch" and "Always". If not specified, "Always" is used.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `seccompProfile` | *[SeccompProfile](./k8s-io-api-core-v1.md#SeccompProfile) | No | The seccomp options to use by the containers in this pod.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `appArmorProfile` | *[AppArmorProfile](./k8s-io-api-core-v1.md#AppArmorProfile) | No | appArmorProfile is the AppArmor options to use by the containers in this pod.<br />Note that this field cannot be set when spec.os.name is windows.<br />+optional |
| `seLinuxChangePolicy` | *[PodSELinuxChangePolicy](./k8s-io-api-core-v1.md#PodSELinuxChangePolicy) | No | seLinuxChangePolicy defines how the container's SELinux label is applied to all volumes used by the Pod.<br />It has no effect on nodes that do not support SELinux or to volumes does not support SELinux.<br />Valid values are "MountOption" and "Recursive".<br /><br />"Recursive" means relabeling of all files on all Pod volumes by the container runtime.<br />This may be slow for large volumes, but allows mixing privileged and unprivileged Pods sharing the same volume on the same node.<br /><br />"MountOption" mounts all eligible Pod volumes with `-o context` mount option.<br />This requires all Pods that share the same volume to use the same SELinux label.<br />It is not possible to share the same volume among privileged and unprivileged Pods.<br />Eligible volumes are in-tree FibreChannel and iSCSI volumes, and all CSI volumes<br />whose CSI driver announces SELinux support by setting spec.seLinuxMount: true in their<br />CSIDriver instance. Other volumes are always re-labelled recursively.<br />"MountOption" value is allowed only when SELinuxMount feature gate is enabled.<br /><br />If not specified and SELinuxMount feature gate is enabled, "MountOption" is used.<br />If not specified and SELinuxMount feature gate is disabled, "MountOption" is used for ReadWriteOncePod volumes<br />and "Recursive" for all other volumes.<br /><br />This field affects only Pods that have SELinux label set, either in PodSecurityContext or in SecurityContext of all containers.<br /><br />All Pods that use the same volume should use the same seLinuxChangePolicy, otherwise some pods can get stuck in ContainerCreating state.<br />Note that this field cannot be set when spec.os.name is windows.<br />+featureGate=SELinuxChangePolicy<br />+optional |

## PodSpec

PodSpec is a description of a pod.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `volumes` | [][Volume](./k8s-io-api-core-v1.md#Volume) | No | List of volumes that can be mounted by containers belonging to the pod.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes<br />+optional<br />+patchMergeKey=name<br />+patchStrategy=merge,retainKeys<br />+listType=map<br />+listMapKey=name |
| `initContainers` | [][Container](./k8s-io-api-core-v1.md#Container) | No | List of initialization containers belonging to the pod.<br />Init containers are executed in order prior to containers being started. If any<br />init container fails, the pod is considered to have failed and is handled according<br />to its restartPolicy. The name for an init container or normal container must be<br />unique among all containers.<br />Init containers may not have Lifecycle actions, Readiness probes, Liveness probes, or Startup probes.<br />The resourceRequirements of an init container are taken into account during scheduling<br />by finding the highest request/limit for each resource type, and then using the max of<br />that value or the sum of the normal containers. Limits are applied to init containers<br />in a similar fashion.<br />Init containers cannot currently be added or removed.<br />Cannot be updated.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/init-containers/<br />+patchMergeKey=name<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=name |
| `containers` | [][Container](./k8s-io-api-core-v1.md#Container) | No | List of containers belonging to the pod.<br />Containers cannot currently be added or removed.<br />There must be at least one container in a Pod.<br />Cannot be updated.<br />+patchMergeKey=name<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=name |
| `ephemeralContainers` | [][EphemeralContainer](./k8s-io-api-core-v1.md#EphemeralContainer) | No | List of ephemeral containers run in this pod. Ephemeral containers may be run in an existing<br />pod to perform user-initiated actions such as debugging. This list cannot be specified when<br />creating a pod, and it cannot be modified by updating the pod spec. In order to add an<br />ephemeral container to an existing pod, use the pod's ephemeralcontainers subresource.<br />+optional<br />+patchMergeKey=name<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=name |
| `restartPolicy` | [RestartPolicy](./k8s-io-api-core-v1.md#RestartPolicy) | No | Restart policy for all containers within the pod.<br />One of Always, OnFailure, Never. In some contexts, only a subset of those values may be permitted.<br />Default to Always.<br />More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#restart-policy<br />+optional |
| `terminationGracePeriodSeconds` | *int64 | No | Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request.<br />Value must be non-negative integer. The value zero indicates stop immediately via<br />the kill signal (no opportunity to shut down).<br />If this value is nil, the default grace period will be used instead.<br />The grace period is the duration in seconds after the processes running in the pod are sent<br />a termination signal and the time when the processes are forcibly halted with a kill signal.<br />Set this value longer than the expected cleanup time for your process.<br />Defaults to 30 seconds.<br />+optional |
| `activeDeadlineSeconds` | *int64 | No | Optional duration in seconds the pod may be active on the node relative to<br />StartTime before the system will actively try to mark it failed and kill associated containers.<br />Value must be a positive integer.<br />+optional |
| `dnsPolicy` | [DNSPolicy](./k8s-io-api-core-v1.md#DNSPolicy) | No | Set DNS policy for the pod.<br />Defaults to "ClusterFirst".<br />Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst', 'Default' or 'None'.<br />DNS parameters given in DNSConfig will be merged with the policy selected with DNSPolicy.<br />To have DNS options set along with hostNetwork, you have to specify DNS policy<br />explicitly to 'ClusterFirstWithHostNet'.<br />+optional |
| `nodeSelector` | map[string]string | No | NodeSelector is a selector which must be true for the pod to fit on a node.<br />Selector which must match a node's labels for the pod to be scheduled on that node.<br />More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/<br />+optional<br />+mapType=atomic |
| `serviceAccountName` | string | No | ServiceAccountName is the name of the ServiceAccount to use to run this pod.<br />More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/<br />+optional |
| `serviceAccount` | string | No | DeprecatedServiceAccount is a deprecated alias for ServiceAccountName.<br />Deprecated: Use serviceAccountName instead.<br />+k8s:conversion-gen=false<br />+optional |
| `automountServiceAccountToken` | *bool | No | AutomountServiceAccountToken indicates whether a service account token should be automatically mounted.<br />+optional |
| `nodeName` | string | No | NodeName indicates in which node this pod is scheduled.<br />If empty, this pod is a candidate for scheduling by the scheduler defined in schedulerName.<br />Once this field is set, the kubelet for this node becomes responsible for the lifecycle of this pod.<br />This field should not be used to express a desire for the pod to be scheduled on a specific node.<br />https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodename<br />+optional |
| `hostNetwork` | bool | No | Host networking requested for this pod. Use the host's network namespace.<br />When using HostNetwork you should specify ports so the scheduler is aware.<br />When `hostNetwork` is true, specified `hostPort` fields in port definitions must match `containerPort`,<br />and unspecified `hostPort` fields in port definitions are defaulted to match `containerPort`.<br />Default to false.<br />+k8s:conversion-gen=false<br />+optional |
| `hostPID` | bool | No | Use the host's pid namespace.<br />Optional: Default to false.<br />+k8s:conversion-gen=false<br />+optional |
| `hostIPC` | bool | No | Use the host's ipc namespace.<br />Optional: Default to false.<br />+k8s:conversion-gen=false<br />+optional |
| `shareProcessNamespace` | *bool | No | Share a single process namespace between all of the containers in a pod.<br />When this is set containers will be able to view and signal processes from other containers<br />in the same pod, and the first process in each container will not be assigned PID 1.<br />HostPID and ShareProcessNamespace cannot both be set.<br />Optional: Default to false.<br />+k8s:conversion-gen=false<br />+optional |
| `securityContext` | *[PodSecurityContext](./k8s-io-api-core-v1.md#PodSecurityContext) | No | SecurityContext holds pod-level security attributes and common container settings.<br />Optional: Defaults to empty.  See type description for default values of each field.<br />+optional |
| `imagePullSecrets` | [][LocalObjectReference](./k8s-io-api-core-v1.md#LocalObjectReference) | No | ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.<br />If specified, these secrets will be passed to individual puller implementations for them to use.<br />More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod<br />+optional<br />+patchMergeKey=name<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=name |
| `hostname` | string | No | Specifies the hostname of the Pod<br />If not specified, the pod's hostname will be set to a system-defined value.<br />+optional |
| `subdomain` | string | No | If specified, the fully qualified Pod hostname will be "<hostname>.<subdomain>.<pod namespace>.svc.<cluster domain>".<br />If not specified, the pod will not have a domainname at all.<br />+optional |
| `affinity` | *[Affinity](./k8s-io-api-core-v1.md#Affinity) | No | If specified, the pod's scheduling constraints<br />+optional |
| `schedulerName` | string | No | If specified, the pod will be dispatched by specified scheduler.<br />If not specified, the pod will be dispatched by default scheduler.<br />+optional |
| `tolerations` | [][Toleration](./k8s-io-api-core-v1.md#Toleration) | No | If specified, the pod's tolerations.<br />+optional<br />+listType=atomic |
| `hostAliases` | [][HostAlias](./k8s-io-api-core-v1.md#HostAlias) | No | HostAliases is an optional list of hosts and IPs that will be injected into the pod's hosts<br />file if specified.<br />+optional<br />+patchMergeKey=ip<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=ip |
| `priorityClassName` | string | No | If specified, indicates the pod's priority. "system-node-critical" and<br />"system-cluster-critical" are two special keywords which indicate the<br />highest priorities with the former being the highest priority. Any other<br />name must be defined by creating a PriorityClass object with that name.<br />If not specified, the pod priority will be default or zero if there is no<br />default.<br />+optional |
| `priority` | *int32 | No | The priority value. Various system components use this field to find the<br />priority of the pod. When Priority Admission Controller is enabled, it<br />prevents users from setting this field. The admission controller populates<br />this field from PriorityClassName.<br />The higher the value, the higher the priority.<br />+optional |
| `dnsConfig` | *[PodDNSConfig](./k8s-io-api-core-v1.md#PodDNSConfig) | No | Specifies the DNS parameters of a pod.<br />Parameters specified here will be merged to the generated DNS<br />configuration based on DNSPolicy.<br />+optional |
| `readinessGates` | [][PodReadinessGate](./k8s-io-api-core-v1.md#PodReadinessGate) | No | If specified, all readiness gates will be evaluated for pod readiness.<br />A pod is ready when all its containers are ready AND<br />all conditions specified in the readiness gates have status equal to "True"<br />More info: https://git.k8s.io/enhancements/keps/sig-network/580-pod-readiness-gates<br />+optional<br />+listType=atomic |
| `runtimeClassName` | *string | No | RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used<br />to run this pod.  If no RuntimeClass resource matches the named class, the pod will not be run.<br />If unset or empty, the "legacy" RuntimeClass will be used, which is an implicit class with an<br />empty definition that uses the default runtime handler.<br />More info: https://git.k8s.io/enhancements/keps/sig-node/585-runtime-class<br />+optional |
| `enableServiceLinks` | *bool | No | EnableServiceLinks indicates whether information about services should be injected into pod's<br />environment variables, matching the syntax of Docker links.<br />Optional: Defaults to true.<br />+optional |
| `preemptionPolicy` | *[PreemptionPolicy](./k8s-io-api-core-v1.md#PreemptionPolicy) | No | PreemptionPolicy is the Policy for preempting pods with lower priority.<br />One of Never, PreemptLowerPriority.<br />Defaults to PreemptLowerPriority if unset.<br />+optional |
| `overhead` | [ResourceList](./k8s-io-api-core-v1.md#ResourceList) | No | Overhead represents the resource overhead associated with running a pod for a given RuntimeClass.<br />This field will be autopopulated at admission time by the RuntimeClass admission controller. If<br />the RuntimeClass admission controller is enabled, overhead must not be set in Pod create requests.<br />The RuntimeClass admission controller will reject Pod create requests which have the overhead already<br />set. If RuntimeClass is configured and selected in the PodSpec, Overhead will be set to the value<br />defined in the corresponding RuntimeClass, otherwise it will remain unset and treated as zero.<br />More info: https://git.k8s.io/enhancements/keps/sig-node/688-pod-overhead/README.md<br />+optional |
| `topologySpreadConstraints` | [][TopologySpreadConstraint](./k8s-io-api-core-v1.md#TopologySpreadConstraint) | No | TopologySpreadConstraints describes how a group of pods ought to spread across topology<br />domains. Scheduler will schedule pods in a way which abides by the constraints.<br />All topologySpreadConstraints are ANDed.<br />+optional<br />+patchMergeKey=topologyKey<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=topologyKey<br />+listMapKey=whenUnsatisfiable |
| `setHostnameAsFQDN` | *bool | No | If true the pod's hostname will be configured as the pod's FQDN, rather than the leaf name (the default).<br />In Linux containers, this means setting the FQDN in the hostname field of the kernel (the nodename field of struct utsname).<br />In Windows containers, this means setting the registry value of hostname for the registry key HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\Tcpip\\Parameters to FQDN.<br />If a pod does not have FQDN, this has no effect.<br />Default to false.<br />+optional |
| `os` | *[PodOS](./k8s-io-api-core-v1.md#PodOS) | No | Specifies the OS of the containers in the pod.<br />Some pod and container fields are restricted if this is set.<br /><br />If the OS field is set to linux, the following fields must be unset:<br />-securityContext.windowsOptions<br /><br />If the OS field is set to windows, following fields must be unset:<br />- spec.hostPID<br />- spec.hostIPC<br />- spec.hostUsers<br />- spec.resources<br />- spec.securityContext.appArmorProfile<br />- spec.securityContext.seLinuxOptions<br />- spec.securityContext.seccompProfile<br />- spec.securityContext.fsGroup<br />- spec.securityContext.fsGroupChangePolicy<br />- spec.securityContext.sysctls<br />- spec.shareProcessNamespace<br />- spec.securityContext.runAsUser<br />- spec.securityContext.runAsGroup<br />- spec.securityContext.supplementalGroups<br />- spec.securityContext.supplementalGroupsPolicy<br />- spec.containers[*].securityContext.appArmorProfile<br />- spec.containers[*].securityContext.seLinuxOptions<br />- spec.containers[*].securityContext.seccompProfile<br />- spec.containers[*].securityContext.capabilities<br />- spec.containers[*].securityContext.readOnlyRootFilesystem<br />- spec.containers[*].securityContext.privileged<br />- spec.containers[*].securityContext.allowPrivilegeEscalation<br />- spec.containers[*].securityContext.procMount<br />- spec.containers[*].securityContext.runAsUser<br />- spec.containers[*].securityContext.runAsGroup<br />+optional |
| `hostUsers` | *bool | No | Use the host's user namespace.<br />Optional: Default to true.<br />If set to true or not present, the pod will be run in the host user namespace, useful<br />for when the pod needs a feature only available to the host user namespace, such as<br />loading a kernel module with CAP_SYS_MODULE.<br />When set to false, a new userns is created for the pod. Setting false is useful for<br />mitigating container breakout vulnerabilities even allowing users to run their<br />containers as root without actually having root privileges on the host.<br />+k8s:conversion-gen=false<br />+optional |
| `schedulingGates` | [][PodSchedulingGate](./k8s-io-api-core-v1.md#PodSchedulingGate) | No | SchedulingGates is an opaque list of values that if specified will block scheduling the pod.<br />If schedulingGates is not empty, the pod will stay in the SchedulingGated state and the<br />scheduler will not attempt to schedule the pod.<br /><br />SchedulingGates can only be set at pod creation time, and be removed only afterwards.<br /><br />+patchMergeKey=name<br />+patchStrategy=merge<br />+listType=map<br />+listMapKey=name<br />+optional |
| `resourceClaims` | [][PodResourceClaim](./k8s-io-api-core-v1.md#PodResourceClaim) | No | ResourceClaims defines which ResourceClaims must be allocated<br />and reserved before the Pod is allowed to start. The resources<br />will be made available to those containers which consume them<br />by name.<br /><br />This is a stable field but requires that the<br />DynamicResourceAllocation feature gate is enabled.<br /><br />This field is immutable.<br /><br />+patchMergeKey=name<br />+patchStrategy=merge,retainKeys<br />+listType=map<br />+listMapKey=name<br />+featureGate=DynamicResourceAllocation<br />+optional |
| `resources` | *[ResourceRequirements](./k8s-io-api-core-v1.md#ResourceRequirements) | No | Resources is the total amount of CPU and Memory resources required by all<br />containers in the pod. It supports specifying Requests and Limits for<br />"cpu", "memory" and "hugepages-" resource names only. ResourceClaims are not supported.<br /><br />This field enables fine-grained control over resource allocation for the<br />entire pod, allowing resource sharing among containers in a pod.<br />TODO: For beta graduation, expand this comment with a detailed explanation.<br /><br />This is an alpha field and requires enabling the PodLevelResources feature<br />gate.<br /><br />+featureGate=PodLevelResources<br />+optional |
| `hostnameOverride` | *string | No | HostnameOverride specifies an explicit override for the pod's hostname as perceived by the pod.<br />This field only specifies the pod's hostname and does not affect its DNS records.<br />When this field is set to a non-empty string:<br />- It takes precedence over the values set in `hostname` and `subdomain`.<br />- The Pod's hostname will be set to this value.<br />- `setHostnameAsFQDN` must be nil or set to false.<br />- `hostNetwork` must be set to false.<br /><br />This field must be a valid DNS subdomain as defined in RFC 1123 and contain at most 64 characters.<br />Requires the HostnameOverride feature gate to be enabled.<br /><br />+featureGate=HostnameOverride<br />+optional |
| `schedulingGroup` | *[PodSchedulingGroup](./k8s-io-api-core-v1.md#PodSchedulingGroup) | No | SchedulingGroup provides a reference to the immediate scheduling runtime<br />grouping object that this Pod belongs to.<br />This field is used by the scheduler to identify the group and apply the<br />correct group scheduling policies. The association with a group also<br />impacts other lifecycle aspects of a Pod that are relevant in a wider context<br />of scheduling like preemption, resource attachment, etc. If not specified,<br />the Pod is treated as a single unit in all of these aspects.<br />The group object referenced by this field may not exist at the time the<br />Pod is created.<br />This field is immutable, but a group object with the same name may be<br />recreated with different policies. Doing this during pod scheduling<br />may result in the placement not conforming to the expected policies.<br /><br />+featureGate=GenericWorkload<br />+optional |

## PortworxVolumeSource

PortworxVolumeSource represents a Portworx volume resource.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `volumeID` | string | Yes | volumeID uniquely identifies a Portworx volume |
| `fsType` | string | No | fSType represents the filesystem type to mount<br />Must be a filesystem type supported by the host operating system.<br />Ex. "ext4", "xfs". Implicitly inferred to be "ext4" if unspecified. |
| `readOnly` | bool | No | readOnly defaults to false (read/write). ReadOnly here will force<br />the ReadOnly setting in VolumeMounts.<br />+optional |

## PreemptionPolicy

PreemptionPolicy describes a policy for if/when to preempt a pod.<br />+enum



## PreferredSchedulingTerm

An empty preferred scheduling term matches all objects with implicit weight 0<br />(i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

| Stanza | Type | Required | Description |
|---|---|---|---|
| `weight` | int32 | Yes | Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100. |
| `preference` | [NodeSelectorTerm](./k8s-io-api-core-v1.md#NodeSelectorTerm) | Yes | A node selector term, associated with the corresponding weight. |

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



## QuobyteVolumeSource

Represents a Quobyte mount that lasts the lifetime of a pod.<br />Quobyte volumes do not support ownership management or SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `registry` | string | Yes | registry represents a single or multiple Quobyte Registry services<br />specified as a string as host:port pair (multiple entries are separated with commas)<br />which acts as the central registry for volumes |
| `volume` | string | Yes | volume is a string that references an already created Quobyte volume by name. |
| `readOnly` | bool | No | readOnly here will force the Quobyte volume to be mounted with read-only permissions.<br />Defaults to false.<br />+optional |
| `user` | string | No | user to map volume access to<br />Defaults to serivceaccount user<br />+optional |
| `group` | string | No | group to map volume access to<br />Default is no group<br />+optional |
| `tenant` | string | No | tenant owning the given Quobyte volume in the Backend<br />Used with dynamically provisioned Quobyte volumes, value is set by the plugin<br />+optional |

## RBDVolumeSource

Represents a Rados Block Device mount that lasts the lifetime of a pod.<br />RBD volumes support ownership management and SELinux relabeling.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `monitors` | []string | No | monitors is a collection of Ceph monitors.<br />More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br />+listType=atomic |
| `image` | string | Yes | image is the rados image name.<br />More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it |
| `fsType` | string | No | fsType is the filesystem type of the volume that you want to mount.<br />Tip: Ensure that the filesystem type is supported by the host operating system.<br />Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#rbd<br />TODO: how do we prevent errors in the filesystem from compromising the machine<br />+optional |
| `pool` | string | No | pool is the rados pool name.<br />Default is rbd.<br />More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br />+optional<br />+default="rbd" |
| `user` | string | No | user is the rados user name.<br />Default is admin.<br />More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br />+optional<br />+default="admin" |
| `keyring` | string | No | keyring is the path to key ring for RBDUser.<br />Default is /etc/ceph/keyring.<br />More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br />+optional<br />+default="/etc/ceph/keyring" |
| `secretRef` | *[LocalObjectReference](./k8s-io-api-core-v1.md#LocalObjectReference) | No | secretRef is name of the authentication secret for RBDUser. If provided<br />overrides keyring.<br />Default is nil.<br />More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br />+optional |
| `readOnly` | bool | No | readOnly here will force the ReadOnly setting in VolumeMounts.<br />Defaults to false.<br />More info: https://examples.k8s.io/volumes/rbd/README.md#how-to-use-it<br />+optional |

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

## ResourceResizeRestartPolicy

ResourceResizeRestartPolicy specifies how to handle container resource resize.



## RestartPolicy

RestartPolicy describes how the container should be restarted.<br />Only one of the following restart policies may be specified.<br />If none of the following policies is specified, the default one<br />is RestartPolicyAlways.<br />+enum



## SELinuxOptions

SELinuxOptions are the labels to be applied to the container

| Stanza | Type | Required | Description |
|---|---|---|---|
| `user` | string | No | User is a SELinux user label that applies to the container.<br />+optional |
| `role` | string | No | Role is a SELinux role label that applies to the container.<br />+optional |
| `type` | string | No | Type is a SELinux type label that applies to the container.<br />+optional |
| `level` | string | No | Level is SELinux level label that applies to the container.<br />+optional |

## ScaleIOVolumeSource

ScaleIOVolumeSource represents a persistent ScaleIO volume

| Stanza | Type | Required | Description |
|---|---|---|---|
| `gateway` | string | Yes | gateway is the host address of the ScaleIO API Gateway. |
| `system` | string | Yes | system is the name of the storage system as configured in ScaleIO. |
| `secretRef` | *[LocalObjectReference](./k8s-io-api-core-v1.md#LocalObjectReference) | No | secretRef references to the secret for ScaleIO user and other<br />sensitive information. If this is not provided, Login operation will fail. |
| `sslEnabled` | bool | No | sslEnabled Flag enable/disable SSL communication with Gateway, default false<br />+optional |
| `protectionDomain` | string | No | protectionDomain is the name of the ScaleIO Protection Domain for the configured storage.<br />+optional |
| `storagePool` | string | No | storagePool is the ScaleIO Storage Pool associated with the protection domain.<br />+optional |
| `storageMode` | string | No | storageMode indicates whether the storage for a volume should be ThickProvisioned or ThinProvisioned.<br />Default is ThinProvisioned.<br />+optional<br />+default="ThinProvisioned" |
| `volumeName` | string | No | volumeName is the name of a volume already created in the ScaleIO system<br />that is associated with this volume source. |
| `fsType` | string | No | fsType is the filesystem type to mount.<br />Must be a filesystem type supported by the host operating system.<br />Ex. "ext4", "xfs", "ntfs".<br />Default is "xfs".<br />+optional<br />+default="xfs" |
| `readOnly` | bool | No | readOnly Defaults to false (read/write). ReadOnly here will force<br />the ReadOnly setting in VolumeMounts.<br />+optional |

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



## StorageOSVolumeSource

Represents a StorageOS persistent volume resource.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `volumeName` | string | No | volumeName is the human-readable name of the StorageOS volume.  Volume<br />names are only unique within a namespace. |
| `volumeNamespace` | string | No | volumeNamespace specifies the scope of the volume within StorageOS.  If no<br />namespace is specified then the Pod's namespace will be used.  This allows the<br />Kubernetes name scoping to be mirrored within StorageOS for tighter integration.<br />Set VolumeName to any name to override the default behaviour.<br />Set to "default" if you are not using namespaces within StorageOS.<br />Namespaces that do not pre-exist within StorageOS will be created.<br />+optional |
| `fsType` | string | No | fsType is the filesystem type to mount.<br />Must be a filesystem type supported by the host operating system.<br />Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br />+optional |
| `readOnly` | bool | No | readOnly defaults to false (read/write). ReadOnly here will force<br />the ReadOnly setting in VolumeMounts.<br />+optional |
| `secretRef` | *[LocalObjectReference](./k8s-io-api-core-v1.md#LocalObjectReference) | No | secretRef specifies the secret to use for obtaining the StorageOS API<br />credentials.  If not specified, default values will be attempted.<br />+optional |

## SupplementalGroupsPolicy

SupplementalGroupsPolicy defines how supplemental groups<br />of the first container processes are calculated.<br />+enum



## Sysctl

Sysctl defines a kernel parameter to be set

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | Name of a property to set |
| `value` | string | Yes | Value of a property to set |

## TCPSocketAction

TCPSocketAction describes an action based on opening a socket

| Stanza | Type | Required | Description |
|---|---|---|---|
| `port` | [IntOrString](./k8s-io-apimachinery-pkg-util-intstr.md#IntOrString) | Yes | Number or name of the port to access on the container.<br />Number must be in the range 1 to 65535.<br />Name must be an IANA_SVC_NAME. |
| `host` | string | No | Optional: Host name to connect to, defaults to the pod IP.<br />+optional |

## TaintEffect

+enum



## TerminationMessagePolicy

TerminationMessagePolicy describes how termination messages are retrieved from a container.<br />+enum



## Toleration

The pod this Toleration is attached to tolerates any taint that matches<br />the triple <key,value,effect> using the matching operator <operator>.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `key` | string | No | Key is the taint key that the toleration applies to. Empty means match all taint keys.<br />If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br />+optional |
| `operator` | [TolerationOperator](./k8s-io-api-core-v1.md#TolerationOperator) | No | Operator represents a key's relationship to the value.<br />Valid operators are Exists, Equal, Lt, and Gt. Defaults to Equal.<br />Exists is equivalent to wildcard for value, so that a pod can<br />tolerate all taints of a particular category.<br />Lt and Gt perform numeric comparisons (requires feature gate TaintTolerationComparisonOperators).<br />+optional |
| `value` | string | No | Value is the taint value the toleration matches to.<br />If the operator is Exists, the value should be empty, otherwise just a regular string.<br />+optional |
| `effect` | [TaintEffect](./k8s-io-api-core-v1.md#TaintEffect) | No | Effect indicates the taint effect to match. Empty means match all taint effects.<br />When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br />+optional |
| `tolerationSeconds` | *int64 | No | TolerationSeconds represents the period of time the toleration (which must be<br />of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,<br />it is not set, which means tolerate the taint forever (do not evict). Zero and<br />negative values will be treated as 0 (evict immediately) by the system.<br />+optional |

## TolerationOperator

A toleration operator is the set of operators that can be used in a toleration.<br />+enum



## TopologySpreadConstraint

TopologySpreadConstraint specifies how to spread matching pods among the given topology.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `maxSkew` | int32 | Yes | MaxSkew describes the degree to which pods may be unevenly distributed.<br />When `whenUnsatisfiable=DoNotSchedule`, it is the maximum permitted difference<br />between the number of matching pods in the target topology and the global minimum.<br />The global minimum is the minimum number of matching pods in an eligible domain<br />or zero if the number of eligible domains is less than MinDomains.<br />For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same<br />labelSelector spread as 2/2/1:<br />In this case, the global minimum is 1.<br />+-------+-------+-------+<br />| zone1 | zone2 | zone3 |<br />+-------+-------+-------+<br />|  P P  |  P P  |   P   |<br />+-------+-------+-------+<br />- if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 2/2/2;<br />scheduling it onto zone1(zone2) would make the ActualSkew(3-1) on zone1(zone2)<br />violate MaxSkew(1).<br />- if MaxSkew is 2, incoming pod can be scheduled onto any zone.<br />When `whenUnsatisfiable=ScheduleAnyway`, it is used to give higher precedence<br />to topologies that satisfy it.<br />It's a required field. Default value is 1 and 0 is not allowed. |
| `topologyKey` | string | Yes | TopologyKey is the key of node labels. Nodes that have a label with this key<br />and identical values are considered to be in the same topology.<br />We consider each <key, value> as a "bucket", and try to put balanced number<br />of pods into each bucket.<br />We define a domain as a particular instance of a topology.<br />Also, we define an eligible domain as a domain whose nodes meet the requirements of<br />nodeAffinityPolicy and nodeTaintsPolicy.<br />e.g. If TopologyKey is "kubernetes.io/hostname", each Node is a domain of that topology.<br />And, if TopologyKey is "topology.kubernetes.io/zone", each zone is a domain of that topology.<br />It's a required field. |
| `whenUnsatisfiable` | [UnsatisfiableConstraintAction](./k8s-io-api-core-v1.md#UnsatisfiableConstraintAction) | Yes | WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy<br />the spread constraint.<br />- DoNotSchedule (default) tells the scheduler not to schedule it.<br />- ScheduleAnyway tells the scheduler to schedule the pod in any location,<br />  but giving higher precedence to topologies that would help reduce the<br />  skew.<br />A constraint is considered "Unsatisfiable" for an incoming pod<br />if and only if every possible node assignment for that pod would violate<br />"MaxSkew" on some topology.<br />For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same<br />labelSelector spread as 3/1/1:<br />+-------+-------+-------+<br />| zone1 | zone2 | zone3 |<br />+-------+-------+-------+<br />| P P P |   P   |   P   |<br />+-------+-------+-------+<br />If WhenUnsatisfiable is set to DoNotSchedule, incoming pod can only be scheduled<br />to zone2(zone3) to become 3/2/1(3/1/2) as ActualSkew(2-1) on zone2(zone3) satisfies<br />MaxSkew(1). In other words, the cluster can still be imbalanced, but scheduler<br />won't make it *more* imbalanced.<br />It's a required field. |
| `labelSelector` | *[LabelSelector](./k8s-io-apimachinery-pkg-apis-meta-v1.md#LabelSelector) | No | LabelSelector is used to find matching pods.<br />Pods that match this label selector are counted to determine the number of pods<br />in their corresponding topology domain.<br />+optional |
| `minDomains` | *int32 | No | MinDomains indicates a minimum number of eligible domains.<br />When the number of eligible domains with matching topology keys is less than minDomains,<br />Pod Topology Spread treats "global minimum" as 0, and then the calculation of Skew is performed.<br />And when the number of eligible domains with matching topology keys equals or greater than minDomains,<br />this value has no effect on scheduling.<br />As a result, when the number of eligible domains is less than minDomains,<br />scheduler won't schedule more than maxSkew Pods to those domains.<br />If value is nil, the constraint behaves as if MinDomains is equal to 1.<br />Valid values are integers greater than 0.<br />When value is not nil, WhenUnsatisfiable must be DoNotSchedule.<br /><br />For example, in a 3-zone cluster, MaxSkew is set to 2, MinDomains is set to 5 and pods with the same<br />labelSelector spread as 2/2/2:<br />+-------+-------+-------+<br />| zone1 | zone2 | zone3 |<br />+-------+-------+-------+<br />|  P P  |  P P  |  P P  |<br />+-------+-------+-------+<br />The number of domains is less than 5(MinDomains), so "global minimum" is treated as 0.<br />In this situation, new pod with the same labelSelector cannot be scheduled,<br />because computed skew will be 3(3 - 0) if new Pod is scheduled to any of the three zones,<br />it will violate MaxSkew.<br />+optional |
| `nodeAffinityPolicy` | *[NodeInclusionPolicy](./k8s-io-api-core-v1.md#NodeInclusionPolicy) | No | NodeAffinityPolicy indicates how we will treat Pod's nodeAffinity/nodeSelector<br />when calculating pod topology spread skew. Options are:<br />- Honor: only nodes matching nodeAffinity/nodeSelector are included in the calculations.<br />- Ignore: nodeAffinity/nodeSelector are ignored. All nodes are included in the calculations.<br /><br />If this value is nil, the behavior is equivalent to the Honor policy.<br />+optional |
| `nodeTaintsPolicy` | *[NodeInclusionPolicy](./k8s-io-api-core-v1.md#NodeInclusionPolicy) | No | NodeTaintsPolicy indicates how we will treat node taints when calculating<br />pod topology spread skew. Options are:<br />- Honor: nodes without taints, along with tainted nodes for which the incoming pod<br />has a toleration, are included.<br />- Ignore: node taints are ignored. All nodes are included.<br /><br />If this value is nil, the behavior is equivalent to the Ignore policy.<br />+optional |
| `matchLabelKeys` | []string | No | MatchLabelKeys is a set of pod label keys to select the pods over which<br />spreading will be calculated. The keys are used to lookup values from the<br />incoming pod labels, those key-value labels are ANDed with labelSelector<br />to select the group of existing pods over which spreading will be calculated<br />for the incoming pod. The same key is forbidden to exist in both MatchLabelKeys and LabelSelector.<br />MatchLabelKeys cannot be set when LabelSelector isn't set.<br />Keys that don't exist in the incoming pod labels will<br />be ignored. A null or empty list means only match against labelSelector.<br /><br />This is a beta field and requires the MatchLabelKeysInPodTopologySpread feature gate to be enabled (enabled by default).<br />+listType=atomic<br />+optional |

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



## UnsatisfiableConstraintAction

+enum



## Volume

Volume represents a named volume in a pod that may be accessed by any container in the pod.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes | name of the volume.<br />Must be a DNS_LABEL and unique within the pod.<br />More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names |
| `hostPath` | *[HostPathVolumeSource](./k8s-io-api-core-v1.md#HostPathVolumeSource) | No | hostPath represents a pre-existing file or directory on the host<br />machine that is directly exposed to the container. This is generally<br />used for system agents or other privileged things that are allowed<br />to see the host machine. Most containers will NOT need this.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath<br />---<br />TODO(jonesdl) We need to restrict who can use host directory mounts and who can/can not<br />mount host directories as read/write.<br />+optional |
| `emptyDir` | *[EmptyDirVolumeSource](./k8s-io-api-core-v1.md#EmptyDirVolumeSource) | No | emptyDir represents a temporary directory that shares a pod's lifetime.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir<br />+optional |
| `gcePersistentDisk` | *[GCEPersistentDiskVolumeSource](./k8s-io-api-core-v1.md#GCEPersistentDiskVolumeSource) | No | gcePersistentDisk represents a GCE Disk resource that is attached to a<br />kubelet's host machine and then exposed to the pod.<br />Deprecated: GCEPersistentDisk is deprecated. All operations for the in-tree<br />gcePersistentDisk type are redirected to the pd.csi.storage.gke.io CSI driver.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk<br />+optional |
| `awsElasticBlockStore` | *[AWSElasticBlockStoreVolumeSource](./k8s-io-api-core-v1.md#AWSElasticBlockStoreVolumeSource) | No | awsElasticBlockStore represents an AWS Disk resource that is attached to a<br />kubelet's host machine and then exposed to the pod.<br />Deprecated: AWSElasticBlockStore is deprecated. All operations for the in-tree<br />awsElasticBlockStore type are redirected to the ebs.csi.aws.com CSI driver.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore<br />+optional |
| `gitRepo` | *[GitRepoVolumeSource](./k8s-io-api-core-v1.md#GitRepoVolumeSource) | No | gitRepo represents a git repository at a particular revision.<br />Deprecated: GitRepo is deprecated. To provision a container with a git repo, mount an<br />EmptyDir into an InitContainer that clones the repo using git, then mount the EmptyDir<br />into the Pod's container.<br />+optional |
| `secret` | *[SecretVolumeSource](./k8s-io-api-core-v1.md#SecretVolumeSource) | No | secret represents a secret that should populate this volume.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#secret<br />+optional |
| `nfs` | *[NFSVolumeSource](./k8s-io-api-core-v1.md#NFSVolumeSource) | No | nfs represents an NFS mount on the host that shares a pod's lifetime<br />More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs<br />+optional |
| `iscsi` | *[ISCSIVolumeSource](./k8s-io-api-core-v1.md#ISCSIVolumeSource) | No | iscsi represents an ISCSI Disk resource that is attached to a<br />kubelet's host machine and then exposed to the pod.<br />More info: https://kubernetes.io/docs/concepts/storage/volumes/#iscsi<br />+optional |
| `glusterfs` | *[GlusterfsVolumeSource](./k8s-io-api-core-v1.md#GlusterfsVolumeSource) | No | glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime.<br />Deprecated: Glusterfs is deprecated and the in-tree glusterfs type is no longer supported.<br />+optional |
| `persistentVolumeClaim` | *[PersistentVolumeClaimVolumeSource](./k8s-io-api-core-v1.md#PersistentVolumeClaimVolumeSource) | No | persistentVolumeClaimVolumeSource represents a reference to a<br />PersistentVolumeClaim in the same namespace.<br />More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims<br />+optional |
| `rbd` | *[RBDVolumeSource](./k8s-io-api-core-v1.md#RBDVolumeSource) | No | rbd represents a Rados Block Device mount on the host that shares a pod's lifetime.<br />Deprecated: RBD is deprecated and the in-tree rbd type is no longer supported.<br />+optional |
| `flexVolume` | *[FlexVolumeSource](./k8s-io-api-core-v1.md#FlexVolumeSource) | No | flexVolume represents a generic volume resource that is<br />provisioned/attached using an exec based plugin.<br />Deprecated: FlexVolume is deprecated. Consider using a CSIDriver instead.<br />+optional |
| `cinder` | *[CinderVolumeSource](./k8s-io-api-core-v1.md#CinderVolumeSource) | No | cinder represents a cinder volume attached and mounted on kubelets host machine.<br />Deprecated: Cinder is deprecated. All operations for the in-tree cinder type<br />are redirected to the cinder.csi.openstack.org CSI driver.<br />More info: https://examples.k8s.io/mysql-cinder-pd/README.md<br />+optional |
| `cephfs` | *[CephFSVolumeSource](./k8s-io-api-core-v1.md#CephFSVolumeSource) | No | cephFS represents a Ceph FS mount on the host that shares a pod's lifetime.<br />Deprecated: CephFS is deprecated and the in-tree cephfs type is no longer supported.<br />+optional |
| `flocker` | *[FlockerVolumeSource](./k8s-io-api-core-v1.md#FlockerVolumeSource) | No | flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running.<br />Deprecated: Flocker is deprecated and the in-tree flocker type is no longer supported.<br />+optional |
| `downwardAPI` | *[DownwardAPIVolumeSource](./k8s-io-api-core-v1.md#DownwardAPIVolumeSource) | No | downwardAPI represents downward API about the pod that should populate this volume<br />+optional |
| `fc` | *[FCVolumeSource](./k8s-io-api-core-v1.md#FCVolumeSource) | No | fc represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.<br />+optional |
| `azureFile` | *[AzureFileVolumeSource](./k8s-io-api-core-v1.md#AzureFileVolumeSource) | No | azureFile represents an Azure File Service mount on the host and bind mount to the pod.<br />Deprecated: AzureFile is deprecated. All operations for the in-tree azureFile type<br />are redirected to the file.csi.azure.com CSI driver.<br />+optional |
| `configMap` | *[ConfigMapVolumeSource](./k8s-io-api-core-v1.md#ConfigMapVolumeSource) | No | configMap represents a configMap that should populate this volume<br />+optional |
| `vsphereVolume` | *[VsphereVirtualDiskVolumeSource](./k8s-io-api-core-v1.md#VsphereVirtualDiskVolumeSource) | No | vsphereVolume represents a vSphere volume attached and mounted on kubelets host machine.<br />Deprecated: VsphereVolume is deprecated. All operations for the in-tree vsphereVolume type<br />are redirected to the csi.vsphere.vmware.com CSI driver.<br />+optional |
| `quobyte` | *[QuobyteVolumeSource](./k8s-io-api-core-v1.md#QuobyteVolumeSource) | No | quobyte represents a Quobyte mount on the host that shares a pod's lifetime.<br />Deprecated: Quobyte is deprecated and the in-tree quobyte type is no longer supported.<br />+optional |
| `azureDisk` | *[AzureDiskVolumeSource](./k8s-io-api-core-v1.md#AzureDiskVolumeSource) | No | azureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.<br />Deprecated: AzureDisk is deprecated. All operations for the in-tree azureDisk type<br />are redirected to the disk.csi.azure.com CSI driver.<br />+optional |
| `photonPersistentDisk` | *[PhotonPersistentDiskVolumeSource](./k8s-io-api-core-v1.md#PhotonPersistentDiskVolumeSource) | No | photonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine.<br />Deprecated: PhotonPersistentDisk is deprecated and the in-tree photonPersistentDisk type is no longer supported. |
| `projected` | *[ProjectedVolumeSource](./k8s-io-api-core-v1.md#ProjectedVolumeSource) | No | projected items for all in one resources secrets, configmaps, and downward API |
| `portworxVolume` | *[PortworxVolumeSource](./k8s-io-api-core-v1.md#PortworxVolumeSource) | No | portworxVolume represents a portworx volume attached and mounted on kubelets host machine.<br />Deprecated: PortworxVolume is deprecated. All operations for the in-tree portworxVolume type<br />are redirected to the pxd.portworx.com CSI driver.<br />+optional |
| `scaleIO` | *[ScaleIOVolumeSource](./k8s-io-api-core-v1.md#ScaleIOVolumeSource) | No | scaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes.<br />Deprecated: ScaleIO is deprecated and the in-tree scaleIO type is no longer supported.<br />+optional |
| `storageos` | *[StorageOSVolumeSource](./k8s-io-api-core-v1.md#StorageOSVolumeSource) | No | storageOS represents a StorageOS volume attached and mounted on Kubernetes nodes.<br />Deprecated: StorageOS is deprecated and the in-tree storageos type is no longer supported.<br />+optional |
| `csi` | *[CSIVolumeSource](./k8s-io-api-core-v1.md#CSIVolumeSource) | No | csi (Container Storage Interface) represents ephemeral storage that is handled by certain external CSI drivers.<br />+optional |
| `ephemeral` | *[EphemeralVolumeSource](./k8s-io-api-core-v1.md#EphemeralVolumeSource) | No | ephemeral represents a volume that is handled by a cluster storage driver.<br />The volume's lifecycle is tied to the pod that defines it - it will be created before the pod starts,<br />and deleted when the pod is removed.<br /><br />Use this if:<br />a) the volume is only needed while the pod runs,<br />b) features of normal volumes like restoring from snapshot or capacity<br />   tracking are needed,<br />c) the storage driver is specified through a storage class, and<br />d) the storage driver supports dynamic volume provisioning through<br />   a PersistentVolumeClaim (see EphemeralVolumeSource for more<br />   information on the connection between this volume type<br />   and PersistentVolumeClaim).<br /><br />Use PersistentVolumeClaim or one of the vendor-specific<br />APIs for volumes that persist for longer than the lifecycle<br />of an individual pod.<br /><br />Use CSI for light-weight local ephemeral volumes if the CSI driver is meant to<br />be used that way - see the documentation of the driver for<br />more information.<br /><br />A pod can use both types of ephemeral volumes and<br />persistent volumes at the same time.<br /><br />+optional |
| `image` | *[ImageVolumeSource](./k8s-io-api-core-v1.md#ImageVolumeSource) | No | image represents an OCI object (a container image or artifact) pulled and mounted on the kubelet's host machine.<br />The volume is resolved at pod startup depending on which PullPolicy value is provided:<br /><br />- Always: the kubelet always attempts to pull the reference. Container creation will fail If the pull fails.<br />- Never: the kubelet never pulls the reference and only uses a local image or artifact. Container creation will fail if the reference isn't present.<br />- IfNotPresent: the kubelet pulls if the reference isn't already present on disk. Container creation will fail if the reference isn't present and the pull fails.<br /><br />The volume gets re-resolved if the pod gets deleted and recreated, which means that new remote content will become available on pod recreation.<br />A failure to resolve or pull the image during pod startup will block containers from starting and may add significant latency. Failures will be retried using normal volume backoff and will be reported on the pod reason and message.<br />The types of objects that may be mounted by this volume are defined by the container runtime implementation on a host machine and at minimum must include all valid types supported by the container image field.<br />The OCI object gets mounted in a single directory (spec.containers[*].volumeMounts.mountPath) by merging the manifest layers in the same way as for container images.<br />The volume will be mounted read-only (ro).<br />Sub path mounts for containers are not supported (spec.containers[*].volumeMounts.subpath) before 1.33.<br />The field spec.securityContext.fsGroupChangePolicy has no effect on this volume type.<br />+optional |

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

## VsphereVirtualDiskVolumeSource

Represents a vSphere volume resource.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `volumePath` | string | Yes | volumePath is the path that identifies vSphere volume vmdk |
| `fsType` | string | No | fsType is filesystem type to mount.<br />Must be a filesystem type supported by the host operating system.<br />Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.<br />+optional |
| `storagePolicyName` | string | No | storagePolicyName is the storage Policy Based Management (SPBM) profile name.<br />+optional |
| `storagePolicyID` | string | No | storagePolicyID is the storage Policy Based Management (SPBM) profile ID associated with the StoragePolicyName.<br />+optional |

## WeightedPodAffinityTerm

The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

| Stanza | Type | Required | Description |
|---|---|---|---|
| `weight` | int32 | Yes | weight associated with matching the corresponding podAffinityTerm,<br />in the range 1-100. |
| `podAffinityTerm` | [PodAffinityTerm](./k8s-io-api-core-v1.md#PodAffinityTerm) | Yes | Required. A pod affinity term, associated with the corresponding weight. |

## WindowsSecurityContextOptions

WindowsSecurityContextOptions contain Windows-specific options and credentials.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `gmsaCredentialSpecName` | *string | No | GMSACredentialSpecName is the name of the GMSA credential spec to use.<br />+optional |
| `gmsaCredentialSpec` | *string | No | GMSACredentialSpec is where the GMSA admission webhook<br />(https://github.com/kubernetes-sigs/windows-gmsa) inlines the contents of the<br />GMSA credential spec named by the GMSACredentialSpecName field.<br />+optional |
| `runAsUserName` | *string | No | The UserName in Windows to run the entrypoint of the container process.<br />Defaults to the user specified in image metadata if unspecified.<br />May also be set in PodSecurityContext. If set in both SecurityContext and<br />PodSecurityContext, the value specified in SecurityContext takes precedence.<br />+optional |
| `hostProcess` | *bool | No | HostProcess determines if a container should be run as a 'Host Process' container.<br />All of a Pod's containers must have the same effective HostProcess value<br />(it is not allowed to have a mix of HostProcess containers and non-HostProcess containers).<br />In addition, if HostProcess is true then HostNetwork must also be set to true.<br />+optional |


