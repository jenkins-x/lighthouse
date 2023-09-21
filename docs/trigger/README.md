# Lighthouse custome triggers

You can enable use LH_CUSTOM_TRIGGER_COMMAND environment variable to add custom triggers. It takes in a comma seperated list of commands.

When these commands are triggered using ChatOps Lighthouse will convert the command arg into TRIGGER_COMMAND_ARG enviroment variable which you can refer from the pipeline.

eg:
`
LH_CUSTOM_TRIGGER_COMMAND=deploy,trigger
`
```
apiVersion: config.lighthouse.jenkins-x.io/v1alpha1
kind: TriggerConfig
spec:
  presubmits:
  - name: pr
    context: "pr"
    always_run: true
    optional: false
    source: "pullrequest.yaml"
  - name: trigger
    context: "trigger"
    always_run: false
    optional: false
    rerun_command: /trigger this
    trigger: (?m)^/trigger( all| this),?(\s+|$)
    source: "trigger.yaml"
  - name: deploy
    context: "deployr"
    always_run: false
    optional: false
    rerun_command: /deploy dev 
    trigger: (?m)^/deploy(?:[ \t]+([-\w]+(?:,[-\w]+)*))?(?:[ \t]+([-\w]+(?:,[-\w]+)*))?
    source: "deploy.yaml"
  postsubmits:
  - name: release
    context: "release"
    source: "release.yaml"
    branches:
    - ^main$
```
you can use ChatOps command to trigger each pipeline
eg:
***/deploy qa***
