package fake

import (
	"github.com/jenkins-x/lighthouse/pkg/plumber"
)

// FakePlumber a fake Plumber
type FakePlumber struct {
	Pipelines []*plumber.PipelineOptions
}

// NewPlumber creates a fake plumber
func NewPlumber() *FakePlumber {
	return &FakePlumber{
		Pipelines: []*plumber.PipelineOptions{},
	}
}

// Create creates a plumber job
func (p *FakePlumber) Create(po *plumber.PipelineOptions) (*plumber.PipelineOptions, error) {
	p.Pipelines = append(p.Pipelines, po)
	return po, nil
}

func (p *FakePlumber) PrependReactor(s string, s2 string, i func(plumberJob *plumber.PipelineOptions) (handled bool, ret *plumber.PipelineOptions, err error)) {
}
