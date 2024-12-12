# Lighthouse with Tekton Pipelines

<!-- MarkdownTOC autolink="true" -->

- [Objective](#objective)
- [Prerequisite](#prerequisite)
- [Installation](#installation)
  - [Lighthouse](#lighthouse)
  - [Tekton Pipelines](#tekton-pipelines)
    - [Prepare the test project](#prepare-the-test-project)
    - [Configure a Tekton Pipeline](#configure-a-tekton-pipeline)
    - [Configure Lighthouse to run the pipeline](#configure-lighthouse-to-run-the-pipeline)
- [Next steps](#next-steps)
- [Webhook types](#webhook-types)
  - [BitBucket Server Hooks](#bitbucket-server-hooks)
  - [GitHub or GitHub Enterprise Hooks](#github-or-github-enterprise-hooks)
  - [GitLab](#gitlab)

<!-- /MarkdownTOC -->

## Objective

The objective of these instructions is to create a sample project where pull requests merge by GitOps commands like `lgtm` or `approve`, placed as comments on pull requests of your SCM provider.
Also, each time a pull request merges, a Tekton Pipeline is triggered to build the `master` of your project.
This is achieved by using Lighthouse and Tekton Pipelines together.

The configuration of Lighthouse will be managed in a dedicated repository which itself is managed via GitOps.
Changes to the configuration in this repository will automatically synchronize to the in-cluster configuration of Lighthouse.

---

**NOTE**:
The examples are using GitHub as SCM provider, but you can choose any of the Lighthouse supported SCM providers.
In this case you need to modify the relevant settings accordingly.

---

## Prerequisite

- [Helm 3](https://helm.sh/docs/intro/install/)
- Optionally [gh](https://github.com/cli/cli) to automate the required GitHub interaction.
  Instead of `gh` you can use the web UI to create repositories, setup webhooks, etc.
- Optionally [pack](https://buildpacks.io/docs/install-pack/) for using Cloud Native Buildpacks to build the sample application.
- A Kubernetes cluster version 1.16 or later (provided you want to use the latest version of Tekton).
  - Ingress enabled, for example, by using the [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/deploy/).
    You can use any other ingress solution instead.
  - The domain name for your cluster's external IP. If you don't have a domain, you can use [nip.io](https://nip.io/).  
- [Tekton Pipelines](https://github.com/tektoncd/pipeline/blob/master/docs/install.md) installed on the cluster.
  - Optionally [Tekton Dashboard](https://github.com/tektoncd/dashboard/blob/master/docs/install.md).
- A dedicated user for the SCM provider of your choice used as _bot user_.
  - An OAuth token for this bot user. In the case of GitHub refer to [Creating a personal access token
](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token).
- A self-generated secret for securing webhook delivery.
  You can, for example, generate a secret by running:

  ```bash
    ruby -rsecurerandom -e 'puts SecureRandom.hex(42)'
   ```

   See also [Securing your webhooks](https://developer.github.com/webhooks/securing/).

## Installation

### Lighthouse

Let's start with the Lighthouse.
Create a git repository with an initial Lighthouse configuration and install Lighthouse using Helm.
After the installation, any changes merged into the _master_ branch of this configuration repository will be applied to the in-cluster configuration of Lighthouse.

- Let's start with creating a Git repository containing a minimal Lighthouse configuration.
  In this example, the bot user is the owner of the repository.
  This is not mandatory, but the bot user needs to have access to the repository.
  Note that we are providing a second username `approver`.
  This is the user allowed to merge pull-request via the `approve` command.
  The approver must differ from the bot user.
  The bot user cannot approve pull-requests.

    ```bash
    bot_user=<github-username-of-bot-user>
    repo_name=lighthouse-config
    approver=<github-username-of-initial-approver>
  
    mkdir $repo_name
    cd $repo_name
    git init
  
    cat > config.yaml <<EOF
    pod_namespace: lighthouse
    prowjob_namespace: lighthouse
    tide:
      queries:
      - labels:
        - approved
        repos:
        - $bot_user/$repo_name
    EOF
  
    cat > plugins.yaml <<EOF
    approve:
    - lgtm_acts_as_approve: false
      repos:
      - $bot_user/$repo_name
      require_self_approval: true
    config_updater:
      gzip: false
      maps:
        config.yaml:
          name: config
        plugins.yaml:
          name: plugins  
    plugins:
      $bot_user/$repo_name:
      - config-updater
      - approve
      - lgtm  
    EOF
  
    cat > OWNERS <<EOF
    approvers:
    - $approver
    reviewers:
    - $approver
    EOF
  
    git add .
    git commit -m "Initial Lighthouse config"
    ```

- So far the Lighthouse configuration repository is only local, let's push it to GitHub.
  The _approver_ needs to become a collaborator on the repository as well.

    ```bash
    bot_user=<github-username-of-bot-user>
    repo_name=lighthouse-config
    GITHUB_TOKEN=<oauthtoken-bot-user>

    gh repo create $repo_name --public -y
    gh api -X PUT repos/$bot_user/$repo_name/collaborators/$approver
    git push origin master
    ```

- Manually create the initial Lighthouse configuration maps _config_ and _plugins_ to seed the installation.
  Once the installation succeeded, we will make further changes to the configuration via pull request to the configuration repository.

    ```bash
    install_namespace=lighthouse

    kubectl create namespace $install_namespace
    kubectl create cm config -n $install_namespace --from-file=config.yaml
    kubectl create cm plugins -n $install_namespace --from-file=plugins.yaml
    ```

- Configure the Helm chart repository

    ```bash
    helm repo add lighthouse https://jenkins-x-charts.github.io/repo
    helm repo update
    ```

- Install Lighthouse using Helm.

    ```bash
    install_namespace=lighthouse
    bot_user=<github-username-of-bot-user>
    bot_token=<oauthtoken-bot-user>
    webhook_secret=<generated-webhook-secret>
    tekton_dash_url=<tekton-url>
    domain=<k8s-cluster-domain>
  
    helm install lighthouse lighthouse/lighthouse -n ${install_namespace} -f <(cat <<EOF
    git:
      kind: github
      name: github
      server: https://github.com

    user: "${bot_user}"
    oauthToken: "${bot_token}"
    hmacToken: "${webhook_secret}"

    cluster:
      crds:
        create: true

    tektoncontroller:
      dashboardURL: $tekton_dash_url

    engines:
      jx: false
      tekton: true
  
    webhooks:
      ingress:
        enabled: true
        annotations:
          kubernetes.io/ingress.class: "nginx"
        hosts:
        - "$install_namespace.$domain"
    EOF
    )
    ```

- Last but not least, make sure that any events in the configuration repository are propagated to Lighthouse via a webhook.

    ```bash
    GITHUB_TOKEN=<oauthtoken-bot-user>

    install_namespace=lighthouse
    bot_user=<github-username-of-bot-user>
    repo_name=lighthouse-config
    webhook_url=$(kubectl get ingress -n $install_namespace -o=jsonpath='http://{.items[0].spec.rules[0].host}'/hook)
    webhook_secret=<generated-webhook-secret>

    cat <<EOF | gh api -X POST repos/$bot_user/$repo_name/hooks --input -
    {
      "name": "web",
      "active": true,
      "events": [
        "*"
      ],
      "config": {
        "url": "$webhook_url",
        "content_type": "json",
        "insecure_ssl": "0",
        "secret": "$webhook_secret"
      }
    }
    EOF
    ```

### Tekton Pipelines

Now that we have Lighthouse installed, we can create a sample project and configure it to be used with Tekton and Lighthouse.

#### Prepare the test project

- Let start with creating a minimal hello world web server written in Go.

    ```bash
    bot_user=<github-username-of-bot-user>
    repo_name=hello
    approver=<github-username-of-initial-approver>
  
    mkdir $repo_name
    cd $repo_name
    git init
  
    cat > main.go <<EOF
    package main

    import (
        "fmt"
        "net/http"
    )
  
    func main() {
        http.HandleFunc("/", HelloServer)
        http.ListenAndServe(":8080", nil)
    }
  
    func HelloServer(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, %s!", r.URL.Path[1:])
    }
    EOF
  
    cat > OWNERS <<EOF
    approvers:
    - $approver
    reviewers:
    - $approver
    EOF
  
    git add .
    git commit -m "Initial import"
    ```

- Push the code to GitHub.

    ```bash
    GITHUB_TOKEN=<oauthtoken-bot-user>

    gh repo create $repo_name --public -y
    gh api -X PUT repos/$bot_user/$repo_name/collaborators/$approver
    git push origin master
    ```

- You can verify that the project builds.

  ```bash
  pack build hello --builder paketobuildpacks/builder:tiny
  docker run --rm -p 8080:8080 hello
  curl http://localhost:8080/world
  ```

  You should be able to see "Hello world!".

#### Configure a Tekton Pipeline  

- To build the project using Tekton Pipelines first create two Kubernetes Secrets, one for giving Tekton access to the Git repository and the other to be able to push images to DockerHub.
  For more information, refer to the [Authentication at Run Time](https://github.com/tektoncd/pipeline/blob/master/docs/auth.md#basic-authentication-git) section of the Tekton Pipelines documentation.

    ```bash
    github_user=<github-username-of-bot-user>
    github_pass=<oauthtoken-bot-user>
    docker_user=<docker-hub-user>
    docker_pass=<docker-hub-pass>
  
    cat <<EOF | kubectl apply -f -
    apiVersion: v1
    kind: Secret
    metadata:
      name: dockerhub
      annotations:
        tekton.dev/docker-0: https://index.docker.io/v1/
    type: kubernetes.io/basic-auth
    stringData:
      username: $docker_user
      password: $docker_pass
    EOF
  
    cat <<EOF | kubectl apply -f -
    apiVersion: v1
    kind: Secret
    metadata:
      name: github
      annotations:
        tekton.dev/git-0: https://github.com
    type: kubernetes.io/basic-auth
    stringData:
      username: $github_user
      password: $github_pass
    EOF  
    ```

- To build the pipeline, you need the _git-clone_ and _buildpack_ Tasks from the Tekton Task Catalog.

    ```bash
    install_namespace=lighthouse

    kubectl apply -n $install_namespace -f https://raw.githubusercontent.com/tektoncd/catalog/master/task/git-clone/0.2/git-clone.yaml
    kubectl apply -n $install_namespace -f https://raw.githubusercontent.com/tektoncd/catalog/master/task/buildpacks/0.1/buildpacks.yaml  
    ```

- Now you can create the actual Tekton Pipeline

    ```bash
    docker_user=<docker-hub-user>
    image_name=<image-name>
    repo_owner=<github-username-of-bot-user>
    repo_name=hello

    cat <<EOF | kubectl apply -f -
    apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: app-builder
    secrets:
      - name: github
      - name: dockerhub
    ---
    apiVersion: tekton.dev/v1
    kind: PipelineResource
    metadata:
      name: buildpacks-app-image
    spec:
      type: image
      params:
        - name: url
          value: ${docker_user}/${image_name}:latest
    ---
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: buildpacks-source-pvc
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 500Mi
    ---
    apiVersion: v1
    kind: PersistentVolumeClaim
    metadata:
      name: buildpacks-cache-pvc
    spec:
      accessModes:
        - ReadWriteOnce
      resources:
        requests:
          storage: 500Mi
    ---
    apiVersion: tekton.dev/v1
    kind: Pipeline
    metadata:
      name: ${repo_name}-pipeline
    spec:
      workspaces:
        - name: shared-workspace
      resources:
        - name: build-image
          type: image
      tasks:
        - name: fetch-repository
          taskRef:
            name: git-clone
          workspaces:
            - name: output
              workspace: shared-workspace
          params:
            - name: url
              value: https://github.com/$repo_owner/$repo_name
            - name: deleteExisting
              value: "true"
        - name: buildpacks
          taskRef:
            name: buildpacks
          runAfter:
            - fetch-repository
          workspaces:
            - name: source
              workspace: shared-workspace
          params:
            - name: BUILDER_IMAGE
              value: paketobuildpacks/builder:tiny
            - name: CACHE
              value: buildpacks-cache
          resources:
            outputs:
              - name: image
                resource: build-image
    EOF
    ```

   TIP: It is highly recommended that each `Pipeline`'s first `Task` be either the [`git-clone` ](https://github.com/tektoncd/catalog/tree/master/task/git-clone/0.1) task, for postsubmit pipelines, or the [`git-batch-merge`](https://github.com/tektoncd/catalog/tree/master/task/git-batch-merge/0.2) task, for presubmit pipelines, from the [Tekton Catalog](https://github.com/tektoncd/catalog).

- You can verify that the pipeline works by manually creating a `PipelineRun`.

    ```bash
    repo_name=hello

    cat <<EOF | kubectl apply -f -
    apiVersion: tekton.dev/v1
    kind: PipelineRun
    metadata:
      name: ${repo_name}-pipeline-run
    spec:
      serviceAccountName: app-builder
      pipelineRef:
        name: ${repo_name}-pipeline
      workspaces:
        - name: shared-workspace
          persistentvolumeclaim:
            claimName: buildpacks-source-pvc
      resources:
        - name: build-image
          resourceRef:
            name: buildpacks-app-image
      podTemplate:
        volumes:
          - name: buildpacks-cache
            persistentVolumeClaim:
              claimName: buildpacks-cache-pvc
    EOF  
    ```

    You can follow the pipeline execution in the Tekton dashboard.

#### Configure Lighthouse to run the pipeline

Now we need to add the sample project to the Lighthouse configuration and setup the webhook ensuring that events occurring in the sample project are sent to Lighthouse.

- Update `config.yaml` and `plugins.yaml` in your config repository.
  For the sample project, a _postsubmit_ is configured, which will trigger the Tekton build after merges to the _master_ branch.
  Besides the sample project is added to the _tide_ configuration to enable the use of GitOps commands.

    ```bash
    bot_user=<github-username-of-bot-user>
    config_repo_name=lighthouse-config
    sample_repo_name=hello

    git checkout -b config-update
    cat > config.yaml <<EOF
    pod_namespace: lighthouse
    prowjob_namespace: lighthouse
    postsubmits:
      $bot_user/$sample_repo_name:
        - agent: tekton-pipeline
          branches:
            - master
          context: $sample_repo_name
          name: $sample_repo_name
          pipeline_run_spec:
            serviceAccountName: app-builder
            pipelineRef:
              name: ${sample_repo_name}-pipeline
            workspaces:
              - name: shared-workspace
                persistentvolumeclaim:
                  claimName: buildpacks-source-pvc
            resources:
              - name: build-image
                resourceRef:
                  name: buildpacks-app-image
            podTemplate:
              volumes:
                - name: buildpacks-cache
                  persistentVolumeClaim:
                    claimName: buildpacks-cache-pvc
    tide:
      queries:
      - labels:
        - approved
        repos:
        - $bot_user/$config_repo_name
        - $bot_user/$sample_repo_name
    EOF
  
    cat > plugins.yaml <<EOF
    approve:
    - lgtm_acts_as_approve: false
      repos:
      - $bot_user/$config_repo_name
      - $bot_user/$sample_repo_name
      require_self_approval: true
    config_updater:
      gzip: false
      maps:
        config.yaml:
          name: config
        plugins.yaml:
          name: plugins
    triggers:
    - repos:
      - $bot_user/$sample_repo_name
      ignore_ok_to_test: false
      elide_skipped_contexts: false
      skip_draft_pr: false
    plugins:
      $bot_user/$config_repo_name:
      - config-updater
      - approve
      - lgtm  
      $bot_user/$sample_repo_name:
      - approve
      - lgtm
      - trigger
    EOF
    ```

- Open a pull request for this change

    ```bash
    bot_user=<github-username-of-bot-user>
    sample_repo_name=hello
    GITHUB_TOKEN=<oauthtoken-bot-user>

    git commit -a -m "feat: adding bot_user/$sample_repo_name to Lighhouse config"
    gh pr create --title "adding bot_user/$sample_repo_name to Lighhouse config" --body ""
    gh pr view --web
   ```

- You should be able to approve the pull request by adding a pull request comment with the `/approve` command.
  After adding the comment, the bot user will first add the `approve` label, which in turn will then trigger the merge.
  Give this process some time.
  Once the pull request merged, you can verify the Lighthouse configuration in the cluster.

    ```bash
    install_namespace=lighthouse

    kubectl -n $install_namespace get cm config -o yaml
    kubectl -n $install_namespace get cm plugins -o yaml
    ```

- Last but not least, let's create a webhook for the sample project.

    ```bash
    bot_user=<github-username-of-bot-user>
    sample_repo_name=hello
    webhook_url=$(kubectl get ingress -n $install_namespace -o=jsonpath='http://{.items[0].spec.rules[0].host}'/hook)
    webhook_secret=<generated-webhook-secret>

    cat <<EOF | gh api -X POST repos/$bot_user/$sample_repo_name/hooks --input -
    {
      "name": "web",
      "active": true,
      "events": [
        "*"
      ],
      "config": {
        "url": "$webhook_url",
        "content_type": "json",
        "insecure_ssl": "0",
        "secret": "$webhook_secret"
      }
    }
    EOF
    ```

- Try opening a PR against the sample project repository!

## Next steps

From here, you have multiple possibilities to expand on this sample setup.
You can explore more of the Lighthouse plugins and update the Lighthouse configuration.
For more information refer to [PLUGINS](./PLUGINS.md).
Alternatively, you can further explore the Tekton Pipeline configuration.
At the moment there is only an unparameterized postsubmit pipeline configured.
You can make this pipeline more dynamic by parameterizing it, or you can create a pipeline to build pull requests and configure it as a presubmit action in Lighthouse.

## Webhook types

The following sections describe which webhooks events should be delivered to Lighthouse depending on the SCM provider.

### BitBucket Server Hooks

- `repo:refs_changed`
- `repo:modified`
- `repo:forked`
- `repo:comment:added`
- `repo:comment:edited`
- `repo:comment:deleted`
- `pr:opened`
- `pr:reviewer:approved`
- `pr:reviewer:unapproved`
- `pr:reviewer:needs_work`
- `pr:merged`
- `pr:declined`
- `pr:deleted`
- `pr:comment:added`
- `pr:comment:edited`
- `pr:comment:deleted`
- `pr:modified`
- `pr:from_ref_updated`

### GitHub or GitHub Enterprise Hooks

- `Send me everything`

Or you can exclude the following:

- `Deploy keys`
- `Deployment statuses`
- `Wiki`
- `Packages`
- `Page builds`
- `Project cards`
- `Project columns`
- `Registry packages`
- `Releases`
- `Repository vulnerability alerts`
- `Stars`
- `Watches`

### GitLab

- `Push events`
- `Tag push events`
- `Comments`
- `Confidential comments`
- `Issue events`
- `Confidential issue events`
- `Merge request events`
