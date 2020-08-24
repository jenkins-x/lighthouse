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

// Config is config for all prow jobs
type Config struct {
	// Presets apply to all job types.
	Presets []Preset `json:"presets,omitempty"`
	// Full repo name (such as "kubernetes/kubernetes") -> list of jobs.
	Presubmits  map[string][]Presubmit  `json:"presubmits,omitempty"`
	Postsubmits map[string][]Postsubmit `json:"postsubmits,omitempty"`

	// Periodics are not associated with any repo.
	Periodics []Periodic `json:"periodics,omitempty"`
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
	listPeriodic := func(ps []Periodic) []Periodic {
		var res []Periodic
		res = append(res, ps...)
		return res
	}

	return listPeriodic(c.Periodics)
}

// SetPresubmits updates c.Presubmits to jobs, after compiling and validating their regexes.
func (c *Config) SetPresubmits(jobs map[string][]Presubmit) error {
	nj := map[string][]Presubmit{}
	for k, v := range jobs {
		nj[k] = make([]Presubmit, len(v))
		copy(nj[k], v)
		if err := SetPresubmitRegexes(nj[k]); err != nil {
			return err
		}
	}
	c.Presubmits = nj
	return nil
}

// SetPostsubmits updates c.Postsubmits to jobs, after compiling and validating their regexes.
func (c *Config) SetPostsubmits(jobs map[string][]Postsubmit) error {
	nj := map[string][]Postsubmit{}
	for k, v := range jobs {
		nj[k] = make([]Postsubmit, len(v))
		copy(nj[k], v)
		if err := SetPostsubmitRegexes(nj[k]); err != nil {
			return err
		}
	}
	c.Postsubmits = nj
	return nil
}

// SetPresubmitRegexes compiles and validates all the regular expressions for
// the provided presubmits.
func SetPresubmitRegexes(ps []Presubmit) error {
	for i := range ps {
		if err := ps[i].SetRegexes(); err != nil {
			return err
		}
	}
	return nil
}

// SetPostsubmitRegexes compiles and validates all the regular expressions for
// the provided postsubmits.
func SetPostsubmitRegexes(ps []Postsubmit) error {
	for i := range ps {
		if err := ps[i].SetRegexes(); err != nil {
			return err
		}
	}
	return nil
}

// ClearCompiledRegexes removes compiled regexes from the presubmits,
// useful for testing when deep equality is needed between presubmits
func ClearCompiledRegexes(presubmits []Presubmit) {
	for i := range presubmits {
		presubmits[i].re = nil
		presubmits[i].Brancher.re = nil
		presubmits[i].Brancher.reSkip = nil
		presubmits[i].RegexpChangeMatcher.reChanges = nil
	}
}
