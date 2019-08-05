package fake

import (
	"github.com/jenkins-x/lighthouse/pkg/plumber"
)

// FakePlumber a fake Plumber
type FakePlumber struct {
	Jobs []*plumber.PlumberArguments
}

// NewPlumber creates a fake plumber
func NewPlumber() *FakePlumber {
	return &FakePlumber{
		Jobs: []*plumber.PlumberArguments{},
	}
}

// Create creates a plumber job
func (p *FakePlumber) Create(job *plumber.PlumberArguments) (*plumber.PlumberArguments, error) {
	p.Jobs = append(p.Jobs, job)
	return job, nil
}

func (p *FakePlumber) PrependReactor(s string, s2 string, i func(plumberJob *plumber.PlumberArguments) (handled bool, ret *plumber.PlumberArguments, err error)) {
}
