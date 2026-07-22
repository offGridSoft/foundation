//go:build darwin || linux

package hostresource

import (
	"syscall"

	"github.com/offGridSoft/foundation/v2026/core"
)

func probeDisk(path core.AbsoluteDirectoryPath) (DiskCapacity, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path.String(), &stat); err != nil {
		return DiskCapacity{}, err
	}
	if stat.Bsize <= 0 {
		return DiskCapacity{}, core.ErrHostResourceContract
	}
	free, err := blocksToBytes(stat.Bavail, uint64(stat.Bsize))
	if err != nil {
		return DiskCapacity{}, err
	}
	total, err := blocksToBytes(stat.Blocks, uint64(stat.Bsize))
	if err != nil {
		return DiskCapacity{}, err
	}
	return DiskCapacity{FreeBytes: free, TotalBytes: total}, nil
}
