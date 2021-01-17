package filebrowser

import (
	"fmt"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitFileBrowser(t *testing.T) {
	const verbose = false

	cf, err := git.NewClientFactory()
	require.NoError(t, err, "failed to create git client factory")
	fb := NewFileBrowserFromGitClient(cf)

	owner := "jenkins-x-quickstarts"
	repo := "node-http"
	path := "/"
	ref := "5067522b7ed292bef46570ffe3ab75d3a5428769"
	fileName := "LICENSE"

	message := func() string {
		return fmt.Sprintf("for %s/%s path %s ref %s", owner, repo, path, ref)
	}

	fileMessage := func() string {
		return message() + " file " + fileName
	}

	files, err := fb.ListFiles(owner, repo, path, ref)
	require.NoError(t, err, "failed to list files "+message())
	assert.NotEmpty(t, files, "should not be empty")
	for _, f := range files {
		t.Logf("file %s type %s\n", f.Name, f.Type)
	}

	data, err := fb.GetFile(owner, repo, fileName, ref)
	require.NoError(t, err, "failed to get file "+fileMessage())
	text := string(data)

	if verbose {
		t.Logf("loaded file %s content: %s\n", path, text)
	}
	require.Contains(t, text, "Apache", message())

	// switch the main branch
	ref = ""
	fileName = "package.json"

	data, err = fb.GetFile(owner, repo, fileName, ref)
	require.NoError(t, err, "failed to get file "+fileMessage())
	text = string(data)

	t.Logf("loaded file %s content: %s\n", path, text)
	require.Contains(t, text, "server.js", message())

	// switch back to a old sha
	ref = "5067522b7ed292bef46570ffe3ab75d3a5428769"
	files, err = fb.ListFiles(owner, repo, path, ref)
	require.NoError(t, err, "failed to list files "+message())
	assert.NotEmpty(t, files, "should not be empty")
	for _, f := range files {
		t.Logf("file %s type %s\n", f.Name, f.Type)
	}

	assertNoScmFileExists(t, files, fileName, message())

	ref = ""
	files, err = fb.ListFiles(owner, repo, path, ref)
	require.NoError(t, err, "failed to list files "+message())
	assertScmFileExists(t, files, fileName, message())
}

func assertScmFileExists(t *testing.T, files []*scm.FileEntry, name, message string) {
	for _, f := range files {
		if f.Name == name {
			t.Logf("includes expected file name %s for %s", name, message)
			return
		}
	}
	require.Fail(t, "should have found file %s %s", name, message)
}

func assertNoScmFileExists(t *testing.T, files []*scm.FileEntry, name, message string) {
	for _, f := range files {
		if f.Name == name {
			require.Fail(t, "should not have found file %s %s", name, message)
		}
	}
	t.Logf("correctly does not include file name %s for %s", name, message)
}
