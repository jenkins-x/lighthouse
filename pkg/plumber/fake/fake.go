package fake

import (
	"github.com/jenkins-x/lighthouse/pkg/plumber"
)

// FakePlumber a fake Plumber
type FakePlumber struct {
	Jobs []*plumber.PipelineOptions
}

// NewPlumber creates a fake plumber
func NewPlumber() *FakePlumber {
	return &FakePlumber{
		Jobs: []*plumber.PipelineOptions{},
	}
}

// Create creates a plumber job
func (p *FakePlumber) Create(job *plumber.PipelineOptions) (*plumber.PipelineOptions, error) {
	p.Jobs = append(p.Jobs, job)
	return job, nil
}

func (p *FakePlumber) PrependReactor(s string, s2 string, i func(plumberJob *plumber.PipelineOptions) (handled bool, ret *plumber.PipelineOptions, err error)) {
}
