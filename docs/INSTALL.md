## Lighthouse with Tekton Pipelines installation

* [Install Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) and [Tekton Dashboard](https://github.com/tektoncd/dashboard/blob/master/docs/install.md).
* Choose a bot user on the SCM provider and create an OAuth token for that user.
* Create a Kubernetes `Secret` for the bot user to be able to access git, [as described here](https://github.com/tektoncd/pipeline/blob/master/docs/auth.md#basic-authentication-git).
* Generate an HMAC secret key, which will be used for webhook configuration.
  * On Linux, you can generate the HMAC key by running:
```shell script
tr -dc 'A-F0-9' < /dev/urandom | head -c42
```
* Optionally, install the [nginx ingress controller](https://kubernetes.github.io/ingress-nginx/deploy/).
You can use any other ingress solution instead.
* Get the domain name for your cluster's external IP. If you don't have a domain, you can use nip.io.
* Create a `myvalues.yaml` Helm values file like the following:

```yaml
git:
  kind: github # or gitlab, or bitbucketserver
  name: github # or any other name you choose - this is just a label
  server: https://github.com # the base URL for your SCM provider

hmacToken: (the HMAC token you generated earlier)

user: (your bot user name)
oauthToken: (your bot's oauth token)

engines:
  tekton: true

tektoncontroller:
  dashboardURL: "https://url.for.your.tekton.dashboard/"

# If you want to set up your own ingress or use TLS, etc, set createIngress to false,
# and don't specify a domainName
createIngress: true
domainName: 1.2.3.4.nip.io

# If you don't want to create initial configmaps for config.yaml and plugins.yaml, omit this section.
configMaps:
  create: true
  configUpdater:
    orgAndRepo: foo/bar # The organization/owner and repository names for the repository that will contain
                        # the config.yaml and plugins.yaml to be automatically updated into the configmaps
                        # when they're changed in a merged PR going forward.
    path: a/b/c # The directory in that repository containing the config.yaml and plugins.yaml
```

* With Helm 3 installed, run:
```shell script
helm repo add lighthouse http://chartmuseum.jenkins-x.io
helm repo update
helm install -f myvalues.yaml --namespace (the namespace you want to install Lighthouse into) lighthouse lighthouse/lighthouse
```
* Note - to upgrade in the future, you will want to change `configMaps.create` in your values file to `false` or the configmaps will get overwritten. We're working on making this smarter.
* Create the config repository that will contain and update your ConfigMaps, matching the `orgAndRepo` and `path` specified above.
  * Copy the data within the `config` and `plugins` `ConfigMap`s that have been created in your cluster into `a/b/c/config.yaml` and `a/b/c/plugins.yaml` respectively.
* Push this repository and create a webhook for it, as described at [Webhook Creation](#webhook-creation).
* Create a new, or use an existing, repository for your project. Add an [`OWNERS`](https://github.com/jenkins-x/lighthouse/blob/master/OWNERS) file listing users able to review and approve PRs to this repository. Push that change to master for that repository.
* Create/add one or more `Pipeline`s and their relevant `Task`s for the pre and post submit builds you want to run for your repository. It is highly recommended that each `Pipeline`'s first `Task` be either [the `git-clone` task](https://github.com/tektoncd/catalog/tree/master/task/git-clone/0.1), for postsubmit pipelines, or [the `git-batch-merge` task](https://github.com/tektoncd/catalog/tree/master/task/git-batch-merge/0.2), for presubmit pipelines, from the [Tekton Catalog](https://github.com/tektoncd/catalog).
* Add your project repository to the `config.yaml` and `plugins.yaml` in your config repository, with `agent: tekton-pipeline` and a `pipeline_run_spec` specified. Open a PR with that change - when it's merged, the in-cluster `config` and `plugins` `ConfigMaps` will automatically be updated.
* [Create the webhook](#webhook-creation) for the project repository.
* Try opening a PR against the project repository!

## Webhook Creation

Create the webhook for a repository using the HMAC key you created earlier, and `(hook ingress URL)/hook` for the URL.
Make sure to use JSON for the delivery. For the event types for each provider:

### BitBucket Server Hooks
* `repo:refs_changed`
* `repo:modified`
* `repo:forked`
* `repo:comment:added`
* `repo:comment:edited`
* `repo:comment:deleted`
* `pr:opened`
* `pr:reviewer:approved`
* `pr:reviewer:unapproved`
* `pr:reviewer:needs_work`
* `pr:merged`
* `pr:declined`
* `pr:deleted`
* `pr:comment:added`
* `pr:comment:edited`
* `pr:comment:deleted`
* `pr:modified`
* `pr:from_ref_updated`

### GitHub or GitHub Enterprise Hooks
* `Send me everything`

Or you can exclude the following:

* `Deploy keys`
* `Deployment statuses`
* `Wiki`
* `Packages`
* `Page builds`
* `Project cards`
* `Project columns`
* `Registry packages`
* `Releases`
* `Repository vulnerability alerts`
* `Stars`
* `Watches`

### GitLab

* `Push events`
* `Tag push events`
* `Comments`
* `Confidential comments`
* `Issue events`
* `Confidential issue events`
* `Merge request events`

