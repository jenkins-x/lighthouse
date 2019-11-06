package githubapp

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ownerTokensDir struct {
	gitServer string
	dir       string
}

// NewOwnerTokensDir creates a new dir token scanner
func NewOwnerTokensDir(gitServer, dir string) *ownerTokensDir {
	return &ownerTokensDir{gitServer, dir}
}

// FindToken finds the token for the given owner
func (o *ownerTokensDir) FindToken(owner string) (string, error) {
	dir := o.dir
	ownerURL := util.UrlJoin(o.gitServer, owner)
	prefix := ownerURL + "="
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to list files in dir %s", dir)
	}
	for _, f := range files {
		localName := f.Name()
		if localName == "username" {
			continue
		}
		name := filepath.Join(dir, localName)
		logrus.Tracef("loading file %s", name)
		data, err := ioutil.ReadFile(name)
		if err != nil {
			return "", errors.Wrapf(err, "failed to load file %s", name)
		}
		text := strings.TrimSpace(string(data))
		if strings.HasPrefix(text, prefix) {
			return strings.TrimPrefix(text, prefix), nil
		}
	}
	return "", errors.Errorf("no github app secret found for owner URL %s", ownerURL)
}
