# Package github.com/jenkins-x/lighthouse/pkg/config/job

- [PipelineKind](#PipelineKind)
- [PipelineRunParam](#PipelineRunParam)


## PipelineKind

PipelineKind specifies how the job is triggered.



## PipelineRunParam

PipelineRunParam represents a param used by the pipeline run

| Stanza | Type | Required | Description |
|---|---|---|---|
| `name` | string | No | Name is the name of the param |
| `value_template` | string | No | ValueTemplate is the template used to build the value from well know variables |


