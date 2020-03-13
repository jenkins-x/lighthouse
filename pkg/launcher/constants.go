package launcher

const (
	// TektonAgent the default agent name
	TektonAgent = "tekton"
	// LighthouseJobTypeLabel is added in resources created by lighthouse and
	// carries the job type (presubmit, postsubmit, periodic, batch)
	// that the pod is running.
	LighthouseJobTypeLabel = "lighthouse.jenkins-x.io/type"
	// LighthouseJobIDLabel is added in resources created by lighthouse and
	// carries the ID of the PipelineOptions that the pod is fulfilling.
	// We also name resources after the PipelineOptions that spawned them but
	// this allows for multiple resources to be linked to one
	// PipelineOptions.
	LighthouseJobIDLabel = "lighthouse.jenkins-x.io/id"
	// LighthouseJobAnnotation is added in resources created by lighthouse and
	// carries the name of the job that the pod is running. Since
	// job names can be arbitrarily long, this is added as
	// an annotation instead of a label.
	LighthouseJobAnnotation = "lighthouse.jenkins-x.io/job"
	// CreatedByLighthouse is added on resources created by Lighthosue.
	// Since resources often live in another cluster/namespace,
	// the k8s garbage collector would immediately delete these
	// resources
	CreatedByLighthouse = "created-by-lighthouse"
	// OrgLabel is added in resources created by Lighthouse and
	// carries the org associated with the job, eg kubernetes-sigs.
	OrgLabel = "lighthouse.jenkins-x.io/refs.org"
	// RepoLabel is added in resources created by Lighthouse and
	// carries the repo associated with the job, eg test-infra
	RepoLabel = "lighthouse.jenkins-x.io/refs.repo"
	// PullLabel is added in resources created by Lighthouse and
	// carries the PR number associated with the job, eg 321.
	PullLabel = "lighthouse.jenkins-x.io/refs.pull"
	// DefaultClusterAlias specifies the default context for resources owned by jobs (pods/builds).
	DefaultClusterAlias = "default"
)
