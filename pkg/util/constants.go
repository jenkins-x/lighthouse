package util

const (
	// CommitStatusPendingDescription is the description used for PR commit status for pipelines we have just kicked off.
	CommitStatusPendingDescription = "Pipeline pending"

	// OverriddenByPrefix is the beginning of the description for commit statuses set by /override
	OverriddenByPrefix = "Overridden by"

	// GitHubAppGitRemoteUsername Username for git https URLs when using a GitHub App token.
	// see https://developer.github.com/apps/building-github-apps/authenticating-with-github-apps/#http-based-git-access-by-an-installation
	GitHubAppGitRemoteUsername = "x-access-token"

	// GitHubAppSecretDirEnvVar is the name of the environment variable which would contain secrets when this is configured
	// with the GitHub App.
	GitHubAppSecretDirEnvVar = "GITHUB_APP_SECRET_DIR" // #nosec

	// GitHubAppAPIUserFilename is the filename inside the GitHub App secrets dir which will contain the user we will
	// use for GitHub API calls when present.
	GitHubAppAPIUserFilename = "username"

	// TektonAgent the default agent name
	TektonAgent = "tekton"

	// LighthouseJobTypeLabel is added in resources created by lighthouse and
	// carries the job type (presubmit, postsubmit, periodic, batch)
	// that the pod is running.
	LighthouseJobTypeLabel = "lighthouse.jenkins-x.io/type"

	// LighthouseJobIDLabel is added in resources created by lighthouse and
	// carries the ID of the LighthouseJob that the pod is fulfilling.
	// We also name resources after the LighthouseJob that spawned them but
	// this allows for multiple resources to be linked to one
	// LighthouseJob.
	LighthouseJobIDLabel = "lighthouse.jenkins-x.io/id"

	// LighthousePipelineActivityNameLabel is added to the LighthouseJob with
	// the name of the PipelineActivity corresponding to it.
	LighthousePipelineActivityNameLabel = "lighthouse.jenkins-x.io/activityName"

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

	// ContextLabel is added in resources created by Lighthouse and contains the job context.
	ContextLabel = "lighthouse.jenkins-x.io/context"

	// BranchLabel is added in resources created by Lighthouse and contains the branch name for the job.
	BranchLabel = "lighthouse.jenkins-x.io/branch"

	// BuildNumLabel is added in resources created by Lighthouse and contains the build number for the job.
	BuildNumLabel = "lighthouse.jenkins-x.io/buildNum"

	// DefaultClusterAlias specifies the default context for resources owned by jobs (pods/builds).
	DefaultClusterAlias = "default"

	// ActivityOwnerLabel is the label for the org/owner on the PipelineActivity
	ActivityOwnerLabel = "owner"
	// ActivityRepositoryLabel is the label for the repo name on the PipelineActivity
	ActivityRepositoryLabel = "repository"
	// ActivityBranchLabel is the label for the branch name on the PipelineActivity
	ActivityBranchLabel = "branch"
	// ActivityBuildLabel is the label for the build number on the PipelineActivity
	ActivityBuildLabel = "build"
	// ActivityContextLabel is the label for the (optional) pipeline context on the PipelineActivity
	ActivityContextLabel = "context"

	// GithubServer the default github server URL
	GithubServer = "https://github.com"

	// ProwConfigMapName name of the ConfgMap holding the config
	ProwConfigMapName = "config"
	// ProwPluginsConfigMapName name of the ConfigMap holding the plugins config
	ProwPluginsConfigMapName = "plugins"
	// ProwConfigFilename config file name
	ProwConfigFilename = "config.yaml"
	// ProwPluginsFilename plugins file name
	ProwPluginsFilename = "plugins.yaml"

	// LighthouseCommandPrefix is an optional prefix for commands to deal with things like GitLab hijacking /approve
	LighthouseCommandPrefix = "lh-"
)
