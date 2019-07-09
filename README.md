# Lighthouse

A lightweight webhook handler to trigger Jenkins X Pipelines from webhooks and support ChatOps for multiple git providers such as: GitHub, GitHub Enterprise, BitBucket Server, BitBucket Cloud, GitLab, Gitea etc

## Building

To build the code clone git then type:

    make build
    
Then to run it:

    ./bin/lighthouse


## Environment variables

The following environment variables are used:

| Name  |  Description |
| ------------- | ------------- |
| `BOT_NAME` | the bot user name to use on git operations |
| `GIT_KIND` | the kind of git server: `github, bitbucket, gitea, stash` |
| `GIT_SERVER` | the URL of the server if not using the public hosted git providers: https://github.com or https://bitbucket.org https://gitlab.com |
| `${GIT_KIND}_TOKEN` | the git token to perform operations on git (add comments, labels etc) |
| `HMAC_TOKEN` | the token sent from the git provider in webhooks |

    
## Comparisons to Prow

Lighthouse is very prow-like and currently reuses the Prow plugin mechanism and a bunch of [plugins]()

Its got a few differences though:

* rather than be GitHub specific lighthouse uses [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) to be able to support any git provider 
* lighthouse is mostly like `hook` from Prow; an auto scaling webhook handler - to keep the footprint small
* lighthouse focuses purely on Tekton pipelines so it does not require a `ProwJob` CRD; instead a push webhook to a release or pull request branch can trigger zero to many `PipelineRun` CRDs instead   