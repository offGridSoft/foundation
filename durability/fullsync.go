package durability

import "os"

// CommitFile establishes the strongest stable-file barrier available on the
// host operating system. The caller retains descriptor ownership.
func CommitFile(file *os.File) error {
	if file == nil {
		return os.ErrInvalid
	}
	return fullSyncPlatform(file)
}
