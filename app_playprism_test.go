package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kapital/backend/models"
	"kapital/backend/services"
)

// playApp is an app whose Play gets as far as starting Prism, from an
// executable that does not exist: a launch that is not refused fails with
// "start prism", which no refusal says.
func playApp(t *testing.T) *App {
	t.Helper()
	app := newTestApp(t)
	root := t.TempDir()
	if err := app.SaveSettings(models.AppSettings{Theme: "dark", PrismRoot: root}); err != nil {
		t.Fatal(err)
	}
	app.engine = models.EngineInfo{Found: true, Executable: filepath.Join(root, "no-such-prism"), Root: root}
	return app
}

// A Prism already open would take the launch, and Stop could not end the game
// it starts there: Play is refused before Prism is run. The launcher's own
// Prisms left by the other chapters' runs are closed first, then the look is
// made, for the engine's executable.
func TestPlayIsRefusedWhilePrismIsOpen(t *testing.T) {
	app := playApp(t)
	probe := &prismProbe{ownGone: true, other: true}
	probe.install(app)

	err := app.LaunchChapter("lichdenstein")
	if err == nil || err.Error() != "Prism is open; close it, then press Play" {
		t.Fatalf("got %v", err)
	}
	if got := strings.Join(probe.calls, ","); got != "close own luxemburg,close own frangfurd,look lichdenstein" {
		t.Fatalf("the other chapters' leftover Prisms are closed before the look: %s", got)
	}
	if probe.exes[0] != app.engine.Executable {
		t.Fatalf("looked for %q, the engine's executable is %q", probe.exes[0], app.engine.Executable)
	}
	if app.games.Active("lichdenstein") {
		t.Fatal("no run is followed")
	}

	// Prism closed: Play goes on to start Prism.
	probe.other = false
	if err := app.LaunchChapter("lichdenstein"); err == nil || !strings.Contains(err.Error(), "start prism") {
		t.Fatalf("Prism is started: %v", err)
	}
}

// Where the processes cannot be listed Play goes ahead, as it did before.
func TestPlayGoesAheadWhenAnOpenPrismCannotBeTold(t *testing.T) {
	app := playApp(t)
	probe := &prismProbe{ownGone: true, err: errors.New("no snapshot")}
	probe.install(app)
	if err := app.LaunchChapter("lichdenstein"); err == nil || !strings.Contains(err.Error(), "start prism") {
		t.Fatalf("Prism is started: %v", err)
	}
}

// Another chapter's game starting or running has its Prism open on the same
// root, which would take this launch: Play says which chapter, and closes
// nothing.
func TestPlayIsRefusedWhileAnotherChapterIsRunning(t *testing.T) {
	app := playApp(t)
	probe := &prismProbe{ownGone: true}
	probe.install(app)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	// No process has pid -1, so the tracker has nothing to find and stays starting.
	if err := app.games.Track(ctx, services.TrackRequest{
		ChapterID: "frangfurd",
		Prism:     services.PrismProcess{PID: -1},
		StartedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	err := app.LaunchChapter("lichdenstein")
	if err == nil || err.Error() != "Frangfurd is running; close it, then press Play" {
		t.Fatalf("got %v", err)
	}
	for _, call := range probe.calls {
		if call == "close own frangfurd" {
			t.Fatalf("a running chapter's Prism is not closed: %v", probe.calls)
		}
	}
}
