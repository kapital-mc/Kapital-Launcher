package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"kapital/backend/models"
	"kapital/backend/services"
)

// A Frangfurd instance with the pack's server list, under a Prism root saved
// in settings.
func tailscaleInstance(t *testing.T, app *App) (models.AppSettings, string) {
	t.Helper()
	root := t.TempDir()
	settings := models.AppSettings{Theme: "dark", PrismRoot: root}
	if err := app.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	game := filepath.Join(root, "instances", "kapital-frangfurd", "minecraft")
	if err := os.MkdirAll(game, 0o755); err != nil {
		t.Fatal(err)
	}
	pack, err := os.ReadFile(filepath.Join("backend", "services", "testdata", "serverlist", "frangfurd-servers.dat"))
	if err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(game, "servers.dat")
	if err := os.WriteFile(list, pack, 0o644); err != nil {
		t.Fatal(err)
	}
	return settings, list
}

func TestTheTailscaleSwitchEditsTheInGameServerList(t *testing.T) {
	app := newTestApp(t)
	settings, list := tailscaleInstance(t, app)
	before, err := os.ReadFile(list)
	if err != nil {
		t.Fatal(err)
	}
	settings.TailscaleServers = []string{"frangfurd"}
	if err := app.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	on, err := os.ReadFile(list)
	if err != nil || len(on) <= len(before) {
		t.Fatalf("the switch added nothing: %v", err)
	}
	settings.TailscaleServers = nil
	if err := app.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	off, err := os.ReadFile(list)
	if err != nil || string(off) != string(before) {
		t.Fatalf("the switch off did not give the list back: %v", err)
	}
}

func TestSaveSettingsHoldsTheTailscaleChoiceToItsSwitch(t *testing.T) {
	app := newTestApp(t)
	if err := app.SaveSettings(models.AppSettings{Theme: "dark", TailscaleServers: []string{"luxemburg"}}); err == nil {
		t.Fatal("a chapter with no Tailscale address was switched on")
	}
	chosen := map[string]string{"frangfurd": "Tailscale"}
	if err := app.SaveSettings(models.AppSettings{Theme: "dark", ServerChoices: chosen, TailscaleServers: []string{"frangfurd"}}); err != nil {
		t.Fatal(err)
	}
	if got := app.serverAddress(frangfurd(t, app)); got != "ts.frangfurd.click" {
		t.Fatalf("switch on: %q", got)
	}
	// The switch turned off takes the choice with it.
	if err := app.SaveSettings(models.AppSettings{Theme: "dark", ServerChoices: chosen}); err != nil {
		t.Fatal(err)
	}
	got, err := app.GetSettings()
	if err != nil || got.ServerChoices["frangfurd"] != "" || len(got.TailscaleServers) != 0 {
		t.Fatalf("%v %+v", err, got)
	}
	if got := app.serverAddress(frangfurd(t, app)); got != frangfurdGlobal {
		t.Fatalf("switch off: %q", got)
	}
}

func TestConnectTailscaleTakesOnlyAChapterWithATailscaleAddress(t *testing.T) {
	app := newTestApp(t)
	for _, id := range []string{"nope", "luxemburg"} {
		if err := app.ConnectTailscale(id); err == nil {
			t.Errorf("%s: accepted", id)
		}
	}
}

func TestTailscalePreviewsAnswerForTheCLI(t *testing.T) {
	app := newTestApp(t)
	var emitted []models.TailscaleInfo
	app.tailscale.emitState = func(i models.TailscaleInfo) { emitted = append(emitted, i) }
	if _, err := app.StartPreview("", services.PreviewTailscaleDisconnected); err != nil {
		t.Fatal(err)
	}
	info, err := app.GetTailscale()
	if err != nil || info.State != services.TailscaleRunning || len(info.Connected) != 0 || len(emitted) != 1 {
		t.Fatalf("%v %+v %d", err, info, len(emitted))
	}
	// Connect opens nothing under the preview: it becomes the connected one.
	if err := app.ConnectTailscale("frangfurd"); err != nil {
		t.Fatal(err)
	}
	info, err = app.GetTailscale()
	if err != nil || !slices.Equal(info.Connected, []string{"frangfurd"}) || app.previews.Launcher() != services.PreviewTailscaleConnected {
		t.Fatalf("%v %+v", err, info)
	}
	// A switch is not saved under it, and the preview ends.
	err = app.SaveSettings(models.AppSettings{Theme: "dark", TailscaleServers: []string{"frangfurd"}})
	if err == nil || !strings.Contains(err.Error(), "preview") || app.previews.Launcher() != "" {
		t.Fatalf("saved under a preview: %v", err)
	}
	if got, err := app.GetSettings(); err != nil || len(got.TailscaleServers) != 0 {
		t.Fatalf("%v %+v", err, got)
	}
}

func TestTailscaleInstallPreviewFailsWithoutTheInstaller(t *testing.T) {
	app := newTestApp(t)
	app.previewStep = 0
	var steps []models.TailscaleInstallProgress
	app.tailscale.emitInstall = func(p models.TailscaleInstallProgress) { steps = append(steps, p) }
	if _, err := app.StartPreview("", services.PreviewTailscaleInstall); err != nil {
		t.Fatal(err)
	}
	rel, err := app.GetTailscaleRelease()
	if err != nil || !strings.Contains(rel.File, "preview") {
		t.Fatalf("%v %+v", err, rel)
	}
	if err := app.InstallTailscale(); err == nil || len(steps) != len(services.PreviewTailscaleInstallProgress()) {
		t.Fatalf("%v %d steps", err, len(steps))
	}
	if err := app.ClearPreviews(); err != nil || app.previews.Launcher() != "" {
		t.Fatalf("%v %q", err, app.previews.Launcher())
	}
}
