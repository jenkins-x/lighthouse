package inrepo

import (
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
)

// OverrideTaskSpec lets reuse any TaskSpec resources from the used task
func OverrideTaskSpec(ts *tektonv1beta1.TaskSpec, override *tektonv1beta1.TaskSpec) {
	if override.StepTemplate != nil {
		if ts.StepTemplate == nil {
			ts.StepTemplate = &v1.Container{}
		}
		OverrideContainer(ts.StepTemplate, override.StepTemplate, true)
	}
	ts.Volumes = OverrideVolumes(ts.Volumes, override.Volumes, true)
}

// OverrideStep overrides the step with the given overrides
func OverrideStep(step *tektonv1beta1.Step, override *tektonv1beta1.Step) {
	if len(override.Command) > 0 {
		step.Script = override.Script
		step.Command = override.Command
		step.Args = override.Args
	}
	if override.Script != "" {
		step.Script = override.Script
		step.Command = nil
		step.Args = nil
	}
	if override.Timeout != nil {
		step.Timeout = override.Timeout
	}
	OverrideContainer(&step.Container, &override.Container, true)
}

// OverrideContainer overrides the container properties
func OverrideContainer(c *v1.Container, override *v1.Container, modify bool) {
	c.Env = OverrideEnv(c.Env, override.Env, modify)
	c.EnvFrom = OverrideEnvFrom(c.EnvFrom, override.EnvFrom, modify)
	if string(override.ImagePullPolicy) != "" && (modify || string(c.ImagePullPolicy) == "") {
		c.ImagePullPolicy = override.ImagePullPolicy
	}
	if c.Lifecycle == nil {
		c.Lifecycle = override.Lifecycle
	}
	if c.LivenessProbe == nil {
		c.LivenessProbe = override.LivenessProbe
	}
	if c.ReadinessProbe == nil {
		c.ReadinessProbe = override.ReadinessProbe
	}
	c.Resources = OverrideResources(c.Resources, override.Resources, modify)
	if c.SecurityContext == nil {
		c.SecurityContext = override.SecurityContext
	}
	if c.StartupProbe == nil {
		c.StartupProbe = override.StartupProbe
	}
	if override.WorkingDir != "" && (modify || c.WorkingDir == "") {
		c.WorkingDir = override.WorkingDir
	}
	c.VolumeMounts = OverrideVolumeMounts(c.VolumeMounts, override.VolumeMounts, modify)
}

// OverrideEnv override either replaces or adds the given env vars
func OverrideEnv(from []v1.EnvVar, overrides []v1.EnvVar, modify bool) []v1.EnvVar {
	for _, override := range overrides {
		found := false
		for i := range from {
			f := &from[i]
			if f.Name == override.Name {
				found = true
				if modify {
					*f = override
				}
				break
			}
		}
		if !found {
			from = append(from, override)
		}
	}
	return from
}

// OverrideEnvFrom override either replaces or adds the given env froms
func OverrideEnvFrom(from []v1.EnvFromSource, overrides []v1.EnvFromSource, modify bool) []v1.EnvFromSource {
	for _, override := range overrides {
		found := false
		for i := range from {
			f := &from[i]
			if f.ConfigMapRef != nil && override.ConfigMapRef != nil && f.ConfigMapRef.Name == override.ConfigMapRef.Name {
				found = true
				if modify {
					*f = override
				}
				break
			}
			if f.SecretRef != nil && override.SecretRef != nil && f.SecretRef.Name == override.SecretRef.Name {
				found = true
				if modify {
					*f = override
				}
				break
			}
		}
		if !found {
			from = append(from, override)
		}
	}
	return from
}

// OverrideVolumes override either replaces or adds the given volumes
func OverrideVolumes(from []v1.Volume, overrides []v1.Volume, modify bool) []v1.Volume {
	for _, override := range overrides {
		found := false
		for i := range from {
			f := &from[i]
			if f.Name == override.Name {
				found = true
				if modify {
					*f = override
				}
				break
			}
		}
		if !found {
			from = append(from, override)
		}
	}
	return from
}

// OverrideVolumeMounts override either replaces or adds the given volume mounts
func OverrideVolumeMounts(from []v1.VolumeMount, overrides []v1.VolumeMount, modify bool) []v1.VolumeMount {
	for _, override := range overrides {
		found := false
		for i := range from {
			f := &from[i]
			if f.Name == override.Name {
				found = true
				if modify {
					*f = override
				}
				break
			}
		}
		if !found {
			from = append(from, override)
		}
	}
	return from
}

// OverrideResources overrides any resources
func OverrideResources(resources v1.ResourceRequirements, override v1.ResourceRequirements, modify bool) v1.ResourceRequirements {
	resources.Limits = OverrideResourceList(resources.Limits, override.Limits, modify)
	resources.Requests = OverrideResourceList(resources.Requests, override.Requests, modify)
	return resources
}

// OverrideResourceList overrides resource list properties
func OverrideResourceList(requests v1.ResourceList, override v1.ResourceList, modify bool) v1.ResourceList {
	if override == nil {
		return requests
	}
	if requests == nil {
		requests = v1.ResourceList{}
	}
	for k, v := range override {
		_, ok := requests[k]
		if modify || !ok {
			requests[k] = v
		}
	}
	return requests
}
