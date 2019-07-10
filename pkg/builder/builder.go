package builder

import (
	"github.com/jenkins-x/go-scm/scm"
	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/sirupsen/logrus"
)

// PipelineBuilder default builder
type PipelineBuilder struct {
}

// NewBuilder creates a new builder
func NewBuilder(jxClient jxclient.Interface, ns string) (Builder, error) {
	b := &PipelineBuilder{}
	return b, nil
}

// StartBuild starts a build if there is a SourceRepository and Scheduler available
func (b *PipelineBuilder) StartBuild(hook *scm.PushHook, commonOptions *opts.CommonOptions) (string, error) {
	repository := hook.Repository()
	name := repository.Name
	owner := repository.Namespace
	sourceURL := repository.Clone
	branch := repository.Branch

	// TODO
	pipelineKind := "release"
	pullRefs := ""
	prNumber := ""

	// TODO is this correct?
	job := hook.Ref

	l := logrus.WithFields(logrus.Fields(map[string]interface{}{
		"Owner":             owner,
		"Name":              name,
		"SourceURL":         sourceURL,
		"Branch":            branch,
		"PipelineKind":      pipelineKind,
		"PullRefs":          pullRefs,
		"PullRequestNumber": prNumber,
		"Job":               job,
	}))
	l.Info("about to start Jenkinx X meta pipeline")

	po := create.StepCreatePipelineOptions{
		SourceURL: sourceURL,
		Job:       job,
		PullRefs:  pullRefs,
	}
	po.CommonOptions = commonOptions

	err := po.Run()
	if err != nil {
		l.Errorf("failed to create Jenkinx X meta pipeline %s", err.Error())
		return "failed to create Jenkins X Pipeline %s", err
	}
	return "OK", nil
}
