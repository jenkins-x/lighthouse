# Package github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1

- [PipelineResourceSpec](#PipelineResourceSpec)
- [PipelineResourceType](#PipelineResourceType)
- [ResourceParam](#ResourceParam)
- [SecretParam](#SecretParam)


## PipelineResourceSpec

PipelineResourceSpec defines  an individual resources used in the pipeline.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `description` | string | No | Description is a user-facing description of the resource that may be<br />used to populate a UI.<br />+optional |
| `type` | [PipelineResourceType](./github-com-tektoncd-pipeline-pkg-apis-resource-v1alpha1.md#PipelineResourceType) | Yes |  |
| `params` | [][ResourceParam](./github-com-tektoncd-pipeline-pkg-apis-resource-v1alpha1.md#ResourceParam) | No |  |
| `secrets` | [][SecretParam](./github-com-tektoncd-pipeline-pkg-apis-resource-v1alpha1.md#SecretParam) | No | Secrets to fetch to populate some of resource fields<br />+optional |

## PipelineResourceType

PipelineResourceType represents the type of endpoint the pipelineResource is, so that the<br />controller will know this pipelineResource shouldx be fetched and optionally what<br />additional metatdata should be provided for it.



## ResourceParam

ResourceParam declares a string value to use for the parameter called Name, and is used in<br />the specific context of PipelineResources.

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | Yes |  |
| `value` | string | Yes |  |

## SecretParam

SecretParam indicates which secret can be used to populate a field of the resource

| Stanza | Type | Required | Description |
|---|---|---|---|
| `fieldName` | string | Yes |  |
| `secretKey` | string | Yes |  |
| `secretName` | string | Yes |  |


