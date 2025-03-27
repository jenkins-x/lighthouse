
## Changes in version 1.23.5

### Tests

* make tests pass about activityRecord (JordanGoasdoue)

### Chores

* upgrade tektoncd to latest 0.69.1 (JordanGoasdoue)
* tests: fix remaining tests (JordanGoasdoue)
* periodic: make test pass with new taskruntemplate (JordanGoasdoue)
* breakpoint: make test pass with updated struct (JordanGoasdoue)
* normalize actual and expected data to avoid diff when indent (JordanGoasdoue)
* convert from v1beta1 to v1 pipeline/task loaded from ref (JordanGoasdoue)
* support v1 and v1beta1 (JordanGoasdoue)
* use lighthousev1alpha1 (JordanGoasdoue)
* pass tektonClient from param in ConvertPipelineRun (JordanGoasdoue)
* add taskruns get permission (JordanGoasdoue)
* add scheme on Manager config (JordanGoasdoue)
* remove deprecated PipelineResources (JordanGoasdoue)
* get taskruns from clients instead of pr.Status.TaskRuns (JordanGoasdoue)
* replace ArrayOrString with ParamValue (JordanGoasdoue)
* use Timeouts.Pipeline instead of Timeout (JordanGoasdoue)
* manually edit zz_generated because controller-gen fails (JordanGoasdoue)
* regenerate crds with make crd-manifests (JordanGoasdoue)
* use tektoncd/pipeline/pkg/apis/pipeline/v1 (JordanGoasdoue)
* use pipelinev1beta1 everywhere (JordanGoasdoue)
* upgrade tekton pipeline to 0.65.3 with dependencies upgrades (JordanGoasdoue)

### Other Changes

These commits did not use [Conventional Commits](https://conventionalcommits.org/) formatted messages:

* chore(load_pipelinerun): convert the expected to v1 (JordanGoasdoue)
* chore(load_pipelinerun): show how to validate one uses test (JordanGoasdoue)
