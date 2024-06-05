/*
Copyright 2017 The Kubernetes Authors.

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

package updateconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/commentpruner"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	fake2 "github.com/jenkins-x/lighthouse/pkg/scmprovider/fake"
	"github.com/sirupsen/logrus"
	coreapi "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

const defaultNamespace = "default"

func TestUpdateConfig(t *testing.T) {
	basicPR := scm.PullRequest{
		Number: 1,
		Base: scm.PullRequestBranch{
			Repo: scm.Repository{
				Namespace: "kubernetes",
				Name:      "kubernetes",
			},
		},
		Head: scm.PullRequestBranch{
			Sha: "abcdef",
			Repo: scm.Repository{
				Namespace: "kubernetes",
				Name:      "kubernetes",
			},
		},
		Author: scm.User{
			Login: "foo",
		},
		Link: "https://github.com/kubernetes/kubernetes/pull/1",
	}

	testcases := []struct {
		name                 string
		prAction             scm.Action
		merged               bool
		mergeCommit          string
		changes              []*scm.Change
		existConfigMaps      []runtime.Object
		expectedConfigMaps   []*coreapi.ConfigMap
		config               *plugins.ConfigUpdater
		errorCommentContains []string
		existingComment      string
		expectedStatus       []*scm.StatusInput
	}{
		{
			name:     "Opened PR, no update",
			prAction: scm.ActionOpen,
			merged:   false,
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{},
		},
		{
			name:   "Opened PR, not merged, no update",
			merged: false,
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{},
		},
		{
			name:     "Closed PR, no prow changes, no update",
			prAction: scm.ActionClose,
			merged:   false,
			changes: []*scm.Change{
				{
					Path:      "foo.txt",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{},
		},
		{
			name:     "For whatever reason no merge commit SHA",
			prAction: scm.ActionClose,
			merged:   true,
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{},
		},
		{
			name:        "changed config.yaml, 1 update",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"config.yaml": "old-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"config.yaml": "new-config",
					},
				},
			},
		},
		{
			name:        "changed config.yaml, existed configmap, 1 update",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"config.yaml": "old-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"config.yaml": "new-config",
					},
				},
			},
		},
		{
			name:        "changed plugins.yaml, 1 update with custom key",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/plugins.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "plugins",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"test-key": "old-plugins",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "plugins",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"test-key": "new-plugins",
					},
				},
			},
		},
		{
			name:        "changed resources.yaml, 1 update with custom namespace",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "boskos/resources.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "boskos-config",
						Namespace: "boskos",
					},
					Data: map[string]string{
						"resources.yaml": "old-boskos-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "boskos-config",
						Namespace: "boskos",
					},
					Data: map[string]string{
						"resources.yaml": "new-boskos-config",
					},
				},
			},
		},
		{
			name:        "changed config.yaml, plugins.yaml and resources.yaml, 3 update",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/plugins.yaml",
					Additions: 1,
				},
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
				{
					Path:      "boskos/resources.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"config.yaml": "old-config",
					},
				},
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "plugins",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"test-key": "old-plugins",
					},
				},
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "boskos-config",
						Namespace: "boskos",
					},
					Data: map[string]string{
						"resources.yaml": "old-boskos-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"config.yaml": "new-config",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "plugins",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"test-key": "new-plugins",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "boskos-config",
						Namespace: "boskos",
					},
					Data: map[string]string{
						"resources.yaml": "new-boskos-config",
					},
				},
			},
		},
		{
			name:        "edited both config/foo.yaml and config/bar.yaml, 2 update",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "config/foo.yaml",
					Additions: 1,
				},
				{
					Path:      "config/bar.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multikey-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"foo.yaml": "old-foo-config",
						"bar.yaml": "old-bar-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multikey-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"foo.yaml": "new-foo-config",
						"bar.yaml": "new-bar-config",
					},
				},
			},
		},
		{
			name:        "edited config/foo.yaml, 1 update",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "config/foo.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multikey-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"foo.yaml": "old-foo-config",
						"bar.yaml": "old-bar-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multikey-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"foo.yaml": "new-foo-config",
						"bar.yaml": "old-bar-config",
					},
				},
			},
		},
		{
			name:        "remove config/foo.yaml, 1 update",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:    "config/foo.yaml",
					Deleted: true,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multikey-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"foo.yaml": "old-foo-config",
						"bar.yaml": "old-bar-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multikey-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"bar.yaml": "old-bar-config",
					},
				},
			},
		},
		{
			name:        "edited dir/subdir/fejtaverse/krzyzacy.yaml, 1 update",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "dir/subdir/fejtaverse/krzyzacy.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "glob-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"fejta.yaml":    "old-fejta-config",
						"krzyzacy.yaml": "old-krzyzacy-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "glob-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"fejta.yaml":    "old-fejta-config",
						"krzyzacy.yaml": "new-krzyzacy-config",
					},
				},
			},
		},
		{
			name:        "renamed dir/subdir/fejtaverse/krzyzacy.yaml, 1 update",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "54321",
			changes: []*scm.Change{
				{
					Path:         "dir/subdir/fejtaverse/fejtabot.yaml",
					PreviousPath: "dir/subdir/fejtaverse/krzyzacy.yaml",
					Renamed:      true,
					Additions:    1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "glob-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"krzyzacy.yaml": "old-krzyzacy-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "glob-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"fejtabot.yaml": "new-fejtabot-config",
					},
				},
			},
		},
		{
			name:        "add delete edit glob config, 3 update",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "dir/subdir/fejta.yaml",
					Additions: 1,
				},
				{
					Path:      "dir/subdir/fejtaverse/sig-foo/added.yaml",
					Added:     true,
					Additions: 1,
				},
				{
					Path:    "dir/subdir/fejtaverse/sig-bar/removed.yaml",
					Deleted: true,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "glob-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"fejta.yaml":    "old-fejta-config",
						"krzyzacy.yaml": "old-krzyzacy-config",
						"removed.yaml":  "old-removed-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "glob-config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"fejta.yaml":    "new-fejta-config",
						"krzyzacy.yaml": "old-krzyzacy-config",
						"added.yaml":    "new-added-config",
					},
				},
			},
		},
		{
			name:        "config changes without a backing configmap causes creation",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"config.yaml": "new-config",
					},
				},
			},
		},
		{
			name:        "gzips all content if the top level gzip flag is set",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					BinaryData: map[string][]byte{
						"config.yaml": {31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 75, 45, 215, 77, 206, 207, 75, 203, 76, 7, 4, 0, 0, 255, 255, 84, 214, 231, 87, 10, 0, 0, 0},
					},
				},
			},
			config: &plugins.ConfigUpdater{
				GZIP: true,
				Maps: map[string]plugins.ConfigMapSpec{
					"prow/config.yaml": {
						Name: "config",
					},
					"prow/plugins.yaml": {
						Name: "plugins",
						Key:  "test-key",
					},
				},
			},
		},
		{
			name:        "gzips all content except one marked false if the top level gzip flag is set",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
				{
					Path:      "prow/plugins.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					BinaryData: map[string][]byte{
						"config.yaml": {31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 75, 45, 215, 77, 206, 207, 75, 203, 76, 7, 4, 0, 0, 255, 255, 84, 214, 231, 87, 10, 0, 0, 0},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "plugins",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"plugins.yaml": "new-plugins",
					},
				},
			},
			config: &plugins.ConfigUpdater{
				GZIP: true,
				Maps: map[string]plugins.ConfigMapSpec{
					"prow/config.yaml": {
						Name: "config",
					},
					"prow/plugins.yaml": {
						Name: "plugins",
						GZIP: boolPtr(false),
					},
				},
			},
		},
		{
			name:        "gzips only one marked file if the top level gzip flag is set to false",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
				{
					Path:      "prow/plugins.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					BinaryData: map[string][]byte{
						"config.yaml": {31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 75, 45, 215, 77, 206, 207, 75, 203, 76, 7, 4, 0, 0, 255, 255, 84, 214, 231, 87, 10, 0, 0, 0},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "plugins",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"plugins.yaml": "new-plugins",
					},
				},
			},
			config: &plugins.ConfigUpdater{
				GZIP: false,
				Maps: map[string]plugins.ConfigMapSpec{
					"prow/config.yaml": {
						Name: "config",
						GZIP: boolPtr(true),
					},
					"prow/plugins.yaml": {
						Name: "plugins",
					},
				},
			},
		},
		{
			name:        "adds both binary and text keys for a single configmap",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
				{
					Path:      "prow/binary.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"config.yaml": "new-config",
					},
					BinaryData: map[string][]byte{
						"binary.yaml": []byte("new-binary\x00\xFF\xFF"),
					},
				},
			},
			config: &plugins.ConfigUpdater{
				Maps: map[string]plugins.ConfigMapSpec{
					"prow/*.yaml": {
						Name: "config",
					},
				},
			},
		},
		{
			name:        "converts a text key to a binary key when it becomes binary",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/becoming-binary.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"becoming-binary.yaml": "not-yet-binary",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					BinaryData: map[string][]byte{
						"becoming-binary.yaml": []byte("now-binary\x00\xFF\xFF"),
					},
				},
			},
			config: &plugins.ConfigUpdater{
				Maps: map[string]plugins.ConfigMapSpec{
					"prow/*.yaml": {
						Name: "config",
					},
				},
			},
		},
		{
			name:        "converts a binary key to a text key when it becomes text",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/becoming-text.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					BinaryData: map[string][]byte{
						"becoming-text.yaml": []byte("not-yet-text\x00\xFF\xFF"),
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"becoming-text.yaml": "now-text",
					},
					BinaryData: map[string][]uint8{},
				},
			},
			config: &plugins.ConfigUpdater{
				Maps: map[string]plugins.ConfigMapSpec{
					"prow/*.yaml": {
						Name: "config",
					},
				},
			},
		},
		{
			name:        "simultaneously converts text to binary and binary to text",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/becoming-text.yaml",
					Additions: 1,
				},
				{
					Path:      "prow/becoming-binary.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					BinaryData: map[string][]byte{
						"becoming-text.yaml": []byte("not-yet-text\x00\xFF\xFF"),
					},
					Data: map[string]string{
						"becoming-binary.yaml": "not-yet-binary",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					BinaryData: map[string][]byte{
						"becoming-binary.yaml": []byte("now-binary\x00\xFF\xFF"),
					},
					Data: map[string]string{
						"becoming-text.yaml": "now-text",
					},
				},
			},
			config: &plugins.ConfigUpdater{
				Maps: map[string]plugins.ConfigMapSpec{
					"prow/*.yaml": {
						Name: "config",
					},
				},
			},
		},
		{
			name:        "correctly converts to binary when gzipping",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"config.yaml": "old-config",
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					BinaryData: map[string][]byte{
						"config.yaml": {31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 75, 45, 215, 77, 206, 207, 75, 203, 76, 7, 4, 0, 0, 255, 255, 84, 214, 231, 87, 10, 0, 0, 0},
					},
				},
			},
			config: &plugins.ConfigUpdater{
				GZIP: true,
				Maps: map[string]plugins.ConfigMapSpec{
					"prow/*.yaml": {
						Name: "config",
					},
				},
			},
		},
		{
			name:        "correctly converts to text when ungzipping",
			prAction:    scm.ActionClose,
			merged:      true,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "prow/config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					BinaryData: map[string][]byte{
						"config.yaml": {31, 139, 8, 0, 0, 0, 0, 0, 0, 255, 202, 75, 45, 215, 77, 206, 207, 75, 203, 76, 7, 4, 0, 0, 255, 255, 84, 214, 231, 87, 10, 0, 0, 0},
					},
				},
			},
			expectedConfigMaps: []*coreapi.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"config.yaml": "new-config",
					},
				},
			},
			config: &plugins.ConfigUpdater{
				GZIP: false,
				Maps: map[string]plugins.ConfigMapSpec{
					"prow/*.yaml": {
						Name: "config",
					},
				},
			},
		},
		{
			name:        "invalid YAML in config.yaml, existed configmap",
			prAction:    scm.ActionOpen,
			merged:      false,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "invalid/invalid-yaml-config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"invalid-yaml-config.yaml": "old-config",
					},
				},
			},
			config: &plugins.ConfigUpdater{
				GZIP: false,
				Maps: map[string]plugins.ConfigMapSpec{
					"invalid/invalid-yaml-config.yaml": {
						Name: "config",
					},
				},
			},
			errorCommentContains: []string{
				"Validation error founds in config map file(s):",
				"In file [invalid/invalid-yaml-config.yaml](https://github.com/kubernetes/kubernetes/blob/abcdef/invalid/invalid-yaml-config.yaml) for config map **config**:",
				"> error converting YAML to JSON: yaml: line 2: mapping values are not allowed in this context",
			},
			expectedStatus: []*scm.StatusInput{{
				State: scm.StateFailure,
				Label: configUpdaterContextName,
				Desc:  configUpdaterContextMsgFailed,
			}},
		},
		{
			name:        "can't unmarshal config.yaml",
			prAction:    scm.ActionOpen,
			merged:      false,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "invalid/invalid-config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"invalid-config.yaml": "old-config",
					},
				},
			},
			config: &plugins.ConfigUpdater{
				GZIP: false,
				Maps: map[string]plugins.ConfigMapSpec{
					"invalid/invalid-config.yaml": {
						Name: "config",
					},
				},
			},
			errorCommentContains: []string{
				"Validation error founds in config map file(s):",
				"In file [invalid/invalid-config.yaml](https://github.com/kubernetes/kubernetes/blob/abcdef/invalid/invalid-config.yaml) for config map **config**:",
				"> error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go value of type config.Config",
			},
			expectedStatus: []*scm.StatusInput{{
				State: scm.StateFailure,
				Label: configUpdaterContextName,
				Desc:  configUpdaterContextMsgFailed,
			}},
		},
		{
			name:        "invalid YAML in config.yaml and plugins.yaml",
			prAction:    scm.ActionOpen,
			merged:      false,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "invalid/invalid-yaml-config.yaml",
					Additions: 1,
				},
				{
					Path:      "invalid/invalid-yaml-plugins.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"invalid-yaml-config.yaml": "old-config",
					},
				},
			},
			config: &plugins.ConfigUpdater{
				GZIP: false,
				Maps: map[string]plugins.ConfigMapSpec{
					"invalid/invalid-yaml-config.yaml": {
						Name: "config",
					},
					"invalid/invalid-yaml-plugins.yaml": {
						Name: "plugins",
					},
				},
			},
			errorCommentContains: []string{
				"Validation error founds in config map file(s):",
				"In file [invalid/invalid-yaml-config.yaml](https://github.com/kubernetes/kubernetes/blob/abcdef/invalid/invalid-yaml-config.yaml) for config map **config**:",
				"In file [invalid/invalid-yaml-plugins.yaml](https://github.com/kubernetes/kubernetes/blob/abcdef/invalid/invalid-yaml-plugins.yaml) for config map **plugins**:",
				"> error converting YAML to JSON: yaml: line 2: mapping values are not allowed in this context",
			},
			expectedStatus: []*scm.StatusInput{{
				State: scm.StateFailure,
				Label: configUpdaterContextName,
				Desc:  configUpdaterContextMsgFailed,
			}},
		},
		{
			name:        "valid YAML in config.yaml with existing comment",
			prAction:    scm.ActionOpen,
			merged:      false,
			mergeCommit: "12345",
			changes: []*scm.Change{
				{
					Path:      "valid/config.yaml",
					Additions: 1,
				},
			},
			existConfigMaps: []runtime.Object{
				&coreapi.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "config",
						Namespace: defaultNamespace,
					},
					Data: map[string]string{
						"valid/config.yaml": "old-config",
					},
				},
			},
			config: &plugins.ConfigUpdater{
				GZIP: false,
				Maps: map[string]plugins.ConfigMapSpec{
					"valid/config.yaml": {
						Name: "config",
					},
				},
			},
			expectedStatus: []*scm.StatusInput{{
				State: scm.StateSuccess,
				Label: configUpdaterContextName,
				Desc:  configUpdaterContextMsgSuccess,
			}},
			existingComment: configUpdaterMsgPruneMatch,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			log := logrus.WithField("plugin", pluginName)
			event := scm.PullRequestHook{
				Action:      tc.prAction,
				PullRequest: basicPR,
			}
			event.PullRequest.Merged = tc.merged
			if tc.mergeCommit != "" {
				event.PullRequest.MergeSha = tc.mergeCommit
			}

			invalidYaml, err := os.ReadFile(filepath.Join("test_data", "invalid-yaml.yaml"))
			if err != nil {
				t.Fatalf("couldn't read test_data/invalid-yaml.yaml: %v", err)
			}
			validConfig, err := os.ReadFile(filepath.Join("test_data", "valid-config.yaml"))
			if err != nil {
				t.Fatalf("couldn't read test_data/valid-config.yaml: %v", err)
			}
			invalidConfig, err := os.ReadFile(filepath.Join("test_data", "invalid-config.yaml"))
			if err != nil {
				t.Fatalf("couldn't read test_data/invalid-config.yaml: %v", err)
			}
			fspc := &fake2.SCMClient{
				PullRequests: map[int]*scm.PullRequest{
					basicPR.Number: &basicPR,
				},
				PullRequestChanges: map[int][]*scm.Change{
					basicPR.Number: tc.changes,
				},
				PullRequestComments: map[int][]*scm.Comment{},
				RemoteFiles: map[string]map[string]string{
					"prow/config.yaml": {
						"master": "old-config",
						"12345":  "new-config",
					},
					"prow/binary.yaml": {
						"master": "old-binary\x00\xFF\xFF",
						"12345":  "new-binary\x00\xFF\xFF",
					},
					"prow/becoming-binary.yaml": {
						"master": "not-yet-binary",
						"12345":  "now-binary\x00\xFF\xFF",
					},
					"prow/becoming-text.yaml": {
						"master": "not-yet-text\x00\xFF\xFF",
						"12345":  "now-text",
					},
					"prow/plugins.yaml": {
						"master": "old-plugins",
						"12345":  "new-plugins",
					},
					"boskos/resources.yaml": {
						"master": "old-boskos-config",
						"12345":  "new-boskos-config",
					},
					"config/foo.yaml": {
						"master": "old-foo-config",
						"12345":  "new-foo-config",
					},
					"config/bar.yaml": {
						"master": "old-bar-config",
						"12345":  "new-bar-config",
					},
					"dir/subdir/fejta.yaml": {
						"master": "old-fejta-config",
						"12345":  "new-fejta-config",
					},
					"dir/subdir/fejtaverse/krzyzacy.yaml": {
						"master": "old-krzyzacy-config",
						"12345":  "new-krzyzacy-config",
					},
					"dir/subdir/fejtaverse/fejtabot.yaml": {
						"54321": "new-fejtabot-config",
					},
					"dir/subdir/fejtaverse/sig-foo/added.yaml": {
						"12345": "new-added-config",
					},
					"dir/subdir/fejtaverse/sig-bar/removed.yaml": {
						"master": "old-removed-config",
					},
					"invalid/invalid-yaml-config.yaml": {
						"master": "old-config",
						"abcdef": string(invalidYaml),
					},
					"invalid/invalid-yaml-plugins.yaml": {
						"master": "old-plugins",
						"abcdef": string(invalidYaml),
					},
					"valid/config.yaml": {
						"master": "old-config",
						"abcdef": string(validConfig),
					},
					"invalid/invalid-config.yaml": {
						"master": "old-config",
						"abcdef": string(invalidConfig),
					},
				},
			}
			if tc.existingComment != "" {
				_ = fspc.CreateComment("kubernetes", "kubernetes", basicPR.Number, true, tc.existingComment)
			}
			fkc := fake.NewSimpleClientset(tc.existConfigMaps...)

			m := tc.config
			if m == nil {
				m = &plugins.ConfigUpdater{
					Maps: map[string]plugins.ConfigMapSpec{
						"prow/config.yaml": {
							Name: "config",
						},
						"prow/plugins.yaml": {
							Name: "plugins",
							Key:  "test-key",
						},
						"boskos/resources.yaml": {
							Name:      "boskos-config",
							Namespace: "boskos",
						},
						"config/foo.yaml": {
							Name: "multikey-config",
						},
						"config/bar.yaml": {
							Name: "multikey-config",
						},
						"dir/subdir/**/*.yaml": {
							Name: "glob-config",
						},
					},
				}
			}
			m.SetDefaults()
			cp := commentpruner.NewEventClient(fspc, logrus.WithField("client", "commentpruner"), basicPR.Repository().Namespace, basicPR.Repository().Name, basicPR.Number)

			if err := handle(fspc, fkc.CoreV1(), cp, defaultNamespace, log, event, *m); err != nil {
				t.Fatalf("%s: unexpected error handling: %s", tc.name, err)
			}

			modifiedConfigMaps := sets.NewString()
			for _, action := range fkc.Fake.Actions() {
				var obj runtime.Object
				switch action := action.(type) {
				case clienttesting.CreateActionImpl:
					obj = action.Object
				case clienttesting.UpdateActionImpl:
					obj = action.Object
				default:
					continue
				}
				objectMeta, err := meta.Accessor(obj)
				if err != nil {
					t.Fatalf("%s: client saw an action for something that wasn't an object: %v", tc.name, err)
				}
				modifiedConfigMaps.Insert(objectMeta.GetName())
			}

			if tc.errorCommentContains != nil {
				if len(fspc.PullRequestComments[basicPR.Number]) != 1 {
					t.Errorf("%s: Expect 1 comment, actually got %d", tc.name, len(fspc.PullRequestComments[basicPR.Number]))
				} else {
					comment := fspc.PullRequestComments[basicPR.Number][0].Body
					var missingLines []string
					for _, errLine := range tc.errorCommentContains {
						if !strings.Contains(comment, errLine) {
							missingLines = append(missingLines, fmt.Sprintf(" - '%s'", errLine))
						}
					}
					if len(missingLines) > 0 {
						t.Errorf("Missing lines from comment '%s':\n%s", comment, strings.Join(missingLines, "\n"))
					}
				}
			}

			if tc.expectedStatus != nil {
				createdStatus := fspc.CreatedStatuses[basicPR.Head.Sha]
				if createdStatus == nil {
					t.Errorf("no status set but one was expected")
				} else if d := cmp.Diff(createdStatus, tc.expectedStatus); d != "" {
					t.Errorf("status did not match expected: %s", d)
				}

				if tc.errorCommentContains == nil && len(fspc.PullRequestComments[basicPR.Number]) > 0 {
					t.Errorf("Expected no comments, but found %d", len(fspc.PullRequestComments[basicPR.Number]))
				}
			}

			if tc.expectedConfigMaps != nil {
				if len(fspc.PullRequestComments[basicPR.Number]) != 1 {
					t.Errorf("%s: Expect 1 comment, actually got %d", tc.name, len(fspc.PullRequestComments[basicPR.Number]))
				} else {
					comment := fspc.PullRequestComments[basicPR.Number][0].Body
					if !strings.Contains(comment, "Updated the") {
						t.Errorf("%s: missing Updated the from %s", tc.name, comment)
					}
					for _, configMap := range tc.expectedConfigMaps {
						if modifiedConfigMaps.Has(configMap.Name) {
							if !strings.Contains(comment, configMap.Name) {
								t.Errorf("%s: missing %s from %s", tc.name, configMap.Name, comment)
							}
						} else if strings.Contains(comment, configMap.Name) {
							t.Errorf("%s: should not contain %s in %s", tc.name, configMap.Name, comment)
						}
					}
				}
			}

			expectedConfigMaps := sets.NewString()
			for _, configMap := range tc.expectedConfigMaps {
				expectedConfigMaps.Insert(configMap.Name)
			}
			if missing := expectedConfigMaps.Difference(modifiedConfigMaps); missing.Len() > 0 {
				t.Errorf("%s: did not update expected configmaps: %v", tc.name, missing.List())
			}
			if extra := modifiedConfigMaps.Difference(expectedConfigMaps); extra.Len() > 0 {
				t.Errorf("%s: found unexpectedly updated configmaps: %v", tc.name, extra.List())
			}

			for _, expected := range tc.expectedConfigMaps {
				actual, err := fkc.CoreV1().ConfigMaps(expected.Namespace).Get(context.TODO(), expected.Name, metav1.GetOptions{})
				if err != nil && errors.IsNotFound(err) {
					t.Errorf("%s: Should have updated or created configmap for '%s'", tc.name, expected)
				} else if !equality.Semantic.DeepEqual(expected, actual) {
					t.Errorf("%s: incorrect ConfigMap state after update: %v", tc.name, cmp.Diff(expected, actual))
				}
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
