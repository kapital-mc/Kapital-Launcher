//go:build darwin

package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// launcherBundleID is the identifier the release workflow checks the bundle
// for, and the one codesign must find in an update.
const launcherBundleID = "io.github.sandrogekeler.kapital-launcher"

// launcherBundleName is the bundle's folder name, from wails.json's name.
const launcherBundleName = "Kapital Launcher.app"

// systemUpdatePlatform replaces the whole bundle, the way Sparkle does: only
// the executable would leave a signature that no longer covers the bundle.
func systemUpdatePlatform() updatePlatform {
	return updatePlatform{
		asset:    func(tag string) string { return "Kapital-Launcher-" + tag + "-macos-universal.zip" },
		target:   bundleTarget,
		prepare:  prepareBundle,
		relaunch: openBundle,
	}
}

// bundleTarget is the .app the executable runs from. A bundle macOS
// translocated (opened from Downloads or the disk image without being moved)
// runs from a read-only copy that an update could not reach.
func bundleTarget(exe string) (string, error) {
	if strings.Contains(exe, "/AppTranslocation/") {
		return "", errors.New("move Kapital Launcher to your Applications folder and open it from there")
	}
	bundle := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if filepath.Ext(bundle) != ".app" || filepath.Base(filepath.Dir(exe)) != "MacOS" {
		return "", errors.New("this launcher is not running from its app, so it cannot update itself")
	}
	return bundle, nil
}

// prepareBundle unpacks the zip with ditto, which keeps the bundle's
// symlinks, modes and signature, and checks the result with codesign: a valid
// signature over every part, made for the launcher's identifier. Each tool is
// run with an argument array, never a shell.
func prepareBundle(ctx context.Context, dir, download string) (string, error) {
	unpacked := filepath.Join(dir, "new")
	if out, err := exec.CommandContext(ctx, "/usr/bin/ditto", "-x", "-k", download, unpacked).CombinedOutput(); err != nil {
		return "", fmt.Errorf("unpack the update: %w: %s", err, out)
	}
	removeQuietly(download)
	app := filepath.Join(unpacked, launcherBundleName)
	if info, err := os.Stat(app); err != nil || !info.IsDir() {
		return "", fmt.Errorf("the update has no %s", launcherBundleName)
	}
	requirement := `=identifier "` + launcherBundleID + `"`
	if out, err := exec.CommandContext(ctx, "/usr/bin/codesign", "--verify", "--deep", "--strict", "-R", requirement, app).CombinedOutput(); err != nil {
		return "", fmt.Errorf("the update's signature did not verify: %w: %s", err, out)
	}
	return app, nil
}

// openBundle starts a new instance of the bundle through Launch Services,
// which the running one, still quitting, would otherwise be asked to answer.
func openBundle(target string) error {
	cmd := exec.Command("/usr/bin/open", "-n", target)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start the updated launcher: %w", err)
	}
	return cmd.Process.Release()
}
