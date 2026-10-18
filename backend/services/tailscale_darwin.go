package services

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// tailscaleInstallerSigner is the certificate Tailscale's macOS package is
// signed with, read from Tailscale-1.104.1-macos.pkg (2026-10-18).
const tailscaleInstallerSigner = "Developer ID Installer: Tailscale Inc. (W5364U7YZB)"

// runTailscaleInstaller opens the package in macOS's Installer and waits for
// the Installer to quit. The Installer's steps and the administrator password
// are macOS's; Tailscale then asks for its system extension and VPN
// configuration itself when it first starts.
func runTailscaleInstaller(ctx context.Context, path string) error {
	if err := exec.CommandContext(ctx, "/usr/bin/open", "-W", path).Run(); err != nil {
		return fmt.Errorf("open Tailscale's installer: %w", err)
	}
	return nil
}

// verifyTailscaleInstaller checks the package's signature with pkgutil: signed
// for distribution by Tailscale's own Developer ID.
func verifyTailscaleInstaller(ctx context.Context, path string) error {
	out, err := exec.CommandContext(ctx, "/usr/sbin/pkgutil", "--check-signature", path).CombinedOutput()
	text := string(out)
	if err != nil || !strings.Contains(text, "signed by a developer certificate issued by Apple") ||
		!strings.Contains(text, tailscaleInstallerSigner) {
		return fmt.Errorf("Tailscale's installer is not signed by %s: %s", tailscaleInstallerSigner, strings.TrimSpace(text))
	}
	return nil
}

// OpenTailscaleApp opens the Tailscale app, which starts its network and, on
// its first start, asks for its system extension.
func OpenTailscaleApp() error {
	if err := exec.Command("/usr/bin/open", "-a", "/Applications/Tailscale.app").Run(); err != nil {
		return fmt.Errorf("open Tailscale: %w", err)
	}
	return nil
}
