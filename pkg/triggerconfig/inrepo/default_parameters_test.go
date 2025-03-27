package inrepo

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func TestDefaultWorkspacesEmptyDir(t *testing.T) {

	prs := &pipelinev1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cheese",
		},
		Spec: pipelinev1.PipelineRunSpec{
			PipelineSpec: &pipelinev1.PipelineSpec{},
			Workspaces: []pipelinev1.WorkspaceBinding{
				{Name: "foo"},
				{Name: "bar"},
			},
		},
	}

	rs, err := DefaultPipelineParameters(prs)
	assert.NoError(t, err)

	assert.True(t, rs.Spec.Workspaces[0].EmptyDir != nil)
	assert.True(t, rs.Spec.Workspaces[1].EmptyDir != nil)
}

func TestDefaultFinallyParameters(t *testing.T) {

	prs := &pipelinev1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cheese",
		},
		Spec: pipelinev1.PipelineRunSpec{
			PipelineSpec: &pipelinev1.PipelineSpec{
				Tasks: []pipelinev1.PipelineTask{{
					Name: "maintask",
					TaskSpec: &pipelinev1.EmbeddedTask{TaskSpec: pipelinev1.TaskSpec{
						Steps: []pipelinev1.Step{{
							Name:  "mystep",
							Image: "myimage",
						}},
					}},
				}},

				Finally: []pipelinev1.PipelineTask{{
					Name: "i-should-have-parameters",
					TaskSpec: &pipelinev1.EmbeddedTask{TaskSpec: pipelinev1.TaskSpec{
						Steps: []pipelinev1.Step{{
							Name:  "finallystep",
							Image: "finallyimage",
						}},
					}},
				}},
			},
		},
	}

	rs, err := DefaultPipelineParameters(prs)
	assert.NoError(t, err)

	assert.True(t, rs.Spec.PipelineSpec.Finally[0].Params != nil)
}
