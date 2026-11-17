package util_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The following tests verify that the HMACToken function works correctly
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
			hmacToken: "myhmactokenfromenvvar",
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
