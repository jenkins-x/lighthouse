package inrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	// TektonAPIVersion the default tekton API version
	TektonAPIVersion = "tekton.dev/v1beta1"

	// LoadFileRefPattern the regular expression to match which Pipeline/Task references to load via files
	LoadFileRefPattern = "lighthouse.jenkins-x.io/loadFileRefs"
)

// DefaultValues default values applied to a PipelineRun if wrapping a Pipeline/Task/TaskRun as a PipelineRun
type DefaultValues struct {
	// ServiceAccountName
	ServiceAccountName string
	//  Timeout
	Timeout *metav1.Duration
}

// NewDefaultValues creatse a new default values
func NewDefaultValues() (*DefaultValues, error) {
	answer := &DefaultValues{
		ServiceAccountName: os.Getenv("DEFAULT_PIPELINE_RUN_SERVICE_ACCOUNT"),
	}
	timeout := os.Getenv("DEFAULT_PIPELINE_RUN_TIMEOUT")
	if timeout != "" {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			return answer, errors.Wrapf(err, "failed to parse duration %s", timeout)
		}
		answer.Timeout = &metav1.Duration{
			Duration: duration,
		}
	}
	return answer, nil
}

// LoadTektonResourceAsPipelineRun loads a PipelineRun, Pipeline, Task or TaskRun and convert it to a PipelineRun
// if necessary
func LoadTektonResourceAsPipelineRun(data []byte, dir, message string, getData func(path string) ([]byte, error), defaultValues *DefaultValues) (*tektonv1beta1.PipelineRun, error) {
	if defaultValues == nil {
		var err error
		defaultValues, err = NewDefaultValues()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse default values")
		}
	}
	kindPrefix := "kind:"
	kind := "PipelineRun"
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, kindPrefix) {
			continue
		}
		k := strings.TrimSpace(line[len(kindPrefix):])
		if k != "" {
			kind = k
			break
		}
	}
	switch kind {
	case "Pipeline":
		pipeline := &tektonv1beta1.Pipeline{}
		err := yaml.Unmarshal(data, pipeline)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal Pipeline YAML %s", message)
		}
		prs, err := ConvertPipelineToPipelineRun(pipeline, message, defaultValues)
		if err == nil {
			re, err := loadTektonRefsFromFilesPattern(prs)
			if err != nil {
				return prs, err
			}
			if re != nil {
				return loadPipelineRunRefs(prs, dir, message, re, getData)
			}
		}
		return prs, err

	case "PipelineRun":
		prs := &tektonv1beta1.PipelineRun{}
		err := yaml.Unmarshal(data, prs)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal PipelineRun YAML %s", message)
		}

		re, err := loadTektonRefsFromFilesPattern(prs)
		if err != nil {
			return prs, err
		}
		if re != nil {
			return loadPipelineRunRefs(prs, dir, message, re, getData)
		}
		return prs, err
	case "Task":
		task := &tektonv1beta1.Task{}
		err := yaml.Unmarshal(data, task)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal Task YAML %s", message)
		}
		prs, err := ConvertTaskToPipelineRun(task, message, defaultValues)
		if err == nil {
			re, err := loadTektonRefsFromFilesPattern(prs)
			if err != nil {
				return prs, err
			}
			if re != nil {
				return loadPipelineRunRefs(prs, dir, message, re, getData)
			}
		}
		return prs, err

	case "TaskRun":
		tr := &tektonv1beta1.TaskRun{}
		err := yaml.Unmarshal(data, tr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal TaskRun YAML %s", message)
		}
		prs, err := ConvertTaskRunToPipelineRun(tr, message, defaultValues)
		if err == nil {
			re, err := loadTektonRefsFromFilesPattern(prs)
			if err != nil {
				return prs, err
			}
			if re != nil {
				return loadPipelineRunRefs(prs, dir, message, re, getData)
			}
		}
		return prs, err

	default:
		return nil, errors.Errorf("kind %s is not supported for %s", kind, message)
	}
}

// loadTektonRefsFromFilesPattern returns a regular expression matching the Pipeline/Task references we should load via the file system
// as separate local files
func loadTektonRefsFromFilesPattern(prs *tektonv1beta1.PipelineRun) (*regexp.Regexp, error) {
	if prs.Annotations == nil {
		return nil, nil
	}
	pattern := prs.Annotations[LoadFileRefPattern]
	if pattern == "" {
		return nil, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse annotation %s value %s as a regular expression", LoadFileRefPattern, pattern)
	}
	return re, nil
}

func loadPipelineRunRefs(prs *tektonv1beta1.PipelineRun, dir, message string, re *regexp.Regexp, getData func(path string) ([]byte, error)) (*tektonv1beta1.PipelineRun, error) {
	// if we reference a local
	if prs.Spec.PipelineSpec == nil && prs.Spec.PipelineRef != nil && prs.Spec.PipelineRef.Name != "" && re.MatchString(prs.Spec.PipelineRef.Name) {
		pipelinePath := filepath.Join(dir, prs.Spec.PipelineRef.Name)
		if !strings.HasSuffix(pipelinePath, ".yaml") {
			pipelinePath += ".yaml"
		}
		data, err := getData(pipelinePath)
		if err == nil && len(data) > 0 {
			p := &tektonv1beta1.Pipeline{}
			err = yaml.Unmarshal(data, p)
			if err != nil {
				return prs, errors.Wrapf(err, "failed to unmarshal Pipeline YAML file %s %s", pipelinePath, message)
			}
			prs.Spec.PipelineSpec = &p.Spec
			prs.Spec.PipelineRef = nil
		}
	}

	if prs.Spec.PipelineSpec != nil {
		err := loadTaskRefs(prs.Spec.PipelineSpec, dir, message, re, getData)
		if err != nil {
			return prs, errors.Wrapf(err, "failed to load Task refs for %s", message)
		}
	}
	return prs, nil
}

func loadTaskRefs(pipelineSpec *tektonv1beta1.PipelineSpec, dir, message string, re *regexp.Regexp, getData func(path string) ([]byte, error)) error {
	for i := range pipelineSpec.Tasks {
		t := &pipelineSpec.Tasks[i]
		if t.TaskSpec == nil && t.TaskRef != nil && t.TaskRef.Name != "" && re.MatchString(t.TaskRef.Name) {
			path := filepath.Join(dir, t.TaskRef.Name)
			if !strings.HasSuffix(path, ".yaml") {
				path += ".yaml"
			}
			data, err := getData(path)
			if err == nil && len(data) > 0 {
				t2 := &tektonv1beta1.Task{}
				err = yaml.Unmarshal(data, t2)
				if err != nil {
					return errors.Wrapf(err, "failed to unmarshal Task YAML file %s %s", path, message)
				}
				t.TaskSpec = &t2.Spec
				t.TaskRef = nil
			}
		}
	}
	return nil
}

// ConvertPipelineToPipelineRun converts the Pipeline to a PipelineRun
func ConvertPipelineToPipelineRun(from *tektonv1beta1.Pipeline, message string, defaultValues *DefaultValues) (*tektonv1beta1.PipelineRun, error) {
	prs := &tektonv1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: TektonAPIVersion,
		},
	}
	prs.Name = from.Name
	prs.Annotations = from.Annotations
	prs.Labels = from.Labels

	prs.Spec.PipelineSpec = &from.Spec
	defaultValues.Apply(prs)
	return prs, nil
}

// ConvertTaskToPipelineRun converts the Task to a PipelineRun
func ConvertTaskToPipelineRun(from *tektonv1beta1.Task, message string, defaultValues *DefaultValues) (*tektonv1beta1.PipelineRun, error) {
	prs := &tektonv1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: TektonAPIVersion,
		},
	}
	prs.Name = from.Name
	prs.Annotations = from.Annotations
	prs.Labels = from.Labels

	fs := &from.Spec
	pipelineSpec := &tektonv1beta1.PipelineSpec{
		Description: "",
		Resources:   nil,
		Tasks: []tektonv1beta1.PipelineTask{
			{
				Name:       from.Name,
				TaskSpec:   fs,
				Resources:  ToPipelineResources(fs.Resources),
				Params:     ToParams(fs.Params),
				Workspaces: ToWorkspacePipelineTaskBindingsFromDeclarations(fs.Workspaces),
			},
		},
		Params:     fs.Params,
		Workspaces: ToPipelineWorkspaceDeclarations(fs.Workspaces),
		//Results:    fs.Results,
		Finally: nil,
	}
	prs.Spec.PipelineSpec = pipelineSpec
	prs.Spec.Params = ToParams(fs.Params)
	//prs.Spec.Resources = fs.Resources
	prs.Spec.Workspaces = ToWorkspaceBindings(fs.Workspaces)
	defaultValues.Apply(prs)
	return prs, nil
}

// ConvertTaskRunToPipelineRun converts the TaskRun to a PipelineRun
func ConvertTaskRunToPipelineRun(from *tektonv1beta1.TaskRun, message string, defaultValues *DefaultValues) (*tektonv1beta1.PipelineRun, error) {
	prs := &tektonv1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: TektonAPIVersion,
		},
	}
	prs.Name = from.Name
	prs.Annotations = from.Annotations
	prs.Labels = from.Labels

	fs := &from.Spec
	params := fs.Params
	var paramSpecs []tektonv1beta1.ParamSpec
	if len(params) == 0 && fs.TaskSpec != nil {
		paramSpecs = fs.TaskSpec.Params
		if len(params) == 0 {
			params = ToParams(paramSpecs)
		}
	}
	if len(paramSpecs) == 0 {
		paramSpecs = ToParamSpecs(params)
	}
	pipelineSpec := &tektonv1beta1.PipelineSpec{
		Description: "",
		Resources:   nil,
		Tasks: []tektonv1beta1.PipelineTask{
			{
				Name:     from.Name,
				TaskRef:  fs.TaskRef,
				TaskSpec: fs.TaskSpec,
				//Resources: fs.Resources,
				Params:     params,
				Workspaces: ToWorkspacePipelineTaskBindings(fs.Workspaces),
			},
		},
		Params:     paramSpecs,
		Workspaces: nil,
		Results:    nil,
		Finally:    nil,
	}
	prs.Spec.PipelineSpec = pipelineSpec
	prs.Spec.Params = params
	prs.Spec.PodTemplate = fs.PodTemplate
	//prs.Spec.Resources = fs.Resources
	prs.Spec.ServiceAccountName = fs.ServiceAccountName
	prs.Spec.Workspaces = fs.Workspaces
	defaultValues.Apply(prs)
	return prs, nil
}

// Apply adds any default values that are empty in the generated PipelineRun
func (v *DefaultValues) Apply(prs *tektonv1beta1.PipelineRun) {
	if prs.Spec.ServiceAccountName == "" && v.ServiceAccountName != "" {
		prs.Spec.ServiceAccountName = v.ServiceAccountName
	}
	if prs.Spec.Timeout == nil && v.Timeout != nil {
		prs.Spec.Timeout = v.Timeout
	}
}

// ToParams convers the param specs to params
func ToParams(params []tektonv1beta1.ParamSpec) []tektonv1beta1.Param {
	var answer []tektonv1beta1.Param
	for _, p := range params {
		answer = append(answer, tektonv1beta1.Param{
			Name: p.Name,
			Value: tektonv1beta1.ArrayOrString{
				Type:      tektonv1beta1.ParamTypeString,
				StringVal: fmt.Sprintf("$(params.%s)", p.Name),
			},
		})
	}
	return answer
}

// ToParamSpecs generates param specs from the params
func ToParamSpecs(params []tektonv1beta1.Param) []tektonv1beta1.ParamSpec {
	var answer []tektonv1beta1.ParamSpec
	for _, p := range params {
		answer = append(answer, tektonv1beta1.ParamSpec{
			Name: p.Name,
			// lets assume strings for now
			Type:        tektonv1beta1.ParamTypeString,
			Description: "",
			Default:     nil,
		})
	}
	return answer
}

// ToPipelineResources converts the task resources to piepline resources
func ToPipelineResources(resources *tektonv1beta1.TaskResources) *tektonv1beta1.PipelineTaskResources {
	if resources == nil {
		return nil
	}
	return &tektonv1beta1.PipelineTaskResources{
		Inputs:  ToPipelineInputs(resources.Inputs),
		Outputs: ToPipelineOutputs(resources.Inputs),
	}
}

// ToPipelineInputs converts the task resources into pipeline inputs
func ToPipelineInputs(inputs []tektonv1beta1.TaskResource) []tektonv1beta1.PipelineTaskInputResource {
	var answer []tektonv1beta1.PipelineTaskInputResource
	for _, from := range inputs {
		answer = append(answer, ToPipelineInput(from))
	}
	return answer
}

// ToPipelineOutputs converts the task resources into pipeline outputs
func ToPipelineOutputs(inputs []tektonv1beta1.TaskResource) []tektonv1beta1.PipelineTaskOutputResource {
	var answer []tektonv1beta1.PipelineTaskOutputResource
	for _, from := range inputs {
		answer = append(answer, ToPipelineOutput(from))
	}
	return answer
}

// ToPipelineInput converts the task resource into pipeline inputs
func ToPipelineInput(from tektonv1beta1.TaskResource) tektonv1beta1.PipelineTaskInputResource {
	return tektonv1beta1.PipelineTaskInputResource{
		Name:     from.Name,
		Resource: from.ResourceDeclaration.Name,
		From:     nil,
	}
}

// ToPipelineOutput converts the task resource into pipeline outputs
func ToPipelineOutput(from tektonv1beta1.TaskResource) tektonv1beta1.PipelineTaskOutputResource {
	return tektonv1beta1.PipelineTaskOutputResource{
		Name:     from.Name,
		Resource: from.ResourceDeclaration.Name,
	}
}

// ToWorkspaceBindings converts the workspace declarations to workspaces bindings
func ToWorkspaceBindings(workspaces []tektonv1beta1.WorkspaceDeclaration) []tektonv1beta1.WorkspaceBinding {
	var answer []tektonv1beta1.WorkspaceBinding
	for _, from := range workspaces {
		answer = append(answer, ToWorkspaceBinding(from))
	}
	return answer
}

// ToWorkspaceBinding converts the workspace declaration to a workspaces binding
func ToWorkspaceBinding(from tektonv1beta1.WorkspaceDeclaration) tektonv1beta1.WorkspaceBinding {
	return tektonv1beta1.WorkspaceBinding{
		Name: from.Name,
	}
}

// ToWorkspacePipelineTaskBindings converts the workspace bindings to pipeline task bindings
func ToWorkspacePipelineTaskBindings(workspaces []tektonv1beta1.WorkspaceBinding) []tektonv1beta1.WorkspacePipelineTaskBinding {
	var answer []tektonv1beta1.WorkspacePipelineTaskBinding
	for _, from := range workspaces {
		answer = append(answer, ToWorkspacePipelineTaskBinding(from))
	}
	return answer
}

// ToWorkspacePipelineTaskBinding converts the workspace binding to a pipeline task binding
func ToWorkspacePipelineTaskBinding(from tektonv1beta1.WorkspaceBinding) tektonv1beta1.WorkspacePipelineTaskBinding {
	return tektonv1beta1.WorkspacePipelineTaskBinding{
		Name:      from.Name,
		Workspace: from.Name,
		SubPath:   from.SubPath,
	}
}

// ToWorkspacePipelineTaskBindingsFromDeclarations converts the workspace declarations to pipeline task bindings
func ToWorkspacePipelineTaskBindingsFromDeclarations(workspaces []tektonv1beta1.WorkspaceDeclaration) []tektonv1beta1.WorkspacePipelineTaskBinding {
	var answer []tektonv1beta1.WorkspacePipelineTaskBinding
	for _, from := range workspaces {
		answer = append(answer, ToWorkspacePipelineTaskBindingsFromDeclaration(from))
	}
	return answer
}

// ToWorkspacePipelineTaskBindingsFromDeclaration converts the workspace declaration to a pipeline task binding
func ToWorkspacePipelineTaskBindingsFromDeclaration(from tektonv1beta1.WorkspaceDeclaration) tektonv1beta1.WorkspacePipelineTaskBinding {
	return tektonv1beta1.WorkspacePipelineTaskBinding{
		Name:      from.Name,
		Workspace: from.Name,
		SubPath:   "",
	}
}

// ToPipelineWorkspaceDeclarations converts the workspace declarations to pipeline workspace declarations
func ToPipelineWorkspaceDeclarations(workspaces []tektonv1beta1.WorkspaceDeclaration) []tektonv1beta1.PipelineWorkspaceDeclaration {
	var answer []tektonv1beta1.PipelineWorkspaceDeclaration
	for _, from := range workspaces {
		answer = append(answer, ToPipelineWorkspaceDeclaration(from))
	}
	return answer
}

// ToPipelineWorkspaceDeclaration converts the workspace declaration to a pipeline workspace declaration
func ToPipelineWorkspaceDeclaration(from tektonv1beta1.WorkspaceDeclaration) tektonv1beta1.PipelineWorkspaceDeclaration {
	return tektonv1beta1.PipelineWorkspaceDeclaration{
		Name:        from.Name,
		Description: from.Description,
	}
}
