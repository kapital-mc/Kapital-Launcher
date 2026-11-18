package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kapital/backend/models"
)

// handMadeApp is an app with Prism found and Lichdenstein's instance there
// with no pre-launch command, as the copy issue 244 found; it returns the app
// and the instances folder.
func handMadeApp(t *testing.T, overrides map[string]string) (*App, string) {
	t.Helper()
	app := newTestApp(t)
	root := t.TempDir()
	if err := app.SaveSettings(models.AppSettings{Theme: "dark", PrismRoot: root, PackOverrides: overrides}); err != nil {
		t.Fatal(err)
	}
	app.engine = models.EngineInfo{Found: true, Source: "settings"}
	dir := filepath.Join(root, "instances")
	inst := filepath.Join(dir, "kapital-lichdenstein")
	if err := os.MkdirAll(filepath.Join(inst, "minecraft", "mods"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inst, "instance.cfg"), []byte("[General]\nname=kapital-lichdenstein\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return app, dir
}

// folders is the names in the instances folder.
func folders(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestReinstallChapterRefusesBeforeMovingAnything(t *testing.T) {
	app, dir := handMadeApp(t, nil)
	check := func(name, want string, err error) {
		t.Helper()
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: got %v, want %q", name, err, want)
		}
		if got := folders(t, dir); len(got) != 1 || got[0] != "kapital-lichdenstein" {
			t.Fatalf("%s: the instances folder changed: %v", name, got)
		}
	}
	_, err := app.ReinstallChapter("atlantis")
	check("an unknown chapter", "no chapter", err)
	_, err = app.ReinstallChapter("frangfurd")
	check("a chapter not installed", "not installed", err)

	unhost(app, "lichdenstein")
	_, err = app.ReinstallChapter("lichdenstein")
	check("no hosted pack", "no hosted pack", err)

	app, dir = handMadeApp(t, nil)
	app.engine = models.EngineInfo{}
	_, err = app.ReinstallChapter("lichdenstein")
	check("no Prism", "", err)
}

// ownApp is an app with Prism found and Frangfurd installed by the launcher,
// syncing from line's pack, with a world and an options.txt; it returns the app
// and the instances folder.
func ownApp(t *testing.T, line string, overrides map[string]string) (*App, string) {
	t.Helper()
	app, cfg := packApp(t, line, overrides)
	inst := filepath.Dir(cfg)
	for name, content := range map[string]string{
		"minecraft/saves/World/level.dat": "world",
		"minecraft/options.txt":           "renderDistance=64",
	} {
		path := filepath.Join(inst, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return app, filepath.Dir(inst)
}

// The launcher's own instance is reset from the pack it syncs from now: a
// local pack's install that cannot fetch puts the whole old folder back.
func TestReinstallOwnInstancePutsItBackWhenTheInstallFails(t *testing.T) {
	const local = "http://127.0.0.1:9/pack.toml"
	app, dir := ownApp(t, commandLine(local), map[string]string{"frangfurd": local})
	_, err := app.ReinstallChapter("frangfurd")
	if err == nil || !strings.Contains(err.Error(), "fetch pack.toml") {
		t.Fatalf("expected the install to fail at the fetch of the local pack, got %v", err)
	}
	if got := folders(t, dir); len(got) != 1 || got[0] != "kapital-frangfurd" {
		t.Fatalf("the old instance is not back: %v", got)
	}
	for _, name := range []string{"saves/World/level.dat", "options.txt"} {
		if _, err := os.Stat(filepath.Join(dir, "kapital-frangfurd", "minecraft", filepath.FromSlash(name))); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestReinstallAndUninstallRefuseDuringATerrainDownload(t *testing.T) {
	app, dir := ownApp(t, commandLine("https://example.invalid/pack.toml"), nil)
	app.terrain.mu.Lock()
	app.terrain.chapter, app.terrain.cancel = "frangfurd", func() {}
	app.terrain.mu.Unlock()
	if _, err := app.ReinstallChapter("frangfurd"); err == nil || !strings.Contains(err.Error(), "terrain download") {
		t.Errorf("reinstall: %v", err)
	}
	if _, err := app.UninstallChapter("frangfurd"); err == nil || !strings.Contains(err.Error(), "terrain download") {
		t.Errorf("uninstall: %v", err)
	}
	if got := folders(t, dir); len(got) != 1 {
		t.Fatalf("the instances folder changed: %v", got)
	}
}

func TestUninstallChapterRemovesTheWholeInstance(t *testing.T) {
	app, dir := ownApp(t, commandLine("https://example.invalid/pack.toml"), nil)
	if _, err := app.UninstallChapter("atlantis"); err == nil {
		t.Error("an unknown chapter was not refused")
	}
	if _, err := app.UninstallChapter("lichdenstein"); err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Errorf("a chapter not installed: %v", err)
	}
	report, err := app.UninstallChapter("frangfurd")
	if err != nil {
		t.Fatal(err)
	}
	if report.Present["frangfurd"] {
		t.Error("the report still has Frangfurd installed")
	}
	if got := folders(t, dir); len(got) != 0 {
		t.Fatalf("left in the instances folder: %v", got)
	}
}

func TestUninstallChapterRefusesWithoutPrism(t *testing.T) {
	app, dir := ownApp(t, commandLine("https://example.invalid/pack.toml"), nil)
	app.engine = models.EngineInfo{}
	if _, err := app.UninstallChapter("frangfurd"); err == nil {
		t.Fatal("no Prism was not refused")
	}
	if got := folders(t, dir); len(got) != 1 {
		t.Fatalf("the instances folder changed: %v", got)
	}
}

func TestReinstallChapterPutsTheOldInstanceBackWhenTheInstallFails(t *testing.T) {
	// Port 9 (discard) on loopback: nothing answers, so the install stops at
	// fetching pack.toml, after the stash.
	app, dir := handMadeApp(t, map[string]string{"lichdenstein": "http://127.0.0.1:9/pack.toml"})
	_, err := app.ReinstallChapter("lichdenstein")
	if err == nil || !strings.Contains(err.Error(), "fetch pack.toml") {
		t.Fatalf("expected the install to fail at the fetch, got %v", err)
	}
	if got := folders(t, dir); len(got) != 1 || got[0] != "kapital-lichdenstein" {
		t.Fatalf("the old instance is not back as it was: %v", got)
	}
	if got := readFile(t, filepath.Join(dir, "kapital-lichdenstein", "instance.cfg")); got != "[General]\nname=kapital-lichdenstein\n" {
		t.Fatalf("instance.cfg changed: %q", got)
	}
}

// Open in Prism on the reinstall notice names the folder Go recorded at the
// stash, and nothing else.
func TestShowStashedInstanceOpensOnlyTheRecordedStash(t *testing.T) {
	app, dir := handMadeApp(t, nil)
	var shown []string
	app.showInPrism = func(_ models.EngineInfo, id, _ string) error {
		shown = append(shown, id)
		return nil
	}
	if err := app.ShowStashedInstance("atlantis"); err == nil {
		t.Fatal("unknown chapter")
	}
	if err := app.ShowStashedInstance("lichdenstein"); err == nil || !strings.Contains(err.Error(), "no backed-up instance") {
		t.Fatalf("nothing stashed yet: %v", err)
	}
	const stash = "kapital-lichdenstein-stashed-2026-10-11"
	app.stashes = map[string]string{"lichdenstein": stash}
	if err := app.ShowStashedInstance("lichdenstein"); err == nil || !strings.Contains(err.Error(), "no longer in Prism") {
		t.Fatalf("a stash that is gone: %v", err)
	}
	if err := os.Rename(filepath.Join(dir, "kapital-lichdenstein"), filepath.Join(dir, stash)); err != nil {
		t.Fatal(err)
	}
	if err := app.ShowStashedInstance("lichdenstein"); err != nil {
		t.Fatal(err)
	}
	if len(shown) != 1 || shown[0] != stash {
		t.Fatalf("shown %v", shown)
	}
}

// A carry-over that fails is a finished reinstall: the answer names the stash,
// which still has its instance.cfg, and ShowStashedInstance can open it.
func TestReinstallRecordsTheStashWhenTheCarryOverFails(t *testing.T) {
	app, dir := handMadeApp(t, nil)
	const stash = "kapital-lichdenstein-stashed-2026-10-11"
	if err := os.Rename(filepath.Join(dir, "kapital-lichdenstein"), filepath.Join(dir, stash)); err != nil {
		t.Fatal(err)
	}
	// The new instance already has the entry the carry would move.
	for _, path := range []string{
		filepath.Join(dir, stash, "minecraft", "saves", "World", "level.dat"),
		filepath.Join(dir, "kapital-lichdenstein", "minecraft", "saves", "Other", "level.dat"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("world"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "kapital-lichdenstein", "instance.cfg"), []byte("[General]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	chapter, ok := app.chapter("lichdenstein")
	if !ok {
		t.Fatal("no chapter")
	}
	var shown []string
	app.showInPrism = func(_ models.EngineInfo, id, _ string) error {
		shown = append(shown, id)
		return nil
	}

	res, err := app.finishReinstall(chapter, dir, stash, true)
	if err != nil {
		t.Fatalf("a failed carry over is not an error: %v", err)
	}
	if res.Stash != stash {
		t.Fatalf("stash %q, want %q", res.Stash, stash)
	}
	if got := app.stashes["lichdenstein"]; got != stash {
		t.Fatalf("recorded %q", got)
	}
	if err := app.ShowStashedInstance("lichdenstein"); err != nil {
		t.Fatal(err)
	}
	if len(shown) != 1 || shown[0] != stash {
		t.Fatalf("shown %v", shown)
	}
}
