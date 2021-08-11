package util_test

import (
	"testing"

	"github.com/jenkins-x/go-scm/pkg/hmac"

	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestCreateHMACHeader(t *testing.T) {
	data := []byte(`{ "text": "hello world" }`)

	key := "5c50537bc2a8cb57ad578ba90318392fd71cfa44c"
	header := util.CreateHMACHeader(data, key)

	assert.True(t, hmac.ValidatePrefix(data, []byte(key), header), "failed to validate header of %s", header)

	t.Logf("got generated header %s\n", header)
}
