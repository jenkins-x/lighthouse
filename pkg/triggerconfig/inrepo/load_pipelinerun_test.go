package inrepo

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = true
)

func TestLoadPipelineRunTest(t *testing.T) {
	sourceDir := filepath.Join("test_data", "load_pipelinerun")
	fs, err := ioutil.ReadDir(sourceDir)
	require.NoError(t, err, "failed to read source dir %s", sourceDir)

	getData := func(path string) ([]byte, error) {
		return ioutil.ReadFile(path)
	}
	for _, f := range fs {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		dir := filepath.Join(sourceDir, name)
		path := filepath.Join(dir, "source.yaml")
		expectedPath := filepath.Join(dir, "expected.yaml")

		message := "load file " + path
		data, err := ioutil.ReadFile(path)
		require.NoError(t, err, "failed to load "+message)

		pr, err := LoadTektonResourceAsPipelineRun(data, dir, message, getData, nil)
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
