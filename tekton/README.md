#### Bootstrapping and updating `Task`s and `Pipeline`s

Run `kubectl apply -f tekton/tasks/*.yaml` and `kubectl apply -f tekton/pipelines/*.yaml` to load initial versions of
the resources. Once they're in and the Lighthouse postsubmit is set properly, [the `lh-update-tekton-resources-pipeline`](https://github.com/jenkins-x/lighthouse/blob/master/tekton/pipelines/lh-update-tekton-resources-pipeline.yaml)
will automatically update the resources whenever the relevant directories have changed in a merged PR.

#### Tasks which need to be installed in the cluster from the catalog:
* https://raw.githubusercontent.com/tektoncd/catalog/master/task/git-batch-merge/0.2/git-batch-merge.yaml
* https://raw.githubusercontent.com/tektoncd/catalog/master/task/git-clone/0.2/git-clone.yaml
* https://raw.githubusercontent.com/tektoncd/catalog/master/task/kaniko/0.1/kaniko.yaml

#### Performing a release

Choose the revision to release and run, from the root directory of the repo:
```shell script
tkn pipeline start \
  --prefix-name=lighthouse-release- \
  --serviceaccount=TODO \
  --workspace=name=source,volumeClaimTemplateFile=./tekton/pvc/release-pvc.yaml \
  --param=version=0.1.1 \          # The new release, without the leading "v"
  --param=previous-version=0.1.0 \ # The previous release, also without the leading "v"
  --param=revision=(the SHA to release) \
  lh-release-pipeline
```

Watch the resulting `PipelineRun`'s log to make sure it passes, and then edit the draft release on GitHub.
