/*
Copyright 2016 The Kubernetes Authors.

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

package jenkins

import (
	"strconv"

	"github.com/sirupsen/logrus"
)

const (
	// Key for unique build number across Jenkins builds.
	// Used for allowing tools to group artifacts in GCS.
	statusBuildID = "BUILD_ID"
	// Key for unique build number across Jenkins builds.
	// Used for correlating Jenkins builds to LighthouseJobs.
	lighthouseJobID = "LIGHTHOUSE_JOB_ID"
)

const (
	success  = "SUCCESS"
	failure  = "FAILURE"
	unstable = "UNSTABLE"
	aborted  = "ABORTED"
)

// Action holds a list of parameters
type Action struct {
	Parameters []Parameter `json:"parameters"`
}

// Parameter configures some aspect of the job.
type Parameter struct {
	Name string `json:"name"`
	// This needs to be an interface so we won't clobber
	// json unmarshaling when the Jenkins job has more
	// parameter types than strings.
	Value interface{} `json:"value"`
}

// Build holds information about an instance of a jenkins job.
type Build struct {
	Actions []Action `json:"actions"`
	Task    struct {
		// Used for tracking unscheduled builds for jobs.
		Name string `json:"name"`
	} `json:"task"`
	Number   int     `json:"number"`
	Result   *string `json:"result"`
	enqueued bool
}

// ParameterDefinition holds information about a build parameter
type ParameterDefinition struct {
	DefaultParameterValue Parameter `json:"defaultParameterValue,omitempty"`
	Description           string    `json:"description"`
	Name                  string    `json:"name"`
	Type                  string    `json:"type"`
}

// JobProperty is a generic Jenkins job property,
// but ParameterDefinitions is specific to Build Parameters
type JobProperty struct {
	Class                string                `json:"_class"`
	ParameterDefinitions []ParameterDefinition `json:"parameterDefinitions,omitempty"`
}

// JobInfo holds information about a job from $job/api/json endpoint
type JobInfo struct {
	Builds    []Build       `json:"builds"`
	LastBuild *Build        `json:"lastBuild,omitempty"`
	Property  []JobProperty `json:"property"`
}

// IsRunning means the job started but has not finished.
func (jb *Build) IsRunning() bool {
	return jb.Result == nil
}

// IsSuccess means the job passed
func (jb *Build) IsSuccess() bool {
	return jb.Result != nil && *jb.Result == success
}

// IsFailure means the job completed with problems.
func (jb *Build) IsFailure() bool {
	return jb.Result != nil && (*jb.Result == failure || *jb.Result == unstable)
}

// IsAborted means something stopped the job before it could finish.
func (jb *Build) IsAborted() bool {
	return jb.Result != nil && *jb.Result == aborted
}

// IsEnqueued means the job has created but has not started.
func (jb *Build) IsEnqueued() bool {
	return jb.enqueued
}

// LighthouseJobID extracts the LighthouseJob identifier for the
// Jenkins build in order to correlate the build with
// a LighthouseJob. If the build has an empty LIGHTHOUSE_JOB_ID
// it didn't start by Lighthouse.
func (jb *Build) LighthouseJobID() string {
	for _, action := range jb.Actions {
		for _, p := range action.Parameters {
			if p.Name == lighthouseJobID {
				value, ok := p.Value.(string)
				if !ok {
					logrus.Errorf("Cannot determine %s value for %#v", p.Name, jb)
					continue
				}
				return value
			}
		}
	}
	return ""
}

// BuildID extracts the Jenkins build number
// We return an empty string if we are dealing with
// a build that does not have the LighthouseJobID set
// explicitly, as in that case the Jenkins build has
// not started by Lighthouse.
func (jb *Build) BuildID() string {
	var buildID string
	hasLighthouseJobID := false
	for _, action := range jb.Actions {
		for _, p := range action.Parameters {
			hasLighthouseJobID = hasLighthouseJobID || p.Name == lighthouseJobID
			buildID = strconv.Itoa(jb.Number)
		}
	}

	if !hasLighthouseJobID {
		return ""
	}
	return buildID
}
