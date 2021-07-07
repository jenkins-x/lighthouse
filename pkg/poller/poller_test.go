package poller_test

import (
	fbfake "github.com/jenkins-x/lighthouse/pkg/filebrowser/fake"
	"github.com/jenkins-x/lighthouse/pkg/poller"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	repoNames = []string{"myorg/myrepo"}

	gitServer = "https://github.com"
)

func TestPoller(t *testing.T) {
	var hooks []*scm.PushHook

	fakeNotifier := func(webhook scm.Webhook) error {
		hook := webhook.(*scm.PushHook)
		assert.NotNil(t, hook, "no PushHook for webhook %#v", webhook)
		if hook != nil {
			hooks = append(hooks, hook)
		}
		return nil
	}
	fb := fbfake.NewFakeFileBrowser("test_data", true)

	p, err := poller.NewPollingController(repoNames, gitServer, fb, fakeNotifier)
	require.NoError(t, err, "failed to create PollingController")

	p.PollReleases()

	require.Len(t, hooks, 1, "should have 1 PushHook")

	hook := hooks[0]
	assert.NotNil(t, hook, "no PushHook")
	t.Logf("created PushHook %#v", hook)
}
