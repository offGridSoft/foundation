//go:build darwin || linux

package durability

import (
	"errors"
	"os"
	"syscall"
)

func openBoundedFile(path string) (*os.File, error) {
	descriptor, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NONBLOCK|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(descriptor), path)
	if file != nil {
		return file, nil
	}
	return nil, errors.Join(os.ErrInvalid, syscall.Close(descriptor))
}
