//go:build darwin

package durability

import (
	"errors"
	"os"
	"syscall"
)

const (
	darwinFullFsync    = uintptr(51)
	darwinBarrierFsync = uintptr(85)
)

func fullSyncPlatform(file *os.File) error {
	if err := fullSyncFcntl(file, darwinFullFsync); err == nil {
		return nil
	} else if !isFcntlUnsupported(err) {
		return &os.PathError{Op: "fullsync", Path: file.Name(), Err: err}
	}
	if err := fullSyncFcntl(file, darwinBarrierFsync); err == nil {
		return nil
	} else if !isFcntlUnsupported(err) {
		return &os.PathError{Op: "barrierfsync", Path: file.Name(), Err: err}
	}
	return file.Sync()
}

func fullSyncFcntl(file *os.File, command uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, file.Fd(), command, 0)
	if errno == 0 {
		return nil
	}
	return errno
}

func isFcntlUnsupported(err error) bool {
	return errors.Is(err, syscall.EINVAL) ||
		errors.Is(err, syscall.ENOTTY) ||
		errors.Is(err, syscall.ENOTSUP) ||
		errors.Is(err, syscall.EOPNOTSUPP) ||
		errors.Is(err, syscall.ENODEV) ||
		errors.Is(err, syscall.EBADF)
}
