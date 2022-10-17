package inrepo

import (
	"context"
	"fmt"
	"io"
	"net/http"
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

	// DefaultParameters the annotation used to disable default parameters
	DefaultParameters = "lighthouse.jenkins-x.io/defaultParameters"

	// LoadFileRefPattern the regular expression to match which Pipeline/Task references to load via files
	LoadFileRefPattern = "lighthouse.jenkins-x.io/loadFileRefs"

	// PrependStepURL loads the steps from the given URL and prepends them to the given Task
	PrependStepURL = "lighthouse.jenkins-x.io/prependStepsURL"

	// AppendStepURL loads the steps from the given URL and appends them to the end of the Task steps
	AppendStepURL = "lighthouse.jenkins-x.io/appendStepsURL"
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
func LoadTektonResourceAsPipelineRun(resolver *UsesResolver, data []byte) (*tektonv1beta1.PipelineRun, error) {
	if resolver.DefaultValues == nil {
		var err error
		resolver.DefaultValues, err = NewDefaultValues()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse default values")
		}
	}
	if resolver.Message == "" {
		resolver.Message = fmt.Sprintf("in repo %s/%s with sha %s", resolver.OwnerName, resolver.RepoName, resolver.SHA)
	}
	message := resolver.Message
	dir := resolver.Dir
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
		prs, err := ConvertPipelineToPipelineRun(pipeline, resolver.Message, resolver.DefaultValues)
		if err != nil {
			return prs, err
		}
		re, err := loadTektonRefsFromFilesPattern(prs)
		if err != nil {
			return prs, err
		}
		if re != nil {
			prs, err = loadPipelineRunRefs(resolver, prs, dir, message, re)
			if err != nil {
				return prs, err
			}
		}
		prs, err = inheritTaskSteps(resolver, prs)
		if err != nil {
			return prs, errors.Wrapf(err, "failed to inherit steps")
		}
		return DefaultPipelineParameters(prs)

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
			prs, err = loadPipelineRunRefs(resolver, prs, dir, message, re)
			if err != nil {
				return prs, err
			}
		}
		prs, err = inheritTaskSteps(resolver, prs)
		if err != nil {
			return prs, errors.Wrap(err, "failed to inherit steps")
		}
		return DefaultPipelineParameters(prs)

	case "Task":
		task := &tektonv1beta1.Task{}
		err := yaml.Unmarshal(data, task)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal Task YAML %s", message)
		}
		prs, err := ConvertTaskToPipelineRun(task, message, resolver.DefaultValues)
		if err != nil {
			return prs, err
		}
		re, err := loadTektonRefsFromFilesPattern(prs)
		if err != nil {
			return prs, err
		}
		if re != nil {
			prs, err = loadPipelineRunRefs(resolver, prs, dir, message, re)
			if err != nil {
				return prs, err
			}
		}
		prs, err = inheritTaskSteps(resolver, prs)
		if err != nil {
			return prs, errors.Wrapf(err, "failed to inherit steps")
		}
		defaultTaskName(prs)
		return DefaultPipelineParameters(prs)

	case "TaskRun":
		tr := &tektonv1beta1.TaskRun{}
		err := yaml.Unmarshal(data, tr)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal TaskRun YAML %s", message)
		}
		prs, err := ConvertTaskRunToPipelineRun(tr, message, resolver.DefaultValues)
		if err != nil {
			return prs, err
		}
		re, err := loadTektonRefsFromFilesPattern(prs)
		if err != nil {
			return prs, err
		}
		if re != nil {
			prs, err = loadPipelineRunRefs(resolver, prs, dir, message, re)
			if err != nil {
				return prs, err
			}
		}
		prs, err = inheritTaskSteps(resolver, prs)
		if err != nil {
			return prs, errors.Wrapf(err, "failed to inherit steps")
		}
		defaultTaskName(prs)
		return DefaultPipelineParameters(prs)

	default:
		return nil, errors.Errorf("kind %s is not supported for %s", kind, message)
	}
}

func defaultTaskName(prs *tektonv1beta1.PipelineRun) {
	ps := prs.Spec.PipelineSpec
	if ps != nil && len(ps.Tasks) > 0 {
		t := ps.Tasks[0]
		if t.Name == "" {
			ps.Tasks[0].Name = "default"
		}
	}
}

// inheritTaskSteps allows Task steps to be prepended or appended if the annotations are present
func inheritTaskSteps(resolver *UsesResolver, prs *tektonv1beta1.PipelineRun) (*tektonv1beta1.PipelineRun, error) {
	err := processUsesSteps(resolver, prs)
	if err != nil {
		return prs, errors.Wrap(err, "failed to process uses steps")
	}
	ps := prs.Spec.PipelineSpec
	if ps == nil || len(ps.Tasks) == 0 {
		return prs, nil
	}
	if prs.Annotations == nil {
		return prs, nil
	}
	appendURL := prs.Annotations[AppendStepURL]
	prependURL := prs.Annotations[PrependStepURL]

	var appendTask *tektonv1beta1.Task
	var prependTask *tektonv1beta1.Task

	if appendURL != "" {
		appendTask, err = loadTaskByURL(appendURL)
		if err != nil {
			return prs, errors.Wrapf(err, "failed to load append steps Task")
		}
	}
	if prependURL != "" {
		prependTask, err = loadTaskByURL(prependURL)
		if err != nil {
			return prs, errors.Wrapf(err, "failed to load prepend steps Task")
		}
	}
	if prependTask != nil {
		firstTask := &ps.Tasks[0]
		if firstTask.TaskSpec != nil {
			firstTask.TaskSpec.Steps = append(prependTask.Spec.Steps, firstTask.TaskSpec.Steps...)
		}
	}
	if appendTask != nil {
		lastTask := &ps.Tasks[len(ps.Tasks)-1]
		lastTask.TaskSpec.Steps = append(lastTask.TaskSpec.Steps, appendTask.Spec.Steps...)
	}
	return prs, nil
}

// processUsesSteps handles any step which has an image prefixed with "uses:"
func processUsesSteps(resolver *UsesResolver, prs *tektonv1beta1.PipelineRun) error {
	ps := prs.Spec.PipelineSpec
	if ps == nil {
		return nil
	}
	for i := range ps.Tasks {
		pt := &ps.Tasks[i]
		if pt.TaskSpec != nil {
			ts := &pt.TaskSpec.TaskSpec
			clearStepTemplateImage := false
			var steps []tektonv1beta1.Step
			for j := range ts.Steps {
				step := ts.Steps[j]
				image := step.Image
				if image == "" && ts.StepTemplate != nil {
					// lets default to the step image so we can share uses across steps
					image = ts.StepTemplate.Image
					if strings.HasPrefix(image, "uses:") {
						clearStepTemplateImage = true
					}
				}
				if !strings.HasPrefix(image, "uses:") {
					steps = append(steps, step)
					continue
				}
				sourceURI := strings.TrimPrefix(image, "uses:")

				loc := &UseLocation{
					PipelineRunSpec: &prs.Spec,
					PipelineSpec:    prs.Spec.PipelineSpec,
					PipelineTask:    pt,
					TaskName:        pt.Name,
					TaskSpec:        ts,
				}
				replaceSteps, err := resolver.UsesSteps(sourceURI, pt.Name, step, ts, loc)
				if err != nil {
					return errors.Wrapf(err, "failed to resolve git URI %s for step %s", sourceURI, step.Name)
				}
				steps = append(steps, replaceSteps...)
			}
			ts.Steps = steps
			if clearStepTemplateImage && ts.StepTemplate != nil {
				ts.StepTemplate.Image = ""
			}
		}
	}
	return nil
}

func loadTaskByURL(uri string) (*tektonv1beta1.Task, error) {
	resp, err := http.Get(uri) // #nosec
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read URL %s", uri)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read body from URL %s", uri)
	}

	task := &tektonv1beta1.Task{}
	err = yaml.Unmarshal(data, &task)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshall YAML from URL %s", uri)
	}
	return task, nil
}

// loadTektonRefsFromFilesPattern returns a regular expression matching the Pipeline/Task references we should load
// via the file system as separate local files
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

func loadPipelineRunRefs(resolver *UsesResolver, prs *tektonv1beta1.PipelineRun, dir, message string, re *regexp.Regexp) (*tektonv1beta1.PipelineRun, error) {
	// if we reference a local
	if prs.Spec.PipelineSpec == nil && prs.Spec.PipelineRef != nil && prs.Spec.PipelineRef.Name != "" && re.MatchString(prs.Spec.PipelineRef.Name) {
		pipelinePath := filepath.Join(dir, prs.Spec.PipelineRef.Name)
		if !strings.HasSuffix(pipelinePath, ".yaml") {
			pipelinePath += ".yaml"
		}
		data, err := resolver.GetData(pipelinePath, true)
		if err != nil {
			return prs, errors.Wrapf(err, "failed to find path %s in PipelineRun", pipelinePath)
		}
		if len(data) == 0 {
			return prs, errors.Errorf("no YAML for path %s in PipelineRun", pipelinePath)
		}
		p := &tektonv1beta1.Pipeline{}
		err = yaml.Unmarshal(data, p)
		if err != nil {
			return prs, errors.Wrapf(err, "failed to unmarshal Pipeline YAML file %s %s", pipelinePath, message)
		}
		prs.Spec.PipelineSpec = &p.Spec
		prs.Spec.PipelineRef = nil
	}

	if prs.Spec.PipelineSpec != nil {
		err := loadTaskRefs(resolver, prs.Spec.PipelineSpec, dir, message, re)
		if err != nil {
			return prs, errors.Wrapf(err, "failed to load Task refs for %s", message)
		}
	}
	return prs, nil
}

func loadTaskRefs(resolver *UsesResolver, pipelineSpec *tektonv1beta1.PipelineSpec, dir, message string, re *regexp.Regexp) error {
	for i := range pipelineSpec.Tasks {
		t := &pipelineSpec.Tasks[i]
		if t.TaskSpec == nil && t.TaskRef != nil && t.TaskRef.Name != "" && re.MatchString(t.TaskRef.Name) {
			path := filepath.Join(dir, t.TaskRef.Name)
			if !strings.HasSuffix(path, ".yaml") {
				path += ".yaml"
			}
			data, err := resolver.GetData(path, false)
			if err != nil {
				return errors.Wrapf(err, "failed to find path %s in PipelineSpec", path)
			}
			if len(data) == 0 {
				return errors.Errorf("no YAML for path %s in PipelineSpec", path)
			}
			t2 := &tektonv1beta1.Task{}
			err = yaml.Unmarshal(data, t2)
			if err != nil {
				return errors.Wrapf(err, "failed to unmarshal Task YAML file %s %s", path, message)
			}
			t.TaskSpec = &tektonv1beta1.EmbeddedTask{
				TaskSpec: t2.Spec,
			}
			t.TaskRef = nil
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
				TaskSpec:   &tektonv1beta1.EmbeddedTask{TaskSpec: *fs},
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
	//prs.Spec.Resources = fs.Resources
	prs.Spec.Workspaces = ToWorkspaceBindings(fs.Workspaces)
	defaultValues.Apply(prs)

	// lets copy the params from the task -> pipeline -> pipelinerun
	loc := &UseLocation{
		PipelineRunSpec: &prs.Spec,
		PipelineSpec:    pipelineSpec,
		TaskName:        from.Name,
		TaskRunSpec:     nil,
		TaskSpec:        fs,
	}
	UseParametersAndResults(context.TODO(), loc, fs)
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
				TaskSpec: &tektonv1beta1.EmbeddedTask{TaskSpec: *fs.TaskSpec},
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

// ToParams converts the param specs to params
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
