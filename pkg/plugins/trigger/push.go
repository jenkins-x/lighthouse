/*
Copyright 2016 The Kubernetes Authors.

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

package trigger

import (
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/jobutil"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
)

func createRefs(pe *scm.PushHook) v1alpha1.Refs {
	branch := scmprovider.PushHookBranch(pe)
	return v1alpha1.Refs{
		Org:      pe.Repo.Namespace,
		Repo:     pe.Repo.Name,
		BaseRef:  branch,
		BaseSHA:  pe.After,
		BaseLink: pe.Compare,
		CloneURI: pe.Repo.Clone,
	}
}

func handlePE(c Client, pe scm.PushHook, trigger *plugins.Trigger) error {
	if pe.Deleted {
		// we should not trigger jobs for a branch deletion
		return nil
	}
	mode, err := trigger.ResolvedPushChangedFiles()
	if err != nil {
		return err
	}
	warn := func(format string, args ...interface{}) {
		c.Logger.Warnf(format, args...)
	}
	changes, err := job.NewPushChangedFilesProvider(mode, c.SCMProviderClient, pe, warn)
	if err != nil {
		return err
	}
	for _, j := range c.Config.GetPostsubmits(pe.Repo) {
		branch := scmprovider.PushHookBranch(&pe)
		if shouldRun, err := j.ShouldRun(branch, changes); err != nil {
			return err
		} else if !shouldRun {
			continue
		}
		refs := createRefs(&pe)
		labels := make(map[string]string)
		for k, v := range j.Labels {
			labels[k] = v
		}
		labels[scmprovider.EventGUID] = pe.GUID
		pj := jobutil.NewLighthouseJob(jobutil.PostsubmitSpec(c.Logger, j, refs), labels, j.Annotations)
		c.Logger.WithFields(jobutil.LighthouseJobFields(&pj)).Info("Creating a new LighthouseJob.")
		if _, err := c.LauncherClient.Launch(&pj); err != nil {
			return err
		}
	}
	return nil
}
