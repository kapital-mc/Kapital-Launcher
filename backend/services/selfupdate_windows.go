//go:build windows

package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// uninstallKey is where the NSIS setup records the install for Apps and
// features: Wails' wails_tools.nsh, UNINST_KEY, named from wails.json's
// companyName and productName, under HKCU for the per-user install.
const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\Kapitel KapitalKapital Launcher`

// systemUpdatePlatform swaps the exe: Windows will not let a running exe be
// overwritten or deleted, but it can be renamed, which is the sequence
// minio/selfupdate uses.
func systemUpdatePlatform() updatePlatform {
	return updatePlatform{
		asset:    func(tag string) string { return "Kapital-Launcher-" + tag + "-windows-amd64.exe" },
		target:   exeTarget,
		prepare:  func(_ context.Context, _, download string) (string, error) { return download, nil },
		relaunch: startExe,
		placed:   recordInstalledVersion,
		hide:     hideFolder,
	}
}

// exeTarget is the running exe itself.
func exeTarget(exe string) (string, error) {
	if !strings.EqualFold(filepath.Ext(exe), ".exe") {
		return "", errors.New("this launcher is not running from its program file, so it cannot update itself")
	}
	return exe, nil
}

// startExe starts the updated exe, an argument array with no arguments.
func startExe(target string) error {
	cmd := exec.Command(target)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start the updated launcher: %w", err)
	}
	return cmd.Process.Release()
}

// hideFolder marks the staging folder hidden, as its leading dot does on macOS.
func hideFolder(dir string) {
	p, err := windows.UTF16PtrFromString(dir)
	if err == nil {
		err = windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_HIDDEN)
	}
	if err != nil {
		slog.Info("launcher update: hide staging folder", "error", err)
	}
}

// recordInstalledVersion updates DisplayVersion in the setup's uninstall
// entry, so Apps and features names the version now installed. Only an entry
// whose icon is this exe is the install being updated; a bare exe run from
// elsewhere leaves the setup's entry alone. The setup writes the numeric part
// of the tag, as the release workflow stamps it.
func recordInstalledVersion(target, tag string) {
	k, err := registry.OpenKey(registry.CURRENT_USER, uninstallKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close() //nolint:errcheck // a registry handle, nothing to flush
	icon, _, err := k.GetStringValue("DisplayIcon")
	if err != nil || !strings.EqualFold(filepath.Clean(strings.Trim(icon, `"`)), filepath.Clean(target)) {
		return
	}
	numeric := strings.TrimPrefix(tag, "v")
	if i := strings.IndexByte(numeric, '-'); i >= 0 {
		numeric = numeric[:i]
	}
	if err := k.SetStringValue("DisplayVersion", numeric); err != nil {
		slog.Info("launcher update: record version", "error", err)
	}
}
