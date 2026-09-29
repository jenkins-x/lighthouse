package scmauth

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// File names read from the secret directory.
const (
	AppIDFilename      = "appId"
	PrivateKeyFilename = "privateKey"
	UsernameFilename   = usernameFilename
)

// Options describes how to build a TokenSource.
type Options struct {
	Mode      Mode
	ServerURL string

	// BotName is optional for ModeGitHubApp, where it is derived from the app.
	BotName string

	// SecretDir holds per-owner token files for ModeOwnerTokens, or appId and
	// privateKey for ModeGitHubApp.
	SecretDir string

	StaticToken string
}

// New builds the TokenSource for the configured mode.
func New(opts Options) (TokenSource, error) {
	switch opts.Mode {
	case ModeStaticToken:
		if opts.StaticToken == "" {
			return nil, errors.New("no token available for staticToken mode")
		}
		return NewStaticTokenSource(opts.StaticToken, opts.BotName), nil

	case ModeOwnerTokens:
		if opts.SecretDir == "" {
			return nil, errors.New("no secret directory configured for ownerTokens mode")
		}
		botName := opts.BotName
		if botName == "" {
			if fromDir, err := readUsername(opts.SecretDir); err == nil {
				botName = fromDir
			}
		}
		return NewOwnerTokensSource(opts.ServerURL, opts.SecretDir, botName), nil

	case ModeGitHubApp:
		appCfg, err := loadAppConfig(opts)
		if err != nil {
			return nil, err
		}
		return NewAppTokenSource(*appCfg)

	case "":
		return nil, errors.New("no authentication mode configured")

	default:
		return nil, errors.Errorf("unknown authentication mode %q, want one of %s, %s or %s",
			opts.Mode, ModeStaticToken, ModeOwnerTokens, ModeGitHubApp)
	}
}

func loadAppConfig(opts Options) (*AppConfig, error) {
	if opts.SecretDir == "" {
		return nil, errors.New("no secret directory configured for githubApp mode")
	}

	appIDPath := filepath.Join(opts.SecretDir, AppIDFilename)
	/* #nosec */
	appIDRaw, err := os.ReadFile(appIDPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading GitHub App ID from %s", appIDPath)
	}
	appID, err := strconv.ParseInt(strings.TrimSpace(string(appIDRaw)), 10, 64)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing GitHub App ID from %s", appIDPath)
	}

	keyPath := filepath.Join(opts.SecretDir, PrivateKeyFilename)
	/* #nosec */
	privateKey, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading GitHub App private key from %s", keyPath)
	}

	botName := opts.BotName
	if botName == "" {
		if fromDir, err := readUsername(opts.SecretDir); err == nil {
			botName = fromDir
		}
	}

	return &AppConfig{
		AppID:      appID,
		PrivateKey: privateKey,
		ServerURL:  opts.ServerURL,
		BotName:    botName,
	}, nil
}

func readUsername(dir string) (string, error) {
	path := filepath.Join(dir, UsernameFilename)
	/* #nosec */
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return "", errors.Errorf("%s is empty", path)
	}
	return name, nil
}
