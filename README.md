# Lighthouse

Lighthouse is a lightweight ChatOps based webhook handler which can trigger Jenkins X Pipelines, Tekton Pipelines or Jenkins Jobs based on webhooks from multiple git providers such as GitHub, GitHub Enterprise, BitBucket Server and GitLab.

<!-- MarkdownTOC autolink="true" indent="  " -->

- [Installing](#installing)
- [Background](#background)
  - [Comparisons to Prow](#comparisons-to-prow)
  - [Porting Prow commands](#porting-prow-commands)
- [Development](#development)
  - [Building](#building)
  - [Environment variables](#environment-variables)
  - [Testing](#testing)
  - [Debugging Lighthouse](#debugging-lighthouse)
  - [Using a local go-scm](#using-a-local-go-scm)

<!-- /MarkdownTOC -->

## Installing

Lighthouse is bundled and released as [Helm Chart](https://helm.sh/docs/topics/charts/).
You find the install instruction in the Chart's [README](./charts/lighthouse/README.md).

Depending on the pipeline engine you want to use, you can find more detailed instructions in one of the following documents:

- [Lighthouse + Tekton](./docs/install_lighthouse_with_tekton.md)
- [Lighthouse + Jenkins](./docs/install_lighthouse_with_jenkins.md)

## Background

Lighthouse derived originally from [Prow](https://github.com/kubernetes/test-infra/tree/master/prow) and started with a copy of its essential code.

Currently, Lighthouse supports the standard [Prow plugins](https://github.com/jenkins-x/lighthouse/tree/master/pkg/plugins) and handles push webhooks to branches to then trigger a pipeline execution on the agent of your choice.

Lighthouse uses the same `config.yaml` and `plugins.yaml` for configuration than Prow.

### Comparisons to Prow

Lighthouse reuses the Prow plugin source code and a bunch of [plugins from Prow](https://github.com/jenkins-x/lighthouse/tree/master/pkg/plugins)

Its got a few differences though:

- rather than being GitHub specific Lighthouse uses [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) so it can support any Git provider
- Lighthouse does not use a `ProwJob` CRD; instead, it has its own `LighthouseJob` CRD.

### Porting Prow commands

If there are any prow commands you want which we've not yet ported over, it is relatively easy to port Prow plugins.

We've reused the prow plugin code and configuration code; so it is mostly a case of switching imports of `k8s.io/test-infra/prow` to `github.com/jenkins-x/lighthouse/pkg/prow`, then modifying the GitHub client structs from, say, `github.PullRequest` to `scm.PullRequest`.

Most of the GitHub structs map 1-1 to the [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) equivalents (e.g. Issue, Commit, PullRequest).
However, the go-scm API does tend to return slices to pointers to resources by default.
There are some naming differences in different parts of the API as well.
For example, compare the `githubClient` API for [Prow lgtm](https://github.com/kubernetes/test-infra/blob/344024d30165cda6f4691cc178f25b16f1a1f5af/prow/plugins/lgtm/lgtm.go#L134-L150) versus the [Lighthouse lgtm](https://github.com/jenkins-x/lighthouse/blob/master/pkg/plugins/lgtm/lgtm.go#L146-L163).

## Development

### Building

To build the code, [fork and clone](https://docs.github.com/en/github/getting-started-with-github/fork-a-repo) this git repository, then type:

```bash
make build
```

`make build` will build all relevant Lighthouse binaries natively for your OS which you then can run locally.
For example, to run the webhook controller, you would type:

```bash
./bin/webhooks
```

To see which other Make rules are available, run:

```bash
make help
```

### Environment variables

While Prow only supports GitHub as SCM provider, Lighthouse supports several Git SCM providers.
Lighthouse achieves the abstraction over the SCM provider using the [go-scm](https://github.com/jenkins-x/go-scm) library.
To configure your SCM, go-scm uses the following environment variables :

| Name  |  Description |
| ------------- | ------------- |
| `GIT_KIND` | the kind of git server: `github, gitlab, bitbucket, gitea, stash` |
| `GIT_SERVER` | the URL of the server if not using the public hosted git providers: [https://github.com](https://github.com), [https://bitbucket.org](https://bitbucket.org) or [https://gitlab.com](https://gitlab.com) |
| `GIT_USER` | the git user (bot name) to use on git operations |
| `GIT_TOKEN` | the git token to perform operations on git (add comments, labels etc.) |
| `HMAC_TOKEN` | the token sent from the git provider in webhooks |

### Testing

To run the unit tests, type:

```bash
make test
```

For development purposes, it is also nice to start an instance of the binary you want to work.
Provided you have a connection to a cluster with Lighthouse installed, the locally started controller will join the cluster, and you can test your development changes directly in-cluster.

For example to run the webhook controller locally:

```bash
make build
GIT_TOKEN=<git-token> ./bin/webhooks -namespace <namespace> -bot-name <git-bot-user>
```

In the case of the webhook controller, you can also test webhook deliveries locally using a [ngrok](https://ngrok.com/) tunnel.
Install [ngrok](https://ngrok.com/) and start a new tunnel:

```bash
$ ngrok http 8080
ngrok by @inconshreveable                                                                                                                                                                     (Ctrl+C to quit)

Session Status                online
Account                       ***
Version                       2.3.35
Region                        United States (us)
Web Interface                 http://127.0.0.1:4040
Forwarding                    http://e289dd1e1245.ngrok.io -> http://localhost:8080
Forwarding                    https://e289dd1e1245.ngrok.io -> http://localhost:8080

Connections                   ttl     opn     rt1     rt5     p50     p90
                              0       0       0.00    0.00    0.00    0.00
```
  
Now you can use your ngrok URL to register a webhook handler with your Git provider.

**NOTE** Remember to append `/hook` to the generated ngrok URL.
 In the case of the above example ht<span>tp://e289dd1e1245.ngrok.io/hook

Any events that happen on your Git provider are now sent to your local webhook instance.

### Debugging Lighthouse

You can setup a remote debugger for Lighthouse using [delve](https://github.com/go-delve/delve/blob/master/Documentation/installation/README.md) via:

```bash
dlv --listen=:2345 --headless=true --api-version=2 exec ./bin/lighthouse -- $*  
```

You can then debug from your Go-based IDE (e.g. GoLand / IDEA / VS Code).

### Debugging webhooks

If you want to debug lighthouse locally from webhooks in your cluster there are a couple of tools that could help:

#### Localizer

If you [install localizer](https://github.com/jaredallard/localizer#install-localizer) (see [the blog for more detail](https://blog.jaredallard.me/localizer-an-adventure-in-creating-a-reverse-tunnel-and-tunnel-manager-for-kubernetes/) you can easily debug webhooks on your cluster.

* first run localizer:

```bash 
sudo localizer
```

Then run/debug lighthouse locally. 

e.g. in your IDE run the [cmd/webhooks/main.go](https://github.com/jenkins-x/lighthouse/blob/master/cmd/webhooks/main.go) (passing `--namespace jx` as program arguments)

Then to get the webhooks to trigger your local process:

```bash 
localizer expose jx/hook --map 80:8080
```

when you have finished debugging, return things to normal via:

```bash 
localizer expose jx/hook --stop
```


#### Telepresence 
You can replace the running version in your cluster with the one running locally using [telepresence](https://www.telepresence.io/).  
First install the [telepresence cli](https://www.telepresence.io/docs/latest/install/) on your device then [the traffic-manager](https://www.telepresence.io/docs/latest/install/helm/) into your cluster 
and connect to the cluster:
```bash
telepresence connect
```
For webhooks, just run:
```bash
telepresence intercept lighthouse-webhooks --namespace=jx --port 80 --env-file=/tmp/webhooks-env
```
in another terminal:
```bash
export $(cat /tmp/webhooks-env | xargs)
dlv --listen=:2345 --headless=true --api-version=2 exec ./bin/webhooks -- --namespace=jx
```

You can do the same for any other deployment (keeper, foghorn...), just make sur to check the command args used for it an set them instead of `--namespace=jx`.

to stop intercepting:
```bash
telepresence leave hook-jx # hook-jx is the name of the intercept
```


### Using a local go-scm

If you are hacking on support for a specific Git provider, you may find yourself working on the Lighthouse code and the [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) code together.
Go modules lets you easily swap out the version of a dependency with a local copy of the code; so you can edit code in Lighthouse and [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm) at the same time.

Just add this line to the end of your [go.mod](https://github.com/jenkins-x/lighthouse/blob/master/go.mod) file:

```bash
replace github.com/jenkins-x/go-scm => /workspace/go/src/github.com/jenkins-x/go-scm
```  

Using the exact path to where you cloned [jenkins-x/go-scm](https://github.com/jenkins-x/go-scm)
