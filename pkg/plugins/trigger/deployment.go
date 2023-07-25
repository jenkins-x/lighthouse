package trigger

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
)

func handleDeployment(c Client, ds scm.DeploymentStatusHook) error {
	for _, j := range c.Config.GetDeployments(ds.Repo) {
		if j.State != "" && j.State != ds.DeploymentStatus.State {
			continue
		}
		if j.Environment != "" && j.Environment != ds.Deployment.Environment {
			continue
		}
		labels := make(map[string]string)
		for k, v := range j.Labels {
			labels[k] = v
		}
		refs := v1alpha1.Refs{
			Org:      ds.Repo.Namespace,
			Repo:     ds.Repo.Name,
			BaseRef:  ds.Deployment.Ref,
			BaseSHA:  ds.Deployment.Sha,
			BaseLink: ds.Deployment.RepositoryLink,
			CloneURI: ds.Repo.Clone,
		}
		labels[scmprovider.EventGUID] = ds.DeploymentStatus.ID
		pj := jobutil.NewLighthouseJob(jobutil.DeploymentSpec(c.Logger, j, refs), labels, j.Annotations)
		c.Logger.WithFields(jobutil.LighthouseJobFields(&pj)).Info("Creating a new LighthouseJob.")
		if _, err := c.LauncherClient.Launch(&pj); err != nil {
			return err
		}

	}
	return nil
}
