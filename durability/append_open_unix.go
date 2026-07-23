//go:build darwin || linux

package durability

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

func openAppendFile(path string, _ fs.FileMode) (*os.File, error) {
	descriptor, err := syscall.Open(path, syscall.O_WRONLY|syscall.O_APPEND|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(descriptor), path)
	if file != nil {
		return file, nil
	}
	return nil, errors.Join(os.ErrInvalid, syscall.Close(descriptor))
}
