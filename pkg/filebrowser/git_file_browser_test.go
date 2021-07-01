package filebrowser

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitFileBrowser(t *testing.T) {
	const verbose = false

	cf, err := git.NewClientFactory()
	require.NoError(t, err, "failed to create git client factory")
	fb := NewFileBrowserFromGitClient(cf)

	fc := NewFetchCache()

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

	files, err := fb.ListFiles(owner, repo, path, ref, fc)
	require.NoError(t, err, "failed to list files "+message())
	assert.NotEmpty(t, files, "should not be empty")
	for _, f := range files {
		t.Logf("file %s type %s\n", f.Name, f.Type)
	}

	data, err := fb.GetFile(owner, repo, fileName, ref, fc)
	require.NoError(t, err, "failed to get file "+fileMessage())
	text := string(data)

	if verbose {
		t.Logf("loaded file %s content: %s\n", path, text)
	}
	require.Contains(t, text, "Apache", message())

	// switch the main branch
	ref = ""
	fileName = "package.json"

	data, err = fb.GetFile(owner, repo, fileName, ref, fc)
	require.NoError(t, err, "failed to get file "+fileMessage())
	text = string(data)

	t.Logf("loaded file %s content: %s\n", path, text)
	require.Contains(t, text, "server.js", message())

	// switch back to a old sha
	ref = "5067522b7ed292bef46570ffe3ab75d3a5428769"
	files, err = fb.ListFiles(owner, repo, path, ref, fc)
	require.NoError(t, err, "failed to list files "+message())
	assert.NotEmpty(t, files, "should not be empty")
	for _, f := range files {
		t.Logf("file %s type %s\n", f.Name, f.Type)
	}

	assertNoScmFileExists(t, files, fileName, message())

	ref = ""
	files, err = fb.ListFiles(owner, repo, path, ref, fc)
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

func TestGitFileBrowser_Clone_CreateTag_FetchRef(t *testing.T) {
	logger := logrus.WithField("client", "git")

	baseDir, err := ioutil.TempDir("", "localdir")
	require.NoError(t, err, "failed to find git binary")
	fmt.Println(baseDir)

	fc := NewFetchCache()

	defer os.RemoveAll(baseDir)

	repoDir := filepath.Join(baseDir, "org", "repo")

	err = os.MkdirAll(repoDir, 0700)
	require.NoError(t, err, "failed to make repo dir")

	userGetter := func() (name, email string, err error) {
		return "bot", "bot@example.com", nil
	}
	censor := func(content []byte) []byte { return content }

	executor, err := git.NewCensoringExecutor(repoDir, censor, logger)
	require.NoError(t, err, "failed to find git binary")

	err = ioutil.WriteFile(filepath.Join(repoDir, "README.md"), []byte("README"), 0600)
	require.NoError(t, err, "failed to write README.md file")

	_, err = executor.Run("init", ".")
	require.NoError(t, err, "failed to init git repo")

	_, err = executor.Run("add", "README.md")
	require.NoError(t, err, "failed to add README.md status")

	_, err = executor.Run("commit", "-m", "add README.md")
	require.NoError(t, err, "failed to add README.md status")

	cf, err := git.NewLocalClientFactory(baseDir, userGetter, censor)
	require.NoError(t, err, "failed to create git client factory")
	fb := NewFileBrowserFromGitClient(cf)

	files, err := fb.ListFiles("org", "repo", "", "master", fc)
	require.NoError(t, err, "failed to list files")

	require.True(t, len(files) == 1, "exepecting 1 file")
	require.Equal(t, files[0].Name, "README.md")

	_, err = executor.Run("checkout", "-b", "update-1")
	require.NoError(t, err, "failed to create new branch")

	err = ioutil.WriteFile(filepath.Join(repoDir, "README.md"), []byte("README-update-1"), 0600)
	require.NoError(t, err, "failed write updated README.md")

	_, err = executor.Run("commit", "-a", "-m", "update README.md")
	require.NoError(t, err, "failed to update README.md status")

	_, err = executor.Run("tag", "v0.0.1")
	require.NoError(t, err, "failed to create v0.0.1 tag")

	files, err = fb.ListFiles("org", "repo", "", "v0.0.1", fc)
	require.NoError(t, err, "failed to lst files in v0.0.1 tag")
	require.True(t, len(files) == 1, "exepecting 1 file")
	require.Equal(t, files[0].Name, "README.md")

	data, err := fb.GetFile("org", "repo", "README.md", "v0.0.1", fc)
	require.NoError(t, err, "failed to lst files in v0.0.1 tag")
	require.Equal(t, string(data), "README-update-1")
}
