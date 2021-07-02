package fake

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/lighthouse/pkg/util"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/pkg/errors"
)

type fakeFileBrowser struct {
	dir                      string
	multiRepo                bool
	mainAndCurrentBranchRefs []string
}

// NewFakeFileBrowser a simple fake provider pointing at a folder
func NewFakeFileBrowser(dir string, multiRepo bool) filebrowser.Interface {
	return &fakeFileBrowser{
		dir:                      dir,
		multiRepo:                multiRepo,
		mainAndCurrentBranchRefs: []string{"main"},
	}
}

func (f *fakeFileBrowser) GetMainAndCurrentBranchRefs(owner, repo, ref string) ([]string, error) {
	return f.mainAndCurrentBranchRefs, nil
}

func (f *fakeFileBrowser) GetFile(owner, repo, path, ref string, fc filebrowser.FetchCache) ([]byte, error) {
	fileName := f.getPath(owner, repo, path, ref)
	exists, err := util.FileExists(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check if file exists %s", fileName)
	}
	if !exists {
		return nil, nil
	}
	/* #nosec */
	return ioutil.ReadFile(fileName)
}

func (f *fakeFileBrowser) getPath(owner, repo, path, ref string) string {
	if f.multiRepo {
		refs := []string{ref}
		// lets handle main or master as the default branch name
		if ref == "main" {
			refs = append(refs, "master")
		}
		for _, r := range refs {
			p := filepath.Join(f.dir, owner, repo, "refs", r, path)
			_, err := os.Stat(p)
			if err == nil {
				return p
			}
		}
		return filepath.Join(f.dir, owner, repo, path)
	}
	return filepath.Join(f.dir, path)
}

func (f *fakeFileBrowser) ListFiles(owner, repo, path, ref string, fc filebrowser.FetchCache) ([]*scm.FileEntry, error) {
	dir := f.getPath(owner, repo, path, ref)
	exists, err := util.DirExists(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check dir exists %s", dir)
	}
	if !exists {
		return nil, nil
	}
	fileNames, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read dir %s", dir)
	}

	var answer []*scm.FileEntry
	for _, f := range fileNames {
		name := f.Name()
		t := "file"
		if f.IsDir() {
			t = "dir"
		}
		childPath := filepath.Join(path, name)
		answer = append(answer, &scm.FileEntry{
			Name: name,
			Path: childPath,
			Type: t,
			Size: int(f.Size()),
			Sha:  ref,
			Link: "file://" + childPath,
		})
	}
	return answer, nil
}
