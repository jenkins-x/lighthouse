package plumber

import (
	"fmt"
	"os"
	"strconv"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineBuilder default builder
type PipelineBuilder struct {
	repository scm.Repository
}

// NewPlumber creates a new builder
func NewPlumber(repository scm.Repository) (Plumber, error) {
	b := &PipelineBuilder{
		repository: repository,
	}
	return b, nil
}

// Create creates a pipeline
func (b *PipelineBuilder) Create(request *PipelineOptions, metapipelineClient metapipeline.Client) (*PipelineOptions, error) {
	spec := &request.Spec

	repository := b.repository
	name := repository.Name
	owner := repository.Namespace
	sourceURL := repository.Clone

	pullRefData := b.getPullRefs(sourceURL, spec)
	pullRefs := ""
	if len(spec.Refs.Pulls) > 0 {
		pullRefs = pullRefData.String()
	}

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
	var kind metapipeline.PipelineKind
	if len(spec.Refs.Pulls) > 0 {
		kind = metapipeline.PullRequestPipeline
	} else {
		kind = metapipeline.ReleasePipeline
	}

	l := logrus.WithFields(logrus.Fields(map[string]interface{}{
		"Owner":     owner,
		"Name":      name,
		"SourceURL": sourceURL,
		"Branch":    branch,
		"PullRefs":  pullRefs,
		"Job":       job,
	}))
	l.Info("about to start Jenkinx X meta pipeline")

	sa := os.Getenv("JX_SERVICE_ACCOUNT")
	if sa == "" {
		sa = "tekton-bot"
	}

	pipelineCreateParam := metapipeline.PipelineCreateParam{
		PullRef:      pullRefData,
		PipelineKind: kind,
		Context:      spec.Context,
		// No equivalent to https://github.com/jenkins-x/jx/blob/bb59278c2707e0e99b3c24be926745c324824388/pkg/cmd/controller/pipeline/pipelinerunner_controller.go#L236
		//   for getting environment variables from the prow job here, so far as I can tell (abayer)
		// Also not finding an equivalent to labels from the PipelineRunRequest
		ServiceAccount: sa,
		// I believe we can use an empty string default image?
		DefaultImage: "",
	}

	pipelineActivity, tektonCRDs, err := metapipelineClient.Create(pipelineCreateParam)
	if err != nil {
		return request, errors.Wrap(err, "unable to create Tekton CRDs")
	}

	err = metapipelineClient.Apply(pipelineActivity, tektonCRDs)
	if err != nil {
		return request, errors.Wrap(err, "unable to apply Tekton CRDs")
	}
	return request, nil
}

func (b *PipelineBuilder) getBranch(spec *PipelineOptionsSpec) string {
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

func (b *PipelineBuilder) getPullRefs(sourceURL string, spec *PipelineOptionsSpec) metapipeline.PullRef {
	var pullRef metapipeline.PullRef
	if len(spec.Refs.Pulls) > 0 {
		var prs []metapipeline.PullRequestRef
		for _, pull := range spec.Refs.Pulls {
			prs = append(prs, metapipeline.PullRequestRef{ID: strconv.Itoa(pull.Number), MergeSHA: pull.SHA})
		}

		pullRef = metapipeline.NewPullRefWithPullRequest(sourceURL, spec.Refs.BaseRef, spec.Refs.BaseSHA, prs...)
	} else {
		pullRef = metapipeline.NewPullRef(sourceURL, spec.Refs.BaseRef, spec.Refs.BaseSHA)
	}

	return pullRef
}

func (b *PipelineBuilder) List(opts metav1.ListOptions) (*PipelineOptionsList, error) {
	panic("implement me")
}
