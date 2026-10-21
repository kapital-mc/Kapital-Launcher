package services

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"
)

// The tracker's view of the OS's processes: one row of the process table and
// the calls it makes on processes, implemented per platform in gameproc_*.go.

var errGameProcUnsupported = errors.New("process lookup is not supported on this OS")

// procInfo is one row of the OS's process table.
type procInfo struct {
	PID, PPID int
	Name      string
}

// gameOS is what the tracker asks of the OS, injected so a fake answers in
// tests. The real one is systemGameOS, per platform (gameproc_*.go).
type gameOS struct {
	// list returns every process.
	list func() ([]procInfo, error)
	// started is when a process was created, false when unknown.
	started func(pid int) (time.Time, bool)
	// wait blocks until the process exits or ctx is done. The exit code is
	// known only where the platform reports it.
	wait func(ctx context.Context, pid int) (code int, known bool, err error)
	// terminate ends a process by pid: asks it to (macOS SIGTERM) or, with
	// force, makes it (SIGKILL). Windows has no gentler process-level ask, so
	// both are TerminateProcess. Only for the two processes the tracker found
	// itself, the launcher's Prism and the game's Java (S3.9).
	terminate func(pid int, force bool) error
	// askClose asks a Prism to close: WM_CLOSE to its visible top-level
	// windows on Windows, SIGTERM on macOS. An error means nothing took it.
	askClose func(pid int) error
	// image is the path of the executable a process runs, false when it cannot
	// be read. Only OtherPrismOpen asks, and only about processes that carry
	// Prism's name (gametracker_prismopen.go); nil reads none.
	image func(pid int) (string, bool)
}

// isJavaName is whether a process name is a Java runtime: javaw.exe or
// java.exe on Windows, java on macOS. All three are accepted everywhere, as no
// one OS has more than one of them.
func isJavaName(name string) bool {
	switch strings.ToLower(name) {
	case "javaw.exe", "java.exe", "java":
		return true
	}
	return false
}

// sameExe compares a process name with the Prism executable's base name,
// ignoring case and a ".exe" either side lacks.
func sameExe(name, exe string) bool {
	trim := func(s string) string { return strings.TrimSuffix(strings.ToLower(s), ".exe") }
	return exe != "" && trim(name) == trim(filepath.Base(exe))
}
