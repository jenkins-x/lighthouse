package foghorn

// We need to empty import all enabled plugins so that they will be linked into
// the foghorn binary.
import (
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/approve" // Import all enabled plugins.
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/assign"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/blockade"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/branchcleaner"
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
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/skip"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/stage"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/trigger"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/updateconfig"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/welcome"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/wip"
	_ "github.com/jenkins-x/lighthouse/pkg/plugins/yuks"
)
