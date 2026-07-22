//go:build !darwin

package durability

import "os"

func fullSyncPlatform(file *os.File) error {
	return file.Sync()
}
