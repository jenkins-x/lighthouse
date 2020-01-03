package fake

import (
	"errors"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Plumber a fake Plumber
type Plumber struct {
	Pipelines []*plumber.PipelineOptions
	FailJobs  sets.String
}

// implements interface
var _ plumber.Plumber = &Plumber{}

// NewPlumber creates a fake plumber
func NewPlumber() *Plumber {
	return &Plumber{
		Pipelines: []*plumber.PipelineOptions{},
	}
}

// Create creates a plumber job
func (p *Plumber) Create(po *plumber.PipelineOptions, metapipelineClient metapipeline.Client, repo scm.Repository) (*plumber.PipelineOptions, error) {
	if p.FailJobs.Has(po.Spec.Job) {
		return po, errors.New("failed to create job")
	}
	p.Pipelines = append(p.Pipelines, po)
	po.Status.State = plumber.SuccessState
	return po, nil
}

// PrependReactor prepends a reactor
func (p *Plumber) PrependReactor(s string, s2 string, i func(plumberJob *plumber.PipelineOptions) (handled bool, ret *plumber.PipelineOptions, err error)) {
}

// List lists the pipelines and ttheir options
func (p *Plumber) List(opts metav1.ListOptions) (*plumber.PipelineOptionsList, error) {
	list := plumber.PipelineOptionsList{}
	for _, p := range p.Pipelines {
		list.Items = append(list.Items, *p)
	}
	return &list, nil
}
