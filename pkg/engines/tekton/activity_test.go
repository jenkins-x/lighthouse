package tekton_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/engines/tekton"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// testLogger returns a discarded logrus entry suitable for unit tests.
func testLogger() *logrus.Entry {
	l := logrus.New()
	l.SetOutput(os.Stderr)
	l.SetLevel(logrus.WarnLevel)
	return logrus.NewEntry(l)
}

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

			taskRuns := loadTaskRuns(t, testDir)
			assert.NotEmpty(t, taskRuns, "TaskRuns should not be empty")

			tektonfakeClient := tektonfake.NewSimpleClientset()
			for _, taskRun := range taskRuns {
				_, err := tektonfakeClient.TektonV1().TaskRuns(ns).Create(context.Background(), taskRun, metav1.CreateOptions{})
				assert.NoError(t, err, "Failed to create TaskRun %s in the fake client", taskRun.Name)
			}

			converted, err := tekton.ConvertPipelineRun(context.Background(), testLogger(), tektonfakeClient, pr, ns)
			assert.NoError(t, err)
			expected := loadRecord(t, testDir)

			if d := cmp.Diff(expected, converted); d != "" {
				t.Errorf("Converted PipelineRun record did not match expected record:\n%s", d)
			}
		})
	}
}

func TestConvertPipelineRunWithCustomRuns(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "mixed_taskrun_and_customrun_success",
		},
		{
			name: "customrun_failed",
		},
		{
			name: "customrun_running",
		},
		{
			name: "customrun_no_status",
		},
		{
			name: "unknown_child_kind_ignored",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testDir := filepath.Join("test_data", "activity", tc.name)
			pr := loadPipelineRun(t, testDir)
			ns := "jx"

			tektonfakeClient := tektonfake.NewSimpleClientset()

			taskRuns := loadTaskRuns(t, testDir)
			for _, taskRun := range taskRuns {
				_, err := tektonfakeClient.TektonV1().TaskRuns(ns).Create(context.Background(), taskRun, metav1.CreateOptions{})
				assert.NoError(t, err, "Failed to create TaskRun %s in the fake client", taskRun.Name)
			}

			customRuns := loadCustomRuns(t, testDir)
			for _, customRun := range customRuns {
				_, err := tektonfakeClient.TektonV1beta1().CustomRuns(ns).Create(context.Background(), customRun, metav1.CreateOptions{})
				assert.NoError(t, err, "Failed to create CustomRun %s in the fake client", customRun.Name)
			}

			converted, err := tekton.ConvertPipelineRun(context.Background(), testLogger(), tektonfakeClient, pr, ns)
			assert.NoError(t, err)
			expected := loadRecord(t, testDir)

			if d := cmp.Diff(expected, converted); d != "" {
				t.Errorf("Converted PipelineRun record did not match expected record:\n%s", d)
			}
		})
	}
}

func loadCustomRuns(t *testing.T, dir string) []*tektonv1beta1.CustomRun {
	files, err := filepath.Glob(filepath.Join(dir, "cr*.yaml"))
	if assert.NoError(t, err, "Failed to list files in directory %s", dir) {
		var customRuns []*tektonv1beta1.CustomRun
		for _, fileName := range files {
			data, err := os.ReadFile(fileName)
			if assert.NoError(t, err, "Failed to load file %s", fileName) {
				customRun := &tektonv1beta1.CustomRun{}
				err = yaml.Unmarshal(data, customRun)
				if assert.NoError(t, err, "Failed to unmarshal YAML file %s", fileName) {
					customRuns = append(customRuns, customRun)
				}
			}
		}
		return customRuns
	}
	return nil
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

func loadTaskRuns(t *testing.T, dir string) []*pipelinev1.TaskRun {
	files, err := filepath.Glob(filepath.Join(dir, "tr*.yaml"))
	if assert.NoError(t, err, "Failed to list files in directory %s", dir) {
		var taskRuns []*pipelinev1.TaskRun
		for _, fileName := range files {
			data, err := os.ReadFile(fileName)
			if assert.NoError(t, err, "Failed to load file %s", fileName) {
				taskRun := &pipelinev1.TaskRun{}
				err = yaml.Unmarshal(data, taskRun)
				if assert.NoError(t, err, "Failed to unmarshal YAML file %s", fileName) {
					taskRuns = append(taskRuns, taskRun)
				}
			}
		}
		return taskRuns
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
