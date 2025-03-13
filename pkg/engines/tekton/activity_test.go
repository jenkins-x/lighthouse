package tekton_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/engines/tekton"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/assert"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"sigs.k8s.io/yaml"
)

func TestConvertPipelineRun(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "successful_single_task",
		},
		{
			name: "failed_single_task",
		},
		{
			name: "running_single_task",
		},
		{
			name: "running_multiple_tasks",
		},
		{
			name: "successful_multiple_tasks",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := filepath.Join("test_data", "activity", tc.name)
			pr := loadPipelineRun(t, testDir)
			ns := "jx"

			tektonfakeClient := tektonfake.NewSimpleClientset()
			converted, err := tekton.ConvertPipelineRun(tektonfakeClient, pr, ns)
			assert.NoError(t, err)
			expected := loadRecord(t, testDir)

			if d := cmp.Diff(expected, converted); d != "" {
				t.Errorf("Converted PipelineRun record did not match expected record:\n%s", d)
			}
		})
	}
}

func loadPipelineRun(t *testing.T, dir string) *pipelinev1.PipelineRun {
	fileName := filepath.Join(dir, "pr.yaml")
	if assertFileExists(t, fileName) {
		pr := &pipelinev1.PipelineRun{}
		data, err := os.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, pr)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return pr
			}
		}
	}
	return nil
}

func loadRecord(t *testing.T, dir string) *v1alpha1.ActivityRecord {
	fileName := filepath.Join(dir, "record.yaml")
	if assertFileExists(t, fileName) {
		record := &v1alpha1.ActivityRecord{}
		data, err := os.ReadFile(fileName)
		if assert.NoError(t, err, "Failed to load file %s", fileName) {
			err = yaml.Unmarshal(data, record)
			if assert.NoError(t, err, "Failed to unmarshall YAML file %s", fileName) {
				return record
			}
		}
	}
	return nil
}

// assertFileExists asserts that the given file exists
func assertFileExists(t *testing.T, fileName string) bool {
	exists, err := util.FileExists(fileName)
	assert.NoError(t, err, "Failed checking if file exists %s", fileName)
	assert.True(t, exists, "File %s should exist", fileName)
	return exists
}
