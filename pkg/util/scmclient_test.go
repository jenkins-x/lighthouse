package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test ensures that we can retrieve the Git token from the environment
func TestGetSCMToken(t *testing.T) {
	os.Setenv("GIT_TOKEN", "mytokenfromenvvar")

	token, err := util.GetSCMToken("github")
	require.NoError(t, err, "failed to get SCM token")
	assert.Equal(t, "mytokenfromenvvar", token, "failed to get expected SCM token: %s", token)
}

// This test ensures that we can retrieve the Git token from the filesystem
func TestGetSCMTokenPath(t *testing.T) {
	err := os.Unsetenv("GIT_TOKEN")
	require.NoError(t, err, "failed to unset environment variable GIT_TOKEN")

	os.Setenv("GIT_TOKEN_PATH", filepath.Join("test_data", "secret_dir", "git-token"))

	token, err := util.GetSCMToken("github")
	require.NoError(t, err, "failed to get SCM token")
	assert.Equal(t, "mytokenfrompath", token, "failed to get expected SCM token: %s", token)
}

// This test ensures that GIT_TOKEN takes priority over GIT_TOKEN_PATH for backwards compatibility
func TestGetSCMTokenPathFallback(t *testing.T) {
	os.Setenv("GIT_TOKEN", "mytokenfromenvvar")
	os.Setenv("GIT_TOKEN_PATH", filepath.Join("test_data", "secret_dir", "git-token"))

	token, err := util.GetSCMToken("github")
	require.NoError(t, err, "failed to get SCM token")
	assert.Equal(t, "mytokenfromenvvar", token, "failed to get expected SCM token: %s", token)
}

// This test ensures that we receive an error if path GIT_TOKEN_PATH does not exist
func TestGetSCMTokenPathMissingError(t *testing.T) {
	err := os.Unsetenv("GIT_TOKEN")
	require.NoError(t, err, "failed to unset environment variable GIT_TOKEN")

	os.Setenv("GIT_TOKEN_PATH", filepath.Join("test_data", "secret_dir", "does-not-exist"))

	_, err = util.GetSCMToken("github")
	require.Error(t, err)
}

// This test ensures that we receive an error if neither GIT_TOKEN or GIT_TOKEN_PATH are set
func TestGetSCMTokenUnsetError(t *testing.T) {
	err := os.Unsetenv("GIT_TOKEN")
	require.NoError(t, err, "failed to unset environment variable GIT_TOKEN")

	err = os.Unsetenv("GIT_TOKEN_PATH")
	require.NoError(t, err, "failed to unset environment variable GIT_TOKEN_PATH")

	_, err = util.GetSCMToken("github")
	require.Error(t, err)
}

func TestHMACToken(t *testing.T) {
	tests := map[string]struct {
		envVars   map[string]string
		wantError bool
		hmacToken string
	}{
		"missing env vars": {
			envVars:   map[string]string{},
			hmacToken: "",
		},
		"hmac token env var": {
			envVars: map[string]string{
				"HMAC_TOKEN": "myhmactokenfromenvvar",
			},
			hmacToken: "myhmactokenfromenvvar",
		},
		"hmac token path env var": {
			envVars: map[string]string{
				"HMAC_TOKEN_PATH": filepath.Join("test_data", "secret_dir", "hmac-token"),
			},
			hmacToken: "myhmactokenfrompath",
		},
		"hmac token env var and path env var": {
			envVars: map[string]string{
				"HMAC_TOKEN":      "myhmactokenfromenvvar",
				"HMAC_TOKEN_PATH": filepath.Join("test_data", "secret_dir", "hmac-token"),
			},
			hmacToken: "myhmactokenfrompath",
		},
		"hmac token missing path env var": {
			envVars: map[string]string{
				"HMAC_TOKEN_PATH": filepath.Join("test_data", "secret_dir", "does-not-exist"),
			},
			hmacToken: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// Set environment variables
			for envVarName, envVarValue := range test.envVars {
				os.Setenv(envVarName, envVarValue)
			}

			// Attempt to retrieve HMAC token
			hmacToken := util.HMACToken()

			// Verify HMAC token value
			assert.Equal(t, test.hmacToken, hmacToken, "failed to get expected HMAC token: %s", hmacToken)

			// Unset environment variables
			for envVarName := range test.envVars {
				err := os.Unsetenv(envVarName)
				require.NoErrorf(t, err, "failed to unset environment variable %s", envVarName)
			}
		})
	}
}
