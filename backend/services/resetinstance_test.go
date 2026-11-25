package services

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"kapital/backend/models"
)

// writeTree writes each file, relative to dir, with its content.
func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// The whole reset as ReinstallChapter runs it: stash, a real install from the
// fake pack host, carry over, remove the rest.
func TestResetKeepsThePlayersFilesAndDropsThePacks(t *testing.T) {
	h, c := newFakePackHost(t)
	dir := t.TempDir()
	old := filepath.Join(dir, "kapital-frangfurd")
	writeTree(t, old, map[string]string{
		"instance.cfg":                                                 "[General]\nname=Frangfurd\nMaxMemAlloc=16000\ntotalTimePlayed=7200\nlastLaunchTime=1700000000000\n",
		"kapital-dh-download/part":                                     "staged",
		"minecraft/options.txt":                                        "renderDistance=64",
		"minecraft/config/iris.properties":                             "enableShaders=true",
		"minecraft/mods/broken.jar":                                    "jar",
		"minecraft/shaderpacks/Photon.zip.txt":                         "COLORED_LIGHTS=true",
		"minecraft/saves/World/level.dat":                              "world",
		"minecraft/screenshots/a.png":                                  "png",
		"minecraft/schematics/bridge.nbt":                              "nbt",
		"minecraft/enxv_aeronautics_structures/Bus.excraft":            "excraft",
		"minecraft/Distant_Horizons_server_data/1_frangfurd/dh.sqlite": "terrain",
		"minecraft/xaero/minimap/waypoints.txt":                        "wp",
		"minecraft/flashback/replays/r.zip":                            "replay",
		"minecraft/servers.dat":                                        "servers",
		"minecraft/essential/settings.json":                            "{}",
	})
	stashed, err := StashInstance(dir, "kapital-frangfurd", time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	report := models.InstanceReport{Dir: dir, Present: map[string]bool{"frangfurd": false}}
	if err := c.Install(context.Background(), frangfurdChapter(h.srv.URL+"/frangfurd/pack.toml"), report, ""); err != nil {
		t.Fatal(err)
	}
	moved, err := CarryOver(dir, stashed, "kapital-frangfurd")
	if err != nil || moved != 10 {
		t.Fatalf("moved %d, %v", moved, err)
	}
	if err := RemoveInstance(dir, stashed, time.Now()); err != nil {
		t.Fatal(err)
	}

	for _, kept := range []string{
		"kapital-dh-download/part",
		"minecraft/saves/World/level.dat",
		"minecraft/screenshots/a.png",
		"minecraft/schematics/bridge.nbt",
		"minecraft/enxv_aeronautics_structures/Bus.excraft",
		"minecraft/Distant_Horizons_server_data/1_frangfurd/dh.sqlite",
		"minecraft/xaero/minimap/waypoints.txt",
		"minecraft/flashback/replays/r.zip",
		"minecraft/servers.dat",
		"minecraft/essential/settings.json",
		"minecraft/packwiz-installer-bootstrap.jar",
	} {
		if !exists(filepath.Join(old, filepath.FromSlash(kept))) {
			t.Errorf("%s is not in the new instance", kept)
		}
	}
	for _, gone := range []string{"minecraft/options.txt", "minecraft/config", "minecraft/mods", "minecraft/shaderpacks"} {
		if exists(filepath.Join(old, filepath.FromSlash(gone))) {
			t.Errorf("%s survived the reset", gone)
		}
	}
	cfg := readTestFile(t, filepath.Join(old, "instance.cfg"))
	if !strings.Contains(cfg, "totalTimePlayed=7200") || !strings.Contains(cfg, "lastLaunchTime=1700000000000") {
		t.Errorf("play time not carried:\n%s", cfg)
	}
	if strings.Contains(cfg, "16000") {
		t.Errorf("the old memory setting survived:\n%s", cfg)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("the instances folder holds more than the new instance: %v %v", entries, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCarryOverNeverPutsAnythingOverAFile(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"old/minecraft/saves/World/level.dat": "old",
		"new/minecraft/saves/World/level.dat": "new",
		"new/instance.cfg":                    "[General]\n",
	})
	if _, err := CarryOver(dir, "old", "new"); err == nil || !strings.Contains(err.Error(), "already in the new instance") {
		t.Fatalf("got %v", err)
	}
	if got := readTestFile(t, filepath.Join(dir, "new/minecraft/saves/World/level.dat")); got != "new" {
		t.Fatalf("the new file was replaced: %q", got)
	}
	if !exists(filepath.Join(dir, "old/minecraft/saves/World/level.dat")) {
		t.Fatal("the old world left the stash")
	}
}

func TestCarryOverFollowsADotMinecraftFolder(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"old/.minecraft/saves/World/level.dat": "old",
		"new/minecraft/packwiz.jar":            "jar",
		"new/instance.cfg":                     "[General]\n",
	})
	if moved, err := CarryOver(dir, "old", "new"); err != nil || moved != 1 {
		t.Fatalf("moved %d, %v", moved, err)
	}
	if !exists(filepath.Join(dir, "new/minecraft/saves/World/level.dat")) {
		t.Fatal("the world is not in the new game folder")
	}
}

func TestCarryOverRefusesAPathForAName(t *testing.T) {
	for _, pair := range [][2]string{{"../x", "new"}, {"old", "a/b"}, {"", "new"}} {
		if _, err := CarryOver(t.TempDir(), pair[0], pair[1]); err == nil {
			t.Errorf("%q -> %q was not refused", pair[0], pair[1])
		}
	}
}

func TestRemoveInstanceRemovesTheWholeFolder(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"kapital-frangfurd/instance.cfg":                    "[General]\n",
		"kapital-frangfurd/minecraft/saves/World/level.dat": "world",
		"other/instance.cfg":                                "[General]\n",
	})
	if err := RemoveInstance(dir, "kapital-frangfurd", time.Now()); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != "other" {
		t.Fatalf("left %v, %v", entries, err)
	}
	// Gone already: nothing to do.
	if err := RemoveInstance(dir, "kapital-frangfurd", time.Now()); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveInstanceRefuses(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{"file": "x"})
	for _, name := range []string{"..", "../x", "a/b", "", "file"} {
		if err := RemoveInstance(dir, name, time.Now()); err == nil {
			t.Errorf("%q was not refused", name)
		}
	}
	if !exists(filepath.Join(dir, "file")) {
		t.Fatal("a file in an instance's place was removed")
	}
}

func TestCarryOverMovesALinkAsALink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a test runner cannot make links on Windows")
	}
	dir := t.TempDir()
	elsewhere := t.TempDir()
	writeTree(t, elsewhere, map[string]string{"World/level.dat": "world"})
	writeTree(t, dir, map[string]string{"new/minecraft/x": "", "new/instance.cfg": "[General]\n"})
	if err := os.MkdirAll(filepath.Join(dir, "old/minecraft"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(dir, "old/minecraft/saves")); err != nil {
		t.Fatal(err)
	}
	if _, err := CarryOver(dir, "old", "new"); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(filepath.Join(dir, "new/minecraft/saves"))
	if err != nil || target != elsewhere {
		t.Fatalf("saves is not the same link: %q %v", target, err)
	}
	if !exists(filepath.Join(elsewhere, "World/level.dat")) {
		t.Fatal("the link's target changed")
	}
}
