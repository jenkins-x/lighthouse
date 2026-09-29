package scmauth

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/h2non/gock"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const githubAPI = "https://api.github.com"

func newTestAppSource(t *testing.T, botName string) TokenSource {
	t.Helper()
	src, err := NewAppTokenSource(AppConfig{
		AppID:      99,
		PrivateKey: pkcs1PEM(t, sharedTestKey(t)),
		BotName:    botName,
	})
	require.NoError(t, err)
	return src
}

func TestExpiringSoon(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) *time.Time {
		v := now.Add(d)
		return &v
	}

	tests := map[string]struct {
		expiresAt *time.Time
		want      bool
	}{
		"fresh hour-long token":        {expiresAt: at(time.Hour), want: false},
		"just outside the buffer":      {expiresAt: at(tokenRefreshBuffer + time.Minute), want: false},
		"just inside the buffer":       {expiresAt: at(tokenRefreshBuffer - time.Minute), want: true},
		"already expired":              {expiresAt: at(-time.Minute), want: true},
		"missing expiry is distrusted": {expiresAt: nil, want: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := expiringSoon(&scm.InstallationToken{ExpiresAt: test.expiresAt}, now)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestAppTokenSourceMintsThenCaches(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		defer gock.Off()

		gock.New(githubAPI).Get("/orgs/myorg/installation").Reply(200).
			JSON(map[string]interface{}{"id": 555})
		gock.New(githubAPI).Post("/app/installations/555/access_tokens").Reply(201).
			JSON(map[string]interface{}{
				"token":      "ghs_minted",
				"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
			})

		src := newTestAppSource(t, "")

		got, err := src.Token("myorg")
		require.NoError(t, err)
		assert.Equal(t, "ghs_minted", got)

		// Only one mint is registered, so a second network call would fail to match.
		got, err = src.Token("myorg")
		require.NoError(t, err)
		assert.Equal(t, "ghs_minted", got)

		assert.True(t, gock.IsDone(), "each request should have been made exactly once")
	})
}

func TestAppTokenSourceRefreshesBeforeExpiry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		defer gock.Off()

		expiry := time.Now().Add(time.Hour)

		gock.New(githubAPI).Get("/orgs/myorg/installation").Reply(200).
			JSON(map[string]interface{}{"id": 555})
		gock.New(githubAPI).Post("/app/installations/555/access_tokens").Reply(201).
			JSON(map[string]interface{}{"token": "first", "expires_at": expiry.Format(time.RFC3339)})
		gock.New(githubAPI).Post("/app/installations/555/access_tokens").Reply(201).
			JSON(map[string]interface{}{
				"token":      "second",
				"expires_at": expiry.Add(time.Hour).Format(time.RFC3339),
			})

		src := newTestAppSource(t, "")

		got, err := src.Token("myorg")
		require.NoError(t, err)
		assert.Equal(t, "first", got)

		// Advance to inside the refresh buffer.
		time.Sleep(time.Hour - tokenRefreshBuffer + time.Second)

		got, err = src.Token("myorg")
		require.NoError(t, err)
		assert.Equal(t, "second", got, "a token close to expiry must be re-minted")

		// The installation lookup is cached, so only the mint repeats.
		assert.True(t, gock.IsDone())
	})
}

func TestAppTokenSourceFallsBackToUserInstallation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		defer gock.Off()

		gock.New(githubAPI).Get("/orgs/someuser/installation").Reply(404).
			JSON(map[string]interface{}{"message": "Not Found"})
		gock.New(githubAPI).Get("/users/someuser/installation").Reply(200).
			JSON(map[string]interface{}{"id": 777})
		gock.New(githubAPI).Post("/app/installations/777/access_tokens").Reply(201).
			JSON(map[string]interface{}{
				"token":      "user_token",
				"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
			})

		src := newTestAppSource(t, "")

		got, err := src.Token("someuser")
		require.NoError(t, err)
		assert.Equal(t, "user_token", got)
		assert.True(t, gock.IsDone())
	})
}

func TestAppTokenSourceCachesMissingInstallation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		defer gock.Off()

		gock.New(githubAPI).Get("/orgs/nope/installation").Reply(404).
			JSON(map[string]interface{}{"message": "Not Found"})
		gock.New(githubAPI).Get("/users/nope/installation").Reply(404).
			JSON(map[string]interface{}{"message": "Not Found"})

		src := newTestAppSource(t, "")

		_, err := src.Token("nope")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not installed")

		_, err = src.Token("nope")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not installed")

		assert.True(t, gock.IsDone(), "the lookup should not be repeated")
	})
}

func TestAppTokenSourceDoesNotCacheTransientFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		defer gock.Off()

		gock.New(githubAPI).Get("/orgs/myorg/installation").Reply(500).
			JSON(map[string]interface{}{"message": "Server Error"})
		gock.New(githubAPI).Get("/orgs/myorg/installation").Reply(200).
			JSON(map[string]interface{}{"id": 555})
		gock.New(githubAPI).Post("/app/installations/555/access_tokens").Reply(201).
			JSON(map[string]interface{}{
				"token":      "recovered",
				"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
			})

		src := newTestAppSource(t, "")

		_, err := src.Token("myorg")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "not installed",
			"a server error must not be reported as a missing installation")

		got, err := src.Token("myorg")
		require.NoError(t, err)
		assert.Equal(t, "recovered", got, "the owner must recover once GitHub does")
		assert.True(t, gock.IsDone())
	})
}

func TestAppTokenSourceBotNameDerivesBotSuffix(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		defer gock.Off()

		gock.New(githubAPI).Get("/app").Reply(200).
			JSON(map[string]interface{}{"slug": "my-lighthouse"})

		src := newTestAppSource(t, "")

		got, err := src.BotName()
		require.NoError(t, err)
		assert.Equal(t, "my-lighthouse[bot]", got)

		// Cached, so a second call issues no request.
		got, err = src.BotName()
		require.NoError(t, err)
		assert.Equal(t, "my-lighthouse[bot]", got)
		assert.True(t, gock.IsDone())
	})
}

func TestAppTokenSourceBotNameRejectsMismatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		defer gock.Off()

		gock.New(githubAPI).Get("/app").Reply(200).
			JSON(map[string]interface{}{"slug": "my-lighthouse"})

		src := newTestAppSource(t, "stale-name[bot]")

		_, err := src.BotName()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match")
		assert.Contains(t, err.Error(), "my-lighthouse[bot]")
	})
}

func TestAppTokenSourceBotNameAcceptsMatchingOverride(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		defer gock.Off()

		gock.New(githubAPI).Get("/app").Reply(200).
			JSON(map[string]interface{}{"slug": "my-lighthouse"})

		src := newTestAppSource(t, "my-lighthouse[bot]")

		got, err := src.BotName()
		require.NoError(t, err)
		assert.Equal(t, "my-lighthouse[bot]", got)
	})
}

func TestAppTokenSourceRequiresOwner(t *testing.T) {
	src := newTestAppSource(t, "")

	assert.True(t, src.RequiresOwner())

	_, err := src.Token("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner is required")
}

func TestAppTokenSourceUsesAppInstallationToken(t *testing.T) {
	src := newTestAppSource(t, "")
	assert.True(t, src.UsesAppInstallationToken(),
		"app tokens are attached to requests differently from ordinary tokens")
}

func TestNewAppTokenSourceValidatesConfig(t *testing.T) {
	tests := map[string]struct {
		cfg     AppConfig
		wantErr string
	}{
		"no app id": {
			cfg:     AppConfig{PrivateKey: pkcs1PEM(t, sharedTestKey(t))},
			wantErr: "no GitHub App ID",
		},
		"no private key": {
			cfg:     AppConfig{AppID: 1},
			wantErr: "no GitHub App private key",
		},
		"unparseable private key": {
			cfg:     AppConfig{AppID: 1, PrivateKey: []byte("garbage")},
			wantErr: "parsing GitHub App private key",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewAppTokenSource(test.cfg)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}
