# Lighthouse

A lightweight webhook handler to trigger Jenkins X Pipelines from webhooks and support ChatOps for multiple git providers such as: GitHub, GitHub Enterprise, BitBucket Server, BitBucket Cloud, GitLab, Gitea etc

## Building

To build the code clone git then type:

    make build
    
Then to run it:

    ./bin/lighthouse hook
    
## Comparisons to Prow

Lighthouse is very prow-like and currently reuses the Prow plugin mechanism and a bunch of [plugins]()

Its got a few differences though:

* rather than be GitHub specific lighthouse uses [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) to be able to support any git provider 
* lighthouse is mostly like `hook` from Prow; an auto scaling webhook handler - to keep the footprint small
* lighthouse focuses purely on Tekton pipelines so it does not require a `ProwJob` CRD; instead a push webhook to a release or pull request branch can trigger zero to many `PipelineRun` CRDs instead
* lighthouse uses the Jenkins X CRDs for configuration: `SourceRepository`, `SourceRepositoryGroup` and `Schedule`       