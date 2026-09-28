package scmauth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strconv"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/pkg/errors"
)

const (
	// jwtTTL stays under GitHub's 10 minute maximum.
	jwtTTL = 9 * time.Minute

	// jwtClockSkew backdates iat because GitHub rejects tokens issued in the future.
	jwtClockSkew = 60 * time.Second
)

// ParsePrivateKey decodes a PEM-encoded RSA private key in PKCS#1 or PKCS#8 form.
func ParsePrivateKey(keyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("no PEM block found in private key")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "private key is neither PKCS#1 nor PKCS#8")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.Errorf("private key is %T, want *rsa.PrivateKey", parsed)
	}
	return key, nil
}

func signJWT(appID int64, key *rsa.PrivateKey, now time.Time) (string, error) {
	issued := now.Add(-jwtClockSkew)
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]interface{}{
		"iat": issued.Unix(),
		"exp": issued.Add(jwtTTL).Unix(),
		"iss": strconv.FormatInt(appID, 10),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", errors.Wrap(err, "marshalling JWT header")
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", errors.Wrap(err, "marshalling JWT claims")
	}

	enc := base64.RawURLEncoding
	signingInput := enc.EncodeToString(headerJSON) + "." + enc.EncodeToString(claimsJSON)

	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", errors.Wrap(err, "signing JWT")
	}
	return signingInput + "." + enc.EncodeToString(signature), nil
}

type jwtSource struct {
	appID int64
	key   *rsa.PrivateKey
}

func (s *jwtSource) Token(context.Context) (*scm.Token, error) {
	assertion, err := signJWT(s.appID, s.key, time.Now())
	if err != nil {
		return nil, err
	}
	return &scm.Token{Token: assertion}, nil
}
