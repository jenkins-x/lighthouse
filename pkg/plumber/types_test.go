package plumber_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/lighthouse/pkg/plumber"
)

func TestPipelineOptionsSpec_GetEnvVars(t *testing.T) {
	tests := []struct {
		name string
		spec *plumber.PipelineOptionsSpec
		env  map[string]string
	}{
		{
			name: "periodic",
			spec: &plumber.PipelineOptionsSpec{
				Type:      plumber.PeriodicJob,
				Namespace: "jx",
				Job:       "some-job",
			},
			env: map[string]string{
				plumber.JobNameEnv: "some-job",
				plumber.JobTypeEnv: string(plumber.PeriodicJob),
				plumber.JobSpecEnv: fmt.Sprintf("type:%s", plumber.PeriodicJob),
			},
		},
		{
			name: "postsubmit",
			spec: &plumber.PipelineOptionsSpec{
				Type:      plumber.PostsubmitJob,
				Namespace: "jx",
				Job:       "some-release-job",
				Refs: &plumber.Refs{
					Org:     "some-org",
					Repo:    "some-repo",
					BaseRef: "master",
					BaseSHA: "1234abcd",
				},
			},
			env: map[string]string{
				plumber.JobNameEnv:     "some-release-job",
				plumber.JobTypeEnv:     string(plumber.PostsubmitJob),
				plumber.JobSpecEnv:     fmt.Sprintf("type:%s", plumber.PostsubmitJob),
				plumber.RepoNameEnv:    "some-repo",
				plumber.RepoOwnerEnv:   "some-org",
				plumber.PullBaseRefEnv: "master",
				plumber.PullBaseShaEnv: "1234abcd",
				plumber.PullRefsEnv:    "master:1234abcd",
			},
		},
		{
			name: "presubmit",
			spec: &plumber.PipelineOptionsSpec{
				Type:      plumber.PresubmitJob,
				Namespace: "jx",
				Job:       "some-pr-job",
				Refs: &plumber.Refs{
					Org:     "some-org",
					Repo:    "some-repo",
					BaseRef: "master",
					BaseSHA: "1234abcd",
					Pulls: []plumber.Pull{
						{
							Number: 1,
							SHA:    "5678",
						},
					},
				},
			},
			env: map[string]string{
				plumber.JobNameEnv:     "some-pr-job",
				plumber.JobTypeEnv:     string(plumber.PresubmitJob),
				plumber.JobSpecEnv:     fmt.Sprintf("type:%s", plumber.PresubmitJob),
				plumber.RepoNameEnv:    "some-repo",
				plumber.RepoOwnerEnv:   "some-org",
				plumber.PullBaseRefEnv: "master",
				plumber.PullBaseShaEnv: "1234abcd",
				plumber.PullRefsEnv:    "master:1234abcd,1:5678",
				plumber.PullNumberEnv:  "1",
				plumber.PullPullShaEnv: "5678",
			},
		},
		{
			name: "batch",
			spec: &plumber.PipelineOptionsSpec{
				Type:      plumber.BatchJob,
				Namespace: "jx",
				Job:       "some-pr-job",
				Refs: &plumber.Refs{
					Org:     "some-org",
					Repo:    "some-repo",
					BaseRef: "master",
					BaseSHA: "1234abcd",
					Pulls: []plumber.Pull{
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
				plumber.JobNameEnv:     "some-pr-job",
				plumber.JobTypeEnv:     string(plumber.BatchJob),
				plumber.JobSpecEnv:     fmt.Sprintf("type:%s", plumber.BatchJob),
				plumber.RepoNameEnv:    "some-repo",
				plumber.RepoOwnerEnv:   "some-org",
				plumber.PullBaseRefEnv: "master",
				plumber.PullBaseShaEnv: "1234abcd",
				plumber.PullRefsEnv:    "master:1234abcd,1:5678,2:0efg",
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
