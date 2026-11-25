package services

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"kapital/backend/models"
)

const testOptions = "version:3955\r\nfov:0.5\r\nkey_key.attack:key.mouse.left\r\nresourcePacks:[\"vanilla\"]\r\nkey_key.use:key.mouse.right\r\nlang:en_us"

// settingsInstance is an instance folder whose game folder holds the named
// files, each with its own name as content, and an options.txt with two key
// lines.
func settingsInstance(t *testing.T, files ...string) (string, string) {
	t.Helper()
	dir := storageInstance(t, files...)
	game := filepath.Join(dir, "minecraft")
	if err := os.MkdirAll(game, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(game, "options.txt"), []byte(testOptions), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, game
}

func listed(t *testing.T, dir string) map[string]models.GameSettingsItem {
	t.Helper()
	items, err := ListGameSettings(dir)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]models.GameSettingsItem{}
	var order []string
	for _, it := range items {
		byID[it.ID] = it
		order = append(order, it.ID)
	}
	byID["_order"] = models.GameSettingsItem{Name: strings.Join(order, " ")}
	return byID
}

func TestListGameSettingsShowsWhatTheInstanceHas(t *testing.T) {
	dir, _ := settingsInstance(t,
		"minecraft/config/sodium-options.json",
		"minecraft/config/iris.properties",
		"minecraft/config/DistantHorizons.toml",
		"minecraft/shaderpacks/photon_v1.3b.zip.txt",
		"minecraft/shaderpacks/ComplementaryUnbound_r5.9.3 + EuphoriaPatches_1.10.5.txt",
		"minecraft/shaderpacks/photon_v1.3b.zip",
		"minecraft/shaderpacks/notes/readme.txt",
	)
	got := listed(t, dir)
	if order := got["_order"].Name; order != "options keys video iris shaderSettings distantHorizons" {
		t.Fatalf("order %q", order)
	}
	if k := got["keys"]; k.Keys != 2 || strings.Join(k.Files, ",") != "options.txt" {
		t.Fatalf("keys %+v", k)
	}
	if v := got["video"]; strings.Join(v.Files, ",") != "config/sodium-options.json" || v.Name != "Video settings" {
		t.Fatalf("video %+v", v)
	}
	want := "shaderpacks/ComplementaryUnbound_r5.9.3 + EuphoriaPatches_1.10.5.txt,shaderpacks/photon_v1.3b.zip.txt"
	if s := strings.Join(got["shaderSettings"].Files, ","); s != want {
		t.Fatalf("shader settings %q", s)
	}
}

func TestListGameSettingsKeepsAnItemTheSyncInstalled(t *testing.T) {
	dir, game := settingsInstance(t)
	record := `{"cachedFiles":{"config/DistantHorizons.toml":{"cachedLocation":"config/DistantHorizons.toml"},"shaderpacks/photon_v1.3b.zip.txt":{}}}`
	if err := os.WriteFile(filepath.Join(game, "packwiz.json"), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	got := listed(t, dir)
	if order := got["_order"].Name; order != "options keys shaderSettings distantHorizons" {
		t.Fatalf("order %q", order)
	}
	if f := got["distantHorizons"].Files; len(f) != 0 {
		t.Fatalf("files %v", f)
	}
}

func TestListGameSettingsWithoutAGameFolder(t *testing.T) {
	items, err := ListGameSettings(t.TempDir())
	if err != nil || items == nil || len(items) != 0 {
		t.Fatalf("%v %v", items, err)
	}
}

func TestResetGameSettingDeletesTheItemsFilesOnly(t *testing.T) {
	dir, game := settingsInstance(t,
		"minecraft/config/sodium-options.json",
		"minecraft/config/sodium-extra-options.json",
		"minecraft/config/sodium-mixins.properties",
		"minecraft/shaderpacks/photon_v1.3b.zip.txt",
		"minecraft/shaderpacks/photon_v1.3b.zip",
	)
	if err := ResetGameSetting(dir, "video"); err != nil {
		t.Fatal(err)
	}
	if err := ResetGameSetting(dir, GameSettingShaders); err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"config/sodium-options.json", "config/sodium-extra-options.json", "shaderpacks/photon_v1.3b.zip.txt"} {
		if _, err := os.Stat(filepath.Join(game, gone)); !os.IsNotExist(err) {
			t.Fatalf("%s is still there: %v", gone, err)
		}
	}
	for _, kept := range []string{"config/sodium-mixins.properties", "shaderpacks/photon_v1.3b.zip", "options.txt"} {
		if _, err := os.Stat(filepath.Join(game, kept)); err != nil {
			t.Fatalf("%s went: %v", kept, err)
		}
	}
	// Nothing left is nothing to reset.
	if err := ResetGameSetting(dir, "video"); err != nil {
		t.Fatal(err)
	}
	if err := ResetGameSetting(dir, "options"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(game, "options.txt")); !os.IsNotExist(err) {
		t.Fatalf("options.txt is still there: %v", err)
	}
}

func TestResetGameSettingTakesOutOnlyTheKeyLines(t *testing.T) {
	dir, game := settingsInstance(t)
	if err := ResetGameSetting(dir, GameSettingKeys); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(game, "options.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "version:3955\r\nfov:0.5\r\nresourcePacks:[\"vanilla\"]\r\nlang:en_us"; string(raw) != want {
		t.Fatalf("got %q", raw)
	}
	if _, err := os.Stat(filepath.Join(game, optionsTemp)); !os.IsNotExist(err) {
		t.Fatalf("the temporary file stayed: %v", err)
	}
	if k := listed(t, dir)["keys"]; k.Keys != 0 || len(k.Files) != 0 {
		t.Fatalf("keys %+v", k)
	}
}

func TestResetGameSettingRefusesAnythingElse(t *testing.T) {
	dir, game := settingsInstance(t, "minecraft/config/iris.properties")
	for _, id := range []string{"", "mods", "../options", "config/iris.properties"} {
		if err := ResetGameSetting(dir, id); err == nil {
			t.Fatalf("%q was reset", id)
		}
	}
	if err := ResetGameSetting(t.TempDir(), "options"); err == nil {
		t.Fatal("an instance without a game folder was reset")
	}
	if _, err := os.Stat(filepath.Join(game, "config", "iris.properties")); err != nil {
		t.Fatal(err)
	}
}

func TestResetGameSettingLeavesALinkOrFolderInPlace(t *testing.T) {
	dir, game := settingsInstance(t)
	if err := os.MkdirAll(filepath.Join(game, "config", "DistantHorizons.toml"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ResetGameSetting(dir, "distantHorizons"); err == nil || !strings.Contains(err.Error(), "left as it is") {
		t.Fatalf("got %v", err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	outside := filepath.Join(t.TempDir(), "iris.properties")
	if err := os.WriteFile(outside, []byte("enableShaders=true"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(game, "config", "iris.properties")); err != nil {
		t.Fatal(err)
	}
	if err := ResetGameSetting(dir, "iris"); err == nil {
		t.Fatal("a link was removed")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal(err)
	}
	opts := filepath.Join(game, "options.txt")
	if err := os.Remove(opts); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, opts); err != nil {
		t.Fatal(err)
	}
	if err := ResetGameSetting(dir, GameSettingKeys); err == nil {
		t.Fatal("options.txt through a link was rewritten")
	}
}

func TestWithoutKeyLinesKeepsEveryOtherLine(t *testing.T) {
	out, n := withoutKeyLines([]byte("key_a:b\nfov:1\nkey_c:d"))
	if n != 2 || string(out) != "fov:1\n" {
		t.Fatalf("%q %d", out, n)
	}
	if out, n := withoutKeyLines(nil); n != 0 || len(out) != 0 {
		t.Fatalf("%q %d", out, n)
	}
}
