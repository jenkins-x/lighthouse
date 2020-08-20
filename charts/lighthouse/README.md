

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
| `buildNumbers.enable` | bool |  | `false` |
| `buildNumbers.image.pullPolicy` | string |  | `"{{ .Values.image.pullPolicy }}"` |
| `buildNumbers.image.repository` | string |  | `"{{ .Values.image.parentRepository }}/lighthouse-build-numbers"` |
| `buildNumbers.image.tag` | string |  | `"{{ .Values.image.tag }}"` |
| `buildNumbers.livenessProbe.initialDelaySeconds` | int |  | `60` |
| `buildNumbers.livenessProbe.periodSeconds` | int |  | `10` |
| `buildNumbers.livenessProbe.successThreshold` | int |  | `1` |
| `buildNumbers.livenessProbe.timeoutSeconds` | int |  | `1` |
| `buildNumbers.probe.path` | string |  | `"/healthz"` |
| `buildNumbers.probe.port` | int |  | `8081` |
| `buildNumbers.readinessProbe.periodSeconds` | int |  | `10` |
| `buildNumbers.readinessProbe.successThreshold` | int |  | `1` |
| `buildNumbers.readinessProbe.timeoutSeconds` | int |  | `1` |
| `buildNumbers.replicaCount` | int |  | `1` |
| `buildNumbers.resources.limits.cpu` | string |  | `"100m"` |
| `buildNumbers.resources.limits.memory` | string |  | `"256Mi"` |
| `buildNumbers.resources.requests.cpu` | string |  | `"80m"` |
| `buildNumbers.resources.requests.memory` | string |  | `"128Mi"` |
| `buildNumbers.service.externalPort` | int |  | `80` |
| `buildNumbers.service.internalPort` | int |  | `8080` |
| `buildNumbers.service.name` | string |  | `"lighthouse-build-numbers"` |
| `buildNumbers.service.type` | string |  | `"ClusterIP"` |
| `buildNumbers.terminationGracePeriodSeconds` | int |  | `30` |
| `cluster.crds.create` | bool |  | `true` |
| `clusterName` | string |  | `""` |
| `configMaps.config` | string |  | `nil` |
| `configMaps.configUpdater.orgAndRepo` | string |  | `""` |
| `configMaps.configUpdater.path` | string |  | `""` |
| `configMaps.create` | bool |  | `false` |
| `configMaps.plugins` | string |  | `nil` |
| `createIngress` | bool |  | `false` |
| `domainName` | string |  | `""` |
| `engines.jx` | bool |  | `true` |
| `engines.tekton` | bool |  | `false` |
| `env.JX_DEFAULT_IMAGE` | string |  | `""` |
| `foghorn.image.pullPolicy` | string |  | `"{{ .Values.image.pullPolicy }}"` |
| `foghorn.image.repository` | string |  | `"{{ .Values.image.parentRepository }}/lighthouse-foghorn"` |
| `foghorn.image.tag` | string |  | `"{{ .Values.image.tag }}"` |
| `foghorn.replicaCount` | int |  | `1` |
| `foghorn.reportURLBase` | string |  | `""` |
| `foghorn.resources.limits.cpu` | string |  | `"100m"` |
| `foghorn.resources.limits.memory` | string |  | `"256Mi"` |
| `foghorn.resources.requests.cpu` | string |  | `"80m"` |
| `foghorn.resources.requests.memory` | string |  | `"128Mi"` |
| `foghorn.terminationGracePeriodSeconds` | int |  | `180` |
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
| `hmacToken` | string |  | `""` |
| `hook.ingress.annotations` | string |  | `nil` |
| `hook.ingress.class` | string |  | `"nginx"` |
| `hook.ingress.tls.secretName` | string |  | `""` |
| `image.parentRepository` | string |  | `"gcr.io/jenkinsxio"` |
| `image.pullPolicy` | string |  | `"IfNotPresent"` |
| `image.tag` | string |  | `"0.0.749"` |
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
| `keeper.replicaCount` | int |  | `1` |
| `keeper.resources.limits.cpu` | string |  | `"400m"` |
| `keeper.resources.limits.memory` | string |  | `"512Mi"` |
| `keeper.resources.requests.cpu` | string |  | `"100m"` |
| `keeper.resources.requests.memory` | string |  | `"128Mi"` |
| `keeper.service.externalPort` | int |  | `80` |
| `keeper.service.internalPort` | int |  | `8888` |
| `keeper.service.type` | string |  | `"ClusterIP"` |
| `keeper.statusContextLabel` | string |  | `"Lighthouse Merge Status"` |
| `keeper.terminationGracePeriodSeconds` | int |  | `30` |
| `lighthouseJobNamespace` | string |  | `""` |
| `logFormat` | string |  | `"json"` |
| `tektoncontroller.dashboardURL` | string |  | `""` |
| `tektoncontroller.image.pullPolicy` | string |  | `"{{ .Values.image.pullPolicy }}"` |
| `tektoncontroller.image.repository` | string |  | `"{{ .Values.image.parentRepository }}/lighthouse-tekton-controller"` |
| `tektoncontroller.image.tag` | string |  | `"{{ .Values.image.tag }}"` |
| `tektoncontroller.replicaCount` | int |  | `1` |
| `tektoncontroller.resources.limits.cpu` | string |  | `"100m"` |
| `tektoncontroller.resources.limits.memory` | string |  | `"256Mi"` |
| `tektoncontroller.resources.requests.cpu` | string |  | `"80m"` |
| `tektoncontroller.resources.requests.memory` | string |  | `"128Mi"` |
| `tektoncontroller.service.annotations` | object |  | `{}` |
| `tektoncontroller.terminationGracePeriodSeconds` | int |  | `180` |
| `user` | string |  | `""` |
| `vault.enabled` | bool |  | `false` |
| `webhooks.image.pullPolicy` | string |  | `"{{ .Values.image.pullPolicy }}"` |
| `webhooks.image.repository` | string |  | `"{{ .Values.image.parentRepository }}/lighthouse-webhooks"` |
| `webhooks.image.tag` | string |  | `"{{ .Values.image.tag }}"` |
| `webhooks.livenessProbe.initialDelaySeconds` | int |  | `60` |
| `webhooks.livenessProbe.periodSeconds` | int |  | `10` |
| `webhooks.livenessProbe.successThreshold` | int |  | `1` |
| `webhooks.livenessProbe.timeoutSeconds` | int |  | `1` |
| `webhooks.probe.path` | string |  | `"/healthz"` |
| `webhooks.probe.port` | int |  | `8081` |
| `webhooks.readinessProbe.periodSeconds` | int |  | `10` |
| `webhooks.readinessProbe.successThreshold` | int |  | `1` |
| `webhooks.readinessProbe.timeoutSeconds` | int |  | `1` |
| `webhooks.replicaCount` | int |  | `2` |
| `webhooks.resources.limits.cpu` | string |  | `"100m"` |
| `webhooks.resources.limits.memory` | string |  | `"256Mi"` |
| `webhooks.resources.requests.cpu` | string |  | `"80m"` |
| `webhooks.resources.requests.memory` | string |  | `"128Mi"` |
| `webhooks.service.externalPort` | int |  | `80` |
| `webhooks.service.internalPort` | int |  | `8080` |
| `webhooks.service.name` | string |  | `"hook"` |
| `webhooks.service.type` | string |  | `"ClusterIP"` |
| `webhooks.terminationGracePeriodSeconds` | int |  | `180` |

You can look directly at the [values.yaml](./values.yaml) file to look at the options and their default values.
