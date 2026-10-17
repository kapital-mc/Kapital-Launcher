//go:build windows

package services

import "golang.org/x/sys/windows"

// diskFree is the space the current user may still write on the volume that
// holds dir.
func diskFree(dir string) (int64, error) {
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0, err
	}
	var free, total, totalFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &free, &total, &totalFree); err != nil {
		return 0, err
	}
	return int64(free), nil //nolint:gosec // no volume holds 8 EiB
}
