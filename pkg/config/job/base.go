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
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// DefaultClusterAlias specifies the default context for resources owned by jobs (pods/builds).
	DefaultClusterAlias = "default"
)

var jobNameRegex = regexp.MustCompile(`^[A-Za-z0-9-._]+$`)

// Base contains attributes common to all job types
type Base struct {
	UtilityConfig
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
	// SourcePath contains the path where the tekton pipeline run is defined
	SourcePath string `json:"source,omitempty"`
	// Spec is the Kubernetes pod spec used if Agent is kubernetes.
	Spec *v1.PodSpec `json:"spec,omitempty"`
	// PipelineRunSpec is the Tekton PipelineRun spec used if agent is tekton-pipeline
	PipelineRunSpec *pipelinev1.PipelineRunSpec `json:"pipeline_run_spec,omitempty"`
	// PipelineRunParams are the params used by the pipeline run
	PipelineRunParams []PipelineRunParam `json:"pipeline_run_params,omitempty"`
	// lets us register a loader
	pipelineLoader func(*Base) error
}

// LoadPipeline() loads the pipeline specification if its not already been loaded
func (b *Base) LoadPipeline(logger *logrus.Entry) error {
	if b.PipelineRunSpec != nil || b.pipelineLoader == nil {
		return nil
	}
	logger.Debugf("lazy loading the PipelineRunSpec")
	answer := b.pipelineLoader(b)
	// lets gc the function
	b.pipelineLoader = nil
	return answer
}

// SetPipelineLoader sets the function to lazy load the pipeline spec
func (b *Base) SetPipelineLoader(fn func(b *Base) error) {
	b.pipelineLoader = fn
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
		b.Cluster = DefaultClusterAlias
	}
}

// Validate validates job base
func (b *Base) Validate(jobType PipelineKind, podNamespace string) error {
	if !jobNameRegex.MatchString(b.Name) {
		return fmt.Errorf("name: must match regex %q", jobNameRegex.String())
	}
	// Ensure max_concurrency is non-negative.
	if b.MaxConcurrency < 0 {
		return fmt.Errorf("max_concurrency: %d must be a non-negative number", b.MaxConcurrency)
	}
	if err := b.ValidateAgent(podNamespace); err != nil {
		return err
	}
	if err := b.ValidatePodSpec(jobType); err != nil {
		return err
	}
	if err := ValidateLabels(b.Labels); err != nil {
		return err
	}
	if b.Spec == nil || len(b.Spec.Containers) == 0 {
		return nil // knative-build and jenkins jobs have no spec
	}
	return nil
}

// ValidateAgent validates job agent
func (b *Base) ValidateAgent(podNamespace string) error {
	agents := sets.NewString(AvailablePipelineAgentTypes()...)
	agent := b.Agent
	switch {
	case !agents.Has(agent):
		return fmt.Errorf("agent must be one of %s (found %q)", strings.Join(agents.List(), ", "), agent)
		/*	case b.Spec != nil && agent != k:
				return fmt.Errorf("job specs require agent: %s (found %q)", k, agent)
			case agent == k && b.Spec == nil:
				return errors.New("kubernetes jobs require a spec")
			case b.BuildSpec != nil && agent != b:
				return fmt.Errorf("job build_specs require agent: %s (found %q)", b, agent)
			case agent == b && b.BuildSpec == nil:
				return errors.New("knative-build jobs require a build_spec")
			case b.DecorationConfig != nil && agent != k && agent != b:
				// TODO(fejta): only source decoration supported...
				return fmt.Errorf("decoration requires agent: %s or %s (found %q)", k, b, agent)
			case b.ErrorOnEviction && agent != k:
				return fmt.Errorf("error_on_eviction only applies to agent: %s (found %q)", k, agent)
			case b.Namespace == nil || *b.Namespace == "":
				return fmt.Errorf("failed to default namespace")
			case *b.Namespace != podNamespace && agent != b:
				// TODO(fejta): update plank to allow this (depends on client change)
				return fmt.Errorf("namespace customization requires agent: %s (found %q)", b, agent)
		*/
	}
	return nil
}

// ValidatePodSpec validates job pod spec
func (b *Base) ValidatePodSpec(jobType PipelineKind) error {
	if b.Spec == nil {
		return nil
	}
	if len(b.Spec.InitContainers) != 0 {
		return errors.New("pod spec may not use init containers")
	}
	if n := len(b.Spec.Containers); n != 1 {
		return fmt.Errorf("pod spec must specify exactly 1 container, found: %d", n)
	}
	/*	for _, env := range spec.Containers[0].Env {
			for _, prowEnv := range downwardapi.EnvForType(jobType) {
				if env.Name == prowEnv {
					// TODO(fejta): consider allowing this
					return fmt.Errorf("env %s is reserved", env.Name)
				}
			}
		}
	*/
	// for _, mount := range b.Spec.Containers[0].VolumeMounts {
	// 	for _, prowMount := range VolumeMounts() {
	// 		if mount.Name == prowMount {
	// 			return fmt.Errorf("volumeMount name %s is reserved for decoration", prowMount)
	// 		}
	// 	}
	// 	for _, prowMountPath := range VolumeMountPaths() {
	// 		if strings.HasPrefix(mount.MountPath, prowMountPath) || strings.HasPrefix(prowMountPath, mount.MountPath) {
	// 			return fmt.Errorf("mount %s at %s conflicts with decoration mount at %s", mount.Name, mount.MountPath, prowMountPath)
	// 		}
	// 	}
	// }
	// for _, volume := range spec.Volumes {
	// 	for _, prowVolume := range VolumeMounts() {
	// 		if volume.Name == prowVolume {
	// 			return fmt.Errorf("volume %s is a reserved for decoration", volume.Name)
	// 		}
	// 	}
	// }
	return nil
}
