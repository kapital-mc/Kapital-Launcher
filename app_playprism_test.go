package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kapital/backend/models"
	"kapital/backend/services"
)

// playApp is an app whose Play gets as far as starting Prism, from an
// executable that does not exist: a launch that goes ahead fails with
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
// it starts there. Play closes the launcher's own leftover Prisms of the other
// chapters, then the idle ones of the engine's executable, and goes ahead,
// whether or not one stayed open.
func TestPlayClosesTheOpenPrismsAndGoesAhead(t *testing.T) {
	for _, stillOpen := range []bool{false, true} {
		app := playApp(t)
		probe := &prismProbe{ownGone: true}
		probe.install(app)
		var exes []string
		app.closeIdle = func(exe string) bool {
			probe.calls = append(probe.calls, "close idle")
			exes = append(exes, exe)
			return stillOpen
		}

		err := app.LaunchChapter("lichdenstein")
		if err == nil || !strings.Contains(err.Error(), "start prism") {
			t.Fatalf("still open %v: Prism is started: %v", stillOpen, err)
		}
		if got := strings.Join(probe.calls, ","); !strings.HasPrefix(got, "close own luxemburg,close own frangfurd,close idle") {
			t.Fatalf("the leftovers first, then the idle Prisms, before Prism starts: %s", got)
		}
		if len(exes) != 1 || exes[0] != app.engine.Executable {
			t.Fatalf("closed for %v, the engine's executable is %q", exes, app.engine.Executable)
		}
	}
}

// Another chapter's game starting or running has its Prism open on the same
// root, which would take this launch: Play says which chapter, and closes
// nothing.
func TestPlayIsRefusedWhileAnotherChapterIsRunning(t *testing.T) {
	app := playApp(t)
	probe := &prismProbe{ownGone: true}
	probe.install(app)
	app.closeIdle = func(string) bool {
		probe.calls = append(probe.calls, "close idle")
		return false
	}
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
		if call == "close own frangfurd" || call == "close idle" {
			t.Fatalf("nothing is closed: %v", probe.calls)
		}
	}
}
