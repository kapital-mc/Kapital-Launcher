//go:build !windows

package services

// fileBusy is never true here: macOS and Linux rename a file whoever has it open.
func fileBusy(error) bool { return false }
