# Foghorn

Foghorn is a fully Kubernetes-native CI system, inspired by [prow](https://github.com/kubernetes/test-infra/tree/master/prow). It's powered by [Tekton Pipelines](https://github.com/tektoncd/pipeline), which provides a number of handy abstractions for defining CI pipelines in a k8s-native way. Foghorn provides webhook handling and some git-centric ChatOps-y features, and it supports multiple git providers.

## Building

The fastest way to get started hacking on Foghorn is with [ko](https://github.com/google/ko).

If you take a look in `deployments`, everything should look normal except for the following field in `200-webhook-deployment.yaml`:

```yaml
image: github.com/foghornci/foghorn/cmd/webhook
```

The magic of `ko` is that it will: 

* automatically build the go binary whose code is specified in the `image` field
* package it into an image
* push the image to a configured image registry
* update the deployment manifest with the newly build image name and tag
* `kubectl apply` a collection of k8s manifest files

Once you've installed and configured `ko`, simply run the following from the project root:

```sh
ko apply -f deployments/
```

This will build and deploy Foghorn to your Kubernetes cluster. In addition to a deployment, it will create a service and ingress so that you can start receiving webhooks right away, and it will put all of it in a namespace called `foghorn`. Simply re-run the command to build and deploy your latest changes. It's idempotent, so running it multiple times with no changes in between will have no effect.
