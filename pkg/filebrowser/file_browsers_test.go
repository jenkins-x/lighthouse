package filebrowser_test

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/stretchr/testify/require"
)

func TestFileBrowsers(t *testing.T) {
	cf, err := git.NewClientFactory()
	require.NoError(t, err, "failed to create git client factory")

	testCases := []struct {
		serverURL string
	}{
		{
			serverURL: "https://something.com",
		},
		{
			serverURL: "https://github.com",
		},
		{
			serverURL: "http://gitlab.something.com",
		},
	}

	for _, tc := range testCases {
		fb, err := filebrowser.NewFileBrowsers(tc.serverURL, filebrowser.NewFileBrowserFromGitClient(cf))
		require.NoError(t, err, "failed to create file browser for  %s", tc.serverURL)
		require.NotNil(t, fb.LighthouseGitFileBrowser(), "failed to get default file browser for server %s", tc.serverURL)

		names := []string{filebrowser.Lighthouse, filebrowser.GitHub}
		for _, name := range names {
			require.NotNil(t, fb.GetFileBrowser(name), "failed to get file browser for server name %s", name)
		}
	}
}
