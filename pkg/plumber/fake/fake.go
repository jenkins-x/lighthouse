package fake

import (
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FakePlumber a fake FakePlumber
type FakePlumber struct {
	Pipelines []*plumber.PipelineOptions
}

// implements interface
var _ plumber.Plumber = &FakePlumber{}

// NewPlumber creates a fake plumber
func NewPlumber() *FakePlumber {
	return &FakePlumber{
		Pipelines: []*plumber.PipelineOptions{},
	}
}

// Create creates a plumber job
func (p *FakePlumber) Create(po *plumber.PipelineOptions, metapipelineClient metapipeline.Client) (*plumber.PipelineOptions, error) {
	p.Pipelines = append(p.Pipelines, po)
	po.Status.State = plumber.SuccessState
	return po, nil
}

// PrependReactor prepends a reactor
func (p *FakePlumber) PrependReactor(s string, s2 string, i func(plumberJob *plumber.PipelineOptions) (handled bool, ret *plumber.PipelineOptions, err error)) {
}

// List lists the pipelines and ttheir options
func (p *FakePlumber) List(opts metav1.ListOptions) (*plumber.PipelineOptionsList, error) {
	list := plumber.PipelineOptionsList{}
	for _, p := range p.Pipelines {
		list.Items = append(list.Items, *p)
	}
	return &list, nil
}
