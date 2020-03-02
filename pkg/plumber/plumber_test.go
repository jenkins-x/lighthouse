package plumber_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/go-cmp/cmp"
	v1 "github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
	"github.com/stretchr/testify/require"
)

func TestPipelineActivityConvert(t *testing.T) {
	testcases := []struct {
		name string
	}{
		{
			name: "presubmit",
		},
		{
			name: "batch",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join("test_data", "convert", tc.name, "activity.yaml")
			require.FileExists(t, file)

			expectedFile := filepath.Join("test_data", "convert", tc.name, "expected.yaml")
			require.FileExists(t, expectedFile)

			data, err := ioutil.ReadFile(file)
			require.NoError(t, err, "cannot read file %s", file)

			activity := v1.PipelineActivity{}
			err = yaml.Unmarshal(data, &activity)
			require.NoError(t, err, "cannot unmarshal YAML in file %s", file)

			data, err = ioutil.ReadFile(expectedFile)
			require.NoError(t, err, "cannot read file %s", expectedFile)

			expected := plumber.PipelineOptions{}
			err = yaml.Unmarshal(data, &expected)
			require.NoError(t, err, "cannot unmarshal YAML in file %s", expectedFile)

			actual := plumber.ToPipelineOptions(&activity)

			data, err = yaml.Marshal(&actual)
			require.NoError(t, err, "failed to marshal PipelineOption")

			if d := cmp.Diff(&expected, &actual); d != "" {
				t.Errorf("Generated PipelineOptions did not match expected: \n%s", d)
			}
		})
	}
}
