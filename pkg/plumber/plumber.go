package plumber

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	jxclient "github.com/jenkins-x/jx/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineBuilder default builder
type PipelineBuilder struct {
	jxClient  jxclient.Interface
	namespace string
}

// NewPlumber creates a new builder
func NewPlumber(jxClient jxclient.Interface, namespace string) (Plumber, error) {
	b := &PipelineBuilder{jxClient, namespace}
	return b, nil
}

// Create creates a pipeline
func (b *PipelineBuilder) Create(request *PipelineOptions, metapipelineClient metapipeline.Client, repository scm.Repository) (*PipelineOptions, error) {
	spec := &request.Spec

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
		EnvVariables: spec.GetEnvVars(),
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

// List list current pipelines
func (b *PipelineBuilder) List(opts metav1.ListOptions) (*PipelineOptionsList, error) {
	list, err := b.jxClient.JenkinsV1().PipelineActivities(b.namespace).List(metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	answer := &PipelineOptionsList{}
	for _, pa := range list.Items {
		item := ToPipelineOptions(&pa)
		answer.Items = append(answer.Items, item)
	}
	return answer, nil
}

// ToPipelineOptions converts the PipelineActivity to a PipelineOptions object
func ToPipelineOptions(activity *v1.PipelineActivity) PipelineOptions {
	spec := activity.Spec
	baseRef := "master"

	ref := &Refs{
		Org:      spec.GitOwner,
		Repo:     spec.GitRepository,
		RepoLink: spec.GitURL,
		BaseRef:  baseRef,
		BaseSHA:  spec.BaseSHA,
	}

	kind := PresubmitJob

	// TODO: Something for periodic.
	if spec.GitBranch == "master" {
		kind = PostsubmitJob
	} else if len(spec.BatchPipelineActivity.ComprisingPulLRequests) > 0 {
		kind = BatchJob
	}

	if strings.HasPrefix(spec.GitBranch, "PR-") {
		nt := strings.TrimPrefix(spec.GitBranch, "PR-")
		if nt != "" {
			n, err := strconv.Atoi(nt)
			if err == nil {
				ref.Pulls = append(ref.Pulls, Pull{
					Number: n,
					SHA:    spec.LastCommitSHA,
					Title:  spec.PullTitle,
					Ref:    "pull/" + nt + "/head",

					// TODO
					// Link: spec.LastCommitURL,
					CommitLink: spec.LastCommitURL,
				})
			}
		}
	}

	return PipelineOptions{
		ObjectMeta: activity.ObjectMeta,
		Spec: PipelineOptionsSpec{
			Type:           kind,
			Namespace:      activity.Namespace,
			Job:            spec.Pipeline,
			Refs:           ref,
			Context:        spec.Context,
			RerunCommand:   "",
			MaxConcurrency: 0,
		},
		Status: PipelineStatus{State: ToPipelineState(spec.Status)},
	}
}

// ToPipelineState converts the PipelineActivity state to plumber state
func ToPipelineState(status v1.ActivityStatusType) PipelineState {
	switch status {
	case v1.ActivityStatusTypePending:
		return PendingState
	case v1.ActivityStatusTypeAborted:
		return AbortedState
	case v1.ActivityStatusTypeRunning:
		return RunningState
	case v1.ActivityStatusTypeSucceeded:
		return SuccessState
	case v1.ActivityStatusTypeFailed, v1.ActivityStatusTypeError:
		return FailureState
	default:
		return FailureState
	}
}
