package launcher

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/v2/pkg/tekton/metapipeline"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
)

// PipelineLauncher the interface is the service which creates Pipelines
type PipelineLauncher interface {
	// Launch creates new tekton pipelines
	Launch(*v1alpha1.LighthouseJob, metapipeline.Client, scm.Repository) (*v1alpha1.LighthouseJob, error)
}
