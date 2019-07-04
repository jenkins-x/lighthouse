package builder

import (
	"fmt"

	"github.com/drone/go-scm/scm"
	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/jenkins-x/lighthouse/pkg/caches"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// PipelineBuilder default builder
type PipelineBuilder struct {
	repoCache *caches.SourceRepositoryCache
}

// NewBuilder creates a new builder
func NewBuilder(jxClient jxclient.Interface, ns string) (Builder, error) {
	repoCache, err := caches.NewSourceRepositoryCache(jxClient, ns)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create SourceRepositoryCache")
	}

	return &PipelineBuilder{
		repoCache: repoCache,
	}, nil
}

// StartBuild starts a build if there is a SourceRepository and Scheduler available
func (b *PipelineBuilder) StartBuild(hook *scm.PushHook, commonOptions *opts.CommonOptions) (string, error) {
	repository := hook.Repository()
	name := repository.Name
	owner := repository.Namespace
	sourceURL := repository.Clone
	branch := repository.Branch
	sr := b.repoCache.FindRepository(owner, name)

	// TODO
	pipelineKind := "release"
	pullRefs := ""
	prNumber := ""

	// TODO is this correct?
	job := hook.Ref

	if sr == nil {
		logrus.Warnf("could not find SourceRepository for owner %s name %s", owner, name)
		return fmt.Sprintf("no Pipeline setup for repository %s/%s", owner, name), nil
	}

	fields := map[string]interface{}{
		"SourceURL":         sourceURL,
		"Branch":            branch,
		"PipelineKind":      pipelineKind,
		"PullRefs":          pullRefs,
		"PullRequestNumber": prNumber,
		"Job":               job,
	}
	logrus.WithFields(logrus.Fields(fields)).Info("about to start Jenkinx X meta pipeline")

	po := create.StepCreatePipelineOptions{
		SourceURL:         sourceURL,
		Branch:            branch,
		Job:               job,
		PipelineKind:      pipelineKind,
		PullRefs:          pullRefs,
		PullRequestNumber: prNumber,
	}
	po.CommonOptions = commonOptions

	err := po.Run()
	if err != nil {
		logrus.WithFields(logrus.Fields(fields)).Errorf("failed to create Jenkinx X meta pipeline %s", err.Error())
		return "failed to create Jenkins X Pipeline %s", err
	}
	return "OK", nil
}
