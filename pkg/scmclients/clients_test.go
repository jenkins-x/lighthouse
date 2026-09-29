package scmclients

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/scmauth"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSource struct {
	token         string
	botName       string
	requiresOwner bool
	appToken      bool
}

func (f *fakeSource) Token(owner string) (string, error) {
	if f.requiresOwner && owner == "" {
		return "", scmauth.ErrNoToken
	}
	return f.token, nil
}

func (f *fakeSource) BotName() (string, error)       { return f.botName, nil }
func (f *fakeSource) RequiresOwner() bool            { return f.requiresOwner }
func (f *fakeSource) UsesAppInstallationToken() bool { return f.appToken }

func noConfig() *config.Config { return nil }

func TestForOwnerCachesPerOwner(t *testing.T) {
	c := NewWithSource(&fakeSource{token: "tok", botName: "bot"}, noConfig)

	first, err := c.ForOwner("acme")
	require.NoError(t, err)
	again, err := c.ForOwner("acme")
	require.NoError(t, err)
	other, err := c.ForOwner("other")
	require.NoError(t, err)

	assert.Same(t, first, again)
	assert.NotSame(t, first, other)
}

func TestForOwnerRejectsEmptyOwnerWhenScopedPerOwner(t *testing.T) {
	c := NewWithSource(&fakeSource{token: "tok", botName: "bot", requiresOwner: true}, noConfig)

	_, err := c.ForOwner("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scoped per owner")

	_, err = c.ForOwner("acme")
	assert.NoError(t, err)
}

func TestForOwnerAllowsEmptyOwnerForSharedCredential(t *testing.T) {
	c := NewWithSource(&fakeSource{token: "tok", botName: "bot"}, noConfig)

	_, err := c.ForOwner("")
	assert.NoError(t, err)
}

func TestTokenResolvesOnEveryCall(t *testing.T) {
	src := &fakeSource{token: "tok", botName: "bot"}
	c := NewWithSource(src, noConfig)

	oc, err := c.ForOwner("acme")
	require.NoError(t, err)

	tokenFn := oc.Token()
	assert.Equal(t, "tok", string(tokenFn()))
	src.token = "rotated"
	assert.Equal(t, "rotated", string(tokenFn()))
}

func TestCloneUserIsPlaceholderForAppTokens(t *testing.T) {
	app, err := NewWithSource(&fakeSource{token: "tok", botName: "app[bot]", appToken: true}, noConfig).ForOwner("acme")
	require.NoError(t, err)
	assert.Equal(t, util.GitHubAppGitRemoteUsername, app.CloneUser)

	plain, err := NewWithSource(&fakeSource{token: "tok", botName: "bot"}, noConfig).ForOwner("acme")
	require.NoError(t, err)
	assert.Equal(t, "bot", plain.CloneUser)
}

func TestOptionsOverrideConfig(t *testing.T) {
	c := NewWithSource(&fakeSource{token: "tok", botName: "bot"}, noConfig,
		WithServerURL("https://ghe.example.com"),
		WithGitKind("gitea"),
		WithBotName("flagbot"))

	assert.Equal(t, "https://ghe.example.com", c.ServerURL())
	assert.Equal(t, "gitea", c.GitKind())

	botName, err := c.BotName()
	require.NoError(t, err)
	assert.Equal(t, "flagbot", botName)
}

func TestEmptyOptionsKeepConfiguredValues(t *testing.T) {
	c := NewWithSource(&fakeSource{token: "tok", botName: "bot"}, noConfig,
		WithServerURL(""), WithGitKind(""), WithBotName(""))

	assert.Equal(t, "https://github.com", c.ServerURL())
	assert.Equal(t, "github", c.GitKind())

	botName, err := c.BotName()
	require.NoError(t, err)
	assert.Equal(t, "bot", botName)
}

func TestBrowsersAreBuiltOnceAndCleaned(t *testing.T) {
	c := NewWithSource(&fakeSource{token: "tok", botName: "bot"}, noConfig)

	oc, err := c.ForOwner("acme")
	require.NoError(t, err)

	browsers, err := oc.Browsers()
	require.NoError(t, err)
	again, err := oc.Browsers()
	require.NoError(t, err)
	assert.Same(t, browsers, again)

	browser, err := oc.Browser()
	require.NoError(t, err)
	assert.Same(t, oc.browser, browser)

	assert.NoError(t, c.Clean())
}
