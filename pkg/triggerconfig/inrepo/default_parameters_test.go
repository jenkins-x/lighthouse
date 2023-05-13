package inrepo

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestDefaultWorkspacesEmptyDir(t *testing.T) {

	prs := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cheese",
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineSpec: &v1beta1.PipelineSpec{},
			Workspaces: []v1beta1.WorkspaceBinding{
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

	prs := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cheese",
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineSpec: &v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{{
					Name: "maintask",
					TaskSpec: &v1beta1.EmbeddedTask{TaskSpec: v1beta1.TaskSpec{
						Steps: []v1beta1.Step{{
							Name:  "mystep",
							Image: "myimage",
						}},
					}},
				}},

				Finally: []v1beta1.PipelineTask{{
					Name: "i-should-have-parameters",
					TaskSpec: &v1beta1.EmbeddedTask{TaskSpec: v1beta1.TaskSpec{
						Steps: []v1beta1.Step{{
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
