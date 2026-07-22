//go:build !darwin && !linux && !windows

package durability

import "os"

func openBoundedFile(path string) (*os.File, error) {
	return os.Open(path)
}
