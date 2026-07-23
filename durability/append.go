package durability

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/offGridSoft/foundation/v2026/contextcheck"
	"github.com/offGridSoft/foundation/v2026/core"
)

type AppendRequest struct {
	Root             core.AbsoluteDirectoryPath
	Target           core.AbsoluteFilePath
	DirectoryMode    fs.FileMode
	FileMode         fs.FileMode
	MaximumFileBytes core.ByteCount
}

func (r AppendRequest) Validate() error {
	if err := r.Target.Validate(); err != nil {
		return fmt.Errorf(core.ErrFmtDurableAppendRequest, errors.Join(core.ErrDurabilityContract, err))
	}
	if err := validateCreationMode(r.DirectoryMode); err != nil {
		return fmt.Errorf(core.ErrFmtDurableAppendRequest, err)
	}
	if err := validateCreationMode(r.FileMode); err != nil {
		return fmt.Errorf(core.ErrFmtDurableAppendRequest, err)
	}
	if err := r.MaximumFileBytes.Validate(); err != nil {
		return fmt.Errorf(core.ErrFmtDurableAppendRequest, errors.Join(core.ErrDurabilityContract, err))
	}
	if _, err := r.MaximumFileBytes.Int64(); err != nil {
		return fmt.Errorf(core.ErrFmtDurableAppendRequest, errors.Join(core.ErrDurabilityContract, err))
	}
	_, err := r.directoryRequest()
	return err
}

func (r AppendRequest) directoryRequest() (DirectoryRequest, error) {
	directory, err := core.ParseAbsoluteDirectoryPath(filepath.Dir(r.Target.String()))
	if err != nil {
		return DirectoryRequest{}, fmt.Errorf(core.ErrFmtDurableAppendRequest, errors.Join(core.ErrDurabilityContract, err))
	}
	request := DirectoryRequest{Root: r.Root, Target: directory, Mode: r.DirectoryMode}
	if err := request.Validate(); err != nil {
		return DirectoryRequest{}, fmt.Errorf(core.ErrFmtDurableAppendRequest, err)
	}
	return request, nil
}

type appendState uint8

const (
	appendStateUnknown appendState = iota
	appendStateOpen
	appendStateClosed
)

func (s appendState) valid() bool {
	return s == appendStateOpen || s == appendStateClosed
}

type appendFile interface {
	io.Writer
	Stat() (fs.FileInfo, error)
	SyncStable() error
	Close() error
}

type AppendWriter struct {
	file    appendFile
	request AppendRequest
	state   appendState
	mu      sync.Mutex
}

func (w *AppendWriter) Validate() error {
	if w == nil || w.file == nil || !w.state.valid() {
		return core.ErrDurabilityContract
	}
	return w.request.Validate()
}

func OpenAppend(ctx context.Context, request AppendRequest) (*AppendWriter, error) {
	if err := contextcheck.Validate(ctx); err != nil {
		return nil, err
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	directoryRequest, _ := request.directoryRequest()
	if err := EnsureDirectory(ctx, directoryRequest); err != nil {
		return nil, err
	}
	file, created, err := openAppendTarget(request)
	if err != nil {
		return nil, err
	}
	writer := &AppendWriter{file: file, request: request, state: appendStateOpen}
	if err := writer.validateCurrentSize(); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if created {
		if err := file.SyncStable(); err != nil {
			return nil, errors.Join(fmt.Errorf(core.ErrFmtDurableAppendSync, err), file.Close())
		}
	}
	if err := syncAppendParent(request.Target); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return writer, nil
}

func openAppendTarget(request AppendRequest) (appendFile, bool, error) {
	info, err := os.Lstat(request.Target.String())
	if err == nil {
		file, openErr := openExistingAppendTarget(request, info)
		return file, false, openErr
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, false, fmt.Errorf(core.ErrFmtDurableAppendInspect, err)
	}
	file, err := os.OpenFile(request.Target.String(), os.O_APPEND|os.O_CREATE|os.O_EXCL|os.O_WRONLY, request.FileMode)
	if errors.Is(err, fs.ErrExist) {
		info, statErr := os.Lstat(request.Target.String())
		if statErr != nil {
			return nil, false, fmt.Errorf(core.ErrFmtDurableAppendInspect, statErr)
		}
		file, openErr := openExistingAppendTarget(request, info)
		return file, false, openErr
	}
	if err != nil {
		return nil, false, fmt.Errorf(core.ErrFmtDurableAppendOpen, err)
	}
	owned := true
	defer func() {
		if owned {
			_ = file.Close()
		}
	}()
	wrapped := operatingFile{File: file}
	if err := file.Chmod(request.FileMode); err != nil {
		return nil, false, fmt.Errorf(core.ErrFmtDurableAppendOpen, err)
	}
	owned = false
	return wrapped, true, nil
}

func openExistingAppendTarget(request AppendRequest, before fs.FileInfo) (appendFile, error) {
	if before.Mode()&fs.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf(core.ErrFmtDurableAppendOpen, core.ErrDurabilityContract)
	}
	file, err := openAppendFile(request.Target.String(), request.FileMode)
	if err != nil {
		return nil, fmt.Errorf(core.ErrFmtDurableAppendOpen, err)
	}
	owned := true
	defer func() {
		if owned {
			_ = file.Close()
		}
	}()
	after, statErr := file.Stat()
	if statErr != nil || !after.Mode().IsRegular() || !os.SameFile(before, after) {
		return nil, fmt.Errorf(core.ErrFmtDurableAppendInspect, errors.Join(core.ErrDurabilityContract, statErr))
	}
	owned = false
	return operatingFile{File: file}, nil
}

func syncAppendParent(target core.AbsoluteFilePath) error {
	directory, err := core.ParseAbsoluteDirectoryPath(filepath.Dir(target.String()))
	if err != nil {
		return err
	}
	if err := CommitDirectory(directory); err != nil {
		return fmt.Errorf(core.ErrFmtDurableAppendSync, err)
	}
	return nil
}

func (w *AppendWriter) Write(data []byte) (int, error) {
	if w == nil {
		return 0, core.ErrDurabilityContract
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.validateOpen(); err != nil {
		return 0, err
	}
	if err := w.ensureWriteFits(len(data)); err != nil {
		return 0, err
	}
	n, err := w.file.Write(data)
	if n != len(data) {
		return n, errors.Join(core.ErrDurableShortWrite, err, io.ErrShortWrite)
	}
	if err != nil {
		return n, fmt.Errorf(core.ErrFmtDurableAppendWrite, err)
	}
	return n, nil
}

func (w *AppendWriter) Sync(ctx context.Context) error {
	if w == nil {
		return core.ErrDurabilityContract
	}
	if err := contextcheck.Validate(ctx); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.validateOpen(); err != nil {
		return err
	}
	if err := w.file.SyncStable(); err != nil {
		return fmt.Errorf(core.ErrFmtDurableAppendSync, err)
	}
	return nil
}

func (w *AppendWriter) Close() error {
	if w == nil {
		return core.ErrDurabilityContract
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.validateOpen(); err != nil {
		return err
	}
	w.state = appendStateClosed
	return errors.Join(
		wrapAppendError(core.ErrFmtDurableAppendSync, w.file.SyncStable()),
		wrapAppendError(core.ErrFmtDurableAppendClose, w.file.Close()),
	)
}

func (w *AppendWriter) validateOpen() error {
	if err := w.Validate(); err != nil {
		return err
	}
	if w.state != appendStateOpen {
		return core.ErrDurabilityContract
	}
	return nil
}

func (w *AppendWriter) validateCurrentSize() error {
	info, err := w.file.Stat()
	if err != nil {
		return fmt.Errorf(core.ErrFmtDurableAppendInspect, err)
	}
	if !info.Mode().IsRegular() || info.Size() < 0 {
		return fmt.Errorf(core.ErrFmtDurableAppendInspect, core.ErrDurabilityContract)
	}
	maximum, _ := w.request.MaximumFileBytes.Int64()
	if info.Size() > maximum {
		return core.ErrDurableSizeLimit
	}
	return nil
}

func (w *AppendWriter) ensureWriteFits(length int) error {
	info, err := w.file.Stat()
	if err != nil {
		return fmt.Errorf(core.ErrFmtDurableAppendInspect, err)
	}
	if !info.Mode().IsRegular() || info.Size() < 0 {
		return fmt.Errorf(core.ErrFmtDurableAppendInspect, core.ErrDurabilityContract)
	}
	maximum, _ := w.request.MaximumFileBytes.Int64()
	remaining := maximum - info.Size()
	if int64(length) > remaining {
		return core.ErrDurableSizeLimit
	}
	return nil
}

func wrapAppendError(format string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf(format, err)
}
