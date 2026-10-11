package main

import (
	"fmt"
	"os"
	"testing"
)

// isolatedEnv is every variable the app reads to find Prism's data root, its
// install or the home folder (services.PrismService: DataRoot, detection).
// A test that leaves PrismRoot unset with an engine that is not the managed
// one resolved the real %APPDATA%\PrismLauncher and rewrote the author's
// Frangfurd instance (issue 248), so no test in this package sees the real ones.
var isolatedEnv = []string{"APPDATA", "LOCALAPPDATA", "ProgramFiles", "XDG_DATA_HOME", "HOME", "USERPROFILE"}

func TestMain(m *testing.M) {
	os.Exit(runIsolated(m))
}

// runIsolated points isolatedEnv at an empty temporary folder for the run and
// removes it after.
func runIsolated(m *testing.M) int {
	dir, err := os.MkdirTemp("", "kapital-test-env-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "test env:", err)
		return 1
	}
	defer os.RemoveAll(dir) //nolint:errcheck // a temp folder left behind is harmless
	for _, key := range isolatedEnv {
		if err := os.Setenv(key, dir); err != nil {
			fmt.Fprintln(os.Stderr, "test env:", err)
			return 1
		}
	}
	return m.Run()
}
