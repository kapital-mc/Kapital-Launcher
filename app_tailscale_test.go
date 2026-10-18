package main

import (
	"os"
	"path/filepath"
	"testing"

	"kapital/backend/models"
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
