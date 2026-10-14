package services

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"kapital/backend/models"
)

// readZip returns a zip's entries by name and their names in stored order.
func readZip(t *testing.T, data []byte) (map[string][]byte, []string) {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	var order []string
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(rc)
		if cerr := rc.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			t.Fatal(err)
		}
		files[f.Name] = body
		order = append(order, f.Name)
	}
	return files, order
}

func TestPrismZipHoldsWhatInstallWrites(t *testing.T) {
	h, c := newFakePackHost(t)
	packURL := h.srv.URL + "/frangfurd/pack.toml"
	chapter := frangfurdChapter(packURL)
	data, err := c.PrismZip(context.Background(), chapter)
	if err != nil {
		t.Fatal(err)
	}
	files, order := readZip(t, data)
	want := []string{
		"instance.cfg",
		"minecraft/packwiz-installer-bootstrap.jar",
		"minecraft/packwiz-installer.jar",
		"mmc-pack.json",
		"packwiz-LICENSE.txt",
	}
	if !slices.Equal(order, want) {
		t.Fatalf("entries %v, want %v", order, want)
	}

	// Every file Install writes is in the zip with the same bytes, and only the
	// pre-launch command of instance.cfg differs.
	instances := t.TempDir()
	if err := c.Create(context.Background(), chapter, instances); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(instances, chapter.Instance.ID)
	for _, name := range want[1:4] {
		installed, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil || !bytes.Equal(installed, files[name]) {
			t.Errorf("%s differs from the installed instance's (%v)", name, err)
		}
	}
	installedCfg, err := os.ReadFile(filepath.Join(dir, instanceConfig))
	if err != nil {
		t.Fatal(err)
	}
	keep := func(cfg []byte) string {
		var out []string
		for _, line := range strings.Split(string(cfg), "\n") {
			if !strings.HasPrefix(line, "PreLaunchCommand=") {
				out = append(out, line)
			}
		}
		return strings.Join(out, "\n")
	}
	if keep(files["instance.cfg"]) != keep(installedCfg) {
		t.Errorf("instance.cfg differs beyond the command:\n%s\nagainst\n%s", files["instance.cfg"], installedCfg)
	}

	keys, err := scanINIKeys(bytes.NewReader(files["instance.cfg"]), "PreLaunchCommand", "OverrideCommands")
	if err != nil {
		t.Fatal(err)
	}
	if keys["OverrideCommands"] != "true" {
		t.Errorf("OverrideCommands = %q", keys["OverrideCommands"])
	}
	if got := classifyPreLaunch(keys["PreLaunchCommand"]); got.kind != kindLegacy || got.url != packURL {
		t.Errorf("the command %q is not the windowed packwiz template for the pack", keys["PreLaunchCommand"])
	}
	if !bytes.Equal(files["packwiz-LICENSE.txt"], packwizLicense) || !bytes.Contains(packwizLicense, []byte("MIT License")) {
		t.Error("the packwiz licence is not in the zip as embedded")
	}
}

func TestPrismZipIsTheSameBytesEachTime(t *testing.T) {
	h, c := newFakePackHost(t)
	chapter := frangfurdChapter(h.srv.URL + "/frangfurd/pack.toml")
	first, err := c.PrismZip(context.Background(), chapter)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.PrismZip(context.Background(), chapter)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Error("two builds from the same inputs differ")
	}
}

func TestPrismZipRefuses(t *testing.T) {
	h, c := newFakePackHost(t)
	noPack := frangfurdChapter("")
	noPack.Pack.Packwiz = nil
	disagrees := frangfurdChapter(h.srv.URL + "/frangfurd/pack.toml")
	disagrees.Pack.Minecraft = "1.20.1"
	offHost := frangfurdChapter("https://elsewhere.example/frangfurd/pack.toml")
	for name, chapter := range map[string]models.Chapter{
		"no hosted pack":             noPack,
		"a pack the manifest denies": disagrees,
		"a host the check refuses":   offHost,
	} {
		if _, err := c.PrismZip(context.Background(), chapter); err == nil {
			t.Errorf("%s: built a zip", name)
		}
	}
}

func TestPrismZipName(t *testing.T) {
	got, err := PrismZipName(models.Chapter{ID: "frangfurd", Name: "Frangfurd"})
	if err != nil || got != "Kapitel-Kapital-Frangfurd.zip" {
		t.Errorf("got %q, %v", got, err)
	}
	for _, name := range []string{"", "Frang furd", "../Frangfurd", "Frangfürd", strings.Repeat("a", 33)} {
		if got, err := PrismZipName(models.Chapter{ID: "x", Name: name}); err == nil {
			t.Errorf("%q made a file name, %q", name, got)
		}
	}
}
