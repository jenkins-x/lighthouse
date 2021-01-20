package fake

import (
	"io/ioutil"
	"path/filepath"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/lighthouse/pkg/filebrowser"
	"github.com/pkg/errors"
)

type fakeFileBrowser struct {
	dir                      string
	mainAndCurrentBranchRefs []string
}

// NewFakeFileBrowser a simple fake provider pointing at a folder
func NewFakeFileBrowser(dir string) filebrowser.Interface {
	return &fakeFileBrowser{
		dir:                      dir,
		mainAndCurrentBranchRefs: []string{"main"},
	}
}

func (f *fakeFileBrowser) GetMainAndCurrentBranchRefs(owner, repo, ref string) ([]string, error) {
	return f.mainAndCurrentBranchRefs, nil
}

func (f *fakeFileBrowser) GetFile(owner, repo, path, ref string) ([]byte, error) {
	fileName := filepath.Join(f.dir, path)
	return ioutil.ReadFile(fileName)
}

func (f *fakeFileBrowser) ListFiles(owner, repo, path, ref string) ([]*scm.FileEntry, error) {
	dir := filepath.Join(f.dir, path)
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
		path := filepath.Join(dir, name)
		answer = append(answer, &scm.FileEntry{
			Name: name,
			Path: path,
			Type: t,
			Size: int(f.Size()),
			Sha:  ref,
			Link: "file://" + path,
		})
	}
	return answer, nil
}
