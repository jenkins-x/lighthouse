package scmclients

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialsFromEnv(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "git-token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("tokenfrompath\n"), 0o600))

	tests := map[string]struct {
		token     string
		tokenPath string
		want      string
		wantErr   bool
	}{
		"token from env": {
			token: "tokenfromenv",
			want:  "tokenfromenv",
		},
		"token from path": {
			tokenPath: tokenFile,
			want:      "tokenfrompath",
		},
		"env takes priority over path": {
			token:     "tokenfromenv",
			tokenPath: tokenFile,
			want:      "tokenfromenv",
		},
		"missing path": {
			tokenPath: filepath.Join(dir, "does-not-exist"),
			wantErr:   true,
		},
		"neither set": {
			wantErr: true,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv("GIT_TOKEN", tc.token)
			t.Setenv("GIT_TOKEN_PATH", tc.tokenPath)

			creds, err := credentialsFromEnv("my-bot")
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			token, err := creds.Token(t.Context())
			require.NoError(t, err)
			assert.Equal(t, tc.want, token.Token)
			assert.Equal(t, "my-bot", creds.BotName())
			assert.Equal(t, "my-bot", creds.CloneUser())
		})
	}
}

func TestCredentialsFromEnvPicksUpRotatedToken(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "git-token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("first"), 0o600))
	t.Setenv("GIT_TOKEN", "")
	t.Setenv("GIT_TOKEN_PATH", tokenFile)

	creds, err := credentialsFromEnv("my-bot")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(tokenFile, []byte("second"), 0o600))
	token, err := creds.Token(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "second", token.Token)
}
