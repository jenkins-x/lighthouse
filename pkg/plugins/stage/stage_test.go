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
	"reflect"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/scmprovider"
	"github.com/sirupsen/logrus"
)

type fakeClient struct {
	// current labels
	labels []string
	// labels that are added
	added []string
	// labels that are removed
	removed []string
}

func (c *fakeClient) AddLabel(owner, repo string, number int, label string, _ bool) error {
	c.added = append(c.added, label)
	c.labels = append(c.labels, label)
	return nil
}

func (c *fakeClient) RemoveLabel(owner, repo string, number int, label string, _ bool) error {
	c.removed = append(c.removed, label)

	// remove from existing labels
	for k, v := range c.labels {
		if label == v {
			c.labels = append(c.labels[:k], c.labels[k+1:]...)
			break
		}
	}

	return nil
}

func (c *fakeClient) GetIssueLabels(owner, repo string, number int, _ bool) ([]*scm.Label, error) {
	la := []*scm.Label{}
	for _, l := range c.labels {
		la = append(la, &scm.Label{Name: l})
	}
	return la, nil
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
			fc := &fakeClient{
				labels:  tc.labels,
				added:   []string{},
				removed: []string{},
			}
			e := &scmprovider.GenericCommentEvent{
				Body:   tc.body,
				Action: scm.ActionCreate,
			}
			err := handle(fc, logrus.WithField("plugin", "fake-lifecyle"), e)
			switch {
			case err != nil:
				t.Errorf("%s: unexpected error: %v", tc.name, err)
			case !reflect.DeepEqual(tc.added, fc.added):
				t.Errorf("%s: added %v != actual %v", tc.name, tc.added, fc.added)
			case !reflect.DeepEqual(tc.removed, fc.removed):
				t.Errorf("%s: removed %v != actual %v", tc.name, tc.removed, fc.removed)
			}
		})
	}
}
