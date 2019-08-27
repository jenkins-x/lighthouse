package plumber

import 	"github.com/jenkins-x/jx/pkg/tekton/metapipeline"

// Plumber the interface is the service which creates Pipelines
type Plumber interface {
	Create(*PipelineOptions, metapipeline.Client) (*PipelineOptions, error)
}
