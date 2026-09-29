{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "webhooks.name" -}}
{{- $name := default "webhooks" .Values.webhooks.nameOverride -}}
{{- printf "%s-%s" .Chart.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "keeper.name" -}}
{{- $name := default "keeper" .Values.keeper.nameOverride -}}
{{- printf "%s-%s" .Chart.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "poller.name" -}}
{{- $name := default "poller" .Values.poller.nameOverride -}}
{{- printf "%s-%s" .Chart.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "foghorn.name" -}}
{{- $name := default "foghorn" .Values.foghorn.nameOverride -}}
{{- printf "%s-%s" .Chart.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "gcJobs.name" -}}
{{- $name := default "gc-jobs" .Values.gcJobs.nameOverride -}}
{{- printf "%s-%s" .Chart.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "tektoncontroller.name" -}}
{{- $name := default "tekton-controller" .Values.tektoncontroller.nameOverride -}}
{{- printf "%s-%s" .Chart.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "jenkinscontroller.name" -}}
{{- $name := default "jenkins-controller" .Values.jenkinscontroller.nameOverride -}}
{{- printf "%s-%s" .Chart.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Reject the removed githubApp values.
*/}}
{{- define "lighthouse.checkRemovedValues" -}}
{{- if hasKey .Values "githubApp" -}}
{{- if .Values.githubApp.enabled -}}
{{- fail "githubApp.* has been removed: set auth.mode: ownerTokens instead of githubApp.enabled" -}}
{{- else -}}
{{- fail "githubApp.* has been removed: delete it from your values" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Validated credential mode.
*/}}
{{- define "lighthouse.authMode" -}}
{{- include "lighthouse.checkRemovedValues" . -}}
{{- $mode := ((.Values.auth).mode) | default "" -}}
{{- if not (has $mode (list "staticToken" "ownerTokens" "githubApp")) -}}
{{- fail (printf "auth.mode must be staticToken, ownerTokens or githubApp; got %q" $mode) -}}
{{- end -}}
{{- $mode -}}
{{- end -}}

{{/*
Whether the mode reads credentials from a mounted directory rather than an env var.
*/}}
{{- define "lighthouse.usesCredentialsDir" -}}
{{- $mode := include "lighthouse.authMode" . -}}
{{- if or (eq $mode "ownerTokens") (eq $mode "githubApp") -}}true{{- end -}}
{{- end -}}

{{/*
Name of the secret mounted as the credentials directory.
*/}}
{{- define "lighthouse.credentialsSecretName" -}}
{{- $mode := include "lighthouse.authMode" . -}}
{{- if eq $mode "githubApp" -}}
{{- (((.Values.auth).githubApp).existingSecretName) | default "lighthouse-githubapp" -}}
{{- else -}}
{{- (((.Values.auth).ownerTokens).secretName) | default "tide-githubapp-tokens" -}}
{{- end -}}
{{- end -}}

{{/*
Login to act as. Empty in githubApp mode, where it is derived from the app.
*/}}
{{- define "lighthouse.botUsername" -}}
{{- $mode := include "lighthouse.authMode" . -}}
{{- if eq $mode "githubApp" -}}
{{- (((.Values.auth).githubApp).username) | default "" -}}
{{- else -}}
{{- (((.Values.auth).ownerTokens).username) | default "jenkins-x[bot]" -}}
{{- end -}}
{{- end -}}
