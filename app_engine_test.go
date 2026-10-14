package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"kapital/backend/models"
)

// The page asks for the engine once, and on Windows it can ask while startup's
// detection is still running (issue 260): the answer waits for that detection
// instead of reporting the empty engine as "not found".
func TestGetEngineWaitsForStartupDetection(t *testing.T) {
	app, err := NewApp(t.TempDir(), bundledManifest, nil)
	if err != nil {
		t.Fatal(err)
	}
	// A file named in settings is what detection tries first; it is not a
	// program, so --version fails at once and the version stays empty.
	exe := filepath.Join(t.TempDir(), "prismlauncher.exe")
	if err := os.WriteFile(exe, []byte("not prism"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := app.settings.Save(models.AppSettings{Theme: "dark", PrismExecutable: exe}); err != nil {
		t.Fatal(err)
	}

	answer := make(chan models.EngineInfo, 1)
	go func() {
		info, err := app.GetEngine()
		if err != nil {
			t.Error(err)
		}
		answer <- info
	}()
	select {
	case info := <-answer:
		t.Fatalf("GetEngine answered before detection ran: %+v", info)
	case <-time.After(100 * time.Millisecond):
	}

	app.detectStartupEngine()
	select {
	case info := <-answer:
		if !info.Found || info.Source != "settings" || info.Executable != exe {
			t.Fatalf("GetEngine answered %+v, want the Prism named in settings", info)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("GetEngine still waits after detection finished")
	}

	// Later asks answer at once.
	done := make(chan struct{})
	go func() {
		if _, err := app.GetEngine(); err != nil {
			t.Error(err)
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GetEngine waits after startup")
	}
}
