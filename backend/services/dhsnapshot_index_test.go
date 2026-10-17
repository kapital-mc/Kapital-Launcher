package services

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"kapital/backend/models"
)

// konnektIndex is the shape Konnekt publishes (its docs/distant-horizons.md),
// with Frangfurd's own names.
const konnektIndex = `{
  "format": 1,
  "createdAt": "2026-10-13T12:03:41Z",
  "partsBase": "../../parts/",
  "dhVersion": "3.3.3",
  "serverFolder": "1790374988_frangfurd",
  "levels": [
    {
      "folder": "ff@minecraft@@overworld",
      "size": 12,
      "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "parts": [
        {"sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "size": 8},
        {"sha256": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "size": 4}
      ]
    },
    {
      "folder": "ff@northstar@@earth_orbit",
      "size": 3,
      "sha256": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
      "parts": [{"sha256": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "size": 3}]
    }
  ]
}`

func TestParseDHIndexReadsKonnektsIndex(t *testing.T) {
	ix, err := ParseDHIndex([]byte(konnektIndex))
	if err != nil {
		t.Fatal(err)
	}
	if ix.ServerFolder != "1790374988_frangfurd" || len(ix.Levels) != 2 || ix.Total() != 15 {
		t.Errorf("%+v", ix)
	}
}

// editIndex returns konnektIndex with one change made to its decoded form.
func editIndex(t *testing.T, edit func(m map[string]any)) []byte {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(konnektIndex), &m); err != nil {
		t.Fatal(err)
	}
	edit(m)
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func level(m map[string]any, i int) map[string]any {
	return m["levels"].([]any)[i].(map[string]any)
}

func TestParseDHIndexRefuses(t *testing.T) {
	cases := map[string]func(m map[string]any){
		"a newer format":              func(m map[string]any) { m["format"] = 2 },
		"an unknown field":            func(m map[string]any) { m["command"] = "rm" },
		"a time that is not one":      func(m map[string]any) { m["createdAt"] = "yesterday" },
		"an absolute parts base":      func(m map[string]any) { m["partsBase"] = "https://evil.example/parts/" },
		"a rooted parts base":         func(m map[string]any) { m["partsBase"] = "/parts/" },
		"a parts base elsewhere":      func(m map[string]any) { m["partsBase"] = "../../other/" },
		"a version that is not one":   func(m map[string]any) { m["dhVersion"] = "latest" },
		"a server folder path":        func(m map[string]any) { m["serverFolder"] = "../1_frangfurd" },
		"a device name":               func(m map[string]any) { m["serverFolder"] = "CON" },
		"a server folder too long":    func(m map[string]any) { m["serverFolder"] = "1_" + strings.Repeat("a", 150) },
		"no levels":                   func(m map[string]any) { m["levels"] = []any{} },
		"a level folder that climbs":  func(m map[string]any) { level(m, 0)["folder"] = "..@minecraft@@overworld" },
		"a level folder with a slash": func(m map[string]any) { level(m, 0)["folder"] = "ff@minecraft@@a/b" },
		"a level folder with no key":  func(m map[string]any) { level(m, 0)["folder"] = "minecraft@@overworld" },
		"a twice named folder":        func(m map[string]any) { level(m, 1)["folder"] = "ff@minecraft@@overworld" },
		"an uppercase hash":           func(m map[string]any) { level(m, 0)["sha256"] = strings.Repeat("A", 64) },
		"a short part hash": func(m map[string]any) {
			level(m, 0)["parts"].([]any)[0].(map[string]any)["sha256"] = "abc"
		},
		"parts that do not add up": func(m map[string]any) { level(m, 0)["size"] = 13 },
		"parts over the level":     func(m map[string]any) { level(m, 0)["size"] = 7 },
		"an empty part": func(m map[string]any) {
			level(m, 1)["parts"].([]any)[0].(map[string]any)["size"] = 0
		},
		"a part over the cap": func(m map[string]any) {
			level(m, 1)["size"] = dhMaxPart + 1
			level(m, 1)["parts"].([]any)[0].(map[string]any)["size"] = dhMaxPart + 1
		},
	}
	for name, edit := range cases {
		if _, err := ParseDHIndex(editIndex(t, edit)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestParseDHIndexRefusesASnapshotOverTheCap(t *testing.T) {
	parts := make([]DHPart, 0, 1100)
	for range 1100 {
		parts = append(parts, DHPart{SHA256: strings.Repeat("b", 64), Size: dhMaxPart})
	}
	ix := DHIndex{
		Format: 1, CreatedAt: "2026-10-13T12:03:41Z", PartsBase: "../../parts/", DHVersion: "3.3.3",
		ServerFolder: "1_frangfurd",
		Levels:       []DHLevel{{Folder: "ff@minecraft@@overworld", Size: 1100 * dhMaxPart, SHA256: strings.Repeat("a", 64), Parts: parts}},
	}
	data, err := json.Marshal(ix)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseDHIndex(data); err == nil {
		t.Error("a 68 GiB snapshot was accepted")
	}
}

func TestParseDHPointer(t *testing.T) {
	good := `{"format":1,"createdAt":"2026-10-13T12:03:41Z","index":"snapshots/20261013T120341Z/index.json"}`
	if _, err := ParseDHPointer([]byte(good)); err != nil {
		t.Fatal(err)
	}
	for _, index := range []string{
		"https://evil.example/snapshots/20261013T120341Z/index.json",
		"//evil.example/index.json",
		"/snapshots/20261013T120341Z/index.json",
		"../snapshots/20261013T120341Z/index.json",
		"snapshots/20261013T120341Z/../../index.json",
		"snapshots/latest/index.json",
	} {
		bad := `{"format":1,"createdAt":"2026-10-13T12:03:41Z","index":"` + index + `"}`
		if _, err := ParseDHPointer([]byte(bad)); err == nil {
			t.Errorf("index %q accepted", index)
		}
	}
	if _, err := ParseDHPointer([]byte(`{"format":2,"createdAt":"x","index":"snapshots/20261013T120341Z/index.json"}`)); err == nil {
		t.Error("format 2 accepted")
	}
	if _, err := ParseDHPointer([]byte(good + good)); err == nil {
		t.Error("two values accepted")
	}
}

func TestDHAddressesStayOnTheSnapshotsHost(t *testing.T) {
	pointer, err := url.Parse("https://dh.frangfurd.click/dh/latest.json")
	if err != nil {
		t.Fatal(err)
	}
	iu, err := dhIndexURL(pointer, DHPointer{Index: "snapshots/20261013T120341Z/index.json"})
	if err != nil || iu.String() != "https://dh.frangfurd.click/dh/snapshots/20261013T120341Z/index.json" {
		t.Fatalf("%v %v", iu, err)
	}
	ix := DHIndex{PartsBase: "../../parts/"}
	pu, err := dhPartURL(iu, ix, DHPart{SHA256: strings.Repeat("b", 64)})
	if err != nil || pu.String() != "https://dh.frangfurd.click/dh/parts/"+strings.Repeat("b", 64) {
		t.Fatalf("%v %v", pu, err)
	}
	for _, ref := range []string{"https://evil.example/x", "//evil.example/x", "/x", "x?y=1"} {
		if _, err := dhResolve(iu, ref); err == nil {
			t.Errorf("%q resolved", ref)
		}
	}
}

func TestManifestHoldsTheSnapshotAddress(t *testing.T) {
	for raw, ok := range map[string]bool{
		"https://dh.frangfurd.click/dh/latest.json":             true,
		"https://dh.frangfurd.click/latest.json":                true,
		"http://dh.frangfurd.click/dh/latest.json":              false,
		"https://evil.example/dh/latest.json":                   false,
		"https://dh.frangfurd.click/dh/index.json":              false,
		"https://dh.frangfurd.click/dh/latest.json?x=1":         false,
		"https://dh.frangfurd.click/dh/latest.json#x":           false,
		"https://dh.frangfurd.click/dh/../latest.json":          false,
		"https://dh.frangfurd.click/dh//latest.json":            false,
		"https://user@dh.frangfurd.click/dh/latest.json":        false,
		"https://dh.frangfurd.click.evil.example/x/latest.json": false,
	} {
		err := checkDHSnapshotURL("pack.dhSnapshot", raw)
		if (err == nil) != ok {
			t.Errorf("%s: %v", raw, err)
		}
	}
}

func TestDHVersionNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"3.3.3", "3.3.3", false},
		{"3.4.0", "3.3.3", true},
		{"3.3.3", "3.4.0", false},
		{"3.3.3.1", "3.3.3", true},
		{"3.3", "3.3.0", false},
		{"3.3.3-b", "3.3.3", false},
		{"10.0.0", "9.9.9", true},
	} {
		got, err := DHVersionNewer(c.a, c.b)
		if err != nil || got != c.want {
			t.Errorf("%s > %s: %v %v", c.a, c.b, got, err)
		}
	}
	if _, err := DHVersionNewer("new", "3.3.3"); err == nil {
		t.Error("a word compared")
	}
}

func TestClientDHVersionReadsTheJarName(t *testing.T) {
	mods := []models.ModFile{
		{Name: "iris-neoforge-1.8.12+mc1.21.1.jar"},
		{Name: "DistantHorizons-3.3.3-1.21.1-fabric-neoforge.jar", Disabled: true},
	}
	if got := ClientDHVersion(mods); got != "3.3.3" {
		t.Errorf("got %q", got)
	}
	if got := ClientDHVersion(mods[:1]); got != "" {
		t.Errorf("got %q with no Distant Horizons", got)
	}
}
