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

func EnsureDirectory(ctx context.Context, request DirectoryRequest) (resultErr error) {
	if err := contextcheck.Validate(ctx); err != nil {
		return err
	}
	if err := request.Validate(); err != nil {
		return err
	}
	root, err := openVerifiedDirectoryRoot(request.Root.String())
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	relative, _ := filepath.Rel(request.Root.String(), request.Target.String())
	if relative == "." {
		return nil
	}
	parent := "."
	for element := range strings.SplitSeq(relative, string(filepath.Separator)) {
		if err := contextcheck.Validate(ctx); err != nil {
			return err
		}
		next := filepath.Join(parent, element)
		created, err := createRootDirectory(root, next, request.Mode)
		if err != nil {
			return err
		}
		if created {
			if err := syncRootDirectory(root, parent); err != nil {
				return err
			}
		}
		parent = next
	}
	return nil
}

func openVerifiedDirectoryRoot(path string) (*os.Root, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("lstat durable root: %w", err)
	}
	if before.Mode()&fs.ModeSymlink != 0 || !before.IsDir() {
		return nil, fmt.Errorf("durable root is not a real directory: %w", core.ErrDurabilityContract)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, fmt.Errorf("open durable root: %w", err)
	}
	after, statErr := root.Stat(".")
	if statErr == nil && os.SameFile(before, after) {
		return root, nil
	}
	return nil, errors.Join(core.ErrDurabilityContract, statErr, root.Close())
}

func createRootDirectory(root *os.Root, path string, mode fs.FileMode) (bool, error) {
	err := root.Mkdir(path, mode)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, fs.ErrExist) {
		return false, fmt.Errorf("create durable directory: %w", err)
	}
	info, statErr := root.Lstat(path)
	if statErr != nil {
		return false, fmt.Errorf("lstat durable directory: %w", statErr)
	}
	if info.Mode()&fs.ModeSymlink != 0 || !info.IsDir() {
		return false, fmt.Errorf("durable directory element is not a real directory: %w", core.ErrDurabilityContract)
	}
	return false, nil
}

func syncRootDirectory(root *os.Root, path string) error {
	directory, err := root.Open(path)
	if err != nil {
		return fmt.Errorf("open rooted durable directory: %w", err)
	}
	info, statErr := directory.Stat()
	if statErr != nil || !info.IsDir() {
		return errors.Join(core.ErrDurabilityContract, statErr, directory.Close())
	}
	syncErr := FullSync(directory)
	return errors.Join(syncErr, directory.Close())
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
