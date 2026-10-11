package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var stashDay = time.Date(2026, 10, 11, 23, 40, 0, 0, time.Local)

// handMadeInstance writes a folder like the one issue 244 found: an
// instance.cfg with no pre-launch command and a world in it.
func handMadeInstance(t *testing.T, dir, id string) {
	t.Helper()
	world := filepath.Join(dir, id, "minecraft", "saves", "World")
	if err := os.MkdirAll(world, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id, instanceConfig), []byte("[General]\nname=kapital-lichdenstein\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(world, "level.dat"), []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStashInstanceMovesTheFolderAsideWholeUnderTheDay(t *testing.T) {
	dir := t.TempDir()
	handMadeInstance(t, dir, "kapital-lichdenstein")
	name, err := StashInstance(dir, "kapital-lichdenstein", stashDay)
	if err != nil {
		t.Fatal(err)
	}
	if name != "kapital-lichdenstein-stashed-2026-10-11" {
		t.Fatalf("stash name %q", name)
	}
	if _, err := os.Stat(filepath.Join(dir, "kapital-lichdenstein")); !os.IsNotExist(err) {
		t.Fatalf("the old name is still there: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, name, "minecraft", "saves", "World", "level.dat"))
	if err != nil || string(got) != "world" {
		t.Fatalf("the world did not move with the folder: %q, %v", got, err)
	}
}

func TestStashInstanceNumbersASecondStashOfTheDay(t *testing.T) {
	dir := t.TempDir()
	handMadeInstance(t, dir, "kapital-lichdenstein")
	if _, err := StashInstance(dir, "kapital-lichdenstein", stashDay); err != nil {
		t.Fatal(err)
	}
	handMadeInstance(t, dir, "kapital-lichdenstein")
	name, err := StashInstance(dir, "kapital-lichdenstein", stashDay)
	if err != nil {
		t.Fatal(err)
	}
	if name != "kapital-lichdenstein-stashed-2026-10-11-2" {
		t.Fatalf("second stash name %q", name)
	}
	if _, err := os.Stat(filepath.Join(dir, "kapital-lichdenstein-stashed-2026-10-11", instanceConfig)); err != nil {
		t.Fatalf("the first stash was touched: %v", err)
	}
}

func TestStashInstanceRefuses(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "kapital-file"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for name, id := range map[string]string{
		"a path":      "../kapital-lichdenstein",
		"missing":     "kapital-lichdenstein",
		"a file":      "kapital-file",
		"too long id": "kapital-" + strings.Repeat("a", 60),
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := StashInstance(dir, id, stashDay); err == nil {
				t.Fatalf("expected a refusal, got %q", got)
			}
		})
	}
}

func TestUnstashInstancePutsTheFolderBackAndNeverOverAnother(t *testing.T) {
	dir := t.TempDir()
	handMadeInstance(t, dir, "kapital-lichdenstein")
	name, err := StashInstance(dir, "kapital-lichdenstein", stashDay)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "kapital-lichdenstein"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := UnstashInstance(dir, "kapital-lichdenstein", name); err == nil {
		t.Fatal("unstashed over a folder that is there again")
	}
	if err := os.Remove(filepath.Join(dir, "kapital-lichdenstein")); err != nil {
		t.Fatal(err)
	}
	if err := UnstashInstance(dir, "kapital-lichdenstein", name); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "kapital-lichdenstein", instanceConfig)); err != nil {
		t.Fatalf("not back: %v", err)
	}
}
