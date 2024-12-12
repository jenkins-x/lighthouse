# Configuring Pipelines

We want to make it easy to configure pipelines inside each git repository. We also want to make it easy to share pipelines across 

## In Repo configuration

If your repository is configured in lighthouse for in repo so that your repository `myorg/myowner` appears in the configuration like this:


```yaml 
branch-protection:
  protect-tested-repos: true
github:
  LinkURL: null
in_repo_config:
  enabled:
    ...
    myorg/myowner: true
    ...
```

Then lighthouse will find all of the `.lighthouse/*/triggers.yaml` files and use those to setup `presubmits` and `postsubmits`.


## Using existing pipeline tasks and steps

### Using a versioned `Task` / `Pipeline` / `PipelineRun`

Inside the `triggers.yaml` you can use the `source` property to reference a local file or a versioned file in git

```yaml
  presubmits:
  # use a local file
  - name: local-file
    source: pullrequest.yaml
    
  # use any URL  
  - name: url
    source: https://foo.bar.com/something/mytask.yaml
   
  # use a git URI with version information  
  - name: git
    source: jenkins-x/jx3-pipeline-catalog/packs/javascript/.lighthouse/jenkins-x/pullrequest.yaml@v1.2.3
```

That is a good approach for referencing a versioned `Task` / `Pipeline` / `PipelineRun` but it doesn't offer any ability to make local changes or reuse individual steps.


### Source URI

The source URI syntax we use is:
        
* treat it as a URL if it contains `://` 
* if the string contains `@` then it's a git URI of the form: `owner/repository/pathToFile@versionBranchOrSha`
  * the repository is assumed to live on `github.com` like GitHub Actions. If you wish to use your local repository prefix the sourceURI with the server name. e.g.  `lighthouse:owner/repository/pathToFile@versionBranchOrSha` will reference a git repository in the current lighthouse git repository instead of github.com
  * you can use `@HEAD` to mean the latest version from the main branch
  * you can use `@versionStream` to mean the git SHA of this git repository configured inside your version stream if available; otherwise it defaults to `@HEAD`
* otherwise assume the path is a local relative file in git


### Referencing Steps inside a `Task` / `Pipeline` / `PipelineRun`

With this approach we reference a `Task` / `Pipeline` / `PipelineRun` inside a single Step using the following syntax.

This borrows on the idea from [ko](https://github.com/google/ko) and [mink](https://github.com/mattmoor/mink) of reusing the **source URI** in the `image:` property using the special **uses:** prefix. 

So we use `image: uses:sourceURI` to enable including versioned steps inside a `Task` / `Pipeline` / `PipelineRun` :
     
```yaml 
steps:

# use a local file
- image: uses:pullrequest.yaml 

# use a URL 
- image: uses:https://foo.bar.com/something/mytask.yaml 

# use a git URI with version information
- image: uses:jenkins-x/jx3-pipeline-catalog/packs/javascript/.lighthouse/jenkins-x/pullrequest.yaml@v1.2.3
```

e.g. here is [an example](../pkg/triggerconfig/inrepo/test_data/load_pipelinerun/uses-all-steps/source.yaml#L8) which generates this [PipelineRun](../pkg/triggerconfig/inrepo/test_data/load_pipelinerun/uses-all-steps/expected.yaml#L154)

### Referencing named steps

But what if we want to override / customise a specific step or add extra steps before, after or in between?

If we include the `name:`  property we can list each individual step to import (and change the order if required).

```yaml 
apiVersion: tekton.dev/v1
kind: PipelineRun
spec:
  pipelineSpec:
    tasks:
    - name: from-build-pack
      taskSpec:
        steps:
        - image: uses:jenkins-x/jx3-pipeline-catalog/packs/javascript/.lighthouse/jenkins-x/pullrequest.yaml@v1.2.3
          name: jx-variables
        - image: uses:jenkins-x/jx3-pipeline-catalog/packs/javascript/.lighthouse/jenkins-x/pullrequest.yaml@v1.2.3
          name: build-npm-install
        - image: uses:jenkins-x/jx3-pipeline-catalog/packs/javascript/.lighthouse/jenkins-x/pullrequest.yaml@v1.2.3
          name: build-npm-test
        - image: uses:jenkins-x/jx3-pipeline-catalog/packs/javascript/.lighthouse/jenkins-x/pullrequest.yaml@v1.2.3
          name: build-container-build
        - image: uses:jenkins-x/jx3-pipeline-catalog/packs/javascript/.lighthouse/jenkins-x/pullrequest.yaml@v1.2.3
          name: promote-jx-preview
```
        

### Concise syntax 

Copying the `image: uses:` line for every step can be a little noisy so you can use the more concise format where you use the `stepTemplate.image` to configure the uses image. 

Then any step which does not have an image will default to reuse the `stepTemplate.image` value:


```yaml 
apiVersion: tekton.dev/v1
kind: PipelineRun
spec:
  pipelineSpec:
    tasks:
    - name: from-build-pack
      taskSpec:
        stepTemplate:
          image: uses:jenkins-x/jx3-pipeline-catalog/packs/javascript/.lighthouse/jenkins-x/pullrequest.yaml@v1.2.3
        steps:
        - name: jx-variables
        - name: build-npm-install
        - name: build-npm-test
        - name: build-container-build
        - name: promote-jx-preview
```


e.g. here is [an example](../pkg/triggerconfig/inrepo/test_data/load_pipelinerun/uses-steps/source.yaml#L8) which uses different URI formats and generates this [PipelineRun](../pkg/triggerconfig/inrepo/test_data/load_pipelinerun/uses-steps/expected.yaml#L154)


#### Including new steps in between

You can then add extra steps in between these `uses:` steps if you wish to customise the steps in any way - e.g. see `my-prefix-step` which has an explicit `image:` value so isn't inherited from the `stepTemplate.image`


```yaml 
apiVersion: tekton.dev/v1
kind: PipelineRun
spec:
  pipelineSpec:
    tasks:
    - name: from-build-pack
      taskSpec:
        stepTemplate:
          image: uses:jenkins-x/jx3-pipeline-catalog/packs/javascript/.lighthouse/jenkins-x/pullrequest.yaml@v1.2.3
        steps:
        - image: node:12-slim
          name: my-prefix-step
          script: |
            #!/bin/sh
            npm something        
        - name: jx-variables
        - name: build-npm-install
        - name: build-npm-test
        - name: build-container-build
        - name: promote-jx-preview
```

e.g. here is [an example](../pkg/triggerconfig/inrepo/test_data/load_pipelinerun/uses-steps-add-custom/source.yaml#L8) which generates this [PipelineRun](../pkg/triggerconfig/inrepo/test_data/load_pipelinerun/uses-steps-add-custom/expected.yaml#L154)



#### Overriding steps

You may wish to modify a step while still reusing the volumes, environment variables and so forth. e.g. to modify just the command line you can do the following:

```yaml 
apiVersion: tekton.dev/v1
kind: PipelineRun
spec:
  pipelineSpec:
    tasks:
    - name: from-build-pack
      taskSpec:
        steps:
        - image: uses:jenkins-x/jx3-pipeline-catalog/packs/javascript/.lighthouse/jenkins-x/pullrequest.yaml@v1.2.3
          name: jx-variables
          script: |
            #!/usr/bin/env sh
            echo my replacement command script goes here
```

Any extra properties in the steps are used to override the underlying uses step.

If you wish to change the `image:`  then currently just copy and paste the entire step inline in your task.

e.g. here is [an example](../pkg/triggerconfig/inrepo/test_data/load_pipelinerun/uses-steps-override/source.yaml#L8) which generates this [PipelineRun](../pkg/triggerconfig/inrepo/test_data/load_pipelinerun/uses-steps-override/expected.yaml#L154)


#### Including multiple copies of a step

You may wish to reuse a step multiple times in a pipeline with different parameters. e.g. override the `script`, `command`, `args` or add a different `env` variable value.

For example if you want to build multiple images reusing the same container build step but just changing the image name and/or `Dockefile`

You can use this by using a custom `name` syntax. Use the name of the step you wish to inherit and then add `:something` as a suffix.

```yaml 
apiVersion: tekton.dev/v1
kind: PipelineRun
spec:
  pipelineSpec:
    tasks:
    - name: from-build-pack
      taskSpec:
        steps:
        # lets reuse the 'build-container-build' step 3 times with different values
        - name: build-container-build
        - name: build-container-build:cheese
          env:
            - name: IMAGE
              value: gcr.io/myproject/myimage
        - name: build-container-build:wine
          env:
            - name: IMAGE
              value: gcr.io/myproject/wine
        - name: promote-jx-preview
```

e.g. here is [an example](../pkg/triggerconfig/inrepo/test_data/load_pipelinerun/uses-steps-multiple-copies/source.yaml#L8) which generates this [PipelineRun](../pkg/triggerconfig/inrepo/test_data/load_pipelinerun/uses-steps-multiple-copies/expected.yaml#L154)
