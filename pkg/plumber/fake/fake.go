package fake

import (
	"github.com/jenkins-x/lighthouse/pkg/plumber"
)

// FakePlumber a fake Plumber
type FakePlumber struct {
	Jobs []*plumber.PlumberJob
}

// NewPlumber creates a fake plumber
func NewPlumber() *FakePlumber {
	return &FakePlumber{
		Jobs: []*plumber.PlumberJob{},
	}
}

// Create creates a plumber job
func (p *FakePlumber) Create(job *plumber.PlumberJob) (*plumber.PlumberJob, error) {
	p.Jobs = append(p.Jobs, job)
	return job, nil
}

func (p *FakePlumber) PrependReactor(s string, s2 string, i func(plumberJob *plumber.PlumberJob) (handled bool, ret *plumber.PlumberJob, err error)) {
	panic("TODO")
}
