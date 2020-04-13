/*
Copyright 2016 The Kubernetes Authors.

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

package webhook

// We need to empty import all enabled plugins so that they will be linked into
// any hook binary.
import (
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/approve" // Import all enabled plugins.
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/assign"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/blockade"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/cat"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/cherrypickunapproved"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/dog"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/help"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/hold"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/label"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/lgtm"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/lifecycle"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/milestone"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/milestonestatus"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/override"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/owners-label"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/pony"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/shrug"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/sigmention"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/size"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/stage"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/trigger"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/welcome"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/wip"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/yuks"
)
