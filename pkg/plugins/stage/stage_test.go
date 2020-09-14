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

package stage

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
)

func issueLabels(labels ...string) []string {
	ls := []string{}
	for _, label := range labels {
		ls = append(ls, fmt.Sprintf("org/repo#0:%s", label))
	}
	return ls
}

func TestStageLabels(t *testing.T) {
	var testcases = []struct {
		name    string
		body    string
		added   []string
		removed []string
		labels  []string
	}{
		{
			name:    "random command -> no-op",
			body:    "/random-command",
			added:   []string{},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "remove stage but don't specify state -> no-op",
			body:    "/remove-stage",
			added:   []string{},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "add stage but don't specify state -> no-op",
			body:    "/stage",
			added:   []string{},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "add stage random -> no-op",
			body:    "/stage random",
			added:   []string{},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "remove stage random -> no-op",
			body:    "/remove-stage random",
			added:   []string{},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "add alpha and beta with single command -> no-op",
			body:    "/stage alpha beta",
			added:   []string{},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "add alpha and random with single command -> no-op",
			body:    "/stage alpha random",
			added:   []string{},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "add alpha, don't have it -> alpha added",
			body:    "/stage alpha",
			added:   []string{stageAlpha},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "add alpha with prefix, don't have it -> alpha added",
			body:    "/lh-stage alpha",
			added:   []string{stageAlpha},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "add beta, don't have it -> beta added",
			body:    "/stage beta",
			added:   []string{stageBeta},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "add stable, don't have it -> stable added",
			body:    "/stage stable",
			added:   []string{stageStable},
			removed: []string{},
			labels:  []string{},
		},
		{
			name:    "remove alpha, have it -> alpha removed",
			body:    "/remove-stage alpha",
			added:   []string{},
			removed: []string{stageAlpha},
			labels:  []string{stageAlpha},
		},
		{
			name:    "remove alpha with prefix, have it -> alpha removed",
			body:    "/lh-remove-stage alpha",
			added:   []string{},
			removed: []string{stageAlpha},
			labels:  []string{stageAlpha},
		},
		{
			name:    "remove beta, have it -> beta removed",
			body:    "/remove-stage beta",
			added:   []string{},
			removed: []string{stageBeta},
			labels:  []string{stageBeta},
		},
		{
			name:    "remove stable, have it -> stable removed",
			body:    "/remove-stage stable",
			added:   []string{},
			removed: []string{stageStable},
			labels:  []string{stageStable},
		},
		{
			name:    "add alpha but have it -> no-op",
			body:    "/stage alpha",
			added:   []string{},
			removed: []string{},
			labels:  []string{stageAlpha},
		},
		{
			name:    "add beta, have alpha -> beta added, alpha removed",
			body:    "/stage beta",
			added:   []string{stageBeta},
			removed: []string{stageAlpha},
			labels:  []string{stageAlpha},
		},
		{
			name:    "add stable, have beta -> stable added, beta removed",
			body:    "/stage stable",
			added:   []string{stageStable},
			removed: []string{stageBeta},
			labels:  []string{stageBeta},
		},
		{
			name:    "add stable, have alpha and beta -> stable added, alpha and beta removed",
			body:    "/stage stable",
			added:   []string{stageStable},
			removed: []string{stageAlpha, stageBeta},
			labels:  []string{stageAlpha, stageBeta},
		},
		{
			name:    "remove alpha, then remove beta and then add stable -> alpha and beta removed, stable added",
			body:    "/remove-stage alpha\n/remove-stage beta\n/stage stable",
			added:   []string{stageStable},
			removed: []string{stageAlpha, stageBeta},
			labels:  []string{stageAlpha, stageBeta},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeScmClient, fc := fake.NewDefault()
			fakeClient := scmprovider.ToTestClient(fakeScmClient)
			e := &scmprovider.GenericCommentEvent{
				Body:   tc.body,
				Action: scm.ActionCreate,
				Repo: scm.Repository{
					Namespace: "org",
					Name:      "repo",
					FullName:  "org/repo",
				},
			}
			fc.IssueLabelsExisting = issueLabels(tc.labels...)
			agent := plugins.Agent{
				SCMProviderClient: &fakeClient.Client,
				Logger:            logrus.WithField("plugin", pluginName),
			}
			err := plugin.InvokeCommandHandler(e, func(handler plugins.CommandEventHandler, e *scmprovider.GenericCommentEvent, match plugins.CommandMatch) error {
				return handler(match, agent, *e)
			})
			switch {
			case err != nil:
				t.Errorf("%s: unexpected error: %v", tc.name, err)
			case !reflect.DeepEqual(issueLabels(tc.added...), fc.IssueLabelsAdded):
				t.Errorf("%s: added %v != actual %v", tc.name, tc.added, fc.IssueLabelsAdded)
			case !reflect.DeepEqual(issueLabels(tc.removed...), fc.IssueLabelsRemoved):
				t.Errorf("%s: removed %v != actual %v", tc.name, tc.removed, fc.IssueLabelsRemoved)
			}
		})
	}
}
