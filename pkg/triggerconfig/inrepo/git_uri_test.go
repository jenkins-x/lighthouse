package inrepo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/inrepo"
)

func TestParseGitURI(t *testing.T) {
	testCases := []struct {
		text         string
		expectedErr  bool
		expected     *inrepo.GitURI
		expectedText string
	}{
		{
			text: "myowner/myrepo@v1",
			expected: &inrepo.GitURI{
				Owner:      "myowner",
				Repository: "myrepo",
				Path:       "",
				SHA:        "v1",
			},
		},
		{
			text: "myowner/myrepo/@v1",
			expected: &inrepo.GitURI{
				Owner:      "myowner",
				Repository: "myrepo",
				Path:       "",
				SHA:        "v1",
			},
			expectedText: "myowner/myrepo@v1",
		},
		{
			text: "myowner/myrepo/myfile.yaml@v1",
			expected: &inrepo.GitURI{
				Owner:      "myowner",
				Repository: "myrepo",
				Path:       "myfile.yaml",
				SHA:        "v1",
			},
		},
		{
			text: "myowner/myrepo/javascript/pullrequest.yaml@v1",
			expected: &inrepo.GitURI{
				Owner:      "myowner",
				Repository: "myrepo",
				Path:       "javascript/pullrequest.yaml",
				SHA:        "v1",
			},
		},
		{
			text:     "foo.yaml",
			expected: nil,
		},
		{
			text:     "foo/bar/thingy.yaml",
			expected: nil,
		},
	}

	for _, tc := range testCases {
		text := tc.text
		gitURI, err := inrepo.ParseGitURI(text)
		if tc.expectedErr {
			require.Error(t, err, "should have failed to parse %s", text)
			t.Logf("parsing %s got expected error: %s\n", text, err.Error())
		} else {
			require.NoError(t, err, "should have failed to parse %s", text)
			assert.Equal(t, tc.expected, gitURI, "when parsing %s", text)

			if tc.expected != nil {
				actual := tc.expected.String()
				expectedText := tc.expectedText
				if expectedText == "" {
					expectedText = text
				}
				assert.Equal(t, expectedText, actual, "generated string for GitURI for parsed %s", text)
			}
		}
	}
}
