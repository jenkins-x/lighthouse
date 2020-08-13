package v1alpha1

import (
	"regexp"

	"time"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
)

// These structs are copies of the config package structs but with nicer kubernetes native camel case
// yaml encoding

/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// this file contains copies of structs from the config package
// using different JSON/YAML marshalling so that they look like
// kubernetes CRDs; using camelCase rather than lower case and underscores

// JobBase contains attributes common to all job types
type JobBase struct {
	// The name of the job. Must match regex [A-Za-z0-9-._]+
	// e.g. pull-test-infra-bazel-build
	Name string `json:"name"`
	// Labels are added to LighthouseJobs and pods created for this job.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations are unused by prow itself, but provide a space to configure other automation.
	Annotations map[string]string `json:"annotations,omitempty"`
	// MaximumConcurrency of this job, 0 implies no limit.
	MaxConcurrency int `json:"maxConcurrency,omitempty"`
	// Agent that will take care of running this job.
	Agent string `json:"agent"`
	// Cluster is the alias of the cluster to run this job in.
	// (Default: kube.DefaultClusterAlias)
	Cluster string `json:"cluster,omitempty"`
	// Namespace is the namespace in which pods schedule.
	//   nil: results in config.PodNamespace (aka pod default)
	//   empty: results in config.LighthouseJobNamespace (aka same as LighthouseJob)
	Namespace *string `json:"namespace,omitempty"`
	// ErrorOnEviction indicates that the LighthouseJob should be completed and given
	// the ErrorState status if the pod that is executing the job is evicted.
	// If this field is unspecified or false, a new pod will be created to replace
	// the evicted one.
	ErrorOnEviction bool `json:"errorOnEviction,omitempty"`
	// SourcePath contains the path where this job is defined
	SourcePath string `json:"-"`
	// Spec is the Kubernetes pod spec used if Agent is kubernetes.
	Spec *v1.PodSpec `json:"spec,omitempty"`
	// PipelineRunSpec is the Tekton PipelineRun spec used if agent is tekton-pipeline
	PipelineRunSpec *tektonv1beta1.PipelineRunSpec `json:"pipelineRunSpec,omitempty"`

	UtilityConfig
}

// Presubmit runs on PRs.
type Presubmit struct {
	JobBase

	// AlwaysRun automatically for every PR, or only when a comment triggers it.
	AlwaysRun bool `json:"alwaysRun"`

	// Optional indicates that the job's status context should not be required for merge.
	Optional bool `json:"optional,omitempty"`

	// Trigger is the regular expression to trigger the job.
	// e.g. `@k8s-bot e2e test this`
	// RerunCommand must also be specified if this field is specified.
	// (Default: `(?m)^/test (?:.*? )?<job name>(?: .*?)?$`)
	Trigger string `json:"trigger,omitempty"`

	// The RerunCommand to give users. Must match Trigger.
	// Trigger must also be specified if this field is specified.
	// (Default: `/test <job name>`)
	RerunCommand string `json:"rerunCommand,omitempty"`

	Brancher

	RegexpChangeMatcher

	Reporter

	// We'll set these when we load it.
	//re *regexp.Regexp // from Trigger.
}

// Postsubmit runs on push events.
type Postsubmit struct {
	JobBase

	RegexpChangeMatcher

	Brancher

	// TODO(krzyzacy): Move existing `Report` into `SkipReport` once this is deployed
	Reporter
}

// Periodic runs on a timer.
type Periodic struct {
	JobBase

	// (deprecated)Interval to wait between two runs of the job.
	Interval string `json:"interval"`
	// Cron representation of job trigger time
	Cron string `json:"cron"`
	// Tags for config entries
	Tags []string `json:"tags,omitempty"`

	interval time.Duration
}

// Brancher is for shared code between jobs that only run against certain
// branches. An empty brancher runs against all branches.
type Brancher struct {
	// Do not run against these branches. Default is no branches.
	SkipBranches []string `json:"skipBranches,omitempty"`
	// Only run against these branches. Default is all branches.
	Branches []string `json:"branches,omitempty"`

	// We'll set these when we load it.
	re     *regexp.Regexp
	reSkip *regexp.Regexp
}

// RegexpChangeMatcher is for code shared between jobs that run only when certain files are changed.
type RegexpChangeMatcher struct {
	// RunIfChanged defines a regex used to select which subset of file changes should trigger this job.
	// If any file in the changeset matches this regex, the job will be triggered
	RunIfChanged string         `json:"runIfChanged,omitempty"`
	reChanges    *regexp.Regexp // from RunIfChanged
}

// Reporter keeps various details for status reporting
type Reporter struct {
	// Context is the name of the GitHub status context for the job.
	// Defaults: the same as the name of the job.
	Context string `json:"context,omitempty"`
	// SkipReport skips commenting and setting status on GitHub.
	SkipReport bool `json:"skipReport,omitempty"`
}

// ChangedFilesProvider returns a slice of modified files.
type ChangedFilesProvider func() ([]string, error)

// UtilityConfig holds decoration metadata, such as how to clone and additional containers/etc
type UtilityConfig struct {
	// Decorate determines if we decorate the PodSpec or not
	Decorate bool `json:"decorate,omitempty"`

	// PathAlias is the location under <root-dir>/src
	// where the repository under test is cloned. If this
	// is not set, <root-dir>/src/github.com/org/repo will
	// be used as the default.
	PathAlias string `json:"pathAlias,omitempty"`
	// CloneURI is the URI that is used to clone the
	// repository. If unset, will default to
	// `https://github.com/org/repo.git`.
	CloneURI string `json:"cloneUri,omitempty"`
	// SkipSubmodules determines if submodules should be
	// cloned when the job is run. Defaults to true.
	SkipSubmodules bool `json:"skipSubmodules,omitempty"`
	// CloneDepth is the depth of the clone that will be used.
	// A depth of zero will do a full clone.
	CloneDepth int `json:"cloneDepth,omitempty"`
}
