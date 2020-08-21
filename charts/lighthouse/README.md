

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
| `clusterName` | string |  | `""` |
| `configMaps.config` | string |  | `nil` |
| `configMaps.configUpdater.orgAndRepo` | string |  | `""` |
| `configMaps.configUpdater.path` | string |  | `""` |
| `configMaps.create` | bool |  | `false` |
| `configMaps.plugins` | string |  | `nil` |
| `createIngress` | bool |  | `false` |
| `domainName` | string |  | `""` |
| `engines.jx` | bool | Enables the jx engine | `true` |
| `engines.tekton` | bool | Enables the tekton engine | `false` |
| `env` | object | Environment variables | `{"JX_DEFAULT_IMAGE":""}` |
| `foghorn.image.pullPolicy` | string |  | `"{{ .Values.image.pullPolicy }}"` |
| `foghorn.image.repository` | string |  | `"{{ .Values.image.parentRepository }}/lighthouse-foghorn"` |
| `foghorn.image.tag` | string |  | `"{{ .Values.image.tag }}"` |
| `foghorn.replicaCount` | int | Number of replicas | `1` |
| `foghorn.reportURLBase` | string |  | `""` |
| `foghorn.resources.limits` | object | Resource limits applied to the foghorn pods | `{"cpu":"100m","memory":"256Mi"}` |
| `foghorn.resources.requests` | object | Resource requests applied to the foghorn pods | `{"cpu":"80m","memory":"128Mi"}` |
| `foghorn.terminationGracePeriodSeconds` | int | Termination grace period for foghorn pods | `180` |
| `gcJobs.concurrencyPolicy` | string |  | `"Forbid"` |
| `gcJobs.failedJobsHistoryLimit` | int |  | `1` |
| `gcJobs.image.pullPolicy` | string |  | `"{{ .Values.image.pullPolicy }}"` |
| `gcJobs.image.repository` | string |  | `"{{ .Values.image.parentRepository }}/lighthouse-gc-jobs"` |
| `gcJobs.image.tag` | string |  | `"{{ .Values.image.tag }}"` |
| `gcJobs.maxAge` | string |  | `"168h"` |
| `gcJobs.schedule` | string |  | `"0/30 * * * *"` |
| `gcJobs.successfulJobsHistoryLimit` | int |  | `3` |
| `git.kind` | string |  | `"github"` |
| `git.name` | string |  | `"github"` |
| `git.server` | string |  | `""` |
| `githubApp.enabled` | bool |  | `false` |
| `githubApp.username` | string |  | `"jenkins-x[bot]"` |
| `hmacToken` | string | Secret used for webhooks | `""` |
| `hook.ingress.annotations` | string |  | `nil` |
| `hook.ingress.class` | string |  | `"nginx"` |
| `hook.ingress.tls.secretName` | string |  | `""` |
| `image.parentRepository` | string | Docker registry to pull images from | `"gcr.io/jenkinsxio"` |
| `image.pullPolicy` | string | Image pull policy | `"IfNotPresent"` |
| `image.tag` | string | Docker images tag | `"0.0.750"` |
| `keeper.datadog.enabled` | string |  | `"true"` |
| `keeper.image.repository` | string |  | `"{{ .Values.image.parentRepository }}/lighthouse-keeper"` |
| `keeper.image.tag` | string |  | `"{{ .Values.image.tag }}"` |
| `keeper.imagePullPolicy` | string |  | `"IfNotPresent"` |
| `keeper.livenessProbe.initialDelaySeconds` | int |  | `120` |
| `keeper.livenessProbe.periodSeconds` | int |  | `10` |
| `keeper.livenessProbe.successThreshold` | int |  | `1` |
| `keeper.livenessProbe.timeoutSeconds` | int |  | `1` |
| `keeper.probe.path` | string |  | `"/"` |
| `keeper.readinessProbe.periodSeconds` | int |  | `10` |
| `keeper.readinessProbe.successThreshold` | int |  | `1` |
| `keeper.readinessProbe.timeoutSeconds` | int |  | `1` |
| `keeper.replicaCount` | int | Number of replicas | `1` |
| `keeper.resources.limits` | object | Resource limits applied to the keeper pods | `{"cpu":"400m","memory":"512Mi"}` |
| `keeper.resources.requests` | object | Resource requests applied to the keeper pods | `{"cpu":"100m","memory":"128Mi"}` |
| `keeper.service.externalPort` | int |  | `80` |
| `keeper.service.internalPort` | int |  | `8888` |
| `keeper.service.type` | string |  | `"ClusterIP"` |
| `keeper.statusContextLabel` | string |  | `"Lighthouse Merge Status"` |
| `keeper.terminationGracePeriodSeconds` | int | Termination grace period for keeper pods | `30` |
| `lighthouseJobNamespace` | string |  | `""` |
| `logFormat` | string |  | `"json"` |
| `tektoncontroller.dashboardURL` | string | Tekton dashboard URL | `""` |
| `tektoncontroller.image.pullPolicy` | string |  | `"{{ .Values.image.pullPolicy }}"` |
| `tektoncontroller.image.repository` | string | Template for computing the tekton controller docker image pull policy | `"{{ .Values.image.parentRepository }}/lighthouse-tekton-controller"` |
| `tektoncontroller.image.tag` | string | Template for computing the tekton controller docker image tag | `"{{ .Values.image.tag }}"` |
| `tektoncontroller.replicaCount` | int | Number of replicas | `1` |
| `tektoncontroller.resources.limits` | object | Resource limits applied to the tekton controller pods | `{"cpu":"100m","memory":"256Mi"}` |
| `tektoncontroller.resources.requests` | object | Resource requests applied to the tekton controller pods | `{"cpu":"80m","memory":"128Mi"}` |
| `tektoncontroller.service.annotations` | object |  | `{}` |
| `tektoncontroller.terminationGracePeriodSeconds` | int | Termination grace period for tekton controller pods | `180` |
| `user` | string |  | `""` |
| `vault.enabled` | bool |  | `false` |
| `webhooks.image.pullPolicy` | string |  | `"{{ .Values.image.pullPolicy }}"` |
| `webhooks.image.repository` | string |  | `"{{ .Values.image.parentRepository }}/lighthouse-webhooks"` |
| `webhooks.image.tag` | string |  | `"{{ .Values.image.tag }}"` |
| `webhooks.livenessProbe.initialDelaySeconds` | int |  | `60` |
| `webhooks.livenessProbe.periodSeconds` | int |  | `10` |
| `webhooks.livenessProbe.successThreshold` | int |  | `1` |
| `webhooks.livenessProbe.timeoutSeconds` | int |  | `1` |
| `webhooks.probe.path` | string |  | `"/"` |
| `webhooks.readinessProbe.periodSeconds` | int |  | `10` |
| `webhooks.readinessProbe.successThreshold` | int |  | `1` |
| `webhooks.readinessProbe.timeoutSeconds` | int |  | `1` |
| `webhooks.replicaCount` | int | Number of replicas | `2` |
| `webhooks.resources.limits` | object | Resource limits applied to the webhooks pods | `{"cpu":"100m","memory":"256Mi"}` |
| `webhooks.resources.requests` | object | Resource requests applied to the webhooks pods | `{"cpu":"80m","memory":"128Mi"}` |
| `webhooks.service.externalPort` | int |  | `80` |
| `webhooks.service.internalPort` | int |  | `8080` |
| `webhooks.service.name` | string |  | `"hook"` |
| `webhooks.service.type` | string |  | `"ClusterIP"` |
| `webhooks.terminationGracePeriodSeconds` | int | Termination grace period for webhooks pods | `180` |

You can look directly at the [values.yaml](./values.yaml) file to look at the options and their default values.
