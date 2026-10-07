package scmclients_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/lighthouse"
	"github.com/jenkins-x/lighthouse/pkg/scmclients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configGetter(kind, server, botUser string) config.Getter {
	cfg := &config.Config{}
	cfg.ProviderConfig = &lighthouse.ProviderConfig{Kind: kind, Server: server, BotUser: botUser}
	return func() *config.Config { return cfg }
}

func newClientSet(t *testing.T, cfg config.Getter, o scmclients.Options) *scmclients.ClientSet {
	t.Helper()
	cs, err := scmclients.New(cfg, o)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, cs.Clean()) })
	return cs
}

func TestNewResolvesFromConfig(t *testing.T) {
	t.Setenv("GIT_KIND", "")
	t.Setenv("GIT_SERVER", "")
	t.Setenv("GIT_USER", "")
	t.Setenv("GIT_TOKEN", "abc")

	cs := newClientSet(t, configGetter("fake", "https://git.example.com", "config-bot"), scmclients.Options{})

	assert.Equal(t, "fake", cs.GitKind)
	assert.Equal(t, "https://git.example.com", cs.ServerURL.String())
	assert.Equal(t, "config-bot", cs.BotName)
	botName, err := cs.SCMProviderClient.BotName()
	require.NoError(t, err)
	assert.Equal(t, "config-bot", botName)
	assert.NotNil(t, cs.GitClient)
	assert.NotNil(t, cs.GitFactory)
	assert.NotNil(t, cs.FileBrowsers.LighthouseGitFileBrowser())
}

func TestNewOptionsOverrideConfig(t *testing.T) {
	t.Setenv("GIT_KIND", "")
	t.Setenv("GIT_SERVER", "")
	t.Setenv("GIT_USER", "")
	t.Setenv("GIT_TOKEN", "")
	t.Setenv("GIT_TOKEN_PATH", "")

	cs := newClientSet(t, configGetter("github", "https://github.com", "config-bot"), scmclients.Options{
		GitKind:     "fake",
		ServerURL:   "https://other.example.com",
		Credentials: scmclients.NewTokenCredentials("option-bot", "abc"),
	})

	assert.Equal(t, "fake", cs.GitKind)
	assert.Equal(t, "https://other.example.com", cs.ServerURL.String())
	assert.Equal(t, "option-bot", cs.BotName)
}

func TestNewFailsWithoutToken(t *testing.T) {
	t.Setenv("GIT_TOKEN", "")
	t.Setenv("GIT_TOKEN_PATH", "")

	_, err := scmclients.New(configGetter("fake", "https://git.example.com", "bot"), scmclients.Options{})
	require.Error(t, err)
}

func TestForOwnerReturnsSameClientSet(t *testing.T) {
	cs := newClientSet(t, configGetter("fake", "https://git.example.com", "bot"), scmclients.Options{
		Credentials: scmclients.NewTokenCredentials("bot", "abc"),
	})

	for _, owner := range []string{"org-a", "org-b", ""} {
		got, err := cs.ForOwner(owner)
		require.NoError(t, err)
		assert.Same(t, cs, got)
	}
}

func TestSCMClientResolvesTokenPerRequest(t *testing.T) {
	var gotAuth []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"login":"bot"}`))
	}))
	t.Cleanup(server.Close)

	tokenFile := filepath.Join(t.TempDir(), "git-token")
	require.NoError(t, os.WriteFile(tokenFile, []byte("first"), 0o600))
	t.Setenv("GIT_TOKEN", "")
	t.Setenv("GIT_TOKEN_PATH", tokenFile)

	cs := newClientSet(t, configGetter("github", server.URL, "bot"), scmclients.Options{})

	_, _, err := cs.SCMClient.Users.Find(t.Context())
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tokenFile, []byte("second"), 0o600))
	_, _, err = cs.SCMClient.Users.Find(t.Context())
	require.NoError(t, err)

	assert.Equal(t, []string{"Bearer first", "Bearer second"}, gotAuth)
}
