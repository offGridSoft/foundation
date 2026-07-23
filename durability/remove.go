package durability

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/offGridSoft/foundation/v2026/contextcheck"
	"github.com/offGridSoft/foundation/v2026/core"
)

// TreeRemovalRequest binds recursive removal to one validated descendant of a
// separately validated root. The root itself can never be removed.
type TreeRemovalRequest struct {
	Root   core.AbsoluteDirectoryPath
	Target core.AbsoluteDirectoryPath
}

func (r TreeRemovalRequest) Validate() error {
	if err := r.Root.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if err := r.Target.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	return validateRemovalBoundary(r.Root.String(), r.Target.String())
}

// FileRemovalRequest binds one non-directory removal to a verified directory
// root. Terminal symlinks are removed as leaves and are never followed.
type FileRemovalRequest struct {
	Root   core.AbsoluteDirectoryPath
	Target core.AbsoluteFilePath
}

func (r FileRemovalRequest) Validate() error {
	if err := r.Root.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	if err := r.Target.Validate(); err != nil {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	return validateRemovalBoundary(r.Root.String(), r.Target.String())
}

func validateRemovalBoundary(root, target string) error {
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.Join(core.ErrDurabilityContract, err)
	}
	return nil
}

// RemoveTree removes a file or directory tree beneath a verified root. The
// standard library implementation streams directory names in fixed-size
// batches, does not follow a terminal symlink, and uses rooted operations to
// prevent traversal outside Root.
func RemoveTree(ctx context.Context, request TreeRemovalRequest) error {
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
	relative, _ := filepath.Rel(request.Root.String(), request.Target.String())
	info, err := root.Lstat(relative)
	if errors.Is(err, fs.ErrNotExist) {
		return root.Close()
	} else if err != nil {
		return errors.Join(core.ErrDurabilityContract, fmt.Errorf("lstat durable removal target: %w", err), root.Close())
	}
	if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return errors.Join(core.ErrDurabilityContract, root.Close())
	}
	removeErr := root.RemoveAll(relative)
	syncErr := syncRootDirectory(root, filepath.Dir(relative))
	return errors.Join(wrapTreeRemovalError(removeErr), syncErr, root.Close())
}

func wrapTreeRemovalError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("remove durable descendant: %w", err)
}

// RemoveFile removes one non-directory entry beneath a verified root and
// durably syncs its parent. A terminal symlink is unlinked without following
// its target.
func RemoveFile(ctx context.Context, request FileRemovalRequest) error {
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
	relative, _ := filepath.Rel(request.Root.String(), request.Target.String())
	info, err := root.Lstat(relative)
	if errors.Is(err, fs.ErrNotExist) {
		return root.Close()
	} else if err != nil {
		return errors.Join(core.ErrDurabilityContract, fmt.Errorf("lstat durable file removal target: %w", err), root.Close())
	}
	if info.IsDir() && info.Mode()&fs.ModeSymlink == 0 {
		return errors.Join(core.ErrDurabilityContract, root.Close())
	}
	removeErr := root.Remove(relative)
	syncErr := syncRootDirectory(root, filepath.Dir(relative))
	return errors.Join(wrapFileRemovalError(removeErr), syncErr, root.Close())
}

func wrapFileRemovalError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("remove durable file: %w", err)
}
