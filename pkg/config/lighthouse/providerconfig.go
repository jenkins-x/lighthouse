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

package lighthouse

// ProviderConfig is optionally used to configure information about the SCM provider being used. These values will be
// used as fallbacks if environment variables aren't set.
type ProviderConfig struct {
	// Kind is the go-scm driver name
	Kind string `json:"kind,omitempty"`
	// Server is the base URL for the provider, like https://github.com
	Server string `json:"server,omitempty"`
	// BotUser is the username on the provider the bot will use
	BotUser string `json:"botUser,omitempty"`
}
