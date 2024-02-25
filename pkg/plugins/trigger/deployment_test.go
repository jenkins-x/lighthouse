package trigger

import (
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/config"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/launcher/fake"
	fake2 "github.com/jenkins-x/lighthouse/pkg/scmprovider/fake"
	"github.com/sirupsen/logrus"
)

func TestHandleDeployment(t *testing.T) {
	testCases := []struct {
		name      string
		dep       *scm.DeploymentStatusHook
		jobsToRun int
	}{
		{
			name: "deploy to production",
			dep: &scm.DeploymentStatusHook{
				Deployment: scm.Deployment{
					Sha:            "df0442b202e6881b88ab6dad774f63459671ebb0",
					Ref:            "v1.0.1",
					Environment:    "Production",
					RepositoryLink: "https://api.github.com/respos/org/repo",
				},
				DeploymentStatus: scm.DeploymentStatus{
					ID:    "123456",
					State: "success",
				},
				Repo: scm.Repository{
					FullName: "org/repo",
					Clone:    "https://github.com/org/repo.git",
				},
			},
			jobsToRun: 1,
		},
		{
			name: "deploy to production",
			dep: &scm.DeploymentStatusHook{
				Deployment: scm.Deployment{
					Sha:            "df0442b202e6881b88ab6dad774f63459671ebb0",
					Ref:            "v1.0.1",
					Environment:    "Production",
					RepositoryLink: "https://api.github.com/respos/org/repo",
				},
				DeploymentStatus: scm.DeploymentStatus{
					ID:    "123456",
					State: "failed",
				},
				Repo: scm.Repository{
					FullName: "org/repo",
					Clone:    "https://github.com/org/repo.git",
				},
			},
			jobsToRun: 0,
		},
		{
			name: "deploy to production",
			dep: &scm.DeploymentStatusHook{
				Deployment: scm.Deployment{
					Sha:            "df0442b202e6881b88ab6dad774f63459671ebb0",
					Ref:            "v1.0.1",
					Environment:    "Staging",
					RepositoryLink: "https://api.github.com/respos/org/repo",
				},
				DeploymentStatus: scm.DeploymentStatus{
					ID:    "123456",
					State: "success",
				},
				Repo: scm.Repository{
					FullName: "org/repo",
					Clone:    "https://github.com/org/repo.git",
				},
			},
			jobsToRun: 0,
		},
	}

	for _, tc := range testCases {
		g := &fake2.SCMClient{}
		fakeLauncher := fake.NewLauncher()
		c := Client{
			SCMProviderClient: g,
			LauncherClient:    fakeLauncher,
			Config:            &config.Config{ProwConfig: config.ProwConfig{LighthouseJobNamespace: "lighthouseJobs"}},
			Logger:            logrus.WithField("plugin", pluginName),
		}
		deployments := map[string][]job.Deployment{
			"org/repo": {
				{
					Base: job.Base{
						Name: "butter-is-served",
					},
					Reporter:    job.Reporter{},
					State:       "success",
					Environment: "production",
				},
			},
		}
		c.Config.Deployments = deployments
		err := handleDeployment(c, *tc.dep)
		if err != nil {
			t.Errorf("test %q: handlePE returned unexpected error %v", tc.name, err)
		}
		var numStarted int
		for _, job := range fakeLauncher.Pipelines {
			t.Logf("created job with context %s", job.Spec.Context)
			numStarted++
		}
		if numStarted != tc.jobsToRun {
			t.Errorf("test %q: expected %d jobs to run, got %d", tc.name, tc.jobsToRun, numStarted)
		}
	}

}
