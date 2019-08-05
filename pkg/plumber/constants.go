package plumber

const (
	// TektonAgent the default agent name
	TektonAgent = "tekton"
	// PlumberJobTypeLabel is added in resources created by lighthouse and
	// carries the job type (presubmit, postsubmit, periodic, batch)
	// that the pod is running.
	PlumberJobTypeLabel = "lighthouse.jenkins-x.io/type"
	// PlumberJobIDLabel is added in resources created by lighthouse and
	// carries the ID of the PipelineOptions that the pod is fulfilling.
	// We also name resources after the PipelineOptions that spawned them but
	// this allows for multiple resources to be linked to one
	// PipelineOptions.
	PlumberJobIDLabel = "lighthouse.jenkins-x.io/id"
	// PlumberJobAnnotation is added in resources created by lighthouse and
	// carries the name of the job that the pod is running. Since
	// job names can be arbitrarily long, this is added as
	// an annotation instead of a label.
	PlumberJobAnnotation = "lighthouse.jenkins-x.io/job"
)
