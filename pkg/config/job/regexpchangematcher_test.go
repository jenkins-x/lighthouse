/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package job_test

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexChangeMatcher(t *testing.T) {
	testCases := []struct {
		matcher  job.RegexpChangeMatcher
		changes  []string
		expected bool
	}{
		{
			changes:  []string{"cheese.txt"},
			expected: true,
		},
		{
			matcher: job.RegexpChangeMatcher{
				RunIfChanged: `.*\.txt`,
			},
			changes:  []string{"cheese.txt"},
			expected: true,
		},
		{
			matcher: job.RegexpChangeMatcher{
				IgnoreChanges: `.*\.txt`,
			},
			changes:  []string{"cheese.txt"},
			expected: false,
		},
		{
			matcher: job.RegexpChangeMatcher{
				IgnoreChanges: `action\.yml`,
			},
			changes:  []string{"action.yml", "cheese.txt"},
			expected: true,
		},
		{
			matcher: job.RegexpChangeMatcher{
				IgnoreChanges: `action\.yml`,
			},
			changes:  []string{"action.yml"},
			expected: false,
		},
		{
			matcher: job.RegexpChangeMatcher{
				RunIfChanged:  `.*\.txt`,
				IgnoreChanges: `action\.yml`,
			},
			changes:  []string{"cheese.txt"},
			expected: true,
		},
		{
			matcher: job.RegexpChangeMatcher{
				RunIfChanged:  `.*\.txt`,
				IgnoreChanges: `action\.yml`,
			},
			changes:  []string{"cheese.txt", "action.yml"},
			expected: true,
		},
		{
			matcher: job.RegexpChangeMatcher{
				RunIfChanged:  `.*\.txt`,
				IgnoreChanges: `action\.yml`,
			},
			changes:  []string{"action.yml"},
			expected: false,
		},
		{
			matcher: job.RegexpChangeMatcher{
				RunIfChanged:  `.*\.txt`,
				IgnoreChanges: `action\.yml`,
			},
			changes:  []string{"action.yml", "not.text-file.cheese"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		changes := func() ([]string, error) {
			return tc.changes, nil
		}
		determined, got, err := tc.matcher.ShouldRun(changes)
		if !determined {
			got = true
		}
		require.NoError(t, err, "should not have an error invoking matcher %#v", tc.matcher)
		assert.Equal(t, tc.expected, got, "failed to match invoking matcher %#v", tc.matcher)
	}
}
