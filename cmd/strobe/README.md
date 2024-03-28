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

Note that if Strobe misses a schedule time for a particular periodic job due to
crashing or being restarted it will attempt to schedule a job only for the last
missed time.
