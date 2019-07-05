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

package hook

// We need to empty import all enabled plugins so that they will be linked into
// any hook binary.
import (
	_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/approve" // Import all enabled plugins.
	_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/assign"
	_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/blockade"
	_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/cat"
	_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/dog"
	_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/help"
	_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/hold"
	_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/pony"
	_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/yuks"
	/*
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/blunderbuss"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/branchcleaner"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/buildifier"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/cherrypickunapproved"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/cla"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/dco"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/docs-no-retest"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/golint"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/heart"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/label"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/lgtm"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/lifecycle"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/milestone"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/milestonestatus"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/override"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/owners-label"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/releasenote"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/require-matching-label"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/requiresig"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/shrug"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/sigmention"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/size"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/skip"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/slackevents"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/stage"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/trigger"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/updateconfig"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/verify-owners"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/welcome"
		_ "github.com/jenkins-x/lighthouse/pkg/prow/plugins/wip"
	*/)
