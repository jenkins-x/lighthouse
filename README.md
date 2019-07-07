# lighthouse

A lightweight webhook handler to trigger Jenkins X Pipelines from webhooks from GitHub, GitHub Enterprise, BitBucket Cloud / Server, GitLab, Gitea etc

## Building

The fastest way to get started hacking on Lighthouse is with [ko](https://github.com/google/ko).

If you take a look in `deployments/manifests`, everything should look normal except for the following field in `200-webhook.yaml`:

```yaml
image: github.com/jenkins-x/lighthouse/cmd/webhook
```

The magic of `ko` is that it will: 
* automatically build the go binary whose code is specified in the `image` field *
* package it into an image
* push the image to a configured image registry
* update the deployment manifest with the newly build image name and tag
* `kubectl apply` a collection of k8s manifest files

Once you've installed and configured `ko`, simply run the following from the project root:

```sh
ko apply -f deployments/manifests/
```

This will build and deploy lighthouse to your Kubernetes cluster. In addition to a deployment, it will create a service and ingress so that you can start receiving webhooks right away, and it will put all of it in a namespace called `lighthouse`. Simply re-run the command to build and deploy your latest changes. It's idempotent, so running it multiple times with no changes in between will have no effect.
