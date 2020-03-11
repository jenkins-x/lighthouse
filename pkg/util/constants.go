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
)
