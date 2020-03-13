package v1alpha1

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineKind specifies how the job is triggered.
type PipelineKind string

// Various job types.
const (
	// PresubmitJob means it runs on unmerged PRs.
	PresubmitJob PipelineKind = "presubmit"
	// PostsubmitJob means it runs on each new commit.
	PostsubmitJob PipelineKind = "postsubmit"
	// Periodic job means it runs on a time-basis, unrelated to git changes.
	PeriodicJob PipelineKind = "periodic"
	// BatchJob tests multiple unmerged PRs at the same time.
	BatchJob PipelineKind = "batch"
)

// PipelineState specifies the current pipelne status
type PipelineState string

// Various job types.
const (
	// TriggeredState for pipelines that have been triggered
	TriggeredState PipelineState = "triggered"

	// PendingState pipeline is pending
	PendingState PipelineState = "pending"

	// RunningState pipeline is running
	RunningState PipelineState = "running"

	// SuccessState pipeline is successful
	SuccessState PipelineState = "success"

	// FailureState failed
	FailureState PipelineState = "failure"

	// AbortedState aborted
	AbortedState PipelineState = "aborted"
)

// Environment variables to be added to the pipeline we kick off
const (
	// JobSpecEnv is a legacy Prow variable with "type:(type)"
	JobSpecEnv = "JOB_SPEC"
	// JobNameEnv is the name of the job
	JobNameEnv = "JOB_NAME"
	// JobTypeEnv is the type of job
	JobTypeEnv = "JOB_TYPE"
	// RepoOwnerEnv is the org/owner for the repository we're building
	RepoOwnerEnv = "REPO_OWNER"
	// RepoNameEnv is the name of the repository we're building
	RepoNameEnv = "REPO_NAME"
	// PullBaseRefEnv is the base ref (such as master) for a pull request
	PullBaseRefEnv = "PULL_BASE_REF"
	// PullBaseShaEnv is the actual commit sha for the base for a pull request
	PullBaseShaEnv = "PULL_BASE_SHA"
	// PullRefsEnv is the refs and shas for the base and PR, like "master:abcd1234...,123:5678abcd..." for PR-123.
	PullRefsEnv = "PULL_REFS"
	// PullNumberEnv is the pull request number
	PullNumberEnv = "PULL_NUMBER"
	// PullPullShaEnv is the pull request's sha
	PullPullShaEnv = "PULL_PULL_SHA"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LighthouseJob contains the arguments to create a Jenkins X Pipeline and to report on it
type LighthouseJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LighthouseJobSpec   `json:"spec,omitempty"`
	Status LighthouseJobStatus `json:"status,omitempty"`
}

// LighthouseJobStatus represents the status of a pipeline
type LighthouseJobStatus struct {
	State PipelineState `json:"state,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LighthouseJobList represents a list of pipeline options
type LighthouseJobList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LighthouseJob `json:"items"`
}

// LighthouseJobSpec the spec of a pipeline request
type LighthouseJobSpec struct {
	// Type is the type of job and informs how
	// the jobs is triggered
	Type PipelineKind `json:"type,omitempty"`
	// Namespace defines where to create pods/resources.
	Namespace string `json:"namespace,omitempty"`
	// Job is the name of the job
	Job string `json:"job,omitempty"`
	// Refs is the code under test, determined at
	// runtime by Prow itself
	Refs *Refs `json:"refs,omitempty"`
	// Context is the name of the status context used to
	// report back to GitHub
	Context string `json:"context,omitempty"`
	// RerunCommand is the command a user would write to
	// trigger this job on their pull request
	RerunCommand string `json:"rerun_command,omitempty"`
	// MaxConcurrency restricts the total number of instances
	// of this job that can run in parallel at once
	MaxConcurrency int `json:"max_concurrency,omitempty"`
	// LastCommitSHA is the commit that will be/has been reported to on the SCM provider
	LastCommitSHA string `json:"lastCommitSHA,omitempty"`
}

// GetEnvVars gets a map of the environment variables we'll set in the pipeline for this spec.
func (s *LighthouseJobSpec) GetEnvVars() map[string]string {
	env := map[string]string{
		JobNameEnv: s.Job,
		JobTypeEnv: string(s.Type),
	}

	registry := os.Getenv("DOCKER_REGISTRY")
	if registry != "" {
		env["DOCKER_REGISTRY"] = registry
	}

	env[JobSpecEnv] = fmt.Sprintf("type:%s", s.Type)

	if s.Type == PeriodicJob {
		return env
	}

	env[RepoOwnerEnv] = s.Refs.Org
	env[RepoNameEnv] = s.Refs.Repo
	env[PullBaseRefEnv] = s.Refs.BaseRef
	env[PullBaseShaEnv] = s.Refs.BaseSHA
	env[PullRefsEnv] = s.Refs.String()

	if s.Type == PostsubmitJob || s.Type == BatchJob {
		return env
	}

	env[PullNumberEnv] = strconv.Itoa(s.Refs.Pulls[0].Number)
	env[PullPullShaEnv] = s.Refs.Pulls[0].SHA

	return env
}

// Duration is a wrapper around time.Duration that parses times in either
// 'integer number of nanoseconds' or 'duration string' formats and serializes
// to 'duration string' format.
type Duration struct {
	Duration time.Duration
}

// UnmarshalJSON unmarshal a byte array into a Duration object
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

// MarshalJSON marshals a duration object to a byte array
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
	// Launch with ssh-keyscan [-t rsa] host
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

func (r *Refs) String() string {
	rs := []string{}
	if r.BaseSHA != "" {
		rs = append(rs, fmt.Sprintf("%s:%s", r.BaseRef, r.BaseSHA))
	} else {
		rs = append(rs, r.BaseRef)
	}

	for _, pull := range r.Pulls {
		ref := fmt.Sprintf("%d:%s", pull.Number, pull.SHA)

		if pull.Ref != "" {
			ref = fmt.Sprintf("%s:%s", ref, pull.Ref)
		}

		rs = append(rs, ref)
	}
	return strings.Join(rs, ",")
}

// ByNum implements sort.Interface for []Pull to sort by ascending PR number.
type ByNum []Pull

func (prs ByNum) Len() int           { return len(prs) }
func (prs ByNum) Swap(i, j int)      { prs[i], prs[j] = prs[j], prs[i] }
func (prs ByNum) Less(i, j int) bool { return prs[i].Number < prs[j].Number }

// ToPipelineState converts the PipelineActivity state to LighthouseJob state
func ToPipelineState(status v1.ActivityStatusType) PipelineState {
	switch status {
	case v1.ActivityStatusTypePending:
		return PendingState
	case v1.ActivityStatusTypeAborted:
		return AbortedState
	case v1.ActivityStatusTypeRunning:
		return RunningState
	case v1.ActivityStatusTypeSucceeded:
		return SuccessState
	case v1.ActivityStatusTypeFailed, v1.ActivityStatusTypeError:
		return FailureState
	default:
		return FailureState
	}
}
