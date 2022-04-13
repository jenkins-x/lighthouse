package filebrowser_test

import (
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/stretchr/testify/assert"
)

func TestFetchCache(t *testing.T) {
	c := filebrowser.NewFetchCache()

	testCases := []struct {
		owner, repo, sha string
		expected         bool
	}{
		{
			owner:    "myowner",
			repo:     "myrepo",
			sha:      "main",
			expected: true,
		},
		{
			owner:    "myowner",
			repo:     "myrepo",
			sha:      "main",
			expected: false,
		},
		{
			owner:    "myowner",
			repo:     "myrepo",
			sha:      "a1234",
			expected: true,
		},
	}

	for _, tc := range testCases {
		fullName := scm.Join(tc.owner, tc.repo)
		got := c.ShouldFetch(fullName, tc.sha)
		assert.Equal(t, tc.expected, got, "for repo %s and sha %s", fullName, tc.sha)
	}
}
