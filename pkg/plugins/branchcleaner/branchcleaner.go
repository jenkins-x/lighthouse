/*
Copyright 2018 The Kubernetes Authors.

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

package branchcleaner

import (
	"fmt"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
)

const (
	pluginName = "branchcleaner"
)

func init() {
	plugins.RegisterPlugin(
		pluginName,
		plugins.Plugin{
			Description:        "The branchcleaner plugin automatically deletes source branches for merged PRs between two branches on the same repository. This is helpful to keep repos that don't allow forking clean.",
			PullRequestHandler: handlePullRequest,
		},
	)
}

func handlePullRequest(pc plugins.Agent, pre scm.PullRequestHook) error {
	return handle(pc.SCMProviderClient, pc.Logger, pre)
}

type scmProviderClient interface {
	DeleteRef(owner, repo, ref string) error
}

func handle(spc scmProviderClient, log *logrus.Entry, pre scm.PullRequestHook) error {
	// Only consider closed PRs that got merged
	if pre.Action != scm.ActionClose || !pre.PullRequest.Merged {
		return nil
	}

	pr := pre.PullRequest

	//Only consider PRs from the same repo
	if pr.Base.Repo.FullName != pr.Head.Repo.FullName {
		return nil
	}

	if err := spc.DeleteRef(pr.Base.Repo.Namespace, pr.Base.Repo.Name, fmt.Sprintf("heads/%s", pr.Head.Ref)); err != nil {
		return fmt.Errorf("failed to delete branch %s on repo %s/%s after Pull Request #%d got merged: %v",
			pr.Head.Ref, pr.Base.Repo.Namespace, pr.Base.Repo.Name, pre.PullRequest.Number, err)
	}

	return nil
}
