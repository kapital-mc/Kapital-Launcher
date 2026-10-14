package services

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A Windows GUI build has no stderr, so every write to it fails; the log file
// is written all the same (issue 261). A closed file stands in for that stderr.
func TestInitLoggerWritesTheFileWhenStderrFails(t *testing.T) {
	broken, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	if err := broken.Close(); err != nil {
		t.Fatal(err)
	}
	prevStderr, prevLogger := os.Stderr, slog.Default()
	os.Stderr = broken
	t.Cleanup(func() {
		os.Stderr = prevStderr
		slog.SetDefault(prevLogger)
	})

	dataDir := t.TempDir()
	logger, err := InitLogger(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("starting", "version", "test")
	if err := CloseLogger(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(LogPath(dataDir))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "msg=starting version=test") {
		t.Fatalf("the log file is %q, want the line logged", got)
	}
}

// A log past maxLogBytes is moved aside to .1 when the logger opens, and a new
// file is begun.
func TestInitLoggerRotatesALargeLog(t *testing.T) {
	prevLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prevLogger) })

	dataDir := t.TempDir()
	old := strings.Repeat("x", maxLogBytes)
	if err := os.WriteFile(LogPath(dataDir), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := InitLogger(dataDir); err != nil {
		t.Fatal(err)
	}
	if err := CloseLogger(); err != nil {
		t.Fatal(err)
	}

	rotated, err := os.ReadFile(filepath.Join(dataDir, LogFileName+".1"))
	if err != nil || len(rotated) != maxLogBytes {
		t.Fatalf("rotated log: %d bytes, %v", len(rotated), err)
	}
	if info, err := os.Stat(LogPath(dataDir)); err != nil || info.Size() != 0 {
		t.Fatalf("new log: %v %v", info, err)
	}
}
