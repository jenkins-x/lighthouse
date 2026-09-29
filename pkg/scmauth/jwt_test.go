package scmauth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testKeyOnce sync.Once
	testKey     *rsa.PrivateKey
)

// sharedTestKey is generated once because 2048-bit generation is slow.
func sharedTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	testKeyOnce.Do(func() {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		testKey = key
	})
	return testKey
}

func pkcs1PEM(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func pkcs8PEM(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func TestParsePrivateKey(t *testing.T) {
	key := sharedTestKey(t)

	tests := map[string]struct {
		input   []byte
		wantErr string
	}{
		"pkcs1 as issued by github": {input: pkcs1PEM(t, key)},
		"pkcs8 after conversion":    {input: pkcs8PEM(t, key)},
		"not pem":                   {input: []byte("definitely-not-a-key"), wantErr: "no PEM block"},
		"empty":                     {input: nil, wantErr: "no PEM block"},
		"pem but not a key": {
			input:   pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("nope")}),
			wantErr: "neither PKCS#1 nor PKCS#8",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParsePrivateKey(test.input)
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.True(t, got.Equal(key), "parsed key should match the original")
		})
	}
}

func TestSignJWTClaims(t *testing.T) {
	key := sharedTestKey(t)
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)

	token, err := signJWT(4242, key, now)
	require.NoError(t, err)

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3, "a JWT has three dot-separated parts")

	var header map[string]string
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(headerJSON, &header))
	assert.Equal(t, "RS256", header["alg"], "GitHub only accepts RS256")

	var claims map[string]interface{}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(claimsJSON, &claims))

	assert.Equal(t, "4242", claims["iss"], "iss must be the app ID")

	iat := int64(claims["iat"].(float64))
	exp := int64(claims["exp"].(float64))

	assert.Equal(t, now.Add(-jwtClockSkew).Unix(), iat)
	assert.Less(t, iat, now.Unix())

	assert.LessOrEqual(t, exp-iat, int64(600), "JWT lifetime must stay within GitHub's 10 minute limit")
	assert.Greater(t, exp, now.Unix(), "token must still be valid at the current time")
}

func TestSignJWTSignatureVerifies(t *testing.T) {
	key := sharedTestKey(t)

	token, err := signJWT(1, key, time.Now())
	require.NoError(t, err)

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	require.NoError(t, err)

	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	err = rsa.VerifyPKCS1v15(&key.PublicKey, crypto.SHA256, digest[:], signature)
	assert.NoError(t, err, "signature must verify against the app's public key")
}
