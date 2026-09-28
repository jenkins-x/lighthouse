package scmauth

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const usernameFilename = "username"

// OwnerTokensDir finds per-owner tokens in files of the form "<serverURL>/<owner>=<token>".
type OwnerTokensDir struct {
	gitServer string
	dir       string
}

// NewOwnerTokensDir creates a scanner over dir, keyed against gitServer.
func NewOwnerTokensDir(gitServer, dir string) *OwnerTokensDir {
	return &OwnerTokensDir{gitServer, dir}
}

// FindToken returns the token for the given owner.
func (o *OwnerTokensDir) FindToken(owner string) (string, error) {
	ownerURL := urlJoin(o.gitServer, owner)
	prefix := ownerURL + "="
	files, err := os.ReadDir(o.dir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to list files in dir %s", o.dir)
	}
	for _, f := range files {
		localName := f.Name()
		// Skip the ..data symlink and timestamped directory of a projected secret.
		if f.IsDir() || localName == usernameFilename || strings.HasPrefix(localName, ".") {
			continue
		}
		name := filepath.Join(o.dir, localName)

		logrus.Tracef("loading file %s", name)
		/* #nosec */
		data, err := os.ReadFile(name)
		if err != nil {
			return "", errors.Wrapf(err, "failed to load file %s", name)
		}
		text := strings.TrimSpace(string(data))
		if strings.HasPrefix(text, prefix) {
			return strings.TrimPrefix(text, prefix), nil
		}
	}
	return "", errors.Errorf("no token found for owner URL %s in %s", ownerURL, o.dir)
}

func urlJoin(paths ...string) string {
	var b strings.Builder
	last := len(paths) - 1
	for i, path := range paths {
		p := path
		if i > 0 {
			b.WriteString("/")
			p = strings.TrimPrefix(p, "/")
		}
		if i < last {
			p = strings.TrimSuffix(p, "/")
		}
		b.WriteString(p)
	}
	return b.String()
}
