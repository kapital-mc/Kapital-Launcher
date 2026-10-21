//go:build windows

package services

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Windows will not overwrite or delete a running exe, but renames it: the
// swap goes through while the old copy still runs, and the old copy is
// removed once it has exited.
func TestUpdateSwapsARunningExe(t *testing.T) {
	f := newUpdaterFixture(t)
	ping, err := os.ReadFile(filepath.Join(os.Getenv("SystemRoot"), "System32", "PING.EXE"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.target, ping, 0o755); err != nil {
		t.Fatal(err)
	}
	running := exec.Command(f.target, "-n", "30", "127.0.0.1")
	if err := running.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = running.Process.Kill() }) //nolint:errcheck // gone already, or ending now
	if err := os.Remove(f.target); err == nil {
		t.Fatal("Windows deleted a running exe; the test is not testing what it says")
	}

	if _, err := f.download(t); err != nil {
		t.Fatal(err)
	}
	if _, err := f.u.Apply(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, f.target); got != "new build" {
		t.Fatalf("after Apply the launcher holds %q", got)
	}
	if err := running.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = running.Wait() //nolint:errcheck // killed
	f.u.CleanUp(context.Background())
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(f.target), stagingPrefix+"*")); len(left) != 0 {
		t.Fatalf("leftovers after CleanUp: %v", left)
	}
}
