package util_test

import (
	"testing"

	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/jenkins-x/lighthouse/pkg/util"
	"github.com/stretchr/testify/require"
)

func Test_ExternalPluginsForEvent_returns_empty_slice_for_nil_configuration(t *testing.T) {
	configAgent := &plugins.ConfigAgent{}
	plugins := util.ExternalPluginsForEvent(configAgent, util.LighthousePayloadTypeActivity, "myorg/myrepo")
	require.Empty(t, plugins)
}
