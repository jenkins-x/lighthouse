package filebrowser

import (
	"github.com/jenkins-x/lighthouse/pkg/git/v2"
	"github.com/pkg/errors"
	"os"
	"strings"
)

const (
	// GitHub the name used in a Uses Git URI to reference a git server
	GitHub = "github"

	// Lighthouse the name used in a Uses Git URI to reference the git server lighthouse is configured to run against
	Lighthouse = "lighthouse"

	// GitHubURL the github server URL
	GitHubURL = "https://github.com"
)

// FileBrowsers contains the file browsers for the supported git servers
type FileBrowsers struct {
	cache map[string]Interface
}

// NewFileBrowsers creates a new file browsers service for the lighthouse git server URL and file browser
// if the current server URL is not github.com then creates a second file browser so that we can use the uses: Git URI
// syntax for accessing github resources as well as to access the lighthouse git server too
func NewFileBrowsers(serverURL string, fb Interface) (*FileBrowsers, error) {
	if serverURL == "" {
		serverURL = GitHubURL
	}
	isGitHub := strings.TrimSuffix(serverURL, "/") == GitHubURL

	answer := &FileBrowsers{
		cache: map[string]Interface{
			Lighthouse: fb,
		},
	}

	// lets see if we have a custom server name we want to support
	serverName := os.Getenv("GIT_NAME")
	if serverName != "" && serverName != GitHub {
		answer.cache[serverName] = fb
	}
	var githubBrowser Interface
	if isGitHub {
		githubBrowser = fb
	} else {
		// lets create a github browser\
		configureOpts := func(opts *git.ClientFactoryOpts) {
			opts.Token = func() []byte {
				return []byte(os.Getenv("GITHUB_TOKEN"))
			}
			opts.GitUser = func() (name, email string, err error) {
				name = os.Getenv("GITHUB_USER_NAME")
				email = os.Getenv("GITHUB_USER_EMAIL")
				return
			}
			opts.Username = func() (login string, err error) {
				login = os.Getenv("GITHUB_USER")
				return
			}
			opts.Host = "github.com"
			opts.Scheme = "https"
		}
		githubFactory, err := git.NewClientFactory(configureOpts)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create git client factory for github")
		}
		githubBrowser = NewFileBrowserFromGitClient(githubFactory)
	}
	answer.cache[GitHub] = githubBrowser
	return answer, nil
}

// GitHubFileBrowser returns a git file browser for github.com
func (f *FileBrowsers) GitHubFileBrowser() Interface {
	return f.GetFileBrowser(GitHub)
}

// LighthouseGitFileBrowser returns the git file browser for the git server that lighthouse is configured against
func (f *FileBrowsers) LighthouseGitFileBrowser() Interface {
	return f.GetFileBrowser(Lighthouse)
}

// GetFileBrowser returns the file browser for the given git server name.
func (f *FileBrowsers) GetFileBrowser(name string) Interface {
	return f.cache[name]
}
