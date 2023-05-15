package inrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = false

	// disabledTests lets us disable one or more tests when refactoring
	disabledTests = []string{}
)

func TestLoadPipelineRunTest(t *testing.T) {
	sourceDir := filepath.Join("test_data", "load_pipelinerun")
	fs, err := os.ReadDir(sourceDir)
	require.NoError(t, err, "failed to read source Dir %s", sourceDir)

	// lets use a custom version stream sha
	os.Setenv("LIGHTHOUSE_VERSIONSTREAM_JENKINS_X_JX3_PIPELINE_CATALOG", "myversionstreamref")

	require.NoError(t, err, "failed to get cwd")

	// make it easy to run a specific test only
	runTestName := os.Getenv("TEST_NAME")
	for _, f := range fs {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		ignoreTest := false
		for _, ignore := range disabledTests {
			if name == ignore {
				ignoreTest = true
				break
			}
		}
		if ignoreTest {
			t.Logf("ignoring test %s\n", name)
			continue
		}
		if strings.HasPrefix(name, ".") {
			continue
		}
		if runTestName != "" && runTestName != name {
			t.Logf("ignoring test %s\n", name)
			continue
		}

		require.NoError(t, err, "failed to create filebrowsers")

		dir := filepath.Join(sourceDir, name)

		i := 0
		for i <= 10 {
			i++

			suffix := ""
			if i > 1 {
				suffix = fmt.Sprintf("%v", i)
			}
			path := filepath.Join(dir, fmt.Sprintf("source%s.yaml", suffix))

			exists, err := util.FileExists(path)
			require.NoError(t, err, "failed to check for file exists source "+path)

			if !exists && i > 1 {
				break
			}

			expectedPath := filepath.Join(dir, fmt.Sprintf("expected%s.yaml", suffix))

			message := "load file " + path
			data, err := os.ReadFile(path)
			require.NoError(t, err, "failed to load "+message)

			pr, err := LoadTektonResourceAsPipelineRun(data, "myorg", "myRepo", "someSha")

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
				err = os.WriteFile(expectedPath, data, 0666)
				require.NoError(t, err, "failed to save file %s", expectedPath)
				continue
			}
			expectedData, err := os.ReadFile(expectedPath)
			require.NoError(t, err, "failed to load file "+expectedPath)

			text := strings.TrimSpace(string(data))
			expectedText := strings.TrimSpace(string(expectedData))

			assert.Equal(t, expectedText, text, "PipelineRun loaded for "+message)
		}
	}
}
