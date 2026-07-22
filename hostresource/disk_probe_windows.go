//go:build windows

package hostresource

import (
	"github.com/offGridSoft/foundation/v2026/core"
	"golang.org/x/sys/windows"
)

func probeDisk(path core.AbsoluteDirectoryPath) (DiskCapacity, error) {
	pointer, err := windows.UTF16PtrFromString(path.String())
	if err != nil {
		return DiskCapacity{}, err
	}
	var freeForCaller uint64
	var total uint64
	var totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(pointer, &freeForCaller, &total, &totalFree); err != nil {
		return DiskCapacity{}, err
	}
	return DiskCapacity{FreeBytes: freeForCaller, TotalBytes: total}, nil
}
