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
	"regexp"

	"github.com/jenkins-x/lighthouse/pkg/config/util"
)

// Presubmit runs on PRs.
type Presubmit struct {
	Base
	Brancher
	RegexpChangeMatcher
	Reporter
	// AlwaysRun automatically for every PR, or only when a comment triggers it.
	// However if A PR contains files that are included by the ignore changes regex, then a build wont be triggered
	AlwaysRun bool `json:"always_run"`
	// RequireRun if this value is true and AlwaysRun is false then we need to manually trigger this context for the PR to be allowed to auto merge.
	RequireRun bool `json:"require_run,omitempty"`
	// Optional indicates that the job's status context should not be required for merge.
	Optional bool `json:"optional,omitempty"`
	// Trigger is the regular expression to trigger the job.
	// e.g. `@k8s-bot e2e test this`
	// RerunCommand must also be specified if this field is specified.
	// (Default: `(?m)^/test (?:.*? )?<job name>(?: .*?)?$`)
	Trigger string `json:"trigger,omitempty"`
	// The RerunCommand to give users. Must match Trigger.
	// Trigger must also be specified if this field is specified.
	// (Default: `/test <job name>`)
	RerunCommand string       `json:"rerun_command,omitempty"`
	JenkinsSpec  *JenkinsSpec `json:"jenkins_spec,omitempty"`

	// We'll set these when we load it.
	re *regexp.Regexp // from Trigger.
}

// SetDefaults initializes default values
func (p *Presubmit) SetDefaults(namespace string) {
	p.Base.SetDefaults(namespace)
	if p.Context == "" {
		p.Context = p.Name
	}
	// Default the values of Trigger and RerunCommand if both fields are
	// specified. Otherwise let validation fail as both or neither should have
	// been specified.
	if p.Trigger == "" && p.RerunCommand == "" {
		p.Trigger = util.DefaultTriggerFor(p.Name)
		p.RerunCommand = util.DefaultRerunCommandFor(p.Name)
	}
}

// SetRegexes compiles and validates all the regular expressions
func (p *Presubmit) SetRegexes() error {
	if re, err := regexp.Compile(p.Trigger); err == nil {
		p.re = re
	} else {
		return fmt.Errorf("could not compile trigger regex for %s: %v", p.Name, err)
	}
	if !p.re.MatchString(p.RerunCommand) {
		return fmt.Errorf("for job %s, rerun command \"%s\" does not match trigger \"%s\"", p.Name, p.RerunCommand, p.Trigger)
	}
	b, err := p.Brancher.SetBrancherRegexes()
	if err != nil {
		return fmt.Errorf("could not set branch regexes for %s: %v", p.Name, err)
	}
	p.Brancher = b
	c, err := p.RegexpChangeMatcher.SetChangeRegexes()
	if err != nil {
		return fmt.Errorf("could not set change regexes for %s: %v", p.Name, err)
	}
	p.RegexpChangeMatcher = c
	return nil
}

// ClearCompiledRegexes compiles and validates all the regular expressions
func (p *Presubmit) ClearCompiledRegexes() {
	p.re = nil
	p.Brancher.re = nil
	p.Brancher.reSkip = nil
	p.RegexpChangeMatcher.reChanges = nil
}

// CouldRun determines if the presubmit could run against a specific
// base ref
func (p Presubmit) CouldRun(baseRef string) bool {
	return p.Brancher.ShouldRun(baseRef)
}

// ShouldRun determines if the presubmit should run against a specific
// base ref, or in response to a set of changes. The latter mechanism
// is evaluated lazily, if necessary.
func (p Presubmit) ShouldRun(baseRef string, changes ChangedFilesProvider, forced, defaults bool) (bool, error) {
	if !p.CouldRun(baseRef) {
		return false, nil
	}

	// Evaluate regex expressions before checking if pre-submit jobs are always supposed to run
	if !forced {
		if determined, shouldRun, err := p.RegexpChangeMatcher.ShouldRun(changes); err != nil {
			return false, err
		} else if determined {
			return shouldRun, nil
		}
	}

	// TODO temporary disable RequireRun
	// if p.AlwaysRun || p.RequireRun {
	if p.AlwaysRun {
		return true, nil
	}
	if forced {
		return true, nil
	}
	return defaults, nil
}

// TriggersConditionally determines if the presubmit triggers conditionally (if it may or may not trigger).
func (p Presubmit) TriggersConditionally() bool {
	return p.NeedsExplicitTrigger() || p.RegexpChangeMatcher.CouldRun()
}

// NeedsExplicitTrigger determines if the presubmit requires a human action to trigger it or not.
func (p Presubmit) NeedsExplicitTrigger() bool {
	return !p.AlwaysRun && !p.RegexpChangeMatcher.CouldRun()
}

// TriggerMatches returns true if the comment body should trigger this presubmit.
//
// This is usually a /test foo string.
func (p Presubmit) TriggerMatches(body string) bool {
	re := p.re
	if p.Trigger == "" {
		return false
	}
	if re == nil {
		var err error
		re, err = regexp.Compile(p.Trigger)
		if err != nil {
			return body == p.Trigger
		}
	}
	return re != nil && re.MatchString(body)
}

// ContextRequired checks whether a context is required from github points of view (required check).
func (p Presubmit) ContextRequired() bool {
	return !(p.Optional || p.SkipReport)
}

// Validate validates job base
func (p *Presubmit) Validate(podNamespace string) error {
	if err := p.Base.Validate(PresubmitJob, podNamespace); err != nil {
		return fmt.Errorf("invalid presubmit job %s: %v", p.Name, err)
	}
	if p.AlwaysRun && p.RunIfChanged != "" {
		return fmt.Errorf("job %s is set to always run but also declares run_if_changed targets, which are mutually exclusive", p.Name)
	}
	if !p.SkipReport && p.Context == "" {
		return fmt.Errorf("job %s is set to report but has no context configured", p.Name)
	}
	return nil
}
