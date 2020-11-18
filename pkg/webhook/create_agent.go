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

package webhook

import (
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/inrepo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CreateAgent creates an agent for the given repository
// if the repository is configured to use in repository configuration then we create the use the repository specific
// configuration
func (s *Server) CreateAgent(l *logrus.Entry, owner, repo, ref string) (plugins.Agent, error) {
	pc := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, s.ServerURL, l)

	var err error
	pc.Config, pc.PluginConfig, err = inrepo.Generate(pc.SCMProviderClient, pc.Config, pc.PluginConfig, owner, repo, ref)
	if err != nil {
		return pc, errors.Wrapf(err, "failed to calculate in repo config")
	}
	return pc, nil
}
