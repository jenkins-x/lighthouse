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
	"time"
)

// Periodic runs on a timer.
type Periodic struct {
	Base

	// (deprecated)Interval to wait between two runs of the job.
	Interval string `json:"interval"`
	// Cron representation of job trigger time
	Cron string `json:"cron"`
	// Tags for config entries
	Tags []string `json:"tags,omitempty"`

	interval time.Duration
}

// SetDefaults initializes default values
func (p *Periodic) SetDefaults(namespace string) {
	p.Base.SetDefaults(namespace)
}

// SetInterval updates interval, the frequency duration it runs.
func (p *Periodic) SetInterval(d time.Duration) {
	p.interval = d
}

// GetInterval returns interval, the frequency duration it runs.
func (p *Periodic) GetInterval() time.Duration {
	return p.interval
}
