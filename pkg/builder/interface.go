package builder

import (
	"github.com/jenkins-x/go-scm/scm"
	jxv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/plugins"
)

// Builder the interface for the pipeline builder
type Builder interface {
	// StartBuild starts a pipeline if one is configured for the given push event
	StartBuild(hook *scm.PushHook, sr *jxv1.SourceRepository, commonOptions *opts.CommonOptions) (string, error)

	// FindSourceRepository finds the SourceRepository CRD for the given hook
	FindSourceRepository(hook scm.Webhook) *jxv1.SourceRepository

	// CreateChatOpsConfig creates the prow configuration
	CreateChatOpsConfig(hook scm.Webhook, sr *jxv1.SourceRepository) (*config.Config, *plugins.Configuration, error)
}
