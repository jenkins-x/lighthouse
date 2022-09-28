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
	"strings"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/filebrowser"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/triggerconfig/inrepo"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CreateAgent creates an agent for the given repository
// if the repository is configured to use in repository configuration then we create the use the repository specific
// configuration
func (s *Server) CreateAgent(l *logrus.Entry, owner, repo, ref string) (plugins.Agent, error) {
	start := time.Now()
	pc := plugins.NewAgent(s.ConfigAgent, s.Plugins, s.ClientAgent, s.ServerURL, l)
	fullName := scm.Join(owner, repo)
	if pc.Config == nil {
		return pc, errors.Errorf("no config available. maybe the ConfigMap got deleted")
	}
	if !pc.Config.InRepoConfigEnabled(fullName) {
		return pc, nil
	}

	if !IsSHA(ref) {
		err := s.createAgent(&pc, owner, repo, ref)
		if err != nil {
			return pc, errors.Wrapf(err, "failed to calculate in repo config")
		}
		return pc, nil
	}

	key := owner + "/" + repo + "/" + ref
	c := s.InRepoCache
	if x, found := c.Get(key); found {
		pa := x.(*plugins.Agent)
		if pa != nil {
			return *pa, nil
		}
	}
	err := s.createAgent(&pc, owner, repo, ref)
	if err != nil {
		return pc, errors.Wrapf(err, "failed to create agent")
	}
	c.Add(key, &pc)
	duration := time.Since(start)
	l.WithField("Duration", duration.String()).Info("created configAgent")
	return pc, nil
}

func (s *Server) createAgent(pc *plugins.Agent, owner, repo, ref string) error {
	var err error
	cache := inrepo.NewResolverCache()
	fc := filebrowser.NewFetchCache()
	pc.Config, pc.PluginConfig, err = inrepo.Generate(s.FileBrowsers, fc, cache, pc.Config, pc.PluginConfig, owner, repo, ref)
	if err != nil {
		return errors.Wrapf(err, "failed to calculate in repo config")
	}
	return nil
}

// IsSHA returns true if the given ref is a SHA
func IsSHA(ref string) bool {
	return len(ref) > 7 && !strings.Contains(ref, "/")
}
