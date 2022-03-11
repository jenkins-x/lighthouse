package launcher

import (
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
)

// ScmInfo represents a repository in SCM
type ScmInfo interface {
	GetFullRepositoryName() string
}

// PipelineLauncher the interface is the service which creates Pipelines
type PipelineLauncher interface {
	// Launch creates new pipelines
	Launch(*v1alpha1.LighthouseJob, ScmInfo) (*v1alpha1.LighthouseJob, error)
}
