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

	// The launcher's own instance has a pack URL: it is never stashed.
	own := filepath.Join(dir, "kapital-frangfurd")
	if err := os.MkdirAll(own, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(own, "instance.cfg"), []byte("[General]\n"+commandLine(publishedPackURL(t, app))+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = app.ReinstallChapter("frangfurd")
	if err == nil || !strings.Contains(err.Error(), "already set up") {
		t.Errorf("the launcher's own instance: got %v", err)
	}
	if err := os.RemoveAll(own); err != nil {
		t.Fatal(err)
	}

	unhost(app, "lichdenstein")
	_, err = app.ReinstallChapter("lichdenstein")
	check("no hosted pack", "no hosted pack", err)

	app, dir = handMadeApp(t, nil)
	app.engine = models.EngineInfo{}
	_, err = app.ReinstallChapter("lichdenstein")
	check("no Prism", "", err)
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
