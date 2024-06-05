package poller_test

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	scmfake "github.com/jenkins-x/go-scm/scm/driver/fake"

	fbfake "github.com/jenkins-x/lighthouse/pkg/filebrowser/fake"
	"github.com/jenkins-x/lighthouse/pkg/poller"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	repoNames             = []string{"myorg/myrepo"}
	gitServer             = "https://github.com"
	testDataDir           = "test_data"
	contextMatchPattern   = "^Lighthouse$"
	statusLabel           = "Jenkins"
	requireReleaseSuccess = true
)

func TestPollerReleases(t *testing.T) {
	var hooks []*scm.PushHook

	fakeNotifier := func(wrapper *scm.WebhookWrapper) error {
		hook := wrapper.PushHook
		assert.NotNil(t, hook, "no PushHook for webhook %#v", wrapper)
		if hook != nil {
			hooks = append(hooks, hook)
		}
		return nil
	}
	scmClient, fakeData := scmfake.NewDefault()
	fb := fbfake.NewFakeFileBrowser(testDataDir, true)

	contextMatchPatternCompiled, err := regexp.Compile(contextMatchPattern)
	require.NoErrorf(t, err, "failed to compile context match pattern \"%s\"", contextMatchPattern)

	// Load fake status with label that doesn't match our context match pattern
	c := exec.Command("git", "rev-parse", "HEAD")
	c.Dir = testDataDir
	out, err := c.CombinedOutput()
	require.NoError(t, err, "failed to get latest git commit sha")
	sha := strings.TrimSpace(string(out))
	fakeData.Statuses = map[string][]*scm.Status{sha: {{Label: statusLabel}}}

	p, err := poller.NewPollingController(repoNames, gitServer, scmClient, contextMatchPatternCompiled, requireReleaseSuccess, fb, fakeNotifier)
	require.NoError(t, err, "failed to create PollingController")

	p.PollReleases()

	require.Len(t, hooks, 1, "should have 1 PushHook")

	hook := hooks[0]
	assert.NotNil(t, hook, "no PushHook")
	t.Logf("created PushHook %#v", hook)
}

func TestPollerPullRequests(t *testing.T) {
	var prHooks []*scm.PullRequestHook
	var pushHooks []*scm.PushHook

	fakeNotifier := func(wrapper *scm.WebhookWrapper) error {
		if wrapper.PullRequestHook != nil {
			prHooks = append(prHooks, wrapper.PullRequestHook)
		} else if wrapper.PushHook != nil {
			pushHooks = append(pushHooks, wrapper.PushHook)
		} else {
			assert.Fail(t, "unknown webhook %v", wrapper)
		}
		return nil
	}
	scmClient, fakeData := scmfake.NewDefault()
	fb := fbfake.NewFakeFileBrowser("test_data", true)

	contextMatchPatternCompiled, err := regexp.Compile(contextMatchPattern)
	require.NoErrorf(t, err, "failed to compile context match pattern \"%s\"", contextMatchPattern)

	prNumber := 123
	fullName := repoNames[0]
	owner, repo := scm.Split(fullName)
	sha := "mysha1234"
	fakeData.PullRequests[prNumber] = &scm.PullRequest{
		Number: prNumber,
		Base: scm.PullRequestBranch{
			Ref: "master",
			Repo: scm.Repository{
				Namespace: owner,
				Name:      repo,
				FullName:  fullName,
			},
		},
		Head: scm.PullRequestBranch{
			Sha: sha,
		},
		Title:  "fix: some stuff",
		Body:   "the PR comment",
		Closed: false,
		State:  "open",
		Sha:    sha,
	}
	// Load fake status with label that doesn't match our context match pattern
	fakeData.Statuses = map[string][]*scm.Status{sha: {{Label: statusLabel}}}

	p, err := poller.NewPollingController(repoNames, gitServer, scmClient, contextMatchPatternCompiled, requireReleaseSuccess, fb, fakeNotifier)
	require.NoError(t, err, "failed to create PollingController")

	p.PollPullRequests()

	require.Len(t, prHooks, 1, "should have 1 PullRequestHook")
	require.Len(t, pushHooks, 1, "should have 1 PushHook")

	hook := prHooks[0]
	assert.NotNil(t, hook, "no PullRequestHook")
	t.Logf("created PullRequestHook %#v", hook)

	pushHook := pushHooks[0]
	assert.NotNil(t, pushHook, "no PushHook")
	t.Logf("created PushHook %#v", pushHook)
}

func TestListAllStatuses(t *testing.T) {
	sha := "mysha1234"
	fullName := repoNames[0]

	scmClient, fakeData := scmfake.NewDefault()
	fakeData.Statuses = map[string][]*scm.Status{sha: {}}
	// Populate fake data with more entries than the page size to ensure there
	// are multiple pages of data
	statusesCount := poller.StatusesPageSize + 1
	for i := 0; i < statusesCount; i++ {
		status := scm.Status{}
		fakeData.Statuses[sha] = append(fakeData.Statuses[sha], &status)
	}

	p, err := poller.NewPollingController(repoNames, gitServer, scmClient, nil, requireReleaseSuccess, nil, nil)
	require.NoError(t, err, "failed to create PollingController")

	statuses, err := p.ListAllStatuses(context.TODO(), fullName, sha)
	assert.Nil(t, err, "failed to list statuses")
	require.Len(t, statuses, statusesCount, fmt.Sprintf("should have %d statuses", statusesCount))
}
