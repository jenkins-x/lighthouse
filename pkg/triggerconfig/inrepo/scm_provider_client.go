package inrepo

import "github.com/jenkins-x/go-scm/scm"

type scmProviderClient interface {
	GetRepositoryByFullName(string) (*scm.Repository, error)
	GetFile(string, string, string, string) ([]byte, error)
	ListFiles(string, string, string, string) ([]*scm.FileEntry, error)
}
