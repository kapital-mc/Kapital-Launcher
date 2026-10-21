package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kapital/backend/models"
	"kapital/backend/services"
)

const storageServer = "1790374988_frangfurd"

// storageApp is an app with Frangfurd installed and synced, holding one
// server's terrain, a world map and a screenshot.
func storageApp(t *testing.T) (*App, string, string) {
	t.Helper()
	app, cfg, _ := modsApp(t, modJade)
	game := filepath.Join(filepath.Dir(cfg), "minecraft")
	for _, name := range []string{
		"Distant_Horizons_server_data/" + storageServer + "/ff@minecraft@@overworld/DistantHorizons.sqlite",
		"xaero/world-map/Multiplayer_frangfurd/null/0_0.zip",
		"screenshots/2026-10-20_21.38.11.png",
	} {
		p := filepath.Join(game, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return app, cfg, game
}

func storageKinds(s models.ChapterStorage) []string {
	var kinds []string
	for _, it := range s.Items {
		kinds = append(kinds, it.Kind+":"+it.Name)
	}
	return kinds
}

func TestGetChapterStorageMeasuresTheInstance(t *testing.T) {
	app, _, _ := storageApp(t)
	got, err := app.GetChapterStorage("frangfurd")
	if err != nil {
		t.Fatal(err)
	}
	if got.ChapterID != "frangfurd" || got.Running || got.TotalBytes == 0 {
		t.Fatalf("%+v", got)
	}
	want := "terrain:" + storageServer + " worldMap: screenshots: mods: other:"
	if kinds := strings.Join(storageKinds(got), " "); kinds != want {
		t.Fatalf("kinds %q, want %q", kinds, want)
	}
}

func TestStorageMethodsRefuseAChapterThatIsNotThereOrNotInstalled(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.GetChapterStorage("atlantis"); err == nil || !strings.Contains(err.Error(), "no chapter") {
		t.Fatalf("got %v", err)
	}
	if _, err := app.ClearChapterStorage("atlantis", models.StorageLogs, ""); err == nil || !strings.Contains(err.Error(), "no chapter") {
		t.Fatalf("got %v", err)
	}
	if err := app.SaveSettings(models.AppSettings{Theme: "dark", PrismRoot: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.GetChapterStorage("frangfurd"); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("got %v", err)
	}
	if _, err := app.ClearChapterStorage("frangfurd", models.StorageLogs, ""); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("got %v", err)
	}
	if err := app.OpenScreenshotsFolder("frangfurd"); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("got %v", err)
	}
}

func TestClearChapterStorageDeletesAndMeasuresAgain(t *testing.T) {
	app, _, game := storageApp(t)
	got, err := app.ClearChapterStorage("frangfurd", models.StorageTerrain, storageServer)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(game, "Distant_Horizons_server_data", storageServer)); !os.IsNotExist(err) {
		t.Fatalf("the terrain is still there: %v", err)
	}
	if kinds := strings.Join(storageKinds(got), " "); kinds != "worldMap: screenshots: mods: other:" {
		t.Fatalf("measured again: %q", kinds)
	}
	if _, err := app.ClearChapterStorage("frangfurd", models.StorageScreenshots, ""); err == nil {
		t.Fatal("screenshots were deleted")
	}
	if _, err := os.Stat(filepath.Join(game, "screenshots")); err != nil {
		t.Fatal(err)
	}
}

func TestClearChapterStorageIsRefusedWhileTheGameRuns(t *testing.T) {
	app, cfg, game := storageApp(t)
	mapDir := filepath.Join(game, "xaero", "world-map")
	check := func(name, want string) {
		t.Helper()
		_, err := app.ClearChapterStorage("frangfurd", models.StorageWorldMap, "")
		if err == nil || err.Error() != want {
			t.Fatalf("%s: got %v, want %q", name, err, want)
		}
		if _, err := os.Stat(mapDir); err != nil {
			t.Fatalf("%s: the map went: %v", name, err)
		}
	}
	touchGameLog(t, cfg, 0)
	check("the guess", guessRefusal)
	if got, err := app.GetChapterStorage("frangfurd"); err != nil || !got.Running {
		t.Fatalf("the page is told: %v %+v", err, got)
	}
	touchGameLog(t, cfg, 2*time.Minute)
	trackFrangfurd(t, app, cfg)
	check("the tracker", trackerRefusal)
}

func TestClearChapterStorageWaitsForATerrainDownload(t *testing.T) {
	app, _, game := storageApp(t)
	_, done, err := app.terrain.begin(t.Context(), "frangfurd")
	if err != nil {
		t.Fatal(err)
	}
	defer done()
	for _, kind := range []string{models.StorageTerrain, models.StorageTerrainDownload} {
		if _, err := app.ClearChapterStorage("frangfurd", kind, storageServer); err == nil || !strings.Contains(err.Error(), "terrain download") {
			t.Fatalf("%s: got %v", kind, err)
		}
	}
	if _, err := os.Stat(filepath.Join(game, "Distant_Horizons_server_data", storageServer)); err != nil {
		t.Fatal(err)
	}
	// The world map is not the download's.
	if _, err := app.ClearChapterStorage("frangfurd", models.StorageWorldMap, ""); err != nil {
		t.Fatal(err)
	}
}

func TestClearChapterStorageUnderAPreviewIsRefusedAndEndsIt(t *testing.T) {
	app, _, game := storageApp(t)
	mustStart(t, app, "frangfurd", services.PreviewNotInstalled)
	if _, err := app.ClearChapterStorage("frangfurd", models.StorageWorldMap, ""); err == nil || !strings.Contains(err.Error(), "preview") {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(filepath.Join(game, "xaero", "world-map")); err != nil {
		t.Fatal(err)
	}
	if app.previews.Chapter("frangfurd") != "" {
		t.Fatal("the preview was not ended")
	}
}

func TestOpenScreenshotsFolderOpensOnlyThatFolder(t *testing.T) {
	app, _, game := storageApp(t)
	var opened []string
	app.openFolder = func(dir string) error { opened = append(opened, dir); return nil }
	if err := app.OpenScreenshotsFolder("frangfurd"); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != filepath.Join(game, "screenshots") {
		t.Fatalf("opened %v", opened)
	}
	app.openFolder = func(string) error { return errors.New("no file manager") }
	if err := app.OpenScreenshotsFolder("frangfurd"); err == nil || !strings.Contains(err.Error(), "Frangfurd") {
		t.Fatalf("the failure names the chapter: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(game, "screenshots")); err != nil {
		t.Fatal(err)
	}
	if err := app.OpenScreenshotsFolder("frangfurd"); err == nil || !strings.Contains(err.Error(), "no screenshots") {
		t.Fatalf("got %v", err)
	}
}
