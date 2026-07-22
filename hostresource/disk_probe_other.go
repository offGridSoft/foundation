//go:build !darwin && !linux && !windows

package hostresource

import "github.com/offGridSoft/foundation/v2026/core"

func probeDisk(core.AbsoluteDirectoryPath) (DiskCapacity, error) {
	return DiskCapacity{}, core.ErrDiskCapacityUnsupported
}
