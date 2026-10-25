package main

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"kapital/backend/models"
)

// The Camera Tools jars, as the installer's real record in the services
// package's testdata names them.
var cameraJars = []string{
	"Flashback-0.39.10-for-MC1.21.1.jar",
	"connector-2.0.0-beta.17+1.21.1-full.jar",
	"flashback_neoforge_fixed-1.0.17.jar",
	"forgified-fabric-api-0.116.15+2.3.6+1.21.1.jar",
}

// cameraApp is modsApp with Camera Tools synced once: its jars in the folder as
// named, and the installer's record beside the mods folder.
func cameraApp(t *testing.T, files ...string) (*App, string, string) {
	t.Helper()
	app, cfg, mods := modsApp(t, files...)
	raw, err := os.ReadFile(filepath.Join("backend", "services", "testdata", "addon", "kapital-addon-camera.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(mods), "kapital-addon-camera.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return app, cfg, mods
}

func withSuffix(names []string, suffix string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = n + suffix
	}
	return out
}

func sorted(names ...string) []string {
	out := slices.Clone(names)
	slices.Sort(out)
	return out
}

func storedAddons(t *testing.T, app *App) []string {
	t.Helper()
	settings, err := app.settings.Load()
	if err != nil {
		t.Fatal(err)
	}
	return settings.EnabledAddons["frangfurd"]
}

func TestGetChapterModsShowsCameraToolsOffAndLeavesItsJarsOutOfTheList(t *testing.T) {
	app, _, _ := cameraApp(t, append([]string{modJade}, withSuffix(cameraJars, ".disabled")...)...)
	got, err := app.GetChapterMods("frangfurd")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Addons) != 1 || got.Addons[0].ID != "camera" || got.Addons[0].Name != "Camera Tools" || got.Addons[0].Enabled {
		t.Fatalf("add-ons: %+v", got.Addons)
	}
	if !reflect.DeepEqual(got.Addons[0].Jars, cameraJars) {
		t.Fatalf("jars: %q", got.Addons[0].Jars)
	}
	if len(got.Mods) != 1 || got.Mods[0].Name != modJade {
		t.Fatalf("mods: %+v", got.Mods)
	}
}

func TestSetAddonEnabledSavesTheChoiceAndRenamesAtOnce(t *testing.T) {
	app, _, mods := cameraApp(t, append([]string{modJade}, withSuffix(cameraJars, ".disabled")...)...)
	// The player switched one of its jars off in the list before (an old list):
	// it stays off when the add-on comes on.
	if err := app.settings.Update(func(s *models.AppSettings) error {
		s.DisabledMods = map[string][]string{"frangfurd": {cameraJars[1]}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	got, err := app.SetAddonEnabled("frangfurd", "camera", true)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Addons[0].Enabled || !reflect.DeepEqual(storedAddons(t, app), []string{"camera"}) {
		t.Fatalf("on: %+v, stored %q", got.Addons, storedAddons(t, app))
	}
	want := sorted(modJade, cameraJars[0], cameraJars[1]+".disabled", cameraJars[2], cameraJars[3])
	if got := sorted(namesIn(t, mods)...); !reflect.DeepEqual(got, want) {
		t.Fatalf("on disk: %q, want %q", got, want)
	}

	if _, err := app.SetAddonEnabled("frangfurd", "camera", false); err != nil {
		t.Fatal(err)
	}
	if len(storedAddons(t, app)) != 0 {
		t.Fatalf("stored after off: %q", storedAddons(t, app))
	}
	want = sorted(append([]string{modJade}, withSuffix(cameraJars, ".disabled")...)...)
	if got := sorted(namesIn(t, mods)...); !reflect.DeepEqual(got, want) {
		t.Fatalf("on disk after off: %q, want %q", got, want)
	}
}

func TestSetAddonEnabledBeforeItsFirstSyncOnlySaves(t *testing.T) {
	app, cfg := packApp(t, commandLine("https://example.com/pack.toml"), nil)
	got, err := app.SetAddonEnabled("frangfurd", "camera", true)
	if err != nil || !got.Addons[0].Enabled || len(got.Addons[0].Jars) != 0 {
		t.Fatalf("%+v, %v", got.Addons, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(cfg), "minecraft")); !os.IsNotExist(err) {
		t.Fatal("a game folder was made")
	}
}

func TestSetAddonEnabledRefusesWhatTheManifestDoesNotName(t *testing.T) {
	app, _, _ := cameraApp(t, modJade)
	for _, c := range [][2]string{{"atlantis", "camera"}, {"frangfurd", "shaders"}, {"luxemburg", "camera"}, {"frangfurd", "../camera"}} {
		if _, err := app.SetAddonEnabled(c[0], c[1], true); err == nil {
			t.Errorf("%q accepted", c)
		}
	}
	if len(storedAddons(t, app)) != 0 {
		t.Fatal("a choice was stored")
	}
}

func TestSetAddonEnabledIsRefusedWhileTheGameRuns(t *testing.T) {
	app, cfg, mods := cameraApp(t, cameraJars...)
	before := namesIn(t, mods)
	trackFrangfurd(t, app, cfg)
	if _, err := app.SetAddonEnabled("frangfurd", "camera", false); err == nil || !strings.Contains(err.Error(), trackerRefusal) {
		t.Fatalf("got %v", err)
	}
	if !reflect.DeepEqual(namesIn(t, mods), before) || len(storedAddons(t, app)) != 0 {
		t.Fatal("something changed")
	}
}

// The mods list is the pack's: a save of it never turns an add-on that is off
// back on, and never takes an add-on's jar into the player's list.
func TestSetModsDisabledLeavesAnAddonThatIsOffAway(t *testing.T) {
	app, _, mods := cameraApp(t, append([]string{modJade, modDH}, withSuffix(cameraJars, ".disabled")...)...)
	if _, err := app.SetModsDisabled("frangfurd", []string{modDH, cameraJars[0]}); err != nil {
		t.Fatal(err)
	}
	if want := []string{modDH}; !reflect.DeepEqual(storedDisabled(t, app, "frangfurd"), want) {
		t.Fatalf("stored %q, want %q", storedDisabled(t, app, "frangfurd"), want)
	}
	want := sorted(append([]string{modJade, modDH + ".disabled"}, withSuffix(cameraJars, ".disabled")...)...)
	if got := sorted(namesIn(t, mods)...); !reflect.DeepEqual(got, want) {
		t.Fatalf("on disk: %q, want %q", got, want)
	}
}
