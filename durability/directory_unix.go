//go:build !windows

package durability

import (
	"errors"
	"os"
	"path/filepath"
)

func openDirectoryForSync(path string) (*os.File, error) {
	parent := filepath.Dir(path)
	name := filepath.Base(path)
	if parent == path {
		name = "."
	}
	root, err := os.OpenRoot(parent)
	if err != nil {
		return nil, err
	}
	directory, openErr := root.Open(name)
	closeErr := root.Close()
	if openErr == nil && closeErr == nil {
		return directory, nil
	}
	if directory != nil {
		closeErr = errors.Join(closeErr, directory.Close())
	}
	return nil, errors.Join(openErr, closeErr)
}
