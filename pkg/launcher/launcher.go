package launcher

import (
	"context"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	clientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// launcherImpl default launcher
type launcherImpl struct {
	lhClient  clientset.Interface
	namespace string
}

// NewLauncher creates a new builder
func NewLauncher(lhClient clientset.Interface, ns string) PipelineLauncher {
	b := &launcherImpl{
		lhClient:  lhClient,
		namespace: ns,
	}
	return b
}

// Launch creates a pipeline
// TODO: This should be moved somewhere else, probably, and needs some kind of unit testing (apb)
func (b *launcherImpl) Launch(request *v1alpha1.LighthouseJob) (*v1alpha1.LighthouseJob, error) {
	appliedJob, err := b.lhClient.LighthouseV1alpha1().LighthouseJobs(b.namespace).Create(context.TODO(), request, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to apply LighthouseJob")
	}
	// Set status on the job
	appliedJob.Status = v1alpha1.LighthouseJobStatus{
		State: v1alpha1.TriggeredState,
	}
	fullyCreatedJob, err := b.lhClient.LighthouseV1alpha1().LighthouseJobs(b.namespace).UpdateStatus(context.TODO(), appliedJob, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to set status on LighthouseJob %s", appliedJob.Name)
	}

	return fullyCreatedJob, nil
}
