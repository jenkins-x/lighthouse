## Reuse steps in a task and override them

This example shows how we can use `image: uses:sourceURI` and a `name: mystep` to include individual the steps in task and then override (add/replace) properties:

* command to run (either via `script` or `command` and optionally `args`)
* working directory via `workingDir:`  
* environment variables via `env:` or `envFrom:`
* volume mounts via `volumeMounts:`