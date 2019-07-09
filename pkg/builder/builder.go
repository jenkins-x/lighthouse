package builder

import (
	"fmt"

	"github.com/jenkins-x/go-scm/scm"
	jxv1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/pipelinescheduler"
	"github.com/jenkins-x/lighthouse/pkg/caches"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/plugins"
)

// PipelineBuilder default builder
type PipelineBuilder struct {
	jxClient       jxclient.Interface
	ns             string
	envCache       *caches.EnvironmentCache
	repoCache      *caches.SourceRepositoryCache
	schedulerCache *caches.SchedulerCache
}

// NewBuilder creates a new builder
func NewBuilder(jxClient jxclient.Interface, ns string) (Builder, error) {
	b := &PipelineBuilder{
		jxClient: jxClient,
		ns:       ns,
	}
	var err error

	b.repoCache, err = caches.NewSourceRepositoryCache(jxClient, ns)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create SourceRepositoryCache")
	}
	b.envCache, err = caches.NewEnvironmentCache(jxClient, ns)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create EnvironmentCache")
	}
	b.schedulerCache, err = caches.NewSchedulerCache(jxClient, ns)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create SchedulerCache")
	}

	logrus.Info("waiting for the caches to startup")
	caches.WaitForCachesToLoad(b.envCache, b.repoCache, b.schedulerCache)
	logrus.Info("caches loaded")

	return b, nil
}

// FindSourceRepository finds the source repository for the hook
func (b *PipelineBuilder) FindSourceRepository(hook scm.Webhook) *jxv1.SourceRepository {
	repository := hook.Repository()
	return b.repoCache.FindRepository(repository.Namespace, repository.Name)
}

// StartBuild starts a build if there is a SourceRepository and Scheduler available
func (b *PipelineBuilder) StartBuild(hook *scm.PushHook, sr *jxv1.SourceRepository, commonOptions *opts.CommonOptions) (string, error) {
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

	if sr == nil {
		logrus.Warnf("could not find SourceRepository for owner %s name %s", owner, name)
		return fmt.Sprintf("no Pipeline setup for repository %s/%s", owner, name), nil
	}

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
		l.Errorf("failed to create Jenkinx X meta pipeline %s", err.Error())
		return "failed to create Jenkins X Pipeline %s", err
	}
	return "OK", nil
}

// CreateChatOpsConfig creates the prow configuration
func (b *PipelineBuilder) CreateChatOpsConfig(hook scm.Webhook, sr *jxv1.SourceRepository) (*config.Config, *plugins.Configuration, error) {
	devEnv := b.envCache.Get(kube.LabelValueDevEnvironment)
	if devEnv == nil {
		return nil, nil, fmt.Errorf("no Environment called %s in namespace %s", kube.LabelValueDevEnvironment, b.ns)
	}
	teamSchedulerName := devEnv.Spec.TeamSettings.DefaultScheduler.Name

	c, p, err := pipelinescheduler.GenerateProw(false, false, b.jxClient, b.ns, teamSchedulerName, devEnv, nil)
	if err != nil {
		return c, p, errors.Wrap(err, "failed to generate the Prow configuration")
	}
	return c, p, nil
}
