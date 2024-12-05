package inrepo

import (
	"context"
	"fmt"
	"strings"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
)

// UseLocation defines the location where we are using one or more steps where we may need to modify
// the parameters, results and workspaces
type UseLocation struct {
	PipelineRunSpec *pipelinev1.PipelineRunSpec
	PipelineSpec    *pipelinev1.PipelineSpec
	PipelineTask    *pipelinev1.PipelineTask
	TaskName        string
	TaskRunSpec     *pipelinev1.TaskRunSpec
	TaskSpec        *pipelinev1.TaskSpec
}

func getParamsFromTasksResults(loc *UseLocation) map[string]bool {
	areParamsFromTasksResults := make(map[string]bool)
	pipelineTasksAndFinally := loc.PipelineSpec.Tasks
	pipelineTasksAndFinally = append(pipelineTasksAndFinally, loc.PipelineSpec.Finally...)
	for _, pipelineTask := range pipelineTasksAndFinally {
		params := pipelineTask.Params
		for _, param := range params {
			paramValStr := param.Value.StringVal
			if strings.HasPrefix(paramValStr, "$(tasks.") && strings.Contains(paramValStr, ".results.") && strings.HasSuffix(paramValStr, ")") {
				areParamsFromTasksResults[param.Name] = true
			}
		}
	}
	return areParamsFromTasksResults
}

// UseParametersAndResults adds the parameters from the used Task to the PipelineSpec if specified and the PipelineTask
func UseParametersAndResults(ctx context.Context, loc *UseLocation, uses *pipelinev1.TaskSpec) {
	parameterSpecs := uses.Params
	parameters := ToParams(parameterSpecs)
	results := uses.Results
	areParamsFromTasksResults := getParamsFromTasksResults(loc)

	prs := loc.PipelineRunSpec
	if prs != nil {
		prs.Params = useParameters(prs.Params, ToDefaultParams(parameterSpecs), areParamsFromTasksResults)
		prs.Workspaces = useWorkspaceBindings(prs.Workspaces, ToWorkspaceBindings(uses.Workspaces))
	}
	ps := loc.PipelineSpec
	if ps != nil {
		ps.Params = useParameterSpecs(ctx, ps.Params, parameterSpecs, areParamsFromTasksResults)
		ps.Results = usePipelineResults(ps.Results, results, loc.TaskName)
		ps.Workspaces = usePipelineWorkspaces(ps.Workspaces, uses.Workspaces)
	}
	pt := loc.PipelineTask
	if pt != nil {
		pt.Workspaces = useWorkspaceTaskBindings(pt.Workspaces, ToWorkspacePipelineTaskBindingsFromDeclarations(uses.Workspaces))
	}
	trs := loc.TaskRunSpec
	if trs != nil {
		trs.Params = useParameters(trs.Params, parameters, areParamsFromTasksResults)
	}
	ts := loc.TaskSpec
	if ts != nil {
		ts.Params = useParameterSpecs(ctx, ts.Params, parameterSpecs, areParamsFromTasksResults)
		ts.Results = useResults(ts.Results, results)
		ts.Sidecars = useSidecars(ts.Sidecars, uses.Sidecars)
		ts.Workspaces = useWorkspaces(ts.Workspaces, uses.Workspaces)

		// lets create a step template if its not already defined
		if len(parameters) > 0 {
			stepTemplate := ts.StepTemplate
			created := false
			if stepTemplate == nil {
				stepTemplate = &pipelinev1.StepTemplate{}
				created = true
			}
			stepTemplate.Env = useParameterEnvVars(stepTemplate.Env, parameters)
			if len(stepTemplate.Env) > 0 && created {
				ts.StepTemplate = stepTemplate
			}
		}
	}
}

// ToDefaultParams converts the param specs to default params
func ToDefaultParams(params []pipelinev1.ParamSpec) []pipelinev1.Param {
	var answer []pipelinev1.Param
	for _, p := range params {
		value := pipelinev1.ParamValue{
			Type: pipelinev1.ParamTypeString,
		}
		d := p.Default
		if d != nil {
			value.StringVal = d.StringVal
			value.ArrayVal = d.ArrayVal
		}
		answer = append(answer, pipelinev1.Param{
			Name:  p.Name,
			Value: value,
		})
	}
	return answer
}

func useParameterSpecs(ctx context.Context, params []pipelinev1.ParamSpec, uses []pipelinev1.ParamSpec, areParamsFromTasksResults map[string]bool) []pipelinev1.ParamSpec {
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
			if areParamsFromTasksResults == nil || !areParamsFromTasksResults[u.Name] {
				params = append(params, u)
			}
		}
	}
	return params
}

func useParameters(params []pipelinev1.Param, uses []pipelinev1.Param, areParamsFromTasksResults map[string]bool) []pipelinev1.Param {
	for _, u := range uses {
		found := false
		for i := range params {
			p := &params[i]
			if p.Name == u.Name {
				found = true
				if p.Value.Type == u.Value.Type {
					switch p.Value.Type {
					case pipelinev1.ParamTypeString:
						if p.Value.StringVal == "" {
							p.Value.StringVal = u.Value.StringVal
						}
					case pipelinev1.ParamTypeArray:
						if len(p.Value.ArrayVal) == 0 {
							p.Value.ArrayVal = u.Value.ArrayVal
						}
					}
				}
				break
			}
		}
		if !found {
			if areParamsFromTasksResults == nil || !areParamsFromTasksResults[u.Name] {
				params = append(params, u)
			}
		}
	}
	return params
}

func useParameterEnvVars(env []corev1.EnvVar, uses []pipelinev1.Param) []corev1.EnvVar {
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

func usePipelineResults(results []pipelinev1.PipelineResult, uses []pipelinev1.TaskResult, taskName string) []pipelinev1.PipelineResult {
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
			results = append(results, pipelinev1.PipelineResult{
				Name:        u.Name,
				Description: u.Description,
				Value:       *pipelinev1.NewStructuredValues(fmt.Sprintf("$(tasks.%s.results.%s)", taskName, u.Name)),
			})
		}
	}
	return results
}

func useResults(results []pipelinev1.TaskResult, uses []pipelinev1.TaskResult) []pipelinev1.TaskResult {
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

func useWorkspaceTaskBindings(ws []pipelinev1.WorkspacePipelineTaskBinding, uses []pipelinev1.WorkspacePipelineTaskBinding) []pipelinev1.WorkspacePipelineTaskBinding {
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

func usePipelineWorkspaces(ws []pipelinev1.PipelineWorkspaceDeclaration, uses []pipelinev1.WorkspaceDeclaration) []pipelinev1.PipelineWorkspaceDeclaration {
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
			ws = append(ws, pipelinev1.PipelineWorkspaceDeclaration{
				Name:        u.Name,
				Description: u.Description,
				Optional:    u.Optional,
			})
		}
	}
	return ws
}

func useSidecars(ws []pipelinev1.Sidecar, uses []pipelinev1.Sidecar) []pipelinev1.Sidecar {
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

func useWorkspaces(ws []pipelinev1.WorkspaceDeclaration, uses []pipelinev1.WorkspaceDeclaration) []pipelinev1.WorkspaceDeclaration {
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

func useWorkspaceBindings(ws []pipelinev1.WorkspaceBinding, uses []pipelinev1.WorkspaceBinding) []pipelinev1.WorkspaceBinding {
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
