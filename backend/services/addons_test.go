package services

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"kapital/backend/models"
)

// The jars of the Camera Tools add-on, as packwiz-installer v0.5.14 recorded
// them in testdata/addon/kapital-addon-camera.json: a real run of the pinned
// installer, `--meta-file kapital-addon-camera.json`, against the
// frangfurd-camera pack of kapital-packs.
var cameraJars = []string{
	"Flashback-0.39.10-for-MC1.21.1.jar",
	"connector-2.0.0-beta.17+1.21.1-full.jar",
	"flashback_neoforge_fixed-1.0.17.jar",
	"forgified-fabric-api-0.116.15+2.3.6+1.21.1.jar",
}

const cameraPackURL = "https://kapital-packs.alessandrogekeler.workers.dev/frangfurd-camera/pack.toml"

func cameraAddon() models.PackAddon {
	return models.PackAddon{ID: "camera", Name: "Camera Tools", Description: "Flashback. About 220 MB.", Packwiz: cameraPackURL}
}

// writeCameraRecord puts the installer's real record in the game folder.
func writeCameraRecord(t *testing.T, game string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "addon", "kapital-addon-camera.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(game, addonRecordName("camera")), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBundledManifestOffersCameraToolsOnFrangfurdOnly(t *testing.T) {
	m, err := ParseManifest(bundledManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Chapters {
		if c.ID != "frangfurd" && len(c.Pack.Addons) != 0 {
			t.Errorf("%s has add-ons: %+v", c.ID, c.Pack.Addons)
		}
	}
	got := m.Chapters[2].Pack.Addons
	if len(got) != 1 || got[0].ID != "camera" || got[0].Name != "Camera Tools" || got[0].Packwiz != cameraPackURL {
		t.Fatalf("Frangfurd's add-ons: %+v", got)
	}
}

func TestValidateManifestRefusesABadAddon(t *testing.T) {
	str := func(s string) *string { return &s }
	cases := map[string]func(p *models.Pack){
		"no packwiz pack":   func(p *models.Pack) { p.Packwiz = nil },
		"id with a slash":   func(p *models.Pack) { p.Addons[0].ID = "../camera" },
		"upper case id":     func(p *models.Pack) { p.Addons[0].ID = "Camera" },
		"empty name":        func(p *models.Pack) { p.Addons[0].Name = "" },
		"padded name":       func(p *models.Pack) { p.Addons[0].Name = " Camera Tools" },
		"long name":         func(p *models.Pack) { p.Addons[0].Name = strings.Repeat("a", 41) },
		"empty description": func(p *models.Pack) { p.Addons[0].Description = "" },
		"two-line text":     func(p *models.Pack) { p.Addons[0].Description = "one\ntwo" },
		"long description":  func(p *models.Pack) { p.Addons[0].Description = strings.Repeat("a", 161) },
		"http pack": func(p *models.Pack) {
			p.Addons[0].Packwiz = "http://kapital-packs.alessandrogekeler.workers.dev/x/pack.toml"
		},
		"unlisted host":      func(p *models.Pack) { p.Addons[0].Packwiz = "https://evil.example/pack.toml" },
		"command character":  func(p *models.Pack) { p.Addons[0].Packwiz = cameraPackURL + "?a=\"b\"" },
		"the chapter's pack": func(p *models.Pack) { p.Addons[0].Packwiz = *p.Packwiz },
		"too many":           func(p *models.Pack) { p.Addons = append(p.Addons, p.Addons[0], p.Addons[0], p.Addons[0], p.Addons[0]) },
		"duplicate id":       func(p *models.Pack) { p.Addons = append(p.Addons, p.Addons[0]) },
		"duplicate name": func(p *models.Pack) {
			b := cameraAddon()
			b.ID, b.Packwiz = "other", strings.Replace(cameraPackURL, "camera", "other", 1)
			p.Addons = append(p.Addons, b)
		},
		"duplicate url": func(p *models.Pack) {
			b := cameraAddon()
			b.ID, b.Name = "other", "Other"
			p.Addons = append(p.Addons, b)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			m := validManifest()
			m.Chapters[0].Pack.Packwiz = str(testPackURL)
			m.Chapters[0].Pack.Addons = []models.PackAddon{cameraAddon()}
			if err := ValidateManifest(m); err != nil {
				t.Fatalf("the valid one is refused: %v", err)
			}
			mutate(&m.Chapters[0].Pack)
			if err := ValidateManifest(m); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestAddonJarsReadsTheInstallersRecord(t *testing.T) {
	game := gameFixture(t)
	if jars, err := AddonJars(game, "camera"); err != nil || jars != nil {
		t.Fatalf("before any sync: %q, %v", jars, err)
	}
	writeCameraRecord(t, game)
	jars, err := AddonJars(game, "camera")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(jars, cameraJars) {
		t.Fatalf("jars %q, want %q", jars, cameraJars)
	}
}

func TestAddonJarsTakesOnlyJarsDirectlyInTheModsFolder(t *testing.T) {
	game := gameFixture(t)
	record := `{"cachedFiles": {
		"a": {"cachedLocation": "mods/Good-1.0.jar"},
		"b": {"cachedLocation": "mods/../options.txt"},
		"c": {"cachedLocation": "mods/sub/Nested-1.0.jar"},
		"d": {"cachedLocation": "config/flashback.json"},
		"e": {"cachedLocation": "../mods/Escape-1.0.jar"},
		"f": {"cachedLocation": "mods/Good-1.0.jar"}
	}}`
	if err := os.WriteFile(filepath.Join(game, addonRecordName("camera")), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	jars, err := AddonJars(game, "camera")
	if err != nil || !reflect.DeepEqual(jars, []string{"Good-1.0.jar"}) {
		t.Fatalf("jars %q, %v", jars, err)
	}
	if _, err := AddonJars(game, "../x"); err == nil {
		t.Fatal("an id that is a path is read")
	}
	if err := os.WriteFile(filepath.Join(game, addonRecordName("broken")), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := AddonJars(game, "broken"); err == nil {
		t.Fatal("a broken record is read")
	}
}

func TestAddonStatesAndJarSet(t *testing.T) {
	game := gameFixture(t)
	writeCameraRecord(t, game)
	addons := []models.PackAddon{cameraAddon()}
	off := AddonStates("frangfurd", addons, models.AppSettings{}, game)
	if len(off) != 1 || off[0].Enabled || !reflect.DeepEqual(off[0].Jars, cameraJars) {
		t.Fatalf("off by default: %+v", off)
	}
	all, offJars := AddonJarSet(off)
	if len(all) != len(cameraJars) || !reflect.DeepEqual(offJars, cameraJars) {
		t.Fatalf("set %v, off %q", all, offJars)
	}
	on := AddonStates("frangfurd", addons, models.AppSettings{EnabledAddons: map[string][]string{"frangfurd": {"camera"}}}, game)
	if !on[0].Enabled {
		t.Fatalf("on: %+v", on)
	}
	if _, offJars := AddonJarSet(on); offJars != nil {
		t.Fatalf("an add-on that is on has jars off: %q", offJars)
	}
	if fresh := AddonStates("frangfurd", addons, models.AppSettings{}, ""); len(fresh[0].Jars) != 0 {
		t.Fatalf("before the first sync: %+v", fresh)
	}
}

func TestSetAddonJarsKeepsThePlayersDisabledOnesAway(t *testing.T) {
	game := gameFixture(t, cameraJars...)
	if n, err := SetAddonJars(game, cameraJars, false, nil); err != nil || n != len(cameraJars) {
		t.Fatalf("off: %d, %v", n, err)
	}
	for _, name := range modsOnDisk(t, game) {
		if !strings.HasSuffix(name, ".jar.disabled") {
			t.Fatalf("%s is not put away", name)
		}
	}
	keep := []string{cameraJars[1]}
	if _, err := SetAddonJars(game, cameraJars, true, keep); err != nil {
		t.Fatal(err)
	}
	want := []string{cameraJars[0], cameraJars[1] + ".disabled", cameraJars[2], cameraJars[3]}
	if got := modsOnDisk(t, game); !reflect.DeepEqual(got, want) {
		t.Fatalf("on: %q, want %q", got, want)
	}
}

func TestAddonSyncArgsNameTheAddonsOwnRecord(t *testing.T) {
	game := filepath.Join(t.TempDir(), "minecraft")
	args := packwizAddonSyncArgs(game, cameraPackURL, addonRecordName("camera"))
	pack := packwizSyncArgs(game, cameraPackURL)
	// The pack's own arguments, with the record named before the URL, which
	// stays last; and the pack's own slice untouched.
	want := append(append([]string{}, pack[:len(pack)-1]...), "--meta-file", "kapital-addon-camera.json", cameraPackURL)
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args %q, want %q", args, want)
	}
	if again := packwizSyncArgs(game, cameraPackURL); !reflect.DeepEqual(again, pack) {
		t.Fatalf("the pack's arguments changed: %q", again)
	}
}
