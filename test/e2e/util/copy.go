package util

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// Copy copies the the src directory recursively to the destination directory.
// If template is true files ending in '.template' will be templated and copied to the destination directory
// with the .template suffix removed. The context for templating is the current environment.
//
// credit https://gist.github.com/r0l1/92462b38df26839a3ca324697c8cba04
func Copy(src string, dst string, template bool) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = Copy(srcPath, dstPath, template)
			if err != nil {
				return err
			}
		} else {
			// Skip symlinks.
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			if template && strings.HasSuffix(srcPath, ".template") {
				dstPath = strings.TrimSuffix(dstPath, ".template")
				err = TemplateFile(srcPath, dstPath, Env())
				if err != nil {
					return err
				}
			} else {
				err = CopyFile(srcPath, dstPath)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// CopyFile copies the specified source file to the specified destination.
//
// credit https://gist.github.com/r0l1/92462b38df26839a3ca324697c8cba04
func CopyFile(src, dst string) error {
	in, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close()
	}()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	err = out.Sync()
	if err != nil {
		return err
	}

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return err
	}

	return nil
}

// TemplateFile templates the specified file and writes it to the specified destination.
func TemplateFile(templateFile string, destFile string, context map[string]string) error {
	t, err := template.ParseFiles(templateFile)
	if err != nil {
		return errors.Wrapf(err, "unable to parse template file %s", templateFile)
	}

	f, err := os.Create(destFile)
	if err != nil {
		return errors.Wrapf(err, "unable to create file %s", destFile)
	}

	err = t.Execute(f, context)
	if err != nil {
		return errors.Wrapf(err, "unable to template %s", templateFile)
	}
	err = f.Close()
	if err != nil {
		return errors.Wrapf(err, "unable to close templated file %s", destFile)
	}

	return nil
}

func envKeys() []string {
	data := os.Environ()
	var keys []string
	for _, item := range data {
		tokens := strings.Split(item, "=")
		keys = append(keys, tokens[0])
	}
	return keys
}

// Env returns the current environment as map.
func Env() map[string]string {
	keys := envKeys()
	items := make(map[string]string)
	for _, key := range keys {
		items[key] = os.Getenv(key)
	}
	return items
}
