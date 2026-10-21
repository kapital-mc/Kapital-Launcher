//go:build !windows && !darwin

package services

import (
	"context"
	"errors"
)

// errUpdateUnsupported is every other system: the launcher ships for Windows
// and macOS only (docs/adr/0007-target-oses.md).
var errUpdateUnsupported = errors.New("the launcher updates itself on Windows and macOS only")

func systemUpdatePlatform() updatePlatform {
	return updatePlatform{
		asset:    func(string) string { return "" },
		target:   func(string) (string, error) { return "", errUpdateUnsupported },
		prepare:  func(context.Context, string, string) (string, error) { return "", errUpdateUnsupported },
		relaunch: func(string) error { return errUpdateUnsupported },
	}
}
