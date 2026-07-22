package durability

import "os"

func FullSync(file *os.File) error {
	if file == nil {
		return os.ErrInvalid
	}
	return fullSyncPlatform(file)
}
