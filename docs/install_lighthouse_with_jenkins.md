# Lighthouse with Jenkins

<!-- MarkdownTOC autolink="true" indent="  " -->

- [Objective](#objective)
- [Prerequisite](#prerequisite)
- [Installation](#installation)
  - [Lighthouse](#lighthouse)
  - [Test project](#test-project)

<!-- /MarkdownTOC -->

## Objective

The objective of these instructions is to create a sample project where pushes to the master branch (a _post-submit_ in Lighthouse terminology) triggers a Jenkins Job.
The Lighthouse configuration is kept in the Kubernetes ConfigMaps _config_ and _plugins_.

---

**NOTE**:
In this example, the Lighthouse configuration is not backed by a Git repository.
Refer to the [Lighthouse + Tekton](./install_lighthouse_with_tekton.md) installation documentation for an example on how to use GitOps to manage the Lighthouse configuration itself.

---

---

**NOTE**:
The examples are using GitHub as SCM provider, but you can choose any of the Lighthouse supported SCM providers.
In this case you need to modify the relevant settings accordingly.

---

## Prerequisite

- [Helm 3](https://helm.sh/docs/intro/install/)
- Optionally [gh](https://github.com/cli/cli) to automate the required GitHub interaction.
  Instead of `gh`, you can use the web UI to create repositories, setup webhooks, etc.
- [pack](https://buildpacks.io/docs/install-pack/) for using Cloud Native Buildpacks to build the sample application.
- A Kubernetes cluster.
  - Ingress enabled, for example, by using the [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/deploy/).
    You can use any other ingress solution instead.
  - The domain name for your cluster's external IP. If you don't have a domain, you can use [nip.io](https://nip.io/).  
- A dedicated user for the SCM provider of your choice used as _bot user_.
  - An OAuth token for this bot user. In the case of GitHub refer to [Creating a personal access token
](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token).
- A self-generated secret for securing webhook delivery.
  You can, for example, generate a secret by running:

    ```bash
    ruby -rsecurerandom -e 'puts SecureRandom.hex(42)'
    ```

   See also [Securing your webhooks](https://developer.github.com/webhooks/securing/).
- A [Jenkins](https://www.jenkins.io/) instance.
  - [Kubernets agent for Jenkins](https://github.com/jenkinsci/kubernetes-plugin) installed and configured.
  - An [API token](https://www.jenkins.io/doc/book/system-administration/authenticating-scripted-clients/) to invoke operations on the Jenkins instance using its REST API.

## Installation

### Lighthouse

- Configure the Helm chart repository

    ```bash
    helm repo add lighthouse https://storage.googleapis.com/jenkinsxio/charts
    helm repo update
    ```

- Install Lighthouse using Helm

    ```bash
    namespace=<install-namespace>
    domain=<cluster-domain>

    bot_user=<github-username-of-bot-user>
    bot_token=<oauthtoken-bot-user>
    webhook_secret=<generated-webhook-secret>

    jenkins_url=<url-jenkins-instance>
    jenkins_user=<user-for-api-requests>
    jenkins_api_token=<jenkins_api_token>

    helm install lighthouse lighthouse/lighthouse -n ${namespace} -f <(cat <<EOF
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

    jenkinscontroller:
      jenkinsURL: ${jenkins_url}
      jenkinsUser: ${jenkins_user}
      jenkinsToken: ${jenkins_api_token}

    engines:
      jx: false
      tekton: false
      jenkins: true

    configMaps:
      create: true
  
    webhooks:
      ingress:
        enabled: true
        annotations:
          kubernetes.io/ingress.class: "nginx"
        hosts:
        - "$namespace.$domain"
    EOF
    )
    ```

### Test project

- Let's create a sample project with a _hello world_ web server written in Go.

    ```bash
    bot_user=<github-username-of-bot-user>
    repo_name=hello
  
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

    cat > Jenkinsfile <<EOF
    pipeline {
     agent {
         kubernetes {
           label 'build-pod'
           idleMinutes 5
           yamlFile 'build-pod.yaml'
           defaultContainer 'pack'
         }
     }

     parameters {
         string(name: 'JOB_NAME', defaultValue: '', description: 'Job name')
         string(name: 'JOB_TYPE', defaultValue: '', description: 'Job type')
         string(name: 'JOB_SPEC', defaultValue: '', description: 'Job spec')
         string(name: 'BUILD_ID', defaultValue: '', description: 'Build id')
         string(name: 'LIGHTHOUSE_JOB_ID', defaultValue: '', description: 'Lighthouse job id')
     }

     stages {
         stage('Build') {
               steps {
                 sh "pack build hello --builder paketobuildpacks/builder:tiny"
               }
         }  
     }
    }
    EOF

    cat > build-pod.yaml <<EOF
    apiVersion: v1
    kind: Pod
    spec:
      containers:
        - name: pack
          volumeMounts:
            - name: docker
              mountPath: /var/run/docker.sock
          image: buildpacksio/pack:0.12.0
          command: ["tail", "-f", "/dev/null"]
          imagePullPolicy: Always
          resources:
            requests:
              memory: "2Gi"
              cpu: "500m"
            limits:
              memory: "2Gi"
      volumes:
        - name: docker
          hostPath:
            path: /var/run/docker.sock
    EOF

    git add .
    git commit -m "Initial import"
    ```

- Push the code to GitHub.

    ```bash
    GITHUB_TOKEN=<oauthtoken-bot-user>
    repo_name=hello

    gh repo create $repo_name --public -y
    git push origin master
    ```  

- Configure the `config` ConfigMap of Lighthouse to contain a post-submit hook for the test project.

    ```bash
    namespace=<install-namespace>
    bot_user=<github-username-of-bot-user>
    repo_name=hello

    cat <<EOF | kubectl apply -n $namespace -f -
    apiVersion: v1
    data:
      config.yaml: |
        pod_namespace: $namespace
        prowjob_namespace: $namespace
        postsubmits:
          $bot_user/$repo_name:
            - agent: jenkins
              branches:
                - master
              context: $repo_name
              name: $repo_name
        jenkinses:
        - {}
    kind: ConfigMap
    metadata:
      name: config
      namespace: $namespace  
    EOF
    ```  

- Configure the `plugins` ConfigMap of Lighthouse to contain the _trigger_ plugin for the test project.

    ```bash
    namespace=<install-namespace>
    bot_user=<github-username-of-bot-user>
    repo_name=hello

    cat <<EOF | kubectl apply -n $namespace -f -
    apiVersion: v1
    data:
      plugins.yaml: |
        plugins:
          $bot_user/$repo_name:
          - trigger
    kind: ConfigMap
    metadata:
      name: plugins
      namespace: $namespace  
    EOF
    ```

- Last but not least, let's create a webhook for the sample project.

    ```bash
    GITHUB_TOKEN=<oauthtoken-bot-user>

    namespace=<install-namespace>
    bot_user=<github-username-of-bot-user>
    repo_name=hello
    webhook_url=$(kubectl get ingress -n $namespace -o=jsonpath='http://{.items[0].spec.rules[0].host}'/hook)
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

  - Push a change to the master branch and watch the Jenkins Job being triggered!
    
