package filebrowser

import (
	"strings"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/pkg/errors"
)

type scmProviderClient interface {
	GetRepositoryByFullName(string) (*scm.Repository, error)
	GetFile(string, string, string, string) ([]byte, error)
	ListFiles(string, string, string, string) ([]*scm.FileEntry, error)
}

// GetMainAndCurrentBranchRefs a function to return the main branch then the eventRef
func GetMainAndCurrentBranchRefs(scmClient scmProviderClient, fullName, eventRef string) ([]string, error) {
	repository, err := scmClient.GetRepositoryByFullName(fullName)
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
