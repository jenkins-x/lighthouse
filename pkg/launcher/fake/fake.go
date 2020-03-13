package fake

import (
	"errors"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/launcher"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Launcher a fake PipelineLauncher
type Launcher struct {
	Pipelines []*v1alpha1.LighthouseJob
	FailJobs  sets.String
}

// implements interface
var _ launcher.PipelineLauncher = &Launcher{}

// NewLauncher creates a fake launcher
func NewLauncher() *Launcher {
	return &Launcher{
		Pipelines: []*v1alpha1.LighthouseJob{},
	}
}

// Launch creates a launcher job
func (p *Launcher) Launch(po *v1alpha1.LighthouseJob, metapipelineClient metapipeline.Client, repo scm.Repository) (*v1alpha1.LighthouseJob, error) {
	if p.FailJobs.Has(po.Spec.Job) {
		return po, errors.New("failed to create job")
	}
	p.Pipelines = append(p.Pipelines, po)
	po.Status.State = v1alpha1.SuccessState
	return po, nil
}

// PrependReactor prepends a reactor
func (p *Launcher) PrependReactor(s string, s2 string, i func(job *v1alpha1.LighthouseJob) (handled bool, ret *v1alpha1.LighthouseJob, err error)) {
}

// List lists the pipelines and ttheir options
func (p *Launcher) List(opts metav1.ListOptions) (*v1alpha1.LighthouseJobList, error) {
	list := v1alpha1.LighthouseJobList{}
	for _, p := range p.Pipelines {
		list.Items = append(list.Items, *p)
	}
	return &list, nil
}
