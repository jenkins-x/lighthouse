package gittest

import (
	"strings"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/stretchr/testify/require"

	"github.com/sirupsen/logrus"
)

// GetDefaultBranch gets the default branch name for tests
func GetDefaultBranch(t *testing.T) string {
	logger := logrus.WithField("client", "git")
	censor := func(content []byte) []byte { return content }

	executor, err := git.NewCensoringExecutor(".", censor, logger)
	require.NoError(t, err, "failed to find git binary")

	defaultBranch := "master"
	out, err := executor.Run("config", "--global", "--get", "init.defaultBranch")
	if err == nil {
		text := strings.TrimSpace(string(out))
		if text != "" {
			defaultBranch = text
		}
	}
	t.Logf("using default branch: %s\n", defaultBranch)
	return defaultBranch
}
