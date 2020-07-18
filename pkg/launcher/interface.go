package launcher

import (
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
)

// PipelineLauncher the interface is the service which creates Pipelines
type PipelineLauncher interface {
	// Launch creates new pipelines
	Launch(*v1alpha1.LighthouseJob) (*v1alpha1.LighthouseJob, error)
}
