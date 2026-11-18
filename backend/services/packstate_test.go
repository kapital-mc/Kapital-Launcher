package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kapital/backend/models"
)

// packwizState is packwiz-installer's record for a synced pack.toml with
// the given SHA-256, the file list left out.
func packwizState(hash string) string {
	return `{"packFileHash":{"type":"sha256","value":"` + hash + `"},"indexFileHash":{"type":"sha256","value":"x"},"cachedFiles":{},"cachedSide":"client"}`
}

func writeInstanceWithState(t *testing.T, instances, id, state string) {
	t.Helper()
	game := filepath.Join(instances, id, "minecraft")
	if err := os.MkdirAll(game, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instances, id, "instance.cfg"), []byte("[General]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if state != "" {
		if err := os.WriteFile(filepath.Join(game, packwizStateName), []byte(state), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPackStateComparesTheSyncedHashWithTheSource(t *testing.T) {
	h, c := newFakePackHost(t)
	instances := t.TempDir()
	packURL := h.srv.URL + "/frangfurd/pack.toml"
	chapter := frangfurdChapter(packURL)
	sum := sha256.Sum256([]byte(frangfurdPackTOML))
	current := hex.EncodeToString(sum[:])
	ctx := context.Background()

	// Not installed: nothing to compare, but the source's version is read.
	report := models.InstanceReport{Dir: instances, Present: map[string]bool{}, PackURL: map[string]string{}}
	if got := c.PackState(ctx, chapter, report, ""); got.Installed || !got.Checked || got.UpToDate || got.Version != "1.0.0" {
		t.Fatalf("%+v", got)
	}

	// Installed and synced to the current pack: up to date, with its version.
	writeInstanceWithState(t, instances, "kapital-frangfurd", packwizState(current))
	report.Present["frangfurd"] = true
	report.PackURL["frangfurd"] = packURL
	got := c.PackState(ctx, chapter, report, "")
	if !got.Installed || !got.Checked || !got.UpToDate || got.Version != "1.0.0" {
		t.Fatalf("%+v", got)
	}

	// The source moved on: the hashes differ, and the new version is named.
	h.packTOML = strings.Replace(frangfurdPackTOML, `version = "1.0.0"`, `version = "1.1.0"`, 1)
	got = c.PackState(ctx, chapter, report, "")
	if !got.Checked || got.UpToDate || got.Version != "1.1.0" {
		t.Fatalf("%+v", got)
	}

	// Never synced (no packwiz.json): not installed as far as the pack goes.
	writeInstanceWithState(t, instances, "kapital-luxemburg", "")
	lux := models.Chapter{ID: "luxemburg", Instance: models.Instance{ID: "kapital-luxemburg"}}
	report.Present["luxemburg"] = true
	if got := c.PackState(ctx, lux, report, ""); got.Installed {
		t.Fatalf("%+v", got)
	}

	// A source that cannot be read leaves the state unchecked.
	report.PackURL["frangfurd"] = h.srv.URL + "/gone/pack.toml"
	got = c.PackState(ctx, chapter, report, "")
	if !got.Installed || got.Checked {
		t.Fatalf("%+v", got)
	}

	// A URL that passes neither rule is not fetched at all.
	report.PackURL["frangfurd"] = "https://elsewhere.example/pack.toml"
	if got := c.PackState(ctx, chapter, report, ""); got.Checked {
		t.Fatalf("%+v", got)
	}
}

func TestInstalledPackHashReadsOnlyASHA256Record(t *testing.T) {
	dir := t.TempDir()
	if _, ok := installedPackHash(dir); ok {
		t.Fatal("no record")
	}
	game := filepath.Join(dir, ".minecraft")
	if err := os.MkdirAll(game, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(game, packwizStateName), []byte(packwizState("abc")), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, ok := installedPackHash(dir); !ok || got != "abc" {
		t.Fatalf("%q %v", got, ok)
	}
	if err := os.WriteFile(filepath.Join(game, packwizStateName), []byte(`{"packFileHash":{"type":"md5","value":"abc"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := installedPackHash(dir); ok {
		t.Fatal("another hash type cannot be compared")
	}
}

func TestPackStateCountsTheIndexMods(t *testing.T) {
	h, c := newFakePackHost(t)
	index := `hash-format = "sha256"

[[files]]
file = "config/a.toml"
hash = "x"

[[files]]
file = "mods/create.pw.toml"
hash = "x"
metafile = true

[[files]]
file = "mods/frangfurd_menu-0.5.0.jar"
hash = "x"

[[files]]
file = "mods/.connector/cache.json"
hash = "x"
`
	sum := sha256.Sum256([]byte(index))
	h.jarBody["/frangfurd/index.toml"] = []byte(index)
	h.packTOML = strings.Replace(frangfurdPackTOML,
		"1f4a4b7ecc45b042703aafc870ba190e0f24e471a3e6c428d13e7cae51421ee0", hex.EncodeToString(sum[:]), 1)
	chapter := frangfurdChapter(h.srv.URL + "/frangfurd/pack.toml")
	report := models.InstanceReport{Present: map[string]bool{}, PackURL: map[string]string{}}
	ctx := context.Background()

	if got := c.PackState(ctx, chapter, report, ""); got.Mods != 2 {
		t.Fatalf("%+v", got)
	}
	// The same index hash is counted once.
	if got := c.PackState(ctx, chapter, report, ""); got.Mods != 2 || h.jarGets.Load() != 1 {
		t.Fatalf("%+v, %d index reads", got, h.jarGets.Load())
	}

	// An index that does not match its hash is not counted; the version still is.
	h.jarBody["/frangfurd/index.toml"] = []byte(index + "\n")
	h.packTOML = strings.Replace(h.packTOML, `version = "1.0.0"`, `version = "1.0.1"`, 1)
	h.packTOML = strings.Replace(h.packTOML, hex.EncodeToString(sum[:]), strings.Repeat("0", 64), 1)
	if got := c.PackState(ctx, chapter, report, ""); !got.Checked || got.Mods != 0 || got.Version != "1.0.1" {
		t.Fatalf("%+v", got)
	}
}

func TestScanPackIndex(t *testing.T) {
	hash := strings.Repeat("ab", 32)
	ok := "[index]\nfile = \"index.toml\"\nhash-format = \"sha256\"\nhash = \"" + hash + "\"\n"
	if ref, err := scanPackIndex([]byte(ok)); err != nil || ref != (packIndexRef{file: "index.toml", hash: hash}) {
		t.Fatalf("%+v %v", ref, err)
	}
	for _, bad := range []string{
		strings.Replace(ok, "index.toml", "../index.toml", 1),
		strings.Replace(ok, "index.toml", "sub/index.toml", 1),
		strings.Replace(ok, "index.toml", "index.json", 1),
		strings.Replace(ok, "sha256", "md5", 1),
		strings.Replace(ok, hash, "abc", 1),
		strings.Replace(ok, "[index]", "[other]", 1),
	} {
		if ref, err := scanPackIndex([]byte(bad)); err == nil {
			t.Errorf("%q: %+v, want a refusal", bad, ref)
		}
	}
}
