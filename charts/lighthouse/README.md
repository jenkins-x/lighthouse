

# Lighthouse

This chart bootstraps installation of [Lighthouse](https://github.com/jenkins-x/lighthouse).

## Installing

- Add jenkins-x helm charts repo

```bash
helm repo add jenkins-x http://chartmuseum.jenkins-x.io

helm repo update
```

- Install (or upgrade)

```bash
# This will install Lighthouse in the lighthouse namespace (with a my-lighthouse release name)

# Helm v2
helm upgrade --install my-lighthouse --namespace lighthouse jenkins-x/lighthouse
# Helm v3
helm upgrade --install my-lighthouse --namespace lighthouse jenkins-x/lighthouse
```

Look [below](#values) for the list of all available options and their corresponding description.

## Uninstalling

To uninstall the chart, simply delete the release.

```bash
# This will uninstall Lighthouse in the lighthouse namespace (assuming a my-lighthouse release name)

# Helm v2
helm delete --purge my-lighthouse
# Helm v3
helm uninstall my-lighthouse --namespace lighthouse
```

## Version

Current chart version is `0.1.0-SNAPSHOT`

## Values

| Key | Type | Description | Default |
|-----|------|-------------|---------|
| `cluster.crds.create` | bool | Create custom resource definitions | `true` |
| `configMaps.config` | string | Raw `config.yaml` content | `nil` |
| `configMaps.configUpdater` | object | Settings used to configure the `config-updater` plugin | `{"orgAndRepo":"","path":""}` |
| `configMaps.create` | bool | Enables creation of `config.yaml` and `plugins.yaml` config maps | `false` |
| `configMaps.plugins` | string | Raw `plugins.yaml` content | `nil` |
| `engines.jx` | bool | Enables the jx engine | `true` |
| `engines.tekton` | bool | Enables the tekton engine | `false` |
| `env` | object | Environment variables | `{"JX_DEFAULT_IMAGE":""}` |
| `foghorn.image.pullPolicy` | string | Template for computing the foghorn controller docker image pull policy | `"{{ .Values.image.pullPolicy }}"` |
| `foghorn.image.repository` | string | Template for computing the foghorn controller docker image repository | `"{{ .Values.image.parentRepository }}/lighthouse-foghorn"` |
| `foghorn.image.tag` | string | Template for computing the foghorn controller docker image tag | `"{{ .Values.image.tag }}"` |
| `foghorn.replicaCount` | int | Number of replicas | `1` |
| `foghorn.resources.limits` | object | Resource limits applied to the foghorn pods | `{"cpu":"100m","memory":"256Mi"}` |
| `foghorn.resources.requests` | object | Resource requests applied to the foghorn pods | `{"cpu":"80m","memory":"128Mi"}` |
| `foghorn.terminationGracePeriodSeconds` | int | Termination grace period for foghorn pods | `180` |
| `gcJobs.concurrencyPolicy` | string | Drives the job's concurrency policy | `"Forbid"` |
| `gcJobs.failedJobsHistoryLimit` | int | Drives the failed jobs history limit | `1` |
| `gcJobs.image.pullPolicy` | string | Template for computing the gc job docker image pull policy | `"{{ .Values.image.pullPolicy }}"` |
| `gcJobs.image.repository` | string | Template for computing the gc job docker image repository | `"{{ .Values.image.parentRepository }}/lighthouse-gc-jobs"` |
| `gcJobs.image.tag` | string | Template for computing the gc job docker image tag | `"{{ .Values.image.tag }}"` |
| `gcJobs.maxAge` | string | Max age from which `LighthouseJob`s will be deleted | `"168h"` |
| `gcJobs.schedule` | string | Cron expression to periodically delete `LighthouseJob`s | `"0/30 * * * *"` |
| `gcJobs.successfulJobsHistoryLimit` | int | Drives the successful jobs history limit | `3` |
| `git.kind` | string | Git SCM provider (`github`, `gitlab`, `stash`) | `"github"` |
| `git.server` | string | Git server URL | `""` |
| `githubApp.enabled` | bool | Enables GitHub app authentication | `false` |
| `githubApp.username` | string | GitHub app user name  | `"jenkins-x[bot]"` |
| `hmacToken` | string | Secret used for webhooks | `""` |
| `image.parentRepository` | string | Docker registry to pull images from | `"gcr.io/jenkinsxio"` |
| `image.pullPolicy` | string | Image pull policy | `"IfNotPresent"` |
| `image.tag` | string | Docker images tag | `"0.0.750"` |
| `keeper.datadog.enabled` | string | Enables datadog | `"true"` |
| `keeper.image.pullPolicy` | string | Template for computing the keeper controller docker image pull policy | `"{{ .Values.image.pullPolicy }}"` |
| `keeper.image.repository` | string | Template for computing the keeper controller docker image repository | `"{{ .Values.image.parentRepository }}/lighthouse-keeper"` |
| `keeper.image.tag` | string | Template for computing the keeper controller docker image tag | `"{{ .Values.image.tag }}"` |
| `keeper.livenessProbe` | object | Liveness probe configuration | `{"initialDelaySeconds":120,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":1}` |
| `keeper.probe` | object | Liveness and readiness probes settings | `{"path":"/"}` |
| `keeper.readinessProbe` | object | Readiness probe configuration | `{"periodSeconds":10,"successThreshold":1,"timeoutSeconds":1}` |
| `keeper.replicaCount` | int | Number of replicas | `1` |
| `keeper.resources.limits` | object | Resource limits applied to the keeper pods | `{"cpu":"400m","memory":"512Mi"}` |
| `keeper.resources.requests` | object | Resource requests applied to the keeper pods | `{"cpu":"100m","memory":"128Mi"}` |
| `keeper.service` | object | Service settings for the webhooks controller | `{"externalPort":80,"internalPort":8888,"type":"ClusterIP"}` |
| `keeper.statusContextLabel` | string | Label used to report status to git provider | `"Lighthouse Merge Status"` |
| `keeper.terminationGracePeriodSeconds` | int | Termination grace period for keeper pods | `30` |
| `lighthouseJobNamespace` | string | Namespace where `LighthouseJob`s and `Pod`s are created | Deployment namespace |
| `logFormat` | string | Log format | `"json"` |
| `oauthToken` | string | Git token (used when GitHub app authentication is not enabled) | `""` |
| `tektoncontroller.affinity` | object | [Affinity rules](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity) applied to the tekton controller pods | `{}` |
| `tektoncontroller.dashboardURL` | string | Tekton dashboard URL | `""` |
| `tektoncontroller.image.pullPolicy` | string | Template for computing the tekton controller docker image pull policy | `"{{ .Values.image.pullPolicy }}"` |
| `tektoncontroller.image.repository` | string | Template for computing the tekton controller docker image repository | `"{{ .Values.image.parentRepository }}/lighthouse-tekton-controller"` |
| `tektoncontroller.image.tag` | string | Template for computing the tekton controller docker image tag | `"{{ .Values.image.tag }}"` |
| `tektoncontroller.nodeSelector` | object | [Node selector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector) applied to the tekton controller pods | `{}` |
| `tektoncontroller.podAnnotations` | object | Annotations applied to the tekton controller pods | `{}` |
| `tektoncontroller.replicaCount` | int | Number of replicas | `1` |
| `tektoncontroller.resources.limits` | object | Resource limits applied to the tekton controller pods | `{"cpu":"100m","memory":"256Mi"}` |
| `tektoncontroller.resources.requests` | object | Resource requests applied to the tekton controller pods | `{"cpu":"80m","memory":"128Mi"}` |
| `tektoncontroller.service` | object | Service settings for the tekton controller | `{"annotations":{}}` |
| `tektoncontroller.terminationGracePeriodSeconds` | int | Termination grace period for tekton controller pods | `180` |
| `tektoncontroller.tolerations` | list | [Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) applied to the tekton controller pods | `[]` |
| `user` | string | Git user name (used when GitHub app authentication is not enabled) | `""` |
| `webhooks.image.pullPolicy` | string | Template for computing the webhooks controller docker image pull policy | `"{{ .Values.image.pullPolicy }}"` |
| `webhooks.image.repository` | string | Template for computing the webhooks controller docker image repository | `"{{ .Values.image.parentRepository }}/lighthouse-webhooks"` |
| `webhooks.image.tag` | string | Template for computing the webhooks controller docker image tag | `"{{ .Values.image.tag }}"` |
| `webhooks.ingress.annotations` | object | Webhooks ingress annotations | `{}` |
| `webhooks.ingress.enabled` | bool | Enable webhooks ingress | `false` |
| `webhooks.ingress.hosts` | list | Webhooks ingress host names | `[]` |
| `webhooks.livenessProbe` | object | Liveness probe configuration | `{"initialDelaySeconds":60,"periodSeconds":10,"successThreshold":1,"timeoutSeconds":1}` |
| `webhooks.probe` | object | Liveness and readiness probes settings | `{"path":"/"}` |
| `webhooks.readinessProbe` | object | Readiness probe configuration | `{"periodSeconds":10,"successThreshold":1,"timeoutSeconds":1}` |
| `webhooks.replicaCount` | int | Number of replicas | `2` |
| `webhooks.resources.limits` | object | Resource limits applied to the webhooks pods | `{"cpu":"100m","memory":"256Mi"}` |
| `webhooks.resources.requests` | object | Resource requests applied to the webhooks pods | `{"cpu":"80m","memory":"128Mi"}` |
| `webhooks.service` | object | Service settings for the webhooks controller | `{"annotations":{},"externalPort":80,"internalPort":8080,"type":"ClusterIP"}` |
| `webhooks.serviceName` | string | Allows overriding the service name, this is here for compatibility reasons, regular users should clear this out | `"hook"` |
| `webhooks.terminationGracePeriodSeconds` | int | Termination grace period for webhooks pods | `180` |

You can look directly at the [values.yaml](./values.yaml) file to look at the options and their default values.
