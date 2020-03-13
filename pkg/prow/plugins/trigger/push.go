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
	"github.com/jenkins-x/lighthouse/pkg/prow/config"
	"github.com/jenkins-x/lighthouse/pkg/prow/gitprovider"
	"github.com/jenkins-x/lighthouse/pkg/prow/pjutil"
)

func listPushEventChanges(pe scm.PushHook) config.ChangedFilesProvider {
	return func() ([]string, error) {
		changed := make(map[string]bool)
		for _, commit := range pe.Commits {
			for _, added := range commit.Added {
				changed[added] = true
			}
			for _, removed := range commit.Removed {
				changed[removed] = true
			}
			for _, modified := range commit.Modified {
				changed[modified] = true
			}
		}
		var changedFiles []string
		for file := range changed {
			changedFiles = append(changedFiles, file)
		}
		return changedFiles, nil
	}
}

func createRefs(pe *scm.PushHook) v1alpha1.Refs {
	branch := gitprovider.PushHookBranch(pe)
	return v1alpha1.Refs{
		Org:      pe.Repo.Namespace,
		Repo:     pe.Repo.Name,
		BaseRef:  branch,
		BaseSHA:  pe.After,
		BaseLink: pe.Compare,
	}
}

func handlePE(c Client, pe scm.PushHook) error {
	if pe.Deleted {
		// we should not trigger jobs for a branch deletion
		return nil
	}
	for _, j := range c.Config.GetPostsubmits(pe.Repo) {
		branch := gitprovider.PushHookBranch(&pe)
		if shouldRun, err := j.ShouldRun(branch, listPushEventChanges(pe)); err != nil {
			return err
		} else if !shouldRun {
			continue
		}
		refs := createRefs(&pe)
		labels := make(map[string]string)
		for k, v := range j.Labels {
			labels[k] = v
		}
		labels[gitprovider.EventGUID] = pe.GUID
		pj := pjutil.NewLighthouseJob(pjutil.PostsubmitSpec(j, refs), labels, j.Annotations)
		c.Logger.WithFields(pjutil.LighthouseJobFields(&pj)).Info("Creating a new LighthouseJob.")
		if _, err := c.LauncherClient.Launch(&pj, c.MetapipelineClient, pe.Repository()); err != nil {
			return err
		}
	}
	return nil
}
