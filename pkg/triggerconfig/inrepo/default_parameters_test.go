package inrepo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func TestDefaultWorkspacesEmptyDir(t *testing.T) {

	prs := &v1beta1.PipelineRun{
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
