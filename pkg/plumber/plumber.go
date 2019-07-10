package plumber

import (
	"fmt"
	"strconv"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/cmd/opts"
	"github.com/jenkins-x/jx/pkg/cmd/step/create"
	"github.com/jenkins-x/jx/pkg/prow"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// PipelineBuilder default builder
type PipelineBuilder struct {
	repository    scm.Repository
	commonOptions *opts.CommonOptions
}

// NewPlumber creates a new builder
func NewPlumber(repository scm.Repository, commonOptions *opts.CommonOptions) (Plumber, error) {
	b := &PipelineBuilder{
		repository:    repository,
		commonOptions: commonOptions,
	}
	return b, nil
}

// Create creates a pipeline
func (b *PipelineBuilder) Create(request *PlumberJob) (*PlumberJob, error) {
	spec := &request.Spec

	pipelineKind := "release"
	revision := ""
	prNumber := ""
	pullRefData := b.getPullRefs(spec)
	pullRefs := ""
	if len(spec.Refs.Pulls) > 0 {
		pullRefs = pullRefData.String()
	}

	// Only if there is one Pull in Refs, it's a PR build so we are going to pass it
	if len(spec.Refs.Pulls) == 1 {
		revision = spec.Refs.Pulls[0].SHA
		prNumber = strconv.Itoa(spec.Refs.Pulls[0].Number)
	} else {
		//Otherwise it's a Master / Batch build, and we handle it later
		revision = spec.Refs.BaseSHA
	}

	repository := b.repository
	name := repository.Name
	owner := repository.Namespace
	sourceURL := repository.Clone

	branch := b.getBranch(spec)
	if branch == "" {
		branch = repository.Branch
	}
	if branch == "" {
		branch = "master"
	}

	if pullRefs == "" {
		pullRefs = branch + ":"
	}

	job := spec.Job

	l := logrus.WithFields(logrus.Fields(map[string]interface{}{
		"Owner":             owner,
		"Name":              name,
		"SourceURL":         sourceURL,
		"Branch":            branch,
		"PipelineKind":      pipelineKind,
		"PullRefs":          pullRefs,
		"PullRequestNumber": prNumber,
		"Revision":          revision,
		"Job":               job,
	}))
	l.Info("about to start Jenkinx X meta pipeline")

	po := create.StepCreatePipelineOptions{
		SourceURL: sourceURL,
		Job:       job,
		PullRefs:  pullRefs,
		Context:   spec.Context,
	}
	po.CommonOptions = b.commonOptions

	err := po.Run()
	if err != nil {
		l.Errorf("failed to create Jenkinx X meta pipeline %s", err.Error())
		return request, errors.Wrap(err, "failed to create Jenkins X Pipeline")
	}
	return request, nil
}

func (o *PipelineBuilder) getBranch(spec *PlumberJobSpec) string {
	branch := spec.Refs.BaseRef
	if spec.Type == PostsubmitJob {
		return branch
	}
	if spec.Type == BatchJob {
		return "batch"
	}
	if len(spec.Refs.Pulls) > 0 {
		branch = fmt.Sprintf("PR-%v", spec.Refs.Pulls[0].Number)
	}
	return branch
}

func (o *PipelineBuilder) getPullRefs(spec *PlumberJobSpec) *prow.PullRefs {
	toMerge := make(map[string]string)
	for _, pull := range spec.Refs.Pulls {
		toMerge[strconv.Itoa(pull.Number)] = pull.SHA
	}

	pullRef := &prow.PullRefs{
		BaseBranch: spec.Refs.BaseRef,
		BaseSha:    spec.Refs.BaseSHA,
		ToMerge:    toMerge,
	}
	return pullRef
}
