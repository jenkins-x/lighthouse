package merge_test

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/merge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombineTriggerConfig(t *testing.T) {
	v1 := triggerconfig.Config{
		Spec: triggerconfig.ConfigSpec{
			Presubmits: []job.Presubmit{
				{
					Base: job.Base{
						Name: "lint",
					},
					AlwaysRun:    true,
					Optional:     false,
					Trigger:      "/lint",
					RerunCommand: "/relint",
					Reporter: job.Reporter{
						Context: "lint",
					},
				},
			},
			Postsubmits: []job.Postsubmit{
				{
					Base: job.Base{
						Name: "release",
					},
					Reporter: job.Reporter{
						Context: "release",
					},
				},
			},
		},
	}
	v2 := triggerconfig.Config{
		Spec: triggerconfig.ConfigSpec{
			Presubmits: []job.Presubmit{
				{
					Base: job.Base{
						Name: "another",
					},
					AlwaysRun:    true,
					Optional:     false,
					Trigger:      "/another",
					RerunCommand: "/reanother",
					Reporter: job.Reporter{
						Context: "another",
					},
				},
			},
		},
	}

	testCases := []struct {
		name                string
		r1                  *triggerconfig.Config
		r2                  *triggerconfig.Config
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
