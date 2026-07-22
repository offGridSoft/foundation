package durability

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/offGridSoft/foundation/v2026/contextcheck"
	"github.com/offGridSoft/foundation/v2026/core"
)

type DirectoryRequest struct {
	Root   core.AbsoluteDirectoryPath
	Target core.AbsoluteDirectoryPath
	Mode   fs.FileMode
}

func (r DirectoryRequest) Validate() error {
	if err := r.Root.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if err := r.Target.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if err := validateCreationMode(r.Mode); err != nil {
		return err
	}
	relative, err := filepath.Rel(r.Root.String(), r.Target.String())
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return core.ErrDurabilityContract
	}
	return nil
}

func EnsureDirectory(ctx context.Context, request DirectoryRequest) error {
	if err := contextcheck.Validate(ctx); err != nil {
		return err
	}
	if err := request.Validate(); err != nil {
		return err
	}
	relative, _ := filepath.Rel(request.Root.String(), request.Target.String())
	if relative == "." {
		return requireDirectory(request.Root.String())
	}
	parent := request.Root.String()
	if err := requireDirectory(parent); err != nil {
		return err
	}
	for element := range strings.SplitSeq(relative, string(filepath.Separator)) {
		if err := contextcheck.Validate(ctx); err != nil {
			return err
		}
		next := filepath.Join(parent, element)
		created, err := createDirectory(next, request.Mode)
		if err != nil {
			return err
		}
		if created {
			parentPath, _ := core.ParseAbsoluteDirectoryPath(parent)
			if err := SyncDirectory(parentPath); err != nil {
				return err
			}
		}
		parent = next
	}
	return nil
}

func createDirectory(path string, mode fs.FileMode) (bool, error) {
	err := os.Mkdir(path, mode)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, fs.ErrExist) {
		return false, fmt.Errorf("create durable directory: %w", err)
	}
	return false, requireDirectory(path)
}

func requireDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat durable directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("durable directory is not a directory: %w", core.ErrDurabilityContract)
	}
	return nil
}

func SyncDirectory(path core.AbsoluteDirectoryPath) error {
	if err := path.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if err := requireDirectory(path.String()); err != nil {
		return err
	}
	directory, err := openDirectoryForSync(path.String())
	if err != nil {
		return fmt.Errorf("open durable directory: %w", err)
	}
	syncErr := FullSync(directory)
	closeErr := directory.Close()
	return errors.Join(syncErr, closeErr)
}

func validateCreationMode(mode fs.FileMode) error {
	if mode.Perm() == 0 || mode&fs.ModeType != 0 || mode != mode.Perm() {
		return core.ErrDurabilityContract
	}
	return nil
}
