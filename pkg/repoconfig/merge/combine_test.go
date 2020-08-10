package merge_test

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/repoconfig/merge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineRepositoryConfig(t *testing.T) {
	v1 := v1alpha1.RepositoryConfig{
		Spec: v1alpha1.RepositoryConfigSpec{
			Presubmits: []v1alpha1.Presubmit{
				{
					JobBase: v1alpha1.JobBase{
						Name: "lint",
					},
					AlwaysRun:    true,
					Optional:     false,
					Trigger:      "/lint",
					RerunCommand: "/relint",
					Reporter: v1alpha1.Reporter{
						Context: "lint",
					},
				},
			},
			Postsubmits: []v1alpha1.Postsubmit{
				{
					JobBase: v1alpha1.JobBase{
						Name: "release",
					},
					Reporter: v1alpha1.Reporter{
						Context: "release",
					},
				},
			},
		},
	}
	v2 := v1alpha1.RepositoryConfig{
		Spec: v1alpha1.RepositoryConfigSpec{
			Presubmits: []v1alpha1.Presubmit{
				{
					JobBase: v1alpha1.JobBase{
						Name: "another",
					},
					AlwaysRun:    true,
					Optional:     false,
					Trigger:      "/another",
					RerunCommand: "/reanother",
					Reporter: v1alpha1.Reporter{
						Context: "another",
					},
				},
			},
		},
	}

	testCases := []struct {
		name                string
		r1                  *v1alpha1.RepositoryConfig
		r2                  *v1alpha1.RepositoryConfig
		expectedPresubmits  int
		expectedPostsubmits int
	}{
		{
			name: "bothNil",
		},
		{
			name:                "r1",
			r1:                  &v1,
			expectedPostsubmits: 1,
			expectedPresubmits:  1,
		},
		{
			name:                "r2",
			r2:                  &v1,
			expectedPostsubmits: 1,
			expectedPresubmits:  1,
		},
		{
			name:                "combine",
			r1:                  &v1,
			r2:                  &v2,
			expectedPostsubmits: 1,
			expectedPresubmits:  2,
		},
	}

	for _, tc := range testCases {
		name := tc.name
		actual := merge.CombineConfigs(tc.r1, tc.r2)

		if tc.r1 == nil && tc.r2 == nil {
			assert.Nil(t, actual, "expectedPresubmits nil results for %s", name)
		} else {
			require.NotNil(t, actual, "nil results for %s", name)

			assert.Len(t, actual.Spec.Presubmits, tc.expectedPresubmits, "expected presubmits for %s", name)
			assert.Len(t, actual.Spec.Postsubmits, tc.expectedPostsubmits, "expected postsubmits for %s", name)
		}
	}
}
