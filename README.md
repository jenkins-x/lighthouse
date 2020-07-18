# Lighthouse

A lightweight ChatOps based webhook handler which can trigger Jenkins X Pipelines and soon Tekton Pipelines based on webhooks from multiple git providers such as: GitHub, GitHub Enterprise, BitBucket Server, BitBucket Cloud, GitLab, Gitea etc

## Building

To build the code clone git then type:

    make build
    
Then to run it:

    ./bin/lighthouse

## Environment variables

The following environment variables are used:

| Name  |  Description |
| ------------- | ------------- |
| `GIT_KIND` | the kind of git server: `github, bitbucket, gitea, stash` |
| `GIT_SERVER` | the URL of the server if not using the public hosted git providers: https://github.com or https://bitbucket.org https://gitlab.com |
| `GIT_USER` | the git user (bot name) to use on git operations |
| `GIT_TOKEN` | the git token to perform operations on git (add comments, labels etc) |
| `HMAC_TOKEN` | the token sent from the git provider in webhooks |
| `JX_SERVICE_ACCOUNT` | the service account to use for generated pipelines |


## Features 

Currently Lighthouse supports the common [Prow plugins](https://github.com/jenkins-x/lighthouse/tree/master/pkg/plugins) and handles push webhooks to branches to then trigger Jenkins X pipelines. 
    
Lighthouse uses the same `config.yaml` and `plugins.yaml` file structure from Prow so that we can easily migrate from `prow <-> lighthouse`. 

This also means we get to reuse the clean generation of Prow configuration from the `SourceRepository`, `SourceRepositoryGroup` and `Scheduler` CRDs integrated into [jx boot](https://jenkins-x.io/getting-started/boot/). e.g. here's the [default scheduler configuration](https://github.com/jenkins-x/jenkins-x-boot-config/blob/master/env/templates/default-scheduler.yaml) which is used for any project imported into your Jenkins X cluster; without you having to touch the actual prow configuration files. You can create many schedulers and associate them to different `SourceRepository` resources.   

We can also reuse Prow's capability of defining many separate pipelines on a repository (for PRs or releases) via having separate `contexts`. Then on a Pull Request we can use `/test something` or `/test all` to trigger pipelines and use the `/ok-to-test` and `/approve` or `/lgtm` commands 


## Comparisons to Prow

Lighthouse is very prow-like and currently reuses the Prow plugin source code and a bunch of [plugins from Prow](https://github.com/jenkins-x/lighthouse/tree/master/pkg/plugins)

Its got a few differences though:

* rather than be GitHub specific lighthouse uses [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) so it can support any git provider 
* lighthouse is mostly like `hook` from Prow; an auto scaling webhook handler - to keep the footprint small
* lighthouse is also very light. In Jenkins X we have about 10 pods related to prow; with lighthouse we have just 1 along with the tekton controller itself. That one lighthouse pod could easily be auto scaled too from 0 to many as it starts up very quickly.
* lighthouse focuses purely on Tekton pipelines so it does not require a `ProwJob` CRD; instead a push webhook to a release or pull request branch can trigger zero to many `PipelineRun` CRDs instead


## Porting Prow commands

If there are any prow commands you want which we've not yet ported over, its relatively easy to port prow plugins. 

We've reused the prow plugin code and configuration code; so its mostly a case of switching imports of `k8s.io/test-infra/prow` to `github.com/jenkins-x/lighthouse/pkg/prow` - then modifying the github client structs from, say, `github.PullRequest` to `scm.PullRequest`.

Most of the github structs map 1-1 with the [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) equivalents (e.g. Issue, Commit, PullRequest) though the go-scm API does tend to return slices to pointers to resources by default. There are some naming differences at different parts of the API though.

e.g. compare the `githubClient` API for the [prow lgtm](https://github.com/kubernetes/test-infra/blob/344024d30165cda6f4691cc178f25b16f1a1f5af/prow/plugins/lgtm/lgtm.go#L134-L150) versus the [lighthouse lgtm](https://github.com/jenkins-x/lighthouse/blob/master/pkg/prow/plugins/lgtm/lgtm.go#L135-L150).

All the prow plugin related code lives in the [tree of packages](https://github.com/jenkins-x/lighthouse/tree/master/pkg). Mostly all we've done is switch to using [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) and switch out the current prow agents and instead use a single `tekton` agent using the [PlumberClient](https://github.com/jenkins-x/lighthouse/blob/master/pkg/plumber/interface.go#L3-L6) to trigger pipelines.

## Testing Lighthouse

If you want to hack on lighthouse; such as to try it out with a specific git provider from [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) you can run it locally via:

    make build
    
Then to run it:

    ./bin/lighthouse
    
Then if you want to test it out with a git provider running on the cloud or inside Kubernetes you can use [ngrok](https://ngrok.com/) to setup a tunnel to your laptop. e.g.:

    ngrok http 8080
    
Now you can use your personal ngrok URL to register a webhook handler with your git provider. *NOTE* remember to append `/hook` to the generated ngrok URL. e.g. something like: https://7cc3b3ac.ngrok.io/hook

Any events that happen on your git provider should then trigger your local lighthouse.

## Debugging Lighthouse

You can setup a remote debugger for lighthouse using [delve](https://github.com/go-delve/delve/blob/master/Documentation/installation/README.md) via:

``` 
dlv --listen=:2345 --headless=true --api-version=2 exec ./bin/lighthouse -- $*        
```

You can then debug from your go based IDE (e.g. GoLand / IDEA / VS Code).

## Using a local go-scm

If you are hacking on support for a specific git provider you may find yourself hacking on the lighthouse code or the [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) code together.

Go modules lets you easily swap out the version of a dependency with a local copy of the code; so you can edit code in lighthouse and [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) at the same time.

Just add this line to the end of your [go.mod](https://github.com/jenkins-x/lighthouse/blob/master/go.mod) file:

```
replace github.com/jenkins-x/go-scm => /workspace/go/src/github.com/jenkins-x/go-scm
```  

Using the exact path to where you cloned [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm).

Then if you do:

    make build

It will uses your local [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) source.                                                                                              
