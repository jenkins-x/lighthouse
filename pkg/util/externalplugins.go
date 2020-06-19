package util

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/jenkins-x/go-scm/pkg/hmac"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/sirupsen/logrus"
)

// ParseExternalPluginWebhook parses a webhook relayed to an external plugin
func ParseExternalPluginWebhook(req *http.Request, secretToken string) (scm.Webhook, error) {
	data, err := ioutil.ReadAll(
		io.LimitReader(req.Body, 10000000),
	)
	if err != nil {
		return nil, err
	}

	log := logrus.WithFields(map[string]interface{}{
		"URL":     req.URL,
		"Headers": req.Header,
		"Body":    string(data),
	})

	ua := req.Header.Get("User-Agent")
	if ua != LighthouseUserAgent {
		return nil, fmt.Errorf("unknown User-Agent %s, expected %s", ua, LighthouseUserAgent)
	}

	sig := req.Header.Get(LighthouseSignatureHeader)
	if sig == "" {
		return nil, scm.ErrSignatureInvalid
	}
	if !hmac.ValidatePrefix(data, []byte(secretToken), sig) {
		return nil, scm.ErrSignatureInvalid
	}

	kind := req.Header.Get(LighthouseWebhookKindHeader)
	if kind == "" {
		return nil, scm.MissingHeader{Header: LighthouseWebhookKindHeader}
	}
	var hook scm.Webhook
	switch scm.WebhookKind(kind) {
	case scm.WebhookKindBranch:
		hook = new(scm.BranchHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindCheckRun:
		hook = new(scm.CheckRunHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindCheckSuite:
		hook = new(scm.CheckSuiteHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindDeploy:
		hook = new(scm.DeployHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindDeploymentStatus:
		hook = new(scm.DeploymentStatusHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindFork:
		hook = new(scm.ForkHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindInstallation:
		hook = new(scm.InstallationHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindInstallationRepository:
		hook = new(scm.InstallationRepositoryHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindIssue:
		hook = new(scm.IssueHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindIssueComment:
		hook = new(scm.IssueCommentHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindLabel:
		hook = new(scm.LabelHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindPing:
		hook = new(scm.PingHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindPullRequest:
		hook = new(scm.PullRequestHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindPullRequestComment:
		hook = new(scm.PullRequestCommentHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindPush:
		hook = new(scm.PushHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindRelease:
		hook = new(scm.ReleaseHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindRepository:
		hook = new(scm.RepositoryHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindReview:
		hook = new(scm.ReviewHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindReviewCommentHook:
		hook = new(scm.ReviewCommentHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindStar:
		hook = new(scm.StarHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindStatus:
		hook = new(scm.StatusHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindTag:
		hook = new(scm.TagHook)
		err = json.Unmarshal(data, hook)
	case scm.WebhookKindWatch:
		hook = new(scm.WatchHook)
		err = json.Unmarshal(data, hook)
	default:
		log.WithField("Kind", kind).Warnf("unknown webhook")
		return nil, scm.UnknownWebhook{Event: kind}
	}
	if err != nil {
		return nil, err
	}

	return hook, nil
}
