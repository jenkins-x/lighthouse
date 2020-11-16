package inrepo

import (
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

var (
	// defaultParameterSpecs the default Lighthouse Pipeline Parameters which can be injected by the
	// lighthouse tekton engine
	defaultParameterSpecs = []v1beta1.ParamSpec{
		{
			Description: "the unique build number",
			Name:        "BUILD_ID",
			Type:        "string",
		},
		{
			Description: "the name of the job which is the trigger context name",
			Name:        "JOB_NAME",
			Type:        "string",
		},
		{
			Description: "the specification of the job",
			Name:        "JOB_SPEC",
			Type:        "string",
		},
		{
			Description: "'the kind of job: postsubmit or presubmit'",
			Name:        "JOB_TYPE",
			Type:        "string",
		},
		{
			Description: "the base git reference of the pull request",
			Name:        "PULL_BASE_REF",
			Type:        "string",
		},
		{
			Description: "the git sha of the base of the pull request",
			Name:        "PULL_BASE_SHA",
			Type:        "string",
		},
		{
			Description: "git pull request number",
			Name:        "PULL_NUMBER",
			Type:        "string",
			Default: &v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: "",
			},
		},
		{
			Description: "git pull request ref in the form 'refs/pull/$PULL_NUMBER/head'",
			Name:        "PULL_PULL_REF",
			Type:        "string",
			Default: &v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: "",
			},
		},
		{
			Description: "git revision to checkout (branch, tag, sha, ref…)",
			Name:        "PULL_PULL_SHA",
			Type:        "string",
			Default: &v1beta1.ArrayOrString{
				Type:      v1beta1.ParamTypeString,
				StringVal: "",
			},
		},
		{
			Description: "git pull reference strings of base and latest in the form 'master:$PULL_BASE_SHA,$PULL_NUMBER:$PULL_PULL_SHA:refs/pull/$PULL_NUMBER/head'",
			Name:        "PULL_REFS",
			Type:        "string",
		},
		{
			Description: "git repository name",
			Name:        "REPO_NAME",
			Type:        "string",
		},
		{
			Description: "git repository owner (user or organisation)",
			Name:        "REPO_OWNER",
			Type:        "string",
		},
		{
			Description: "git url to clone",
			Name:        "REPO_URL",
			Type:        "string",
		},
	}

	defaultParameters = ToParams(defaultParameterSpecs)
)

// DefaultPipelineParameters defaults the parameter specs and parameter values from lighthouse onto
// the PipelineRun and its nested PipelineSpec and Tasks
func DefaultPipelineParameters(prs *v1beta1.PipelineRun) (*v1beta1.PipelineRun, error) {
	if prs.Annotations != nil && prs.Annotations[DefaultParameters] == "false" {
		return prs, nil
	}
	ps := prs.Spec.PipelineSpec
	if ps == nil {
		return prs, nil
	}

	ps.Params = addDefaultParameterSpecs(ps.Params, defaultParameterSpecs)

	for i := range ps.Tasks {
		task := &ps.Tasks[i]
		task.Params = addDefaultParameters(task.Params, defaultParameters)
		if task.TaskSpec != nil {
			task.TaskSpec.Params = addDefaultParameterSpecs(task.TaskSpec.Params, defaultParameterSpecs)

			// lets create a step template if its not already defined
			if task.TaskSpec.StepTemplate == nil {
				task.TaskSpec.StepTemplate = &corev1.Container{}
			}
			stepTemplate := task.TaskSpec.StepTemplate
			stepTemplate.Env = addDefaultParameterEnvVars(stepTemplate.Env, defaultParameters)
		}
	}
	return prs, nil
}

func addDefaultParameterSpecs(params []v1beta1.ParamSpec, defaults []v1beta1.ParamSpec) []v1beta1.ParamSpec {
	for _, dp := range defaults {
		found := false
		for i := range params {
			param := &params[i]
			if param.Name == dp.Name {
				found = true
				if param.Description == "" {
					param.Description = dp.Description
				}
				if param.Type == "" {
					param.Type = dp.Type
				}
				break
			}
		}
		if !found {
			params = append(params, dp)
		}
	}
	return params
}

func addDefaultParameters(params []v1beta1.Param, defaults []v1beta1.Param) []v1beta1.Param {
	for _, dp := range defaults {
		found := false
		for i := range params {
			p := &params[i]
			if p.Name == dp.Name {
				found = true
				if p.Value.Type == dp.Value.Type {
					switch p.Value.Type {
					case v1beta1.ParamTypeString:
						if p.Value.StringVal == "" {
							p.Value.StringVal = dp.Value.StringVal
						}
					case v1beta1.ParamTypeArray:
						if len(p.Value.ArrayVal) == 0 {
							p.Value.ArrayVal = dp.Value.ArrayVal
						}
					}
				}
				break
			}
		}
		if !found {
			params = append(params, dp)
		}
	}
	return params
}

func addDefaultParameterEnvVars(env []corev1.EnvVar, defaults []v1beta1.Param) []corev1.EnvVar {
	for _, dp := range defaults {
		name := dp.Name
		upperName := strings.ToUpper(name)
		if upperName != name {
			// ignore parameters which are not already suitable environment names being upper case
			// with optional _ characters
			continue
		}
		found := false
		for i := range env {
			p := &env[i]
			if p.Name == name {
				found = true
				if p.Value == "" {
					p.Value = dp.Value.StringVal
				}
				break
			}
		}
		if !found {
			env = append(env, corev1.EnvVar{
				Name:  name,
				Value: dp.Value.StringVal,
			})
		}
	}
	return env
}
