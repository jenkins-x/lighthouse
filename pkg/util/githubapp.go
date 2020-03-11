package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// GetGitHubAppSecretDir returns the location of the GitHub App secrets dir, if defined, and an empty string otherwise
func GetGitHubAppSecretDir() string {
	return os.Getenv(GitHubAppSecretDirEnvVar)
}

// GetGitHubAppAPIUser returns the username to be used for GitHub API calls with the GitHub App, if so configured.
// If there is no configured secrets dir, it returns an empty string.
func GetGitHubAppAPIUser() (string, error) {
	secretsDir := GetGitHubAppSecretDir()
	if secretsDir == "" {
		return "", nil
	}
	dirExist, err := DirExists(secretsDir)
	if err != nil {
		return "", errors.Wrapf(err, "checking if %s exists and is a directory", secretsDir)
	}
	if !dirExist {
		return "", fmt.Errorf("secrets directory for GitHub App integration %s does not exist", secretsDir)
	}
	userFile := filepath.Join(secretsDir, GitHubAppAPIUserFilename)
	fileExist, err := FileExists(userFile)
	if err != nil {
		return "", errors.Wrapf(err, "checking if %s exists and is a file", userFile)
	}
	if !fileExist {
		return "", fmt.Errorf("username file in secrets directory for GitHub App integration %s does not exist", secretsDir)
	}

	/* #nosec */
	data, err := ioutil.ReadFile(userFile)
	if err != nil {
		return "", errors.Wrapf(err, "reading username file in secrets directory for GitHub App integration %s", secretsDir)
	}

	return strings.TrimSpace(string(data)), nil
}
