package v1alpha1_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
)

func TestPipelineOptionsSpec_GetEnvVars(t *testing.T) {
	tests := []struct {
		name string
		spec *v1alpha1.LighthouseJobSpec
		env  map[string]string
	}{
		{
			name: "periodic",
			spec: &v1alpha1.LighthouseJobSpec{
				Type:      v1alpha1.PeriodicJob,
				Namespace: "jx",
				Job:       "some-job",
			},
			env: map[string]string{
				v1alpha1.JobNameEnv: "some-job",
				v1alpha1.JobTypeEnv: string(v1alpha1.PeriodicJob),
				v1alpha1.JobSpecEnv: fmt.Sprintf("type:%s", v1alpha1.PeriodicJob),
			},
		},
		{
			name: "postsubmit",
			spec: &v1alpha1.LighthouseJobSpec{
				Type:      v1alpha1.PostsubmitJob,
				Namespace: "jx",
				Job:       "some-release-job",
				Refs: &v1alpha1.Refs{
					Org:     "some-org",
					Repo:    "some-repo",
					BaseRef: "master",
					BaseSHA: "1234abcd",
				},
			},
			env: map[string]string{
				v1alpha1.JobNameEnv:     "some-release-job",
				v1alpha1.JobTypeEnv:     string(v1alpha1.PostsubmitJob),
				v1alpha1.JobSpecEnv:     fmt.Sprintf("type:%s", v1alpha1.PostsubmitJob),
				v1alpha1.RepoNameEnv:    "some-repo",
				v1alpha1.RepoOwnerEnv:   "some-org",
				v1alpha1.PullBaseRefEnv: "master",
				v1alpha1.PullBaseShaEnv: "1234abcd",
				v1alpha1.PullRefsEnv:    "master:1234abcd",
			},
		},
		{
			name: "presubmit",
			spec: &v1alpha1.LighthouseJobSpec{
				Type:      v1alpha1.PresubmitJob,
				Namespace: "jx",
				Job:       "some-pr-job",
				Refs: &v1alpha1.Refs{
					Org:     "some-org",
					Repo:    "some-repo",
					BaseRef: "master",
					BaseSHA: "1234abcd",
					Pulls: []v1alpha1.Pull{
						{
							Number: 1,
							SHA:    "5678",
						},
					},
				},
			},
			env: map[string]string{
				v1alpha1.JobNameEnv:     "some-pr-job",
				v1alpha1.JobTypeEnv:     string(v1alpha1.PresubmitJob),
				v1alpha1.JobSpecEnv:     fmt.Sprintf("type:%s", v1alpha1.PresubmitJob),
				v1alpha1.RepoNameEnv:    "some-repo",
				v1alpha1.RepoOwnerEnv:   "some-org",
				v1alpha1.PullBaseRefEnv: "master",
				v1alpha1.PullBaseShaEnv: "1234abcd",
				v1alpha1.PullRefsEnv:    "master:1234abcd,1:5678",
				v1alpha1.PullNumberEnv:  "1",
				v1alpha1.PullPullShaEnv: "5678",
			},
		},
		{
			name: "batch",
			spec: &v1alpha1.LighthouseJobSpec{
				Type:      v1alpha1.BatchJob,
				Namespace: "jx",
				Job:       "some-pr-job",
				Refs: &v1alpha1.Refs{
					Org:     "some-org",
					Repo:    "some-repo",
					BaseRef: "master",
					BaseSHA: "1234abcd",
					Pulls: []v1alpha1.Pull{
						{
							Number: 1,
							SHA:    "5678",
						},
						{
							Number: 2,
							SHA:    "0efg",
						},
					},
				},
			},
			env: map[string]string{
				v1alpha1.JobNameEnv:     "some-pr-job",
				v1alpha1.JobTypeEnv:     string(v1alpha1.BatchJob),
				v1alpha1.JobSpecEnv:     fmt.Sprintf("type:%s", v1alpha1.BatchJob),
				v1alpha1.RepoNameEnv:    "some-repo",
				v1alpha1.RepoOwnerEnv:   "some-org",
				v1alpha1.PullBaseRefEnv: "master",
				v1alpha1.PullBaseShaEnv: "1234abcd",
				v1alpha1.PullRefsEnv:    "master:1234abcd,1:5678,2:0efg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectedEnv := make(map[string]string)

			for k, v := range tt.env {
				expectedEnv[k] = v
			}

			// In CI, this will be set, but it may not be set locally, so add it if it's in the env.
			registry := os.Getenv("DOCKER_REGISTRY")
			if registry != "" {
				expectedEnv["DOCKER_REGISTRY"] = registry
			}

			generatedEnv := tt.spec.GetEnvVars()

			if d := cmp.Diff(expectedEnv, generatedEnv); d != "" {
				t.Errorf("Generated environment variables did not match expected: %s", d)
			}
		})
	}
}
