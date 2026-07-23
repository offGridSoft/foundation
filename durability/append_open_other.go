//go:build !darwin && !linux && !windows

package durability

import (
	"io/fs"
	"os"
)

func openAppendFile(path string, mode fs.FileMode) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_WRONLY, mode)
}
