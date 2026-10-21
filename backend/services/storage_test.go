package services

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"kapital/backend/models"
)

// storageInstance is an instance folder holding the named files, each with
// its own path as its content, so every file has a sumLen of its own.
func storageInstance(t *testing.T, files ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const (
	stOverworld = "minecraft/Distant_Horizons_server_data/1790374988_frangfurd/ff@minecraft@@overworld/DistantHorizons.sqlite"
	stNether    = "minecraft/Distant_Horizons_server_data/1790374988_frangfurd/ff@minecraft@@the_nether/DistantHorizons.sqlite"
	stOldServer = "minecraft/Distant_Horizons_server_data/Frangfurd+%28Germany%29/ff@minecraft@@overworld/DistantHorizons.sqlite"
	stMap       = "minecraft/xaero/world-map/Multiplayer_frangfurd/null/0_0.zip"
	stWaypoints = "minecraft/xaero/minimap/Multiplayer_frangfurd/waypoints.txt"
	stWorld     = "minecraft/saves/Convert/region/r.0.0.mca"
	stWorldDat  = "minecraft/saves/Convert/level.dat"
	stTestWorld = "minecraft/saves/Test/level.dat"
	stShot      = "minecraft/screenshots/2026-10-20_21.38.11.png"
	stLatestLog = "minecraft/logs/latest.log"
	stCrash     = "minecraft/crash-reports/crash-2026-10-12_16.19.00-client.txt"
	stModJar    = "minecraft/mods/Jade-1.21.1-NeoForge-15.1.jar"
	stOptions   = "minecraft/options.txt"
	stStaged    = "kapital-dh-download/ff@minecraft@@overworld.sqlite"
	stCfg       = "instance.cfg"
)

func sumLen(names ...string) int64 {
	var n int64
	for _, s := range names {
		n += int64(len(s))
	}
	return n
}

func TestMeasureStorageSortsAnInstanceByKind(t *testing.T) {
	dir := storageInstance(t, stOverworld, stNether, stOldServer, stMap, stWaypoints, stTestWorld,
		stWorld, stWorldDat, stShot, stLatestLog, stCrash, stModJar, stOptions, stStaged, stCfg)
	got, err := MeasureStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		kind, name, label string
		bytes             int64
		files             int
	}
	var rows []row
	for _, it := range got.Items {
		rows = append(rows, row{it.Kind, it.Name, it.Label, it.Bytes, it.Files})
		if it.LastModified == "" {
			t.Errorf("%s has no time", it.Kind)
		}
	}
	want := []row{
		{models.StorageTerrain, "1790374988_frangfurd", "1790374988_frangfurd", sumLen(stOverworld, stNether), 2},
		{models.StorageTerrain, "Frangfurd+%28Germany%29", "Frangfurd (Germany)", sumLen(stOldServer), 1},
		{models.StorageTerrainDownload, "", "", sumLen(stStaged), 1},
		{models.StorageWorldMap, "", "", sumLen(stMap), 1},
		// The minimap's stWaypoints are not the world map: they are the player's own.
		{models.StorageMinimap, "", "", sumLen(stWaypoints), 1},
		// One row per world, largest first.
		{models.StorageSaves, "Convert", "", sumLen(stWorld, stWorldDat), 2},
		{models.StorageSaves, "Test", "", sumLen(stTestWorld), 1},
		{models.StorageScreenshots, "", "", sumLen(stShot), 1},
		{models.StorageLogs, "", "", sumLen(stLatestLog, stCrash), 2},
		{models.StorageMods, "", "", sumLen(stModJar), 1},
		{models.StorageOther, "", "", sumLen(stOptions, stCfg), 2},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Fatalf("got  %+v\nwant %+v", rows, want)
	}
	if got.TotalBytes != sumLen(stOverworld, stNether, stOldServer, stMap, stWaypoints, stTestWorld, stWorld, stWorldDat, stShot, stLatestLog, stCrash, stModJar, stOptions, stStaged, stCfg) {
		t.Errorf("total %d", got.TotalBytes)
	}
	if got.FreeBytes <= 0 || got.Partial {
		t.Errorf("free %d, partial %v", got.FreeBytes, got.Partial)
	}
}

func TestMeasureStorageLeavesOutWhatIsNotThere(t *testing.T) {
	got, err := MeasureStorage(storageInstance(t, stCfg))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Kind != models.StorageOther {
		t.Fatalf("%+v", got.Items)
	}
	if _, err := MeasureStorage(filepath.Join(t.TempDir(), "gone")); err == nil {
		t.Fatal("a missing instance measured")
	}
}

func TestMeasureStorageDoesNotFollowLinks(t *testing.T) {
	outside := storageInstance(t, "big.bin")
	dir := storageInstance(t, stCfg, "minecraft/Distant_Horizons_server_data/keep.txt")
	link := filepath.Join(dir, "minecraft", "Distant_Horizons_server_data", "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("no symlinks here:", err)
	}
	got, err := MeasureStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range got.Items {
		if it.Kind == models.StorageTerrain {
			t.Fatalf("a link was measured as terrain: %+v", it)
		}
	}
}

func TestClearStorageDeletesOnlyItsOwnFolder(t *testing.T) {
	all := []string{stOverworld, stNether, stOldServer, stMap, stWaypoints, stShot, stLatestLog, stCrash, stModJar, stOptions, stStaged, stCfg}
	cases := []struct {
		kind, name string
		gone       []string
	}{
		{models.StorageTerrain, "Frangfurd+%28Germany%29", []string{stOldServer}},
		{models.StorageTerrain, "1790374988_frangfurd", []string{stOverworld, stNether}},
		{models.StorageTerrainDownload, "", []string{stStaged}},
		{models.StorageWorldMap, "", []string{stMap}},
		{models.StorageScreenshots, "", []string{stShot}},
		{models.StorageLogs, "", []string{stLatestLog, stCrash}},
	}
	for _, c := range cases {
		t.Run(c.kind+c.name, func(t *testing.T) {
			dir := storageInstance(t, all...)
			if err := ClearStorage(dir, c.kind, c.name); err != nil {
				t.Fatal(err)
			}
			for _, f := range all {
				_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f)))
				wantGone := false
				for _, g := range c.gone {
					wantGone = wantGone || g == f
				}
				if wantGone != os.IsNotExist(err) {
					t.Errorf("%s: gone %v, want %v (%v)", f, os.IsNotExist(err), wantGone, err)
				}
			}
			// Again: nothing left to delete is not an error.
			if err := ClearStorage(dir, c.kind, c.name); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClearStorageRefusesAnythingElse(t *testing.T) {
	dir := storageInstance(t, stOverworld, stWorld, stShot, stModJar, stOptions, stCfg)
	for _, c := range []struct{ kind, name string }{
		{models.StorageMinimap, ""},
		{models.StorageMods, ""},
		{models.StorageOther, ""},
		{models.StorageSaves, "Convert"},
		{models.StorageSaves, ""},
		{models.StorageTerrain, ""},
		{models.StorageTerrain, ".."},
		{models.StorageTerrain, "../mods"},
		{models.StorageTerrain, `..\mods`},
		{models.StorageTerrain, "C:"},
	} {
		if err := ClearStorage(dir, c.kind, c.name); err == nil {
			t.Errorf("%q %q: deleted", c.kind, c.name)
		}
	}
	for _, f := range []string{stOverworld, stWorld, stShot, stModJar, stOptions, stCfg} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(f))); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
}

func TestClearStorageLeavesALinkOrFileInPlace(t *testing.T) {
	outside := storageInstance(t, "keep.txt")
	dir := storageInstance(t, stCfg, "minecraft/options.txt", "minecraft/logs")
	if err := ClearStorage(dir, models.StorageLogs, ""); err == nil || !strings.Contains(err.Error(), "not a folder") {
		t.Fatalf("a file named logs: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "minecraft", "Distant_Horizons_server_data")); err != nil {
		t.Skip("no symlinks here:", err)
	}
	if err := os.Mkdir(filepath.Join(outside, "server"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ClearStorage(dir, models.StorageTerrain, "server"); err == nil {
		t.Fatal("deleted through a link")
	}
	if _, err := os.Stat(filepath.Join(outside, "server")); err != nil {
		t.Fatalf("the folder outside went: %v", err)
	}
}

func TestClearStorageWithoutAGameFolder(t *testing.T) {
	dir := storageInstance(t, stCfg, stStaged)
	if err := ClearStorage(dir, models.StorageLogs, ""); err == nil {
		t.Fatal("no game folder, yet cleared")
	}
	if err := ClearStorage(dir, models.StorageTerrainDownload, ""); err != nil {
		t.Fatal(err)
	}
}

func TestStorageFolderIsTheKindsOwnFolder(t *testing.T) {
	dir := storageInstance(t, stOverworld, stMap, stWaypoints, stWorld, stShot, stCrash, stModJar, stOptions, stStaged, stCfg)
	game := filepath.Join(dir, "minecraft")
	for _, c := range []struct{ kind, name, want string }{
		{models.StorageTerrain, "1790374988_frangfurd", filepath.Join(game, "Distant_Horizons_server_data", "1790374988_frangfurd")},
		{models.StorageTerrainDownload, "", filepath.Join(dir, "kapital-dh-download")},
		{models.StorageWorldMap, "", filepath.Join(game, "xaero", "world-map")},
		{models.StorageMinimap, "", filepath.Join(game, "xaero", "minimap")},
		{models.StorageSaves, "Convert", filepath.Join(game, "saves", "Convert")},
		{models.StorageScreenshots, "", filepath.Join(game, "screenshots")},
		// No logs folder yet: the crash reports are where the rest is.
		{models.StorageLogs, "", filepath.Join(game, "crash-reports")},
		{models.StorageMods, "", filepath.Join(game, "mods")},
		{models.StorageOther, "", game},
	} {
		got, err := StorageFolder(dir, c.kind, c.name)
		if err != nil || got != c.want {
			t.Errorf("%s %q: %q, %v; want %q", c.kind, c.name, got, err, c.want)
		}
	}
	for _, c := range []struct{ kind, name string }{
		{models.StorageTerrain, "Frangfurd+%28Germany%29"},
		{models.StorageTerrain, ".."},
		{models.StorageTerrain, "../mods"},
		{models.StorageSaves, ""},
		{models.StorageSaves, "../mods"},
		{models.StorageSaves, "Test"},
	} {
		if got, err := StorageFolder(dir, c.kind, c.name); err == nil {
			t.Errorf("%s %q: opened %q", c.kind, c.name, got)
		}
	}
	// Before the first start there is no game folder; everything else is the instance.
	bare := storageInstance(t, stCfg)
	if got, err := StorageFolder(bare, models.StorageOther, ""); err != nil || got != bare {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := StorageFolder(bare, models.StorageScreenshots, ""); err == nil {
		t.Error("screenshots opened without a game folder")
	}
}
