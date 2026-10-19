//go:build !windows && !darwin

package services

import (
	"context"
	"errors"
)

// Off Windows and macOS the launcher does not install Tailscale: Get
// Tailscale opens Tailscale's download page instead.

func runTailscaleInstaller(context.Context, string) error {
	return errors.New("the launcher cannot install Tailscale on this OS")
}

func verifyTailscaleInstaller(context.Context, string) error {
	return errors.New("the launcher cannot install Tailscale on this OS")
}

// OpenTailscaleApp has no app to open here.
func OpenTailscaleApp() error {
	return errors.New("open Tailscale from your system's menu")
}
