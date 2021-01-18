# Package k8s.io/apimachinery/pkg/apis/meta/v1

- [Duration](#Duration)
- [FieldsV1](#FieldsV1)
- [LabelSelector](#LabelSelector)
- [LabelSelectorOperator](#LabelSelectorOperator)
- [LabelSelectorRequirement](#LabelSelectorRequirement)
- [ManagedFieldsEntry](#ManagedFieldsEntry)
- [ManagedFieldsOperationType](#ManagedFieldsOperationType)
- [OwnerReference](#OwnerReference)
- [Time](#Time)


## Duration

Duration is a wrapper around time.Duration which supports correct<br />marshaling to YAML and JSON. In particular, it marshals into strings, which<br />can be used as map keys in json.



## FieldsV1

FieldsV1 stores a set of fields in a data structure like a Trie, in JSON format.<br /><br />Each key is either a '.' representing the field itself, and will always map to an empty set,<br />or a string representing a sub-field or item. The string will follow one of these four formats:<br />'f:<name>', where <name> is the name of a field in a struct, or key in a map<br />'v:<value>', where <value> is the exact json formatted value of a list item<br />'i:<index>', where <index> is position of a item in a list<br />'k:<keys>', where <keys> is a map of  a list item's key fields to their unique values<br />If a key maps to an empty Fields value, the field that key represents is part of the set.<br /><br />The exact format is defined in sigs.k8s.io/structured-merge-diff<br />+protobuf.options.(gogoproto.goproto_stringer)=false



## LabelSelector

A label selector is a label query over a set of resources. The result of matchLabels and<br />matchExpressions are ANDed. An empty label selector matches all objects. A null<br />label selector matches no objects.<br />+structType=atomic

| Stanza | Type | Required | Description |
|---|---|---|---|
| `matchLabels` | map[string]string | No | matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels<br />map is equivalent to an element of matchExpressions, whose key field is "key", the<br />operator is "In", and the values array contains only "value". The requirements are ANDed.<br />+optional |
| `matchExpressions` | [][LabelSelectorRequirement](./k8s-io-apimachinery-pkg-apis-meta-v1.md#LabelSelectorRequirement) | No | matchExpressions is a list of label selector requirements. The requirements are ANDed.<br />+optional |

## LabelSelectorOperator

A label selector operator is the set of operators that can be used in a selector requirement.



## LabelSelectorRequirement

A label selector requirement is a selector that contains values, a key, and an operator that<br />relates the key and values.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `key` | string | Yes | key is the label key that the selector applies to.<br />+patchMergeKey=key<br />+patchStrategy=merge |
| `operator` | [LabelSelectorOperator](./k8s-io-apimachinery-pkg-apis-meta-v1.md#LabelSelectorOperator) | Yes | operator represents a key's relationship to a set of values.<br />Valid operators are In, NotIn, Exists and DoesNotExist. |
| `values` | []string | No | values is an array of string values. If the operator is In or NotIn,<br />the values array must be non-empty. If the operator is Exists or DoesNotExist,<br />the values array must be empty. This array is replaced during a strategic<br />merge patch.<br />+optional |

## ManagedFieldsEntry

ManagedFieldsEntry is a workflow-id, a FieldSet and the group version of the resource<br />that the fieldset applies to.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `manager` | string | No | Manager is an identifier of the workflow managing these fields. |
| `operation` | [ManagedFieldsOperationType](./k8s-io-apimachinery-pkg-apis-meta-v1.md#ManagedFieldsOperationType) | No | Operation is the type of operation which lead to this ManagedFieldsEntry being created.<br />The only valid values for this field are 'Apply' and 'Update'. |
| `apiVersion` | string | No | APIVersion defines the version of this resource that this field set<br />applies to. The format is "group/version" just like the top-level<br />APIVersion field. It is necessary to track the version of a field<br />set because it cannot be automatically converted. |
| `time` | *[Time](./k8s-io-apimachinery-pkg-apis-meta-v1.md#Time) | No | Time is timestamp of when these fields were set. It should always be empty if Operation is 'Apply'<br />+optional |
| `fieldsType` | string | No | FieldsType is the discriminator for the different fields format and version.<br />There is currently only one possible value: "FieldsV1" |
| `fieldsV1` | *[FieldsV1](./k8s-io-apimachinery-pkg-apis-meta-v1.md#FieldsV1) | No | FieldsV1 holds the first JSON version format as described in the "FieldsV1" type.<br />+optional |

## ManagedFieldsOperationType

ManagedFieldsOperationType is the type of operation which lead to a ManagedFieldsEntry being created.



## OwnerReference

OwnerReference contains enough information to let you identify an owning<br />object. An owning object must be in the same namespace as the dependent, or<br />be cluster-scoped, so there is no namespace field.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `apiVersion` | string | Yes | API version of the referent. |
| `kind` | string | Yes | Kind of the referent.<br />More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds |
| `name` | string | Yes | Name of the referent.<br />More info: http://kubernetes.io/docs/user-guide/identifiers#names |
| `uid` | [UID](./k8s-io-apimachinery-pkg-types.md#UID) | Yes | UID of the referent.<br />More info: http://kubernetes.io/docs/user-guide/identifiers#uids |
| `controller` | *bool | No | If true, this reference points to the managing controller.<br />+optional |
| `blockOwnerDeletion` | *bool | No | If true, AND if the owner has the "foregroundDeletion" finalizer, then<br />the owner cannot be deleted from the key-value store until this<br />reference is removed.<br />Defaults to false.<br />To set this field, a user needs "delete" permission of the owner,<br />otherwise 422 (Unprocessable Entity) will be returned.<br />+optional |

## Time

Time is a wrapper around time.Time which supports correct<br />marshaling to YAML and JSON.  Wrappers are provided for many<br />of the factory methods that the time package offers.<br /><br />+protobuf.options.marshal=false<br />+protobuf.as=Timestamp<br />+protobuf.options.(gogoproto.goproto_stringer)=false




