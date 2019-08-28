package fake

import (
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
)

// Plumber a fake Plumber
type Plumber struct {
	Pipelines []*plumber.PipelineOptions
}

// NewPlumber creates a fake plumber
func NewPlumber() *Plumber {
	return &Plumber{
		Pipelines: []*plumber.PipelineOptions{},
	}
}

// Create creates a plumber job
func (p *Plumber) Create(po *plumber.PipelineOptions, metapipelineClient metapipeline.Client) (*plumber.PipelineOptions, error) {
	p.Pipelines = append(p.Pipelines, po)
	return po, nil
}

// PrependReactor prepends a reactor
func (p *Plumber) PrependReactor(s string, s2 string, i func(plumberJob *plumber.PipelineOptions) (handled bool, ret *plumber.PipelineOptions, err error)) {
}
