package plumber

import (
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PlumberJobType specifies how the job is triggered.
type PlumberJobType string

// Various job types.
const (
	// PresubmitJob means it runs on unmerged PRs.
	PresubmitJob PlumberJobType = "presubmit"
	// PostsubmitJob means it runs on each new commit.
	PostsubmitJob PlumberJobType = "postsubmit"
	// Periodic job means it runs on a time-basis, unrelated to git changes.
	PeriodicJob PlumberJobType = "periodic"
	// BatchJob tests multiple unmerged PRs at the same time.
	BatchJob PlumberJobType = "batch"
)

// PlumberJobState specifies whether the job is running
type PlumberJobState string

// Various job states.
const (
	// TriggeredState means the job has been created but not yet scheduled.
	TriggeredState PlumberJobState = "triggered"
	// PendingState means the job is scheduled but not yet running.
	PendingState PlumberJobState = "pending"
	// SuccessState means the job completed without error (exit 0)
	SuccessState PlumberJobState = "success"
	// FailureState means the job completed with errors (exit non-zero)
	FailureState PlumberJobState = "failure"
	// AbortedState means prow killed the job early (new commit pushed, perhaps).
	AbortedState PlumberJobState = "aborted"
	// ErrorState means the job could not schedule (bad config, perhaps).
	ErrorState PlumberJobState = "error"
)

// PlumberJob is used to request a pipeline to be created
// its the lighthouse equivalent of a ProwJob
// though its not a CRD directly; but a set of parameters used to actually create the
// Tekton Pipeline CRDs
//
// By default we tend to turn Webhoks into tekton Pipeline CRDs
type PlumberJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PlumberJobSpec   `json:"spec,omitempty"`
	Status PlumberJobStatus `json:"status,omitempty"`
}

// PlumberJobSpec the spec of a pipeline request
type PlumberJobSpec struct {
	// Type is the type of job and informs how
	// the jobs is triggered
	Type PlumberJobType `json:"type,omitempty"`
	// Cluster is which Kubernetes cluster is used
	// to run the job, only applicable for that
	// specific agent
	Cluster string `json:"cluster,omitempty"`
	// Namespace defines where to create pods/resources.
	Namespace string `json:"namespace,omitempty"`
	// Job is the name of the job
	Job string `json:"job,omitempty"`
	// Refs is the code under test, determined at
	// runtime by Prow itself
	Refs *Refs `json:"refs,omitempty"`
	// ExtraRefs are auxiliary repositories that
	// need to be cloned, determined from config
	ExtraRefs []Refs `json:"extra_refs,omitempty"`
	// Report determines if the result of this job should
	// be posted as a status on GitHub
	Report bool `json:"report,omitempty"`
	// Context is the name of the status context used to
	// report back to GitHub
	Context string `json:"context,omitempty"`
	// RerunCommand is the command a user would write to
	// trigger this job on their pull request
	RerunCommand string `json:"rerun_command,omitempty"`
	// MaxConcurrency restricts the total number of instances
	// of this job that can run in parallel at once
	MaxConcurrency int `json:"max_concurrency,omitempty"`
	// ErrorOnEviction indicates that the PlumberJob should be completed and given
	// the ErrorState status if the pod that is executing the job is evicted.
	// If this field is unspecified or false, a new pod will be created to replace
	// the evicted one.
	ErrorOnEviction bool `json:"error_on_eviction,omitempty"`

	/*
			// Agent determines which controller fulfills
		// this specific PlumberJobSpec and runs the job
		Agent PlumberJobAgent `json:"agent,omitempty"`

		// PodSpec provides the basis for running the test under
		// a Kubernetes agent
		PodSpec *corev1.PodSpec `json:"pod_spec,omitempty"`

		// BuildSpec provides the basis for running the test as
		// a build-crd resource
		// https://github.com/knative/build
		BuildSpec *buildv1alpha1.BuildSpec `json:"build_spec,omitempty"`

		// JenkinsSpec holds configuration specific to Jenkins jobs
		JenkinsSpec *JenkinsSpec `json:"jenkins_spec,omitempty"`

		// PipelineRunSpec provides the basis for running the test as
		// a pipeline-crd resource
		// https://github.com/tektoncd/pipeline
		PipelineRunSpec *pipelinev1alpha1.PipelineRunSpec `json:"pipeline_run_spec,omitempty"`


		// ReporterConfig holds reporter-specific configuration
		ReporterConfig *ReporterConfig `json:"reporter_config,omitempty"`
	*/
	// DecorationConfig holds configuration options for
	// decorating PodSpecs that users provide
	DecorationConfig *DecorationConfig `json:"decoration_config,omitempty"`
}

// Duration is a wrapper around time.Duration that parses times in either
// 'integer number of nanoseconds' or 'duration string' formats and serializes
// to 'duration string' format.
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &d.Duration); err == nil {
		// b was an integer number of nanoseconds.
		return nil
	}
	// b was not an integer. Assume that it is a duration string.

	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	pd, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	d.Duration = pd
	return nil
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

// DecorationConfig specifies how to augment pods.
//
// This is primarily used to provide automatic integration with gubernator
// and testgrid.
type DecorationConfig struct {
	// Timeout is how long the pod utilities will wait
	// before aborting a job with SIGINT.
	Timeout *Duration `json:"timeout,omitempty"`
	// GracePeriod is how long the pod utilities will wait
	// after sending SIGINT to send SIGKILL when aborting
	// a job. Only applicable if decorating the PodSpec.
	GracePeriod *Duration `json:"grace_period,omitempty"`

	/*
		// UtilityImages holds pull specs for utility container
		// images used to decorate a PodSpec.
		UtilityImages *UtilityImages `json:"utility_images,omitempty"`
		// GCSConfiguration holds options for pushing logs and
		// artifacts to GCS from a job.
		GCSConfiguration *GCSConfiguration `json:"gcs_configuration,omitempty"`
	*/
	// GCSCredentialsSecret is the name of the Kubernetes secret
	// that holds GCS push credentials.
	GCSCredentialsSecret string `json:"gcs_credentials_secret,omitempty"`
	// SSHKeySecrets are the names of Kubernetes secrets that contain
	// SSK keys which should be used during the cloning process.
	SSHKeySecrets []string `json:"ssh_key_secrets,omitempty"`
	// SSHHostFingerprints are the fingerprints of known SSH hosts
	// that the cloning process can trust.
	// Create with ssh-keyscan [-t rsa] host
	SSHHostFingerprints []string `json:"ssh_host_fingerprints,omitempty"`
	// SkipCloning determines if we should clone source code in the
	// initcontainers for jobs that specify refs
	SkipCloning *bool `json:"skip_cloning,omitempty"`
	// CookieFileSecret is the name of a kubernetes secret that contains
	// a git http.cookiefile, which should be used during the cloning process.
	CookiefileSecret string `json:"cookiefile_secret,omitempty"`
}

// Validate ensures all the values set in the DecorationConfig are valid.
func (d *DecorationConfig) Validate() error {
	return nil
}

// PlumberJobStatus provides runtime metadata, such as when it finished, whether it is running, etc.
type PlumberJobStatus struct {
	StartTime      metav1.Time     `json:"startTime,omitempty"`
	CompletionTime *metav1.Time    `json:"completionTime,omitempty"`
	State          PlumberJobState `json:"state,omitempty"`
	Description    string          `json:"description,omitempty"`
	URL            string          `json:"url,omitempty"`

	/*	// PodName applies only to PlumberJobs fulfilled by
		// plank. This field should always be the same as
		// the PlumberJob.ObjectMeta.Name field.
		PodName string `json:"pod_name,omitempty"`

		// BuildID is the build identifier vended either by tot
		// or the snowflake library for this job and used as an
		// identifier for grouping artifacts in GCS for views in
		// TestGrid and Gubernator. Idenitifiers vended by tot
		// are monotonically increasing whereas identifiers vended
		// by the snowflake library are not.
		BuildID string `json:"build_id,omitempty"`

		// JenkinsBuildID applies only to PlumberJobs fulfilled
		// by the jenkins-operator. This field is the build
		// identifier that Jenkins gave to the build for this
		// PlumberJob.
		JenkinsBuildID string `json:"jenkins_build_id,omitempty"`

		// PrevReportStates stores the previous reported plumberJob state per reporter
		// So crier won't make duplicated report attempt
		PrevReportStates map[string]PlumberJobState `json:"prev_report_states,omitempty"`
	*/
}

// Pull describes a pull request at a particular point in time.
type Pull struct {
	Number int    `json:"number"`
	Author string `json:"author"`
	SHA    string `json:"sha"`
	Title  string `json:"title,omitempty"`

	// Ref is git ref can be checked out for a change
	// for example,
	// github: pull/123/head
	// gerrit: refs/changes/00/123/1
	Ref string `json:"ref,omitempty"`
	// Link links to the pull request itself.
	Link string `json:"link,omitempty"`
	// CommitLink links to the commit identified by the SHA.
	CommitLink string `json:"commit_link,omitempty"`
	// AuthorLink links to the author of the pull request.
	AuthorLink string `json:"author_link,omitempty"`
}

// Refs describes how the repo was constructed.
type Refs struct {
	// Org is something like kubernetes or k8s.io
	Org string `json:"org"`
	// Repo is something like test-infra
	Repo string `json:"repo"`
	// RepoLink links to the source for Repo.
	RepoLink string `json:"repo_link,omitempty"`

	BaseRef string `json:"base_ref,omitempty"`
	BaseSHA string `json:"base_sha,omitempty"`
	// BaseLink is a link to the commit identified by BaseSHA.
	BaseLink string `json:"base_link,omitempty"`

	Pulls []Pull `json:"pulls,omitempty"`

	// PathAlias is the location under <root-dir>/src
	// where this repository is cloned. If this is not
	// set, <root-dir>/src/github.com/org/repo will be
	// used as the default.
	PathAlias string `json:"path_alias,omitempty"`
	// CloneURI is the URI that is used to clone the
	// repository. If unset, will default to
	// `https://github.com/org/repo.git`.
	CloneURI string `json:"clone_uri,omitempty"`
	// SkipSubmodules determines if submodules should be
	// cloned when the job is run. Defaults to true.
	SkipSubmodules bool `json:"skip_submodules,omitempty"`
	// CloneDepth is the depth of the clone that will be used.
	// A depth of zero will do a full clone.
	CloneDepth int `json:"clone_depth,omitempty"`
}
