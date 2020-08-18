package merge_test

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/repoconfig"
	"github.com/jenkins-x/lighthouse/pkg/repoconfig/merge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineRepositoryConfig(t *testing.T) {
	v1 := repoconfig.RepositoryConfig{
		Spec: repoconfig.RepositoryConfigSpec{
			Presubmits: []repoconfig.Presubmit{
				{
					JobBase: repoconfig.JobBase{
						Name: "lint",
					},
					AlwaysRun:    true,
					Optional:     false,
					Trigger:      "/lint",
					RerunCommand: "/relint",
					Reporter: repoconfig.Reporter{
						Context: "lint",
					},
				},
			},
			Postsubmits: []repoconfig.Postsubmit{
				{
					JobBase: repoconfig.JobBase{
						Name: "release",
					},
					Reporter: repoconfig.Reporter{
						Context: "release",
					},
				},
			},
		},
	}
	v2 := repoconfig.RepositoryConfig{
		Spec: repoconfig.RepositoryConfigSpec{
			Presubmits: []repoconfig.Presubmit{
				{
					JobBase: repoconfig.JobBase{
						Name: "another",
					},
					AlwaysRun:    true,
					Optional:     false,
					Trigger:      "/another",
					RerunCommand: "/reanother",
					Reporter: repoconfig.Reporter{
						Context: "another",
					},
				},
			},
		},
	}

	testCases := []struct {
		name                string
		r1                  *repoconfig.RepositoryConfig
		r2                  *repoconfig.RepositoryConfig
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
