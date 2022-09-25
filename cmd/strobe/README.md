# Strobe

Strobe is a controller that implements the periodic jobs defined in the
Lighthouse config ConfigMap:

```yaml
periodics:
- name: hello-world
  cron: "*/1 * * * *"
  agent: tekton-pipeline
  pipeline_run_spec:
    pipelineSpec:
      tasks:
      - name: hello-world
        taskSpec:
          steps:
          - image: busybox
            script: echo 'Hello World!'
```

This is done by watching the ConfigMap and processing each periodic job by name.
Inspiration is taken from the [Kubernetes CronJob
controller](https://github.com/kubernetes/kubernetes/blob/v1.25.2/pkg/controller/cronjob/cronjob_controllerv2.go).

A key implementation detail is that, since there is no timestamp available for
when a new periodic is added (unlike when creating a
[CronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/),
for example), there is no way to know whether it was added after or before the
most recent schedule time. For this reason, Strobe will schedule a job as soon
as a new periodic job definition is added to serve as this marker and then
subsequent jobs are scheduled according to the specified `cron` as expected.
