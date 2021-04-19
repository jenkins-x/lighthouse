package inrepo

import (
	"context"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// UseLocation defines the location where we are using one or more steps where we may need to modify
// the parameters, results and workspaces
type UseLocation struct {
	PipelineRunSpec *v1beta1.PipelineRunSpec
	PipelineSpec    *v1beta1.PipelineSpec
	PipelineTask    *v1beta1.PipelineTask
	TaskName        string
	TaskRunSpec     *v1beta1.TaskRunSpec
	TaskSpec        *v1beta1.TaskSpec
}

// UseParametersAndResults adds the parameters from the used Task to the PipelineSpec if specified and the PipelineTask
func UseParametersAndResults(ctx context.Context, loc *UseLocation, uses *v1beta1.TaskSpec) error {
	parameterSpecs := uses.Params
	parameters := ToParams(parameterSpecs)
	results := uses.Results

	prs := loc.PipelineRunSpec
	if prs != nil {
		prs.Params = useParameters(prs.Params, ToDefaultParams(parameterSpecs))
		prs.Workspaces = useWorkspaceBindings(prs.Workspaces, ToWorkspaceBindings(uses.Workspaces))
	}
	ps := loc.PipelineSpec
	if ps != nil {
		ps.Params = useParameterSpecs(ctx, ps.Params, parameterSpecs)
		ps.Results = usePipelineResults(ps.Results, results)
		ps.Workspaces = usePipelineWorkspaces(ps.Workspaces, uses.Workspaces)
	}
	pt := loc.PipelineTask
	if pt != nil {
		pt.Workspaces = useWorkspaceTaskBindings(pt.Workspaces, ToWorkspacePipelineTaskBindingsFromDeclarations(uses.Workspaces))
	}
	trs := loc.TaskRunSpec
	if trs != nil {
		trs.Params = useParameters(trs.Params, parameters)
	}
	ts := loc.TaskSpec
	if ts != nil {
		ts.Params = useParameterSpecs(ctx, ts.Params, parameterSpecs)
		ts.Results = useResults(ts.Results, results)
		ts.Sidecars = useSidecars(ts.Sidecars, uses.Sidecars)
		ts.Workspaces = useWorkspaces(ts.Workspaces, uses.Workspaces)

		// lets create a step template if its not already defined
		if len(parameters) > 0 {
			stepTemplate := ts.StepTemplate
			created := false
			if stepTemplate == nil {
				stepTemplate = &corev1.Container{}
				created = true
			}
			stepTemplate.Env = useParameterEnvVars(stepTemplate.Env, parameters)
			if len(stepTemplate.Env) > 0 && created {
				ts.StepTemplate = stepTemplate
			}
		}
	}
	return nil
}

// ToDefaultParams converts the param specs to default params
func ToDefaultParams(params []v1beta1.ParamSpec) []v1beta1.Param {
	var answer []v1beta1.Param
	for _, p := range params {
		value := v1beta1.ArrayOrString{
			Type: v1beta1.ParamTypeString,
		}
		d := p.Default
		if d != nil {
			value.StringVal = d.StringVal
			value.ArrayVal = d.ArrayVal
		}
		answer = append(answer, v1beta1.Param{
			Name:  p.Name,
			Value: value,
		})
	}
	return answer
}

func useParameterSpecs(ctx context.Context, params []v1beta1.ParamSpec, uses []v1beta1.ParamSpec) []v1beta1.ParamSpec {
	for _, u := range uses {
		found := false
		for i := range params {
			param := &params[i]
			if param.Name == u.Name {
				found = true
				if param.Description == "" {
					param.Description = u.Description
				}
				param.SetDefaults(ctx)
				break
			}
		}
		if !found {
			u.SetDefaults(ctx)
			params = append(params, u)
		}
	}
	return params
}

func useParameters(params []v1beta1.Param, uses []v1beta1.Param) []v1beta1.Param {
	for _, u := range uses {
		found := false
		for i := range params {
			p := &params[i]
			if p.Name == u.Name {
				found = true
				if p.Value.Type == u.Value.Type {
					switch p.Value.Type {
					case v1beta1.ParamTypeString:
						if p.Value.StringVal == "" {
							p.Value.StringVal = u.Value.StringVal
						}
					case v1beta1.ParamTypeArray:
						if len(p.Value.ArrayVal) == 0 {
							p.Value.ArrayVal = u.Value.ArrayVal
						}
					}
				}
				break
			}
		}
		if !found {
			params = append(params, u)
		}
	}
	return params
}

func useParameterEnvVars(env []corev1.EnvVar, uses []v1beta1.Param) []corev1.EnvVar {
	for _, u := range uses {
		name := u.Name
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
					p.Value = u.Value.StringVal
				}
				break
			}
		}
		if !found {
			env = append(env, corev1.EnvVar{
				Name:  name,
				Value: u.Value.StringVal,
			})
		}
	}
	return env
}

func usePipelineResults(results []v1beta1.PipelineResult, uses []v1beta1.TaskResult) []v1beta1.PipelineResult {
	for _, u := range uses {
		found := false
		for i := range results {
			param := &results[i]
			if param.Name == u.Name {
				found = true
				if param.Description == "" {
					param.Description = u.Description
				}
				break
			}
		}
		if !found {
			results = append(results, v1beta1.PipelineResult{
				Name:        u.Name,
				Description: u.Description,
			})
		}
	}
	return results
}

func useResults(results []v1beta1.TaskResult, uses []v1beta1.TaskResult) []v1beta1.TaskResult {
	for _, u := range uses {
		found := false
		for i := range results {
			param := &results[i]
			if param.Name == u.Name {
				found = true
				if param.Description == "" {
					param.Description = u.Description
				}
				break
			}
		}
		if !found {
			results = append(results, u)
		}
	}
	return results
}

func useWorkspaceTaskBindings(ws []v1beta1.WorkspacePipelineTaskBinding, uses []v1beta1.WorkspacePipelineTaskBinding) []v1beta1.WorkspacePipelineTaskBinding {
	for _, u := range uses {
		found := false
		for i := range ws {
			param := &ws[i]
			if param.Name == u.Name {
				found = true
				break
			}
		}
		if !found {
			ws = append(ws, u)
		}
	}
	return ws
}

func usePipelineWorkspaces(ws []v1beta1.PipelineWorkspaceDeclaration, uses []v1beta1.WorkspaceDeclaration) []v1beta1.PipelineWorkspaceDeclaration {
	for _, u := range uses {
		found := false
		for i := range ws {
			param := &ws[i]
			if param.Name == u.Name {
				found = true
				if param.Description == "" {
					param.Description = u.Description
				}
				break
			}
		}
		if !found {
			ws = append(ws, v1beta1.PipelineWorkspaceDeclaration{
				Name:        u.Name,
				Description: u.Description,
				Optional:    u.Optional,
			})
		}
	}
	return ws
}

func useSidecars(ws []v1beta1.Sidecar, uses []v1beta1.Sidecar) []v1beta1.Sidecar {
	for _, u := range uses {
		found := false
		for i := range ws {
			param := &ws[i]
			if param.Name == u.Name {
				found = true
				break
			}
		}
		if !found {
			ws = append(ws, u)
		}
	}
	return ws
}

func useWorkspaces(ws []v1beta1.WorkspaceDeclaration, uses []v1beta1.WorkspaceDeclaration) []v1beta1.WorkspaceDeclaration {
	for _, u := range uses {
		found := false
		for i := range ws {
			param := &ws[i]
			if param.Name == u.Name {
				found = true
				if param.Description == "" {
					param.Description = u.Description
				}
				break
			}
		}
		if !found {
			ws = append(ws, u)
		}
	}
	return ws
}

func useWorkspaceBindings(ws []v1beta1.WorkspaceBinding, uses []v1beta1.WorkspaceBinding) []v1beta1.WorkspaceBinding {
	for _, u := range uses {
		found := false
		for i := range ws {
			param := &ws[i]
			if param.Name == u.Name {
				found = true
				break
			}
		}
		if !found {
			ws = append(ws, u)
		}
	}
	return ws
}
