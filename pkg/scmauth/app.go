package scmauth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const tokenRefreshBuffer = 5 * time.Minute

// AppConfig describes the GitHub App to authenticate as.
type AppConfig struct {
	AppID      int64
	PrivateKey []byte
	ServerURL  string

	// BotName, when set, must match the login derived from the app.
	BotName string
}

type appTokenSource struct {
	appID     int64
	serverURL string

	// jwtClient authenticates as the app itself, not as an installation.
	jwtClient *scm.Client

	configuredBotName string

	mu sync.Mutex
	// installations maps owner to installation ID; zero means not installed.
	installations map[string]int64
	tokens        map[string]*scm.InstallationToken
	botName       string
}

// NewAppTokenSource returns a TokenSource that mints GitHub App installation tokens.
func NewAppTokenSource(cfg AppConfig) (TokenSource, error) {
	if cfg.AppID == 0 {
		return nil, errors.New("no GitHub App ID configured")
	}
	if len(cfg.PrivateKey) == 0 {
		return nil, errors.New("no GitHub App private key configured")
	}
	key, err := ParsePrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "parsing GitHub App private key")
	}

	serverURL := cfg.ServerURL
	if serverURL == "" {
		serverURL = defaultGitHubServer
	}

	client, err := factory.NewClientWithTokenSource("github", serverURL,
		&jwtSource{appID: cfg.AppID, key: key})
	if err != nil {
		return nil, errors.Wrapf(err, "creating GitHub client for %s", serverURL)
	}

	return &appTokenSource{
		appID:             cfg.AppID,
		serverURL:         serverURL,
		jwtClient:         client,
		configuredBotName: cfg.BotName,
		installations:     map[string]int64{},
		tokens:            map[string]*scm.InstallationToken{},
	}, nil
}

const defaultGitHubServer = "https://github.com"

func (s *appTokenSource) Token(owner string) (string, error) {
	if owner == "" {
		return "", errors.New("owner is required to mint a GitHub App installation token")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if tok := s.tokens[owner]; tok != nil && !expiringSoon(tok, time.Now()) {
		return tok.Token, nil
	}

	id, err := s.installationID(owner)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	tok, _, err := s.jwtClient.Apps.CreateInstallationToken(ctx, id)
	if err != nil {
		return "", errors.Wrapf(err, "minting installation token for %s (installation %d)", owner, id)
	}
	if tok == nil || tok.Token == "" {
		return "", errors.Errorf("GitHub returned an empty installation token for %s", owner)
	}

	s.tokens[owner] = tok
	logrus.WithFields(logrus.Fields{
		"owner":          owner,
		"installationID": id,
		"expiresAt":      tok.ExpiresAt,
	}).Debug("minted GitHub App installation token")
	return tok.Token, nil
}

func (s *appTokenSource) RequiresOwner() bool { return true }

func expiringSoon(tok *scm.InstallationToken, now time.Time) bool {
	if tok.ExpiresAt == nil {
		return true
	}
	return now.Add(tokenRefreshBuffer).After(*tok.ExpiresAt)
}

// installationID must be called with the mutex held.
func (s *appTokenSource) installationID(owner string) (int64, error) {
	if id, ok := s.installations[owner]; ok {
		if id == 0 {
			return 0, errors.Errorf("GitHub App %d is not installed for owner %s", s.appID, owner)
		}
		return id, nil
	}

	ctx := context.Background()
	install, res, err := s.jwtClient.Apps.GetOrganisationInstallation(ctx, owner)
	if isNotFound(res, err) {
		install, res, err = s.jwtClient.Apps.GetUserInstallation(ctx, owner)
	}
	if isNotFound(res, err) {
		s.installations[owner] = 0
		return 0, errors.Errorf("GitHub App %d is not installed for owner %s", s.appID, owner)
	}
	if err != nil {
		return 0, errors.Wrapf(err, "looking up GitHub App installation for %s", owner)
	}
	if install == nil || install.ID == 0 {
		return 0, errors.Errorf("GitHub returned no installation for owner %s", owner)
	}

	s.installations[owner] = install.ID
	return install.ID, nil
}

func isNotFound(res *scm.Response, err error) bool {
	if err == nil {
		return false
	}
	if res != nil && res.Status == http.StatusNotFound {
		return true
	}
	return strings.Contains(err.Error(), scm.ErrNotFound.Error())
}

func (s *appTokenSource) BotName() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.botName != "" {
		return s.botName, nil
	}

	derived, err := s.derivedBotName()
	if err != nil {
		if s.configuredBotName == "" {
			return "", errors.Wrap(err, "resolving GitHub App bot name")
		}
		logrus.WithError(err).Warnf("could not derive GitHub App bot name, using configured %q", s.configuredBotName)
		s.botName = s.configuredBotName
		return s.botName, nil
	}

	if s.configuredBotName != "" && s.configuredBotName != derived {
		return "", errors.Errorf("configured bot name %q does not match GitHub App login %q",
			s.configuredBotName, derived)
	}

	s.botName = derived
	return s.botName, nil
}

// derivedBotName calls GET /app directly because go-scm has no binding for it.
func (s *appTokenSource) derivedBotName() (string, error) {
	res, err := s.jwtClient.Do(context.Background(), &scm.Request{
		Method: http.MethodGet,
		Path:   "app",
		Header: map[string][]string{"Accept": {"application/vnd.github+json"}},
	})
	if err != nil {
		return "", errors.Wrap(err, "requesting GET /app")
	}
	if res == nil {
		return "", errors.New("GET /app returned no response")
	}
	if res.Body != nil {
		defer func() { _ = res.Body.Close() }()
	}
	if res.Status < 200 || res.Status > 299 {
		return "", errors.Errorf("GET /app returned HTTP %d", res.Status)
	}
	if res.Body == nil {
		return "", errors.New("GET /app returned an empty body")
	}

	out := struct {
		Slug string `json:"slug"`
	}{}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", errors.Wrap(err, "decoding GET /app response")
	}
	if out.Slug == "" {
		return "", errors.New("GET /app returned no slug")
	}
	return out.Slug + "[bot]", nil
}

func (s *appTokenSource) UsesAppInstallationToken() bool { return true }
