package filebrowser

import (
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/pkg/errors"
)

type scmFileBrowser struct {
	scmClient scmProviderClient
}

// NewFileBrowserFromScmClient creates a new file browser from an Scm client
func NewFileBrowserFromScmClient(scmClient scmProviderClient) Interface {
	return &scmFileBrowser{scmClient: scmClient}
}

func (f *scmFileBrowser) GetMainAndCurrentBranchRefs(owner, repo, eventRef string) ([]string, error) {
	fullName := scm.Join(owner, repo)
	repository, err := f.scmClient.GetRepositoryByFullName(fullName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find repository %s", fullName)
	}
	mainBranch := repository.Branch
	if mainBranch == "" {
		mainBranch = "master"
	}

	refs := []string{mainBranch}

	eventRef = strings.TrimPrefix(eventRef, "refs/heads/")
	eventRef = strings.TrimPrefix(eventRef, "refs/tags/")
	if eventRef != mainBranch && eventRef != "" {
		refs = append(refs, eventRef)
	}
	return refs, nil
}

func (f *scmFileBrowser) GetFile(owner, repo, path, ref string) ([]byte, error) {
	return f.scmClient.GetFile(owner, repo, path, ref)
}

func (f *scmFileBrowser) ListFiles(owner, repo, path, ref string) ([]*scm.FileEntry, error) {
	return f.scmClient.ListFiles(owner, repo, path, ref)
}
