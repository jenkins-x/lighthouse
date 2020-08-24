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
	"fmt"

	v1 "k8s.io/api/core/v1"
)

// Preset is intended to match the k8s' PodPreset feature, and may be removed
// if that feature goes beta.
type Preset struct {
	Labels       map[string]string `json:"labels"`
	Env          []v1.EnvVar       `json:"env"`
	Volumes      []v1.Volume       `json:"volumes"`
	VolumeMounts []v1.VolumeMount  `json:"volumeMounts"`
}

// MergePreset merges a preset and labels with a pod spec
func MergePreset(preset Preset, labels map[string]string, pod *v1.PodSpec) error {
	if pod == nil {
		return nil
	}
	for l, v := range preset.Labels {
		if v2, ok := labels[l]; !ok || v2 != v {
			return nil
		}
	}
	for _, e1 := range preset.Env {
		for i := range pod.Containers {
			for _, e2 := range pod.Containers[i].Env {
				if e1.Name == e2.Name {
					return fmt.Errorf("env var duplicated in pod spec: %s", e1.Name)
				}
			}
			pod.Containers[i].Env = append(pod.Containers[i].Env, e1)
		}
	}
	for _, v1 := range preset.Volumes {
		for _, v2 := range pod.Volumes {
			if v1.Name == v2.Name {
				return fmt.Errorf("volume duplicated in pod spec: %s", v1.Name)
			}
		}
		pod.Volumes = append(pod.Volumes, v1)
	}
	for _, vm1 := range preset.VolumeMounts {
		for i := range pod.Containers {
			for _, vm2 := range pod.Containers[i].VolumeMounts {
				if vm1.Name == vm2.Name {
					return fmt.Errorf("volume mount duplicated in pod spec: %s", vm1.Name)
				}
			}
			pod.Containers[i].VolumeMounts = append(pod.Containers[i].VolumeMounts, vm1)
		}
	}
	return nil
}
