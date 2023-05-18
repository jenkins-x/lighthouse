package inrepo

import (
	"fmt"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
	"time"

	"github.com/pkg/errors"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func LoadTektonResourceAsPipelineRun(data []byte, ownerName string, repoName string, sha string) (*tektonv1beta1.PipelineRun, error) {
	message := fmt.Sprintf("in repo %s/%s with sha %s", ownerName, repoName, sha)
	prs := &tektonv1beta1.PipelineRun{}
	// We're doing to replace any instances of versionStream with the versionStream version. Otherwise, we'd have to parse the whole pipeline spec and iterate over it.
	// This is a jenkins-x specific functionality that it doesn't make sense for Tekton to support
	pipelineString := strings.ReplaceAll(string(data), "versionStream", VersionStreamEnvVar(ownerName, repoName))
	err := yaml.Unmarshal([]byte(pipelineString), prs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal PipelineRun YAML %s", message)
	}
	// Now we apply the default values
	prs, err = DefaultPipelineParameters(prs)
	if err != nil {
		return nil, err
	}

	return prs, err
}

// NewDefaultValues creates a new default values
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
