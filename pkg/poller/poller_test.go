package poller_test

import (
	scmfake "github.com/jenkins-x/go-scm/scm/driver/fake"
	"testing"

	fbfake "github.com/jenkins-x/lighthouse/pkg/filebrowser/fake"
	"github.com/jenkins-x/lighthouse/pkg/poller"

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

	fakeNotifier := func(wrapper *scm.WebhookWrapper) error {
		hook := wrapper.PushHook
		assert.NotNil(t, hook, "no PushHook for webhook %#v", wrapper)
		if hook != nil {
			hooks = append(hooks, hook)
		}
		return nil
	}
	scmClient, _ := scmfake.NewDefault()
	fb := fbfake.NewFakeFileBrowser("test_data", true)

	p, err := poller.NewPollingController(repoNames, gitServer, scmClient, fb, fakeNotifier)
	require.NoError(t, err, "failed to create PollingController")

	p.PollReleases()

	require.Len(t, hooks, 1, "should have 1 PushHook")

	hook := hooks[0]
	assert.NotNil(t, hook, "no PushHook")
	t.Logf("created PushHook %#v", hook)
}
