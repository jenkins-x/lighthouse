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

package job

import (
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
)

// Base contains attributes common to all job types
type Base struct {
	// The name of the job. Must match regex [A-Za-z0-9-._]+
	// e.g. pull-test-infra-bazel-build
	Name string `json:"name"`
	// Labels are added to LighthouseJobs and pods created for this job.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations are unused by prow itself, but provide a space to configure other automation.
	Annotations map[string]string `json:"annotations,omitempty"`
	// MaximumConcurrency of this job, 0 implies no limit.
	MaxConcurrency int `json:"max_concurrency,omitempty"`
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
	ErrorOnEviction bool `json:"error_on_eviction,omitempty"`
	// SourcePath contains the path where this job is defined
	SourcePath string `json:"-"`
	// Spec is the Kubernetes pod spec used if Agent is kubernetes.
	Spec *v1.PodSpec `json:"spec,omitempty"`
	// PipelineRunSpec is the Tekton PipelineRun spec used if agent is tekton-pipeline
	PipelineRunSpec *tektonv1beta1.PipelineRunSpec `json:"pipeline_run_spec,omitempty"`
	// PipelineRunParams are the params used by the pipeline run
	PipelineRunParams []PipelineRunParam `json:"pipeline_run_params,omitempty"`

	UtilityConfig
}

// SetDefaults initializes default values
func (b *Base) SetDefaults(namespace string) {
	// Use the Jenkins X type by default
	if b.Agent == "" {
		b.Agent = JenkinsXAgent
	}
	if b.Namespace == nil || *b.Namespace == "" {
		s := namespace
		b.Namespace = &s
	}
	if b.Cluster == "" {
		b.Cluster = "default"
	}
}
