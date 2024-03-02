package util

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	goscmhmac "github.com/jenkins-x/go-scm/pkg/hmac"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/apis/lighthouse/v1alpha1"
	"github.com/jenkins-x/lighthouse/pkg/plugins"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ParseExternalPluginEvent parses a webhook relayed to an external plugin
func ParseExternalPluginEvent(req *http.Request, secretToken string) (scm.Webhook, *v1alpha1.ActivityRecord, error) {
	data, err := io.ReadAll(
		io.LimitReader(req.Body, 10000000),
	)
	if err != nil {
		return nil, nil, err
	}

	log := logrus.WithFields(map[string]interface{}{
		"URL":     req.URL,
		"Headers": req.Header,
		"Body":    string(data),
	})

	ua := req.Header.Get("User-Agent")
	if ua != LighthouseUserAgent {
		return nil, nil, fmt.Errorf("unknown User-Agent %s, expected %s", ua, LighthouseUserAgent)
	}

	sig := req.Header.Get(LighthouseSignatureHeader)
	if sig == "" {
		return nil, nil, scm.ErrSignatureInvalid
	}
	if !goscmhmac.ValidatePrefix(data, []byte(secretToken), sig) {
		return nil, nil, scm.ErrSignatureInvalid
	}

	payloadType := req.Header.Get(LighthousePayloadTypeHeader)
	switch payloadType {
	case LighthousePayloadTypeWebhook:
		hook, err := parseWebhook(log, req, data)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parsing webhook")
		}
		return hook, nil, nil
	case LighthousePayloadTypeActivity:
		ar := new(v1alpha1.ActivityRecord)
		err := json.Unmarshal(data, ar)
		if err != nil {
			return nil, nil, errors.Wrap(err, "parsing activity")
		}
		return nil, ar, nil
	default:
		return nil, nil, fmt.Errorf("unknown Lighthouse payload type %s", payloadType)
	}
}

func parseWebhook(l *logrus.Entry, req *http.Request, data []byte) (scm.Webhook, error) {
	kind := req.Header.Get(LighthouseWebhookKindHeader)
	if kind == "" {
		return nil, scm.MissingHeader{Header: LighthouseWebhookKindHeader}
	}
	var hook scm.Webhook
	var err error
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
		l.WithField("Kind", kind).Warnf("unknown webhook")
		return nil, scm.UnknownWebhook{Event: kind}
	}
	if err != nil {
		return nil, err
	}

	return hook, nil
}

// callExternalPlugins dispatches the provided payload to the external plugins.
func callExternalPlugins(l *logrus.Entry, externalPlugins []plugins.ExternalPlugin, payload []byte, headers http.Header, hmacToken string, wg *sync.WaitGroup) {
	headers.Set("User-Agent", LighthouseUserAgent)
	mac := hmac.New(sha256.New, []byte(hmacToken))
	_, err := mac.Write(payload)
	if err != nil {
		l.WithError(err).Error("Unable to generate signature for relayed payload")
		return
	}
	sum := mac.Sum(nil)
	signature := "sha256=" + hex.EncodeToString(sum)
	headers.Set(LighthouseSignatureHeader, signature)
	for _, p := range externalPlugins {
		wg.Add(1)
		go func(p plugins.ExternalPlugin) {
			defer wg.Done()
			if err := dispatch(p.Endpoint, payload, headers); err != nil {
				l.WithError(err).WithField("external-plugin", p.Name).Warning("Error dispatching event to external plugin.")
			} else {
				l.WithField("external-plugin", p.Name).Info("Dispatched event to external plugin")
			}
		}(p)
	}
}

// CallExternalPluginsWithActivityRecord dispatches the provided activity record to the external plugins.
func CallExternalPluginsWithActivityRecord(l *logrus.Entry, externalPlugins []plugins.ExternalPlugin, activity *v1alpha1.ActivityRecord, hmacToken string, wg *sync.WaitGroup) {
	headers := http.Header{}
	headers.Set(LighthousePayloadTypeHeader, LighthousePayloadTypeActivity)
	payload, err := json.Marshal(activity)
	if err != nil {
		l.WithError(err).Errorf("Unable to marshal activity for relaying to external plugins. Activity is: %v", activity)
		return
	}
	callExternalPlugins(l, externalPlugins, payload, headers, hmacToken, wg)
}

// CallExternalPluginsWithWebhook dispatches the provided webhook to the external plugins.
func CallExternalPluginsWithWebhook(l *logrus.Entry, externalPlugins []plugins.ExternalPlugin, webhook scm.Webhook, hmacToken string, wg *sync.WaitGroup) {
	headers := http.Header{}
	headers.Set(LighthouseWebhookKindHeader, string(webhook.Kind()))
	headers.Set(LighthousePayloadTypeHeader, LighthousePayloadTypeWebhook)
	payload, err := json.Marshal(webhook)
	if err != nil {
		l.WithError(err).Errorf("Unable to marshal webhook for relaying to external plugins. Webhook is: %v", webhook)
		return
	}
	callExternalPlugins(l, externalPlugins, payload, headers, hmacToken, wg)
}

// dispatch creates a new request using the provided payload and headers
// and dispatches the request to the provided endpoint.
func dispatch(endpoint string, payload []byte, h http.Header) error {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header = h
	var resp *http.Response
	backoff := 100 * time.Millisecond
	maxRetries := 5

	c := &http.Client{}
	for retries := 0; retries < maxRetries; retries++ {
		resp, err = c.Do(req)
		if err == nil {
			break
		}
		time.Sleep(backoff)
		backoff *= 2
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("response has status %q and body %q", resp.Status, string(rb))
	}
	return nil
}

// ExternalPluginsForEvent returns whether there are any external plugins that need to
// get the present event.
func ExternalPluginsForEvent(pluginConfig *plugins.ConfigAgent, eventKind string, srcRepo string, disabledExternalPlugins []string) []plugins.ExternalPlugin {
	var matching []plugins.ExternalPlugin
	if pluginConfig.Config() == nil {
		return matching
	}

	srcOrg := strings.Split(srcRepo, "/")[0]

	for repo, extPlugins := range pluginConfig.Config().ExternalPlugins {
		// Make sure the repositories match
		if repo != srcRepo && repo != srcOrg {
			continue
		}

		// Make sure the events match
		for _, p := range extPlugins {
			if StringArrayIndex(disabledExternalPlugins, p.Name) >= 0 {
				continue
			}
			if len(p.Events) == 0 {
				matching = append(matching, p)
			} else {
				for _, et := range p.Events {
					if et == eventKind || et == eventKind+"s" {
						matching = append(matching, p)
						break
					}
				}
			}
		}
	}
	return matching
}
