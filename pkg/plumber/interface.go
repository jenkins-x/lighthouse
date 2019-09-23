package plumber

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Plumber the interface is the service which creates Pipelines
type Plumber interface {
	// Create creates new tekton pipelines
	Create(*PipelineOptions, metapipeline.Client, scm.Repository) (*PipelineOptions, error)

	// lists the status of previously created tekton pipelines
	List(opts metav1.ListOptions) (*PipelineOptionsList, error)
}
