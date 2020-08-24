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
	"sync"
	"time"
)

// Delta represents the before and after states of a Config change detected by the Agent.
type Delta struct {
	Before, After Config
}

// DeltaChan is a channel to receive config delta events when config changes.
type DeltaChan = chan<- Delta

// Agent watches a path and automatically loads the config stored
// therein.
type Agent struct {
	mut           sync.RWMutex // do not export Lock, etc methods
	c             *Config
	subscriptions []DeltaChan
}

// Subscribe registers the channel for messages on config reload.
// The caller can expect a copy of the previous and current config
// to be sent down the subscribed channel when a new configuration
// is loaded.
func (ca *Agent) Subscribe(subscription DeltaChan) {
	ca.mut.Lock()
	defer ca.mut.Unlock()
	ca.subscriptions = append(ca.subscriptions, subscription)
}

// Getter returns the current Config in a thread-safe manner.
type Getter func() *Config

// Config returns the latest config. Do not modify the config.
func (ca *Agent) Config() *Config {
	ca.mut.RLock()
	defer ca.mut.RUnlock()
	return ca.c
}

// Set sets the config. Useful for testing.
func (ca *Agent) Set(c *Config) {
	ca.mut.Lock()
	defer ca.mut.Unlock()
	var oldConfig Config
	if ca.c != nil {
		oldConfig = *ca.c
	}
	delta := Delta{oldConfig, *c}
	ca.c = c
	for _, subscription := range ca.subscriptions {
		go func(sub DeltaChan) { // wait a minute to send each event
			end := time.NewTimer(time.Minute)
			select {
			case sub <- delta:
			case <-end.C:
			}
			if !end.Stop() { // prevent new events
				<-end.C // drain the pending event
			}
		}(subscription)
	}
}
