package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kapital/backend/models"
	"kapital/backend/services"
)

const dhTerrain = "frangfurd overworld"

// onTerrainPart, when set by a test, runs as the test server answers a part.
var onTerrainPart func()

// terrainApp is an app with Frangfurd installed and synced (Distant Horizons
// 3.3.3 in its mods folder), whose manifest points at a snapshot on a TLS test
// server holding one dimension, published with dhVersion.
func terrainApp(t *testing.T, dhVersion string) (*App, string, *[]models.DHDownloadProgress) {
	t.Helper()
	t.Cleanup(func() { onTerrainPart = nil })
	app, cfg, _ := modsApp(t, modDH)
	sum := sha256.Sum256([]byte(dhTerrain))
	hash := hex.EncodeToString(sum[:])
	index := services.DHIndex{
		Format: 1, CreatedAt: "2026-10-13T12:03:41Z", PartsBase: "../../parts/",
		DHVersion: dhVersion, ServerFolder: "1790374988_frangfurd",
		Levels: []services.DHLevel{{
			Folder: "ff@minecraft@@overworld", Size: int64(len(dhTerrain)), SHA256: hash,
			Parts: []services.DHPart{{SHA256: hash, Size: int64(len(dhTerrain))}},
		}},
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body any
		switch r.URL.Path {
		case "/dh/latest.json":
			body = services.DHPointer{Format: 1, CreatedAt: index.CreatedAt, Index: "snapshots/20261013T120341Z/index.json"}
		case "/dh/snapshots/20261013T120341Z/index.json":
			body = index
		case "/dh/parts/" + hash:
			if onTerrainPart != nil {
				onTerrainPart()
			}
			_, _ = w.Write([]byte(dhTerrain)) //nolint:errcheck // test server
			return
		default:
			http.NotFound(w, r)
			return
		}
		data, err := json.Marshal(body)
		if err != nil {
			t.Error(err)
		}
		_, _ = w.Write(data) //nolint:errcheck // test server
	}))
	t.Cleanup(srv.Close)
	app.terrain.source = services.NewDHSourceWith(srv.Client())
	pointer := srv.URL + "/dh/latest.json"
	for i := range app.manifest.Chapters {
		if app.manifest.Chapters[i].ID == "frangfurd" {
			app.manifest.Chapters[i].Pack.DHSnapshot = &pointer
		}
	}
	var events []models.DHDownloadProgress
	app.terrain.emit = func(p models.DHDownloadProgress) { events = append(events, p) }
	return app, cfg, &events
}

func terrainDB(cfg string) string {
	return filepath.Join(filepath.Dir(cfg), "minecraft", "Distant_Horizons_server_data", "1790374988_frangfurd", "ff@minecraft@@overworld", "DistantHorizons.sqlite")
}

func TestGetDHSnapshotDescribesThePublishedTerrain(t *testing.T) {
	app, _, _ := terrainApp(t, "3.3.3")
	info, err := app.GetDHSnapshot("frangfurd")
	if err != nil {
		t.Fatal(err)
	}
	want := models.DHSnapshotInfo{
		ChapterID: "frangfurd", Available: true, CreatedAt: "2026-10-13T12:03:41Z",
		SizeBytes: int64(len(dhTerrain)), Dimensions: 1, ServerDHVersion: "3.3.3", ClientDHVersion: "3.3.3",
	}
	if info != want {
		t.Errorf("got %+v\nwant %+v", info, want)
	}
}

func TestGetDHSnapshotRefusesANewerServer(t *testing.T) {
	app, _, _ := terrainApp(t, "3.4.0")
	info, err := app.GetDHSnapshot("frangfurd")
	if err != nil || info.Available || !strings.Contains(info.Reason, "newer than this pack's 3.3.3") {
		t.Errorf("%+v %v", info, err)
	}
	if _, err := app.DownloadDHSnapshot("frangfurd", false); err == nil {
		t.Error("downloaded a newer server's terrain")
	}
}

func TestGetDHSnapshotForAChapterWithoutOne(t *testing.T) {
	app := newTestApp(t)
	info, err := app.GetDHSnapshot("lichdenstein")
	if err != nil || info.Available || info.Reason == "" {
		t.Errorf("%+v %v", info, err)
	}
	if _, err := app.GetDHSnapshot("atlantis"); err == nil {
		t.Error("an unknown chapter answered")
	}
}

func TestDownloadDHSnapshotPlacesTheTerrain(t *testing.T) {
	app, cfg, events := terrainApp(t, "3.3.3")
	info, err := app.DownloadDHSnapshot("frangfurd", false)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(terrainDB(cfg))
	if err != nil || string(got) != dhTerrain {
		t.Fatalf("%q %v", got, err)
	}
	if info.ExistingBytes != int64(len(dhTerrain)) || info.Downloading {
		t.Errorf("after: %+v", info)
	}
	var phases []string
	for _, e := range *events {
		if e.ChapterID != "frangfurd" {
			t.Errorf("event for %q", e.ChapterID)
		}
		phases = append(phases, e.Phase)
	}
	if phases[0] != "checking" || phases[len(phases)-1] != "done" {
		t.Errorf("phases %v", phases)
	}
}

func TestDownloadDHSnapshotAsksBeforeReplacingTerrain(t *testing.T) {
	app, cfg, _ := terrainApp(t, "3.3.3")
	db := terrainDB(cfg)
	if err := os.MkdirAll(filepath.Dir(db), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(db, []byte("the player's own"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DownloadDHSnapshot("frangfurd", false); err == nil || !strings.Contains(err.Error(), "confirm") {
		t.Fatalf("got %v", err)
	}
	if got, err := os.ReadFile(db); err != nil || string(got) != "the player's own" {
		t.Fatalf("replaced without asking: %q %v", got, err)
	}
	if _, err := app.DownloadDHSnapshot("frangfurd", true); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(db); err != nil || string(got) != dhTerrain {
		t.Errorf("%q %v", got, err)
	}
}

func TestDownloadDHSnapshotIsRefusedWhileTheGameRuns(t *testing.T) {
	app, cfg, _ := terrainApp(t, "3.3.3")
	touchGameLog(t, cfg, time.Second)
	info, err := app.DownloadDHSnapshot("frangfurd", false)
	if err == nil || err.Error() != guessRefusal || !info.Running {
		t.Fatalf("%+v %v", info, err)
	}
	if _, err := os.Stat(terrainDB(cfg)); !os.IsNotExist(err) {
		t.Errorf("terrain written while running: %v", err)
	}
}

func TestDownloadDHSnapshotRunsOneAtATime(t *testing.T) {
	app, _, _ := terrainApp(t, "3.3.3")
	_, done, err := app.terrain.begin(t.Context(), "frangfurd")
	if err != nil {
		t.Fatal(err)
	}
	if info, err := app.GetDHSnapshot("frangfurd"); err != nil || !info.Downloading || info.Available {
		t.Errorf("%+v %v", info, err)
	}
	if _, err := app.DownloadDHSnapshot("frangfurd", false); err == nil {
		t.Error("a second download started")
	}
	if ok, err := app.CancelDHSnapshot("frangfurd"); err != nil || !ok {
		t.Errorf("cancel: %v %v", ok, err)
	}
	done()
	done()
	if ok, err := app.CancelDHSnapshot("frangfurd"); err != nil || ok {
		t.Errorf("cancelled nothing: %v %v", ok, err)
	}
	if _, err := app.CancelDHSnapshot("atlantis"); err == nil {
		t.Error("an unknown chapter cancelled")
	}
}

func TestDownloadDHSnapshotCancelled(t *testing.T) {
	app, cfg, events := terrainApp(t, "3.3.3")
	onTerrainPart = func() {
		if ok, err := app.CancelDHSnapshot("frangfurd"); !ok || err != nil {
			t.Errorf("cancel: %v %v", ok, err)
		}
	}
	info, err := app.DownloadDHSnapshot("frangfurd", false)
	if !errors.Is(err, errDHCancelled) || info.Downloading {
		t.Fatalf("%+v %v", info, err)
	}
	if last := (*events)[len(*events)-1]; last.Phase != "cancelled" {
		t.Errorf("last event %+v", last)
	}
	if _, err := os.Stat(terrainDB(cfg)); !os.IsNotExist(err) {
		t.Errorf("terrain placed after a cancel: %v", err)
	}
	onTerrainPart = nil
	if _, err := app.DownloadDHSnapshot("frangfurd", false); err != nil {
		t.Fatal(err)
	}
}
