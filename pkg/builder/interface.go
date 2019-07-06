package builder

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
)

// Builder the interface for the pipeline builder
type Builder interface {

	// StartBuild starts a pipeline if one is configured for the given push event
	StartBuild(hook *scm.PushHook, commonOptions *opts.CommonOptions) (string, error)
}
