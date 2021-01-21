package inrepo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = false
)

func TestLoadPipelineRunTest(t *testing.T) {
	sourceDir := filepath.Join("test_data", "load_pipelinerun")
	fs, err := ioutil.ReadDir(sourceDir)
	require.NoError(t, err, "failed to read source Dir %s", sourceDir)

	scmClient, _ := fake.NewDefault()
	scmProvider := scmprovider.ToClient(scmClient, "my-bot")

	resolver := &UsesResolver{
		Client:           filebrowser.NewFileBrowserFromScmClient(scmProvider),
		OwnerName:        "myorg",
		LocalFileResolve: true,
	}

	// lets use a custom version stream sha
	os.Setenv("LIGHTHOUSE_VERSIONSTREAM_JENKINS_X_JX3_PIPELINE_CATALOG", "myversionstreamref")

	// make it easy to run a specific test only
	runTestName := os.Getenv("TEST_NAME")
	for _, f := range fs {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if runTestName != "" && runTestName != name {
			t.Logf("ignoring test %s\n", name)
			continue
		}
		dir := filepath.Join(sourceDir, name)
		resolver.Dir = dir

		path := filepath.Join(dir, "source.yaml")
		expectedPath := filepath.Join(dir, "expected.yaml")

		message := "load file " + path
		data, err := ioutil.ReadFile(path)
		require.NoError(t, err, "failed to load "+message)

		pr, err := LoadTektonResourceAsPipelineRun(resolver, data)
		if strings.HasSuffix(name, "-fails") {
			require.Errorf(t, err, "expected failure for test %s", name)
			t.Logf("test %s generated expected error %s\n", name, err.Error())
			continue
		}

		require.NoError(t, err, "failed to load PipelineRun for "+message)
		require.NotNil(t, pr, "no PipelineRun for "+message)

		data, err = yaml.Marshal(pr)
		require.NoError(t, err, "failed to marshal generated PipelineRun for "+message)

		if generateTestOutput {
			err = ioutil.WriteFile(expectedPath, data, 0666)
			require.NoError(t, err, "failed to save file %s", expectedPath)
			continue
		}
		expectedData, err := ioutil.ReadFile(expectedPath)
		require.NoError(t, err, "failed to load file "+expectedPath)

		text := strings.TrimSpace(string(data))
		expectedText := strings.TrimSpace(string(expectedData))

		assert.Equal(t, expectedText, text, "PipelineRun loaded for "+message)
	}
}
