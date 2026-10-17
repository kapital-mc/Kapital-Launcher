//go:build !windows

package services

import "golang.org/x/sys/unix"

// diskFree is the space an unprivileged user may still write on the volume
// that holds dir.
func diskFree(dir string) (int64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(dir, &st); err != nil {
		return 0, err
	}
	return int64(st.Bavail) * int64(st.Bsize), nil //nolint:gosec // block counts fit an int64
}
