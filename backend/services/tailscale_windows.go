package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// runTailscaleInstaller runs the MSI through Windows Installer with its basic
// UI: a progress bar and the administrator prompt the install needs, and no
// wizard. TS_NOLAUNCH (any non-empty value) keeps Tailscale's own window from
// opening at the end, so the launcher's sign-in is the next step (Tailscale's
// MSI properties, tailscale.com/kb/1189). Tailscale documents it for a silent
// install; whether /passive counts is [verify] on a Windows PC.
func runTailscaleInstaller(ctx context.Context, path string) error {
	msiexec := filepath.Join(os.Getenv("SystemRoot"), "System32", "msiexec.exe")
	err := exec.CommandContext(ctx, msiexec, "/i", path, "/passive", "/norestart", "TS_NOLAUNCH=1").Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		switch exit.ExitCode() {
		case 3010: // installed, a restart is needed to finish
			return nil
		case 1602, 1223: // the player cancelled, or declined the administrator prompt
			return errors.New("the Tailscale installation was cancelled")
		}
		return fmt.Errorf("Tailscale's installer ended with code %d", exit.ExitCode())
	}
	return err
}

// verifyTailscaleInstaller has nothing to add on Windows beyond the published
// digest: Windows Installer shows the MSI's verified publisher in its
// administrator prompt.
func verifyTailscaleInstaller(context.Context, string) error { return nil }

// OpenTailscaleApp starts Tailscale's tray app. [verify] the file name.
func OpenTailscaleApp() error {
	cmd := exec.Command(filepath.Join(os.Getenv("ProgramFiles"), "Tailscale", "tailscale-ipn.exe"))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open Tailscale: %w", err)
	}
	go cmd.Wait() //nolint:errcheck // reaped only; the app outlives nothing of ours
	return nil
}
