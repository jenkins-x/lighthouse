package util_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/tide/githubapp"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOwnerTokensDir(t *testing.T) {
	dir := filepath.Join("test_data", "secret_dir")

	tokenFinder := util.NewOwnerTokensDir(githubapp.GithubServer, dir)

	owner := "arcalos-environments"
	token, err := tokenFinder.FindToken(owner)
	require.NoError(t, err, "failed to find token for owner %s in dir %s", owner, dir)
	assert.Equal(t, "mytoken", token, "token for owner %s in dir %s", owner, dir)
	t.Logf("found token %s for owner %s", token, owner)
}
