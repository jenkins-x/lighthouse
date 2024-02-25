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

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/jenkins-x/lighthouse/pkg/config/job"
	"github.com/jenkins-x/lighthouse/pkg/config/secret"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestDefaultJobBase(t *testing.T) {
	bar := "bar"
	filled := job.Base{
		Agent:     "foo",
		Namespace: &bar,
		Cluster:   "build",
	}
	cases := []struct {
		name     string
		config   ProwConfig
		base     func(j *job.Base)
		expected func(j *job.Base)
	}{
		{
			name: "no changes when fields are already set",
		},
		{
			name: "empty agent results in kubernetes",
			base: func(j *job.Base) {
				j.Agent = ""
			},
			expected: func(j *job.Base) {
				j.Agent = string(job.JenkinsXAgent)
			},
		},
		{
			name: "nil namespace becomes PodNamespace",
			config: ProwConfig{
				PodNamespace:           "pod-namespace",
				LighthouseJobNamespace: "wrong",
			},
			base: func(j *job.Base) {
				j.Namespace = nil
			},
			expected: func(j *job.Base) {
				p := "pod-namespace"
				j.Namespace = &p
			},
		},
		{
			name: "empty namespace becomes PodNamespace",
			config: ProwConfig{
				PodNamespace:           "new-pod-namespace",
				LighthouseJobNamespace: "still-wrong",
			},
			base: func(j *job.Base) {
				var empty string
				j.Namespace = &empty
			},
			expected: func(j *job.Base) {
				p := "new-pod-namespace"
				j.Namespace = &p
			},
		},
		{
			name: "empty cluster becomes DefaultClusterAlias",
			base: func(j *job.Base) {
				j.Cluster = ""
			},
			expected: func(j *job.Base) {
				j.Cluster = job.DefaultClusterAlias
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := filled
			if tc.base != nil {
				tc.base(&actual)
			}
			expected := actual
			if tc.expected != nil {
				tc.expected(&expected)
			}
			actual.SetDefaults(tc.config.PodNamespace)
			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("expected %#v\n!=\nactual %#v", expected, actual)
			}
		})
	}
}

func TestValidateAgent(t *testing.T) {
	k := string(job.JenkinsXAgent)
	ns := "default"
	base := job.Base{
		Agent:     k,
		Namespace: &ns,
		Spec:      &v1.PodSpec{},
	}

	cases := []struct {
		name string
		base func(j *job.Base)
		pass bool
	}{
		{
			name: "reject unknown agent",
			base: func(j *job.Base) {
				j.Agent = "random-agent"
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			jb := base
			if tc.base != nil {
				tc.base(&jb)
			}
			switch err := jb.ValidateAgent(ns); {
			case err == nil && !tc.pass:
				t.Error("validation failed to raise an error")
			case err != nil && tc.pass:
				t.Errorf("validation should have passed, got: %v", err)
			}
		})
	}
}

func TestValidateLabels(t *testing.T) {
	cases := []struct {
		name   string
		labels map[string]string
		pass   bool
	}{
		{
			name: "happy case",
			pass: true,
		},
		{
			name: "reject reserved label",
			labels: map[string]string{
				job.Labels()[0]: "anything",
			},
		},
		{
			name: "reject bad label key",
			labels: map[string]string{
				"_underscore-prefix": "annoying",
			},
		},
		{
			name: "reject bad label value",
			labels: map[string]string{
				"whatever": "_private-is-rejected",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			switch err := job.ValidateLabels(tc.labels); {
			case err == nil && !tc.pass:
				t.Error("validation failed to raise an error")
			case err != nil && tc.pass:
				t.Errorf("validation should have passed, got: %v", err)
			}
		})
	}
}

func TestValidateJobBase(t *testing.T) {
	ka := string(job.JenkinsXAgent)
	goodSpec := v1.PodSpec{
		Containers: []v1.Container{
			{},
		},
	}
	ns := "target-namespace"
	cases := []struct {
		name string
		base job.Base
		pass bool
	}{
		{
			name: "valid kubernetes job",
			base: job.Base{
				Name:      "name",
				Agent:     ka,
				Spec:      &goodSpec,
				Namespace: &ns,
			},
			pass: true,
		},
		{
			name: "invalid concurrency",
			base: job.Base{
				Name:           "name",
				MaxConcurrency: -1,
				Agent:          ka,
				Spec:           &goodSpec,
				Namespace:      &ns,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			switch err := tc.base.Validate(job.PresubmitJob, ns); {
			case err == nil && !tc.pass:
				t.Error("validation failed to raise an error")
			case err != nil && tc.pass:
				t.Errorf("validation should have passed, got: %v", err)
			}
		})
	}
}

// integration test for fake config loading
func TestValidConfigLoading(t *testing.T) {
	var testCases = []struct {
		name               string
		prowConfig         string
		jobConfigs         []string
		expectError        bool
		expectPodNameSpace string
		expectEnv          map[string][]v1.EnvVar
		expectContexts     map[string]string
	}{
		{
			name:       "one config",
			prowConfig: ``,
		},

		// TODO get these tests passing...
		/*
							{
						name:       "decorated periodic missing `command`",
						prowConfig: ``,
						jobConfigs: []string{
							`
			periodics:
			- interval: 10m
			  agent: tekton
			  name: foo
			  decorate: true
			  spec:
			    containers:
			    - image: alpine`,
						},
						expectError: true,
					},
					{
						name:       "reject invalid kubernetes periodic",
						prowConfig: ``,
						jobConfigs: []string{
							`
			periodics:
			- interval: 10m
			  agent: tekton
			  build_spec:
			  name: foo`,
						},
						expectError: true,
					},
		*/
		{
			name:       "reject invalid build periodic",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- cron: '* * * * *'
  agent: knative-build
  spec:
  name: foo`,
			},
			expectError: true,
		},
		{
			name:       "one periodic",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- cron: '* * * * *'
  agent: tekton
  name: foo
  spec:
    containers:
    - image: alpine`,
			},
		},
		{
			name:       "one periodic no agent, should default",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- cron: '* * * * *'
  name: foo
  spec:
    containers:
    - image: alpine`,
			},
		},
		{
			name:       "two periodics",
			prowConfig: ``,
			jobConfigs: []string{
				`
periodics:
- cron: '* * * * *'
  agent: tekton
  name: foo
  spec:
    containers:
    - image: alpine`,
				`
periodics:
- cron: '* * * * *'
  agent: tekton
  name: bar
  spec:
    containers:
    - image: alpine`,
			},
		},
		//		{
		//			name:       "duplicated periodics",
		//			prowConfig: ``,
		//			jobConfigs: []string{
		//				`
		//periodics:
		//- cron: '* * * * *'
		//  agent: tekton
		//  name: foo
		//  spec:
		//    containers:
		//    - image: alpine`,
		//				`
		//periodics:
		//- cron: '* * * * *'
		//  agent: tekton
		//  name: foo
		//  spec:
		//    containers:
		//    - image: alpine`,
		//			},
		//			expectError: true,
		//		},
		{
			name:       "one presubmit no context should default",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: tekton
    name: presubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
			expectContexts: map[string]string{
				"presubmit-bar": "presubmit-bar",
			},
		},
		{
			name:       "one presubmit no agent should default",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - context: bar
    name: presubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
			expectContexts: map[string]string{
				"presubmit-bar": "bar",
			},
		},
		{
			name:       "one presubmit, ok",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: tekton
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine`,
			},
			expectContexts: map[string]string{
				"presubmit-bar": "bar",
			},
		},
		{
			name:       "two presubmits",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: tekton
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine`,
				`
presubmits:
  foo/baz:
  - agent: tekton
    name: presubmit-baz
    context: baz
    spec:
      containers:
      - image: alpine`,
			},
			expectContexts: map[string]string{
				"presubmit-bar": "bar",
				"presubmit-baz": "baz",
			},
		},
		{
			name:       "dup presubmits, one file",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: tekton
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine
  - agent: tekton
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine`,
			},
			expectError: true,
		},
		{
			name:       "dup presubmits, two files",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: tekton
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine`,
				`
presubmits:
  foo/bar:
  - agent: tekton
    context: bar
    name: presubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
			expectError: true,
		},
		{
			name:       "dup presubmits not the same branch, two files",
			prowConfig: ``,
			jobConfigs: []string{
				`
presubmits:
  foo/bar:
  - agent: tekton
    name: presubmit-bar
    context: bar
    branches:
    - master
    spec:
      containers:
      - image: alpine`,
				`
presubmits:
  foo/bar:
  - agent: tekton
    context: bar
    branches:
    - other
    name: presubmit-bar2
    spec:
      containers:
      - image: alpine`,
			},
			expectError: false,
		},
		{
			name: "dup presubmits main file",
			prowConfig: `
presubmits:
  foo/bar:
  - agent: tekton
    name: presubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine
  - agent: tekton
    context: bar
    name: presubmit-bar
    spec:
      containers:
      - image: alpine`,
			expectError: true,
		},
		{
			name: "dup presubmits main file not on the same branch",
			prowConfig: `
presubmits:
  foo/bar:
  - agent: tekton
    name: presubmit-bar
    context: bar
    branches:
    - other
    spec:
      containers:
      - image: alpine
  - agent: tekton
    context: bar
    branches:
    - master
    name: presubmit-bar2
    spec:
      containers:
      - image: alpine`,
			expectError: false,
		},

		{
			name:       "one postsubmit, ok",
			prowConfig: ``,
			jobConfigs: []string{
				`
postsubmits:
  foo/bar:
  - agent: tekton
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
			expectContexts: map[string]string{
				"postsubmit-bar": "postsubmit-bar",
			},
		},
		{
			name:       "one postsubmit no agent, should default",
			prowConfig: ``,
			jobConfigs: []string{
				`
postsubmits:
  foo/bar:
  - name: postsubmit-bar
    context: bar
    spec:
      containers:
      - image: alpine`,
			},
			expectContexts: map[string]string{
				"postsubmit-bar": "bar",
			},
		},
		{
			name:       "two postsubmits",
			prowConfig: ``,
			jobConfigs: []string{
				`
postsubmits:
  foo/bar:
  - agent: tekton
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
				`
postsubmits:
  foo/baz:
  - agent: tekton
    name: postsubmit-baz
    spec:
      containers:
      - image: alpine`,
			},
			expectContexts: map[string]string{
				"postsubmit-bar": "postsubmit-bar",
				"postsubmit-baz": "postsubmit-baz",
			},
		},
		{
			name:       "dup postsubmits, one file",
			prowConfig: ``,
			jobConfigs: []string{
				`
postsubmits:
  foo/bar:
  - agent: tekton
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine
  - agent: tekton
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
			expectError: true,
		},
		{
			name:       "dup postsubmits, two files",
			prowConfig: ``,
			jobConfigs: []string{
				`
postsubmits:
  foo/bar:
  - agent: tekton
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
				`
postsubmits:
  foo/bar:
  - agent: tekton
    name: postsubmit-bar
    spec:
      containers:
      - image: alpine`,
			},
			expectError: true,
		},
		{
			name: "test valid presets in main config",
			prowConfig: `
presets:
- labels:
    preset-baz: "true"
  env:
  - name: baz
    value: fejtaverse`,
			jobConfigs: []string{
				`periodics:
- cron: '* * * * *'
  agent: tekton
  name: foo
  labels:
    preset-baz: "true"
  spec:
    containers:
    - image: alpine`,
				`
periodics:
- cron: '* * * * *'
  agent: tekton
  name: bar
  labels:
    preset-baz: "true"
  spec:
    containers:
    - image: alpine`,
			},
			expectEnv: map[string][]v1.EnvVar{
				"foo": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
				},
				"bar": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
				},
			},
		},
		{
			name:       "test valid presets in job configs",
			prowConfig: ``,
			jobConfigs: []string{
				`
presets:
- labels:
    preset-baz: "true"
  env:
  - name: baz
    value: fejtaverse
periodics:
- cron: '* * * * *'
  agent: tekton
  name: foo
  labels:
    preset-baz: "true"
  spec:
    containers:
    - image: alpine`,
				`
periodics:
- cron: '* * * * *'
  agent: tekton
  name: bar
  labels:
    preset-baz: "true"
  spec:
    containers:
    - image: alpine`,
			},
			expectEnv: map[string][]v1.EnvVar{
				"foo": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
				},
				"bar": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
				},
			},
		},
		{
			name: "test valid presets in both main & job configs",
			prowConfig: `
presets:
- labels:
    preset-baz: "true"
  env:
  - name: baz
    value: fejtaverse`,
			jobConfigs: []string{
				`
presets:
- labels:
    preset-k8s: "true"
  env:
  - name: k8s
    value: kubernetes
periodics:
- cron: '* * * * *'
  agent: tekton
  name: foo
  labels:
    preset-baz: "true"
    preset-k8s: "true"
  spec:
    containers:
    - image: alpine`,
				`
periodics:
- cron: '* * * * *'
  agent: tekton
  name: bar
  labels:
    preset-baz: "true"
  spec:
    containers:
    - image: alpine`,
			},
			expectEnv: map[string][]v1.EnvVar{
				"foo": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
					{
						Name:  "k8s",
						Value: "kubernetes",
					},
				},
				"bar": {
					{
						Name:  "baz",
						Value: "fejtaverse",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// save the config
			prowConfigDir := t.TempDir()

			prowConfig := filepath.Join(prowConfigDir, "config.yaml")
			if err := os.WriteFile(prowConfig, []byte(tc.prowConfig), 0666); err != nil {
				t.Fatalf("fail to write prow config: %v", err)
			}

			jobConfig := ""
			if len(tc.jobConfigs) > 0 {
				jobConfigDir := t.TempDir()

				// cover both job config as a file & a dir
				if len(tc.jobConfigs) == 1 {
					// a single file
					jobConfig = filepath.Join(jobConfigDir, "config.yaml")
					if err := os.WriteFile(jobConfig, []byte(tc.jobConfigs[0]), 0666); err != nil {
						t.Fatalf("fail to write job config: %v", err)
					}
				} else {
					// a dir
					jobConfig = jobConfigDir
					for idx, config := range tc.jobConfigs {
						subConfig := filepath.Join(jobConfigDir, fmt.Sprintf("config_%d.yaml", idx))
						if err := os.WriteFile(subConfig, []byte(config), 0666); err != nil {
							t.Fatalf("fail to write job config: %v", err)
						}
					}
				}
			}

			cfg, err := Load(prowConfig, jobConfig)
			if tc.expectError && err == nil {
				t.Errorf("tc %s: Expect error, but got nil", tc.name)
			} else if !tc.expectError && err != nil {
				t.Errorf("tc %s: Expect no error, but got error %v", tc.name, err)
			}

			if err == nil {
				if tc.expectPodNameSpace == "" {
					tc.expectPodNameSpace = "default"
				}

				if cfg.PodNamespace != tc.expectPodNameSpace {
					t.Errorf("tc %s: Expect PodNamespace %s, but got %v", tc.name, tc.expectPodNameSpace, cfg.PodNamespace)
				}

				if len(tc.expectContexts) > 0 {
					for _, j := range cfg.AllPresubmits(nil) {
						ctx, ok := tc.expectContexts[j.Name]
						if !ok {
							t.Errorf("tc %s: job %s has no expected context", tc.name, j.Name)
						} else if ctx != j.Context {
							t.Errorf("tc %s: expect context %s for job %s but got %s", tc.name, ctx, j.Name, j.Context)
						}
					}

					for _, j := range cfg.AllPostsubmits(nil) {
						ctx, ok := tc.expectContexts[j.Name]
						if !ok {
							t.Errorf("tc %s: job %s has no expected context", tc.name, j.Name)
						} else if ctx != j.Context {
							t.Errorf("tc %s: expect context %s for job %s but got %s", tc.name, ctx, j.Name, j.Context)
						}
					}
				}

				if len(tc.expectEnv) > 0 {
					for _, j := range cfg.AllPresubmits(nil) {
						if envs, ok := tc.expectEnv[j.Name]; ok {
							if !reflect.DeepEqual(envs, j.Spec.Containers[0].Env) {
								t.Errorf("tc %s: expect env %v for job %s, got %+v", tc.name, envs, j.Name, j.Spec.Containers[0].Env)
							}
						}
					}

					for _, j := range cfg.AllPostsubmits(nil) {
						if envs, ok := tc.expectEnv[j.Name]; ok {
							if !reflect.DeepEqual(envs, j.Spec.Containers[0].Env) {
								t.Errorf("tc %s: expect env %v for job %s, got %+v", tc.name, envs, j.Name, j.Spec.Containers[0].Env)
							}
						}
					}

					for _, j := range cfg.AllPeriodics() {
						if envs, ok := tc.expectEnv[j.Name]; ok {
							if !reflect.DeepEqual(envs, j.Spec.Containers[0].Env) {
								t.Errorf("tc %s: expect env %v for job %s, got %+v", tc.name, envs, j.Name, j.Spec.Containers[0].Env)
							}
						}
					}
				}
			}
		})
	}
}

func TestLoadYAMLConfig_Defaults(t *testing.T) {
	configYaml := `
postsubmits:
  jenkins-x/jx:
    - agent: tekton
      branches:
        - master
      context: ""
      name: release
    - agent: tekton
      branches:
        - master
      context: whitesource
      name: whitesource
presubmits:
  jenkins-x/jx:
    - agent: tekton
      always_run: true
      context: ""
      name: integration
      rerun_command: /test integration
      trigger: (?m)^/test( all| integration),?(\s+|$)
    - agent: tekton
      always_run: false
      context: bdd
      name: bdd
      rerun_command: /test bdd
      trigger: (?m)^/test( bdd),?(\s+|$)
`
	cfg, err := LoadYAMLConfig([]byte(configYaml))
	assert.NoError(t, err)

	for _, j := range cfg.AllPostsubmits(nil) {
		assert.Equal(t, j.Name, j.Context, "expected context for %s to be same as name but was %s", j.Name, j.Context)
	}
}

func TestBrancher_Intersects(t *testing.T) {
	testCases := []struct {
		name   string
		a, b   job.Brancher
		result bool
	}{
		{
			name: "TwodifferentBranches",
			a: job.Brancher{
				Branches: []string{"a"},
			},
			b: job.Brancher{
				Branches: []string{"b"},
			},
		},
		{
			name: "Opposite",
			a: job.Brancher{
				SkipBranches: []string{"b"},
			},
			b: job.Brancher{
				Branches: []string{"b"},
			},
		},
		{
			name:   "BothRunOnAllBranches",
			a:      job.Brancher{},
			b:      job.Brancher{},
			result: true,
		},
		{
			name: "RunsOnAllBranchesAndSpecified",
			a:    job.Brancher{},
			b: job.Brancher{
				Branches: []string{"b"},
			},
			result: true,
		},
		{
			name: "SkipBranchesAndSet",
			a: job.Brancher{
				SkipBranches: []string{"a", "b", "c"},
			},
			b: job.Brancher{
				Branches: []string{"a"},
			},
		},
		{
			name: "SkipBranchesAndSet",
			a: job.Brancher{
				Branches: []string{"c"},
			},
			b: job.Brancher{
				Branches: []string{"a"},
			},
		},
		{
			name: "BothSkipBranches",
			a: job.Brancher{
				SkipBranches: []string{"a", "b", "c"},
			},
			b: job.Brancher{
				SkipBranches: []string{"d", "e", "f"},
			},
			result: true,
		},
		{
			name: "BothSkipCommonBranches",
			a: job.Brancher{
				SkipBranches: []string{"a", "b", "c"},
			},
			b: job.Brancher{
				SkipBranches: []string{"b", "e", "f"},
			},
			result: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(st *testing.T) {
			r1 := tc.a.Intersects(tc.b)
			r2 := tc.b.Intersects(tc.a)
			for _, result := range []bool{r1, r2} {
				if result != tc.result {
					st.Errorf("Expected %v got %v", tc.result, result)
				}
			}
		})
	}
}

// Integration test for fake secrets loading in a secret agent.
// Checking also if the agent changes the secret's values as expected.
func TestSecretAgentLoading(t *testing.T) {
	tempTokenValue := "121f3cb3e7f70feeb35f9204f5a988d7292c7ba1"
	changedTokenValue := "121f3cb3e7f70feeb35f9204f5a988d7292c7ba0"

	// Creating a temporary directory.
	secretDir := t.TempDir()

	// Launch the first temporary secret.
	firstTempSecret := filepath.Join(secretDir, "firstTempSecret")
	if err := os.WriteFile(firstTempSecret, []byte(tempTokenValue), 0666); err != nil {
		t.Fatalf("fail to write secret: %v", err)
	}

	// Launch the second temporary secret.
	secondTempSecret := filepath.Join(secretDir, "secondTempSecret")
	if err := os.WriteFile(secondTempSecret, []byte(tempTokenValue), 0666); err != nil {
		t.Fatalf("fail to write secret: %v", err)
	}

	tempSecrets := []string{firstTempSecret, secondTempSecret}
	// Starting the agent and add the two temporary secrets.
	secretAgent := &secret.Agent{}
	if err := secretAgent.Start(tempSecrets); err != nil {
		t.Fatalf("Error starting secrets agent. %v", err)
	}

	// Check if the values are as expected.
	for _, tempSecret := range tempSecrets {
		tempSecretValue := secretAgent.GetSecret(tempSecret)
		if string(tempSecretValue) != tempTokenValue {
			t.Fatalf("In secret %s it was expected %s but found %s",
				tempSecret, tempTokenValue, tempSecretValue)
		}
	}

	// Change the values of the files.
	if err := os.WriteFile(firstTempSecret, []byte(changedTokenValue), 0666); err != nil {
		t.Fatalf("fail to write secret: %v", err)
	}
	if err := os.WriteFile(secondTempSecret, []byte(changedTokenValue), 0666); err != nil {
		t.Fatalf("fail to write secret: %v", err)
	}

	retries := 10
	var errors []string

	// Check if the values changed as expected.
	for _, tempSecret := range tempSecrets {
		// Reset counter
		counter := 0
		for counter <= retries {
			tempSecretValue := secretAgent.GetSecret(tempSecret)
			if string(tempSecretValue) != changedTokenValue {
				if counter == retries {
					errors = append(errors, fmt.Sprintf("In secret %s it was expected %s but found %s\n",
						tempSecret, changedTokenValue, tempSecretValue))
				} else {
					// Secret agent needs some time to update the values. So wait and retry.
					time.Sleep(400 * time.Millisecond)
				}
			} else {
				break
			}
			counter++
		}
	}

	if len(errors) > 0 {
		t.Fatal(errors)
	}

}
