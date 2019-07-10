package plumber

const (
	// CreatedByPlumber is added on resources created by prow.
	// Since resources often live in another cluster/namespace,
	// the k8s garbage collector would immediately delete these
	// resources
	// TODO: Namespace this label.
	CreatedByPlumber = "created-by-prow"
	// PlumberJobTypeLabel is added in resources created by prow and
	// carries the job type (presubmit, postsubmit, periodic, batch)
	// that the pod is running.
	PlumberJobTypeLabel = "lighthouse.jenkins-x.io/type"
	// PlumberJobIDLabel is added in resources created by prow and
	// carries the ID of the PlumberJob that the pod is fulfilling.
	// We also name resources after the PlumberJob that spawned them but
	// this allows for multiple resources to be linked to one
	// PlumberJob.
	PlumberJobIDLabel = "lighthouse.jenkins-x.io/id"
	// PlumberJobAnnotation is added in resources created by prow and
	// carries the name of the job that the pod is running. Since
	// job names can be arbitrarily long, this is added as
	// an annotation instead of a label.
	PlumberJobAnnotation = "lighthouse.jenkins-x.io/job"
	// OrgLabel is added in resources created by prow and
	// carries the org associated with the job, eg kubernetes-sigs.
	OrgLabel = "lighthouse.jenkins-x.io/refs.org"
	// RepoLabel is added in resources created by prow and
	// carries the repo associated with the job, eg test-infra
	RepoLabel = "lighthouse.jenkins-x.io/refs.repo"
	// PullLabel is added in resources created by prow and
	// carries the PR number associated with the job, eg 321.
	PullLabel = "lighthouse.jenkins-x.io/refs.pull"
)
