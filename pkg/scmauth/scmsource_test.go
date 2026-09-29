package scmauth

import (
	"context"
	"net/http"
	"testing"

	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/go-scm/scm/transport"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingTransport records the request it was asked to send instead of sending it.
type capturingTransport struct {
	got *http.Request
}

func (c *capturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.got = req
	return &http.Response{
		StatusCode: 200,
		Body:       http.NoBody,
		Header:     http.Header{},
		Request:    req,
	}, nil
}

// rotatingSource returns a different token on each call.
type rotatingSource struct {
	calls  int
	owners []string
}

func (r *rotatingSource) Token(owner string) (string, error) {
	r.calls++
	r.owners = append(r.owners, owner)
	return "token-" + string(rune('0'+r.calls)), nil
}
func (r *rotatingSource) BotName() (string, error)       { return "bot", nil }
func (r *rotatingSource) RequiresOwner() bool            { return false }
func (r *rotatingSource) UsesAppInstallationToken() bool { return false }

type failingSource struct{}

func (failingSource) Token(string) (string, error) {
	return "", errors.New("installation revoked")
}
func (failingSource) BotName() (string, error)       { return "", nil }
func (failingSource) RequiresOwner() bool            { return false }
func (failingSource) UsesAppInstallationToken() bool { return false }

func TestForOwnerResolvesPerCall(t *testing.T) {
	src := &rotatingSource{}
	adapted := ForOwner(src, "myorg")

	for i := 1; i <= 3; i++ {
		tok, err := adapted.Token(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "token-"+string(rune('0'+i)), tok.Token)
	}

	assert.Equal(t, 3, src.calls, "the credential must be resolved on every call")
	assert.Equal(t, []string{"myorg", "myorg", "myorg"}, src.owners,
		"the bound owner must be used for every lookup")
}

func TestForOwnerPropagatesFailure(t *testing.T) {
	_, err := ForOwner(failingSource{}, "myorg").Token(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "installation revoked")
}

func TestFactoryReResolvesPerRequest(t *testing.T) {
	src := &rotatingSource{}
	client, err := factory.NewClientWithTokenSource("github", "https://github.com",
		ForOwner(src, "myorg"))
	require.NoError(t, err)

	capture := &capturingTransport{}
	require.NotNil(t, client.Client)
	auth, ok := client.Client.Transport.(*transport.Auth)
	require.True(t, ok, "the factory should install its token-source transport")
	auth.Base = capture

	for i := 0; i < 2; i++ {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
			"https://api.github.com/user", nil)
		require.NoError(t, err)
		_, err = client.Client.Transport.RoundTrip(req)
		require.NoError(t, err)
	}

	assert.Equal(t, 2, src.calls)
	assert.Equal(t, "Bearer token-2", capture.got.Header.Get("Authorization"))
}
