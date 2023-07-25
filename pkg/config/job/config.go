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

package job

import (
	"fmt"

	"github.com/jenkins-x/lighthouse/pkg/config/lighthouse"
	"gopkg.in/robfig/cron.v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// Config is config for all prow jobs
type Config struct {
	// Presets apply to all job types.
	Presets []Preset `json:"presets,omitempty"`
	// Full repo name (such as "kubernetes/kubernetes") -> list of jobs.
	Presubmits  map[string][]Presubmit  `json:"presubmits,omitempty"`
	Postsubmits map[string][]Postsubmit `json:"postsubmits,omitempty"`
	// Periodics are not associated with any repo.
	Periodics   []Periodic              `json:"periodics,omitempty"`
	Deployments map[string][]Deployment `json:"deployments,omitempty"`
}

func resolvePresets(name string, labels map[string]string, spec *v1.PodSpec, presets []Preset) error {
	for _, preset := range presets {
		if err := MergePreset(preset, labels, spec); err != nil {
			return fmt.Errorf("job %s failed to merge presets: %v", name, err)
		}
	}

	return nil
}

// Merge merges one Config with another one
func (c *Config) Merge(other Config) error {
	c.Presets = append(c.Presets, other.Presets...)
	// validate no duplicated preset key-value pairs
	validLabels := map[string]bool{}
	for _, preset := range c.Presets {
		for label, val := range preset.Labels {
			pair := label + ":" + val
			if _, ok := validLabels[pair]; ok {
				return fmt.Errorf("duplicated preset 'label:value' pair : %s", pair)
			}
			validLabels[pair] = true
		}
	}
	c.Periodics = append(c.Periodics, other.Periodics...)
	if c.Presubmits == nil {
		c.Presubmits = make(map[string][]Presubmit)
	}
	for repo, jobs := range other.Presubmits {
		c.Presubmits[repo] = append(c.Presubmits[repo], jobs...)
	}
	if c.Postsubmits == nil {
		c.Postsubmits = make(map[string][]Postsubmit)
	}
	for repo, jobs := range other.Postsubmits {
		c.Postsubmits[repo] = append(c.Postsubmits[repo], jobs...)
	}
	return nil
}

// Init sets defaults and initializes Config
func (c *Config) Init(lh lighthouse.Config) error {
	for _, ps := range c.Presubmits {
		for i := range ps {
			ps[i].SetDefaults(lh.PodNamespace)
			if err := ps[i].SetRegexes(); err != nil {
				return fmt.Errorf("could not set regex: %v", err)
			}
			if err := resolvePresets(ps[i].Name, ps[i].Labels, ps[i].Spec, c.Presets); err != nil {
				return err
			}
		}
	}
	for _, ps := range c.Postsubmits {
		for i := range ps {
			ps[i].SetDefaults(lh.PodNamespace)
			if err := ps[i].SetRegexes(); err != nil {
				return fmt.Errorf("could not set regex: %v", err)
			}
			if err := resolvePresets(ps[i].Name, ps[i].Labels, ps[i].Spec, c.Presets); err != nil {
				return err
			}
		}
	}
	for i := range c.Periodics {
		c.Periodics[i].SetDefaults(lh.PodNamespace)
		if err := resolvePresets(c.Periodics[i].Name, c.Periodics[i].Labels, c.Periodics[i].Spec, c.Presets); err != nil {
			return err
		}
	}
	return nil
}

// Validate validates Config
func (c *Config) Validate(lh lighthouse.Config) error {
	type orgRepoJobName struct {
		orgRepo, jobName string
	}
	// Validate presubmits.
	// Checking that no duplicate job in prow config exists on the same org / repo / branch.
	validPresubmits := map[orgRepoJobName][]Presubmit{}
	for repo, jobs := range c.Presubmits {
		for _, job := range jobs {
			repoJobName := orgRepoJobName{repo, job.Name}
			for _, existingJob := range validPresubmits[repoJobName] {
				if existingJob.Name == job.Name {
					return fmt.Errorf("duplicated presubmit job: %s", job.Name)
				}
			}
			validPresubmits[repoJobName] = append(validPresubmits[repoJobName], job)
		}
	}
	for _, ps := range c.Presubmits {
		for _, j := range ps {
			if err := j.Validate(lh.PodNamespace); err != nil {
				return err
			}
		}
	}
	// Validate postsubmits.
	// Checking that no duplicate job in prow config exists on the same org / repo / branch.
	validPostsubmits := map[orgRepoJobName][]Postsubmit{}
	for repo, jobs := range c.Postsubmits {
		for _, job := range jobs {
			repoJobName := orgRepoJobName{repo, job.Name}
			for _, existingJob := range validPostsubmits[repoJobName] {
				if existingJob.Brancher.Intersects(job.Brancher) {
					return fmt.Errorf("duplicated postsubmit job: %s", job.Name)
				}
			}
			validPostsubmits[repoJobName] = append(validPostsubmits[repoJobName], job)
		}
	}
	for _, ps := range c.Postsubmits {
		for _, j := range ps {
			if err := j.Base.Validate(PostsubmitJob, lh.PodNamespace); err != nil {
				return fmt.Errorf("invalid postsubmit job %s: %v", j.Name, err)
			}
		}
	}
	// validate no duplicated periodics
	validPeriodics := sets.NewString()
	// Ensure that the periodic durations are valid and specs exist.
	for _, p := range c.Periodics {
		if validPeriodics.Has(p.Name) {
			return fmt.Errorf("duplicated periodic job : %s", p.Name)
		}
		validPeriodics.Insert(p.Name)
		if err := p.Base.Validate(PeriodicJob, lh.PodNamespace); err != nil {
			return fmt.Errorf("invalid periodic job %s: %v", p.Name, err)
		}
	}
	// Set the interval on the periodic jobs. It doesn't make sense to do this
	// for child jobs.
	for _, p := range c.Periodics {
		if p.Cron != "" {
			if _, err := cron.Parse(p.Cron); err != nil {
				return fmt.Errorf("invalid cron string %s in periodic %s: %v", p.Cron, p.Name, err)
			}
		} else {
			return fmt.Errorf("cron cannot be empty in periodic %s", p.Name)
		}
	}
	return nil
}

// DecorationRequested checks if decoration was requested
func (c *Config) DecorationRequested() bool {
	for _, vs := range c.Presubmits {
		for i := range vs {
			if vs[i].Decorate {
				return true
			}
		}
	}
	for _, js := range c.Postsubmits {
		for i := range js {
			if js[i].Decorate {
				return true
			}
		}
	}
	for i := range c.Periodics {
		if c.Periodics[i].Decorate {
			return true
		}
	}
	return false
}

// AllPresubmits returns all prow presubmit jobs in repos.
// if repos is empty, return all presubmits.
func (c *Config) AllPresubmits(repos []string) []Presubmit {
	var res []Presubmit

	for repo, v := range c.Presubmits {
		if len(repos) == 0 {
			res = append(res, v...)
		} else {
			for _, r := range repos {
				if r == repo {
					res = append(res, v...)
					break
				}
			}
		}
	}

	return res
}

// AllPostsubmits returns all prow postsubmit jobs in repos.
// if repos is empty, return all postsubmits.
func (c *Config) AllPostsubmits(repos []string) []Postsubmit {
	var res []Postsubmit

	for repo, v := range c.Postsubmits {
		if len(repos) == 0 {
			res = append(res, v...)
		} else {
			for _, r := range repos {
				if r == repo {
					res = append(res, v...)
					break
				}
			}
		}
	}

	return res
}

// AllPeriodics returns all prow periodic jobs.
func (c *Config) AllPeriodics() []Periodic {
	return c.Periodics
}

// SetPresubmits updates c.Presubmits to jobs, after compiling and validating their regexes.
func (c *Config) SetPresubmits(jobs map[string][]Presubmit) error {
	nj := map[string][]Presubmit{}
	for k, v := range jobs {
		for i := range v {
			if err := v[i].SetRegexes(); err != nil {
				return err
			}
		}
		nj[k] = make([]Presubmit, len(v))
		copy(nj[k], v)
	}
	c.Presubmits = nj
	return nil
}

// SetPostsubmits updates c.Postsubmits to jobs, after compiling and validating their regexes.
func (c *Config) SetPostsubmits(jobs map[string][]Postsubmit) error {
	nj := map[string][]Postsubmit{}
	for k, v := range jobs {
		for i := range v {
			if err := v[i].SetRegexes(); err != nil {
				return err
			}
		}
		nj[k] = make([]Postsubmit, len(v))
		copy(nj[k], v)
	}
	c.Postsubmits = nj
	return nil
}
