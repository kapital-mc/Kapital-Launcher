//go:build windows

package services

import (
	"errors"

	"golang.org/x/sys/windows"
)

// fileBusy is a sharing violation: another process has the file open without
// allowing a delete or rename, which passes once it closes the file.
func fileBusy(err error) bool {
	return errors.Is(err, windows.ERROR_SHARING_VIOLATION)
}
