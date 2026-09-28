package scmauth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}

func TestNewRejectsUnusableModes(t *testing.T) {
	tests := map[string]struct {
		opts    Options
		wantErr string
	}{
		"no mode is not guessed at": {
			opts:    Options{StaticToken: "tok"},
			wantErr: "no authentication mode configured",
		},
		"unknown mode is named": {
			opts:    Options{Mode: Mode("magic")},
			wantErr: `unknown authentication mode "magic"`,
		},
		"static mode without a token": {
			opts:    Options{Mode: ModeStaticToken},
			wantErr: "no token available",
		},
		"owner tokens without a directory": {
			opts:    Options{Mode: ModeOwnerTokens},
			wantErr: "no secret directory",
		},
		"github app without a directory": {
			opts:    Options{Mode: ModeGitHubApp},
			wantErr: "no secret directory",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := New(test.opts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func TestNewStaticMode(t *testing.T) {
	src, err := New(Options{Mode: ModeStaticToken, StaticToken: "tok", BotName: "bot"})
	require.NoError(t, err)

	got, err := src.Token("any-owner")
	require.NoError(t, err)
	assert.Equal(t, "tok", got)
	assert.False(t, src.UsesAppInstallationToken())
	assert.False(t, src.RequiresOwner(), "one credential serves every owner")

	name, err := src.BotName()
	require.NoError(t, err)
	assert.Equal(t, "bot", name)
}

func TestNewOwnerTokensMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "some-generated-name-0", "https://github.com/myorg=owner-token")
	writeFile(t, dir, UsernameFilename, "legacy-bot")

	src, err := New(Options{
		Mode:      ModeOwnerTokens,
		ServerURL: "https://github.com",
		SecretDir: dir,
	})
	require.NoError(t, err)

	got, err := src.Token("myorg")
	require.NoError(t, err)
	assert.Equal(t, "owner-token", got)

	assert.False(t, src.UsesAppInstallationToken())
	assert.True(t, src.RequiresOwner())

	name, err := src.BotName()
	require.NoError(t, err)
	assert.Equal(t, "legacy-bot", name, "the username file should supply the login")
}

func TestNewOwnerTokensModeUnknownOwner(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "tokens-0", "https://github.com/myorg=owner-token")

	src, err := New(Options{Mode: ModeOwnerTokens, ServerURL: "https://github.com", SecretDir: dir})
	require.NoError(t, err)

	_, err = src.Token("other-org")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no token found for owner URL")
}

func TestNewOwnerTokensModeKeysOnServerURL(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "tokens-0", "https://ghe.example.com/myorg=enterprise-token")

	ghSrc, err := New(Options{Mode: ModeOwnerTokens, ServerURL: "https://github.com", SecretDir: dir})
	require.NoError(t, err)
	_, err = ghSrc.Token("myorg")
	require.Error(t, err)

	gheSrc, err := New(Options{Mode: ModeOwnerTokens, ServerURL: "https://ghe.example.com", SecretDir: dir})
	require.NoError(t, err)
	got, err := gheSrc.Token("myorg")
	require.NoError(t, err)
	assert.Equal(t, "enterprise-token", got)
}

func TestNewOwnerTokensModeIgnoresProjectedSecretFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "tokens-0", "https://github.com/myorg=owner-token")
	require.NoError(t, os.Mkdir(filepath.Join(dir, "..2026_08_18_00_00_00.123456"), 0o700))
	writeFile(t, dir, ".hidden", "not a credential")

	src, err := New(Options{Mode: ModeOwnerTokens, ServerURL: "https://github.com", SecretDir: dir})
	require.NoError(t, err)

	got, err := src.Token("myorg")
	require.NoError(t, err)
	assert.Equal(t, "owner-token", got)
}

func TestNewGitHubAppMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, AppIDFilename, "123456\n")
	writeFile(t, dir, PrivateKeyFilename, string(pkcs1PEM(t, sharedTestKey(t))))

	src, err := New(Options{Mode: ModeGitHubApp, ServerURL: "https://github.com", SecretDir: dir})
	require.NoError(t, err)

	assert.True(t, src.UsesAppInstallationToken())
}

func TestNewGitHubAppModeConfigErrors(t *testing.T) {
	validKey := string(pkcs1PEM(t, sharedTestKey(t)))

	tests := map[string]struct {
		files   map[string]string
		wantErr string
	}{
		"missing app id": {
			files:   map[string]string{PrivateKeyFilename: validKey},
			wantErr: "reading GitHub App ID",
		},
		"missing private key": {
			files:   map[string]string{AppIDFilename: "1"},
			wantErr: "reading GitHub App private key",
		},
		"app id is not a number": {
			files:   map[string]string{AppIDFilename: "not-a-number", PrivateKeyFilename: validKey},
			wantErr: "parsing GitHub App ID",
		},
		"private key is not a key": {
			files:   map[string]string{AppIDFilename: "1", PrivateKeyFilename: "nope"},
			wantErr: "parsing GitHub App private key",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			for filename, content := range test.files {
				writeFile(t, dir, filename, content)
			}

			_, err := New(Options{Mode: ModeGitHubApp, SecretDir: dir})
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func TestNewGitHubAppModeTakesUsernameFromDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, AppIDFilename, "123456")
	writeFile(t, dir, PrivateKeyFilename, string(pkcs1PEM(t, sharedTestKey(t))))
	writeFile(t, dir, UsernameFilename, "configured[bot]")

	src, err := New(Options{Mode: ModeGitHubApp, SecretDir: dir})
	require.NoError(t, err)

	app := src.(*appTokenSource)
	assert.Equal(t, "configured[bot]", app.configuredBotName)
}

func TestReadUsernameTreatsEmptyAsMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, UsernameFilename, "   \n")

	_, err := readUsername(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is empty")
}
