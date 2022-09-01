package filebrowser

import "github.com/jenkins-x/go-scm/scm"

// Interface an interface to represent browsing files in a repository
type Interface interface {
	// GetMainAndCurrentBranchRefs returns the main branch then the ref value if its different
	GetMainAndCurrentBranchRefs(owner, repo, ref string) ([]string, error)

	// GetFile returns a file from the given path in the repository with the given sha
	GetFile(owner, repo, path, ref string, fc FetchCache) ([]byte, error)

	// ListFiles returns the file and directory entries in the given path in the repository with the given sha
	ListFiles(owner, repo, path, ref string, fc FetchCache) ([]*scm.FileEntry, error)

	// WithDir processes the given repository and reference at the given directory
	WithDir(owner, repo, ref string, fc FetchCache, sparseCheckoutPatterns []string, f func(dir string) error) error
}
