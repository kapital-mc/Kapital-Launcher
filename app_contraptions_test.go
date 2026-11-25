package main

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"kapital/backend/models"
	"kapital/backend/services"
)

// excraft is the smallest file the Tool Gun's format check takes: gzipped NBT
// with its format and one sub-level.
func excraft(t *testing.T) []byte {
	t.Helper()
	raw := []byte{10, 0, 0} // a compound named ""
	str := func(s string) []byte { return append([]byte{byte(len(s) >> 8), byte(len(s))}, s...) }
	raw = append(raw, 8)
	raw = append(raw, str("format")...)
	raw = append(raw, str("enxv_aeronautics_plot_print_v8")...)
	raw = append(raw, 9)
	raw = append(raw, str("sublevels")...)
	raw = append(raw, 10, 0, 0, 0, 1, 0) // one empty compound
	raw = append(raw, 0)
	var b bytes.Buffer
	zw := gzip.NewWriter(&b)
	if _, err := zw.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// contraptionApp is an app with Frangfurd installed and one Tool Gun save.
func contraptionApp(t *testing.T) (*App, string) {
	t.Helper()
	app, cfg, _ := modsApp(t, modJade)
	game := filepath.Join(filepath.Dir(cfg), "minecraft")
	dir := filepath.Join(game, "enxv_aeronautics_structures")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Bus.excraft"), excraft(t), 0o644); err != nil {
		t.Fatal(err)
	}
	return app, game
}

func TestContraptionsListRenameDelete(t *testing.T) {
	app, _ := contraptionApp(t)
	got, err := app.ListContraptions("frangfurd")
	if err != nil || len(got) != 1 || got[0].Name != "Bus" || got[0].SubLevels != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
	got, err = app.RenameContraption("frangfurd", models.ContraptionToolgun, "Bus.excraft", "Bus Mk2")
	if err != nil || len(got) != 1 || got[0].File != "Bus Mk2.excraft" {
		t.Fatalf("rename: %+v, %v", got, err)
	}
	if _, err := app.RenameContraption("frangfurd", models.ContraptionToolgun, "Bus Mk2.excraft", "a/b"); err == nil {
		t.Error("a bad name was taken")
	}
	got, err = app.DeleteContraption("frangfurd", models.ContraptionToolgun, "Bus Mk2.excraft")
	if err != nil || len(got) != 0 {
		t.Fatalf("delete: %+v, %v", got, err)
	}
}

func TestImportAndExportContraptions(t *testing.T) {
	app, game := contraptionApp(t)
	src := t.TempDir()
	good := filepath.Join(src, "Zeppelin.excraft")
	bad := filepath.Join(src, "Bad.nbt")
	for path, data := range map[string][]byte{good: excraft(t), bad: []byte("nope")} {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	app.pickFiles = func(string) ([]string, error) { return []string{bad, good}, nil }
	n, err := app.ImportContraptions("frangfurd")
	if n != 1 || err == nil || !strings.Contains(err.Error(), "Bad.nbt") {
		t.Fatalf("imported %d, %v", n, err)
	}
	if _, err := os.Stat(filepath.Join(game, "enxv_aeronautics_structures", "Zeppelin.excraft")); err != nil {
		t.Error(err)
	}

	dest := filepath.Join(t.TempDir(), "out.excraft")
	app.pickSaveFile = func(_, name string) (string, error) {
		if name != "Bus.excraft" {
			t.Errorf("suggested %q", name)
		}
		return dest, nil
	}
	if ok, err := app.ExportContraption("frangfurd", models.ContraptionToolgun, "Bus.excraft"); !ok || err != nil {
		t.Fatalf("export: %v, %v", ok, err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Error(err)
	}
	app.pickSaveFile = func(string, string) (string, error) { return "", nil }
	if ok, err := app.ExportContraption("frangfurd", models.ContraptionToolgun, "Bus.excraft"); ok || err != nil {
		t.Errorf("a cancelled export: %v, %v", ok, err)
	}
}

func TestExportContraptionsZipAsksWhereAndCountsTheFiles(t *testing.T) {
	app, _ := contraptionApp(t)
	dest := filepath.Join(t.TempDir(), "all.zip")
	app.pickSaveFile = func(_, name string) (string, error) {
		if !strings.HasSuffix(name, "contraptions.zip") {
			t.Errorf("suggested %q", name)
		}
		return dest, nil
	}
	picks := []models.ContraptionRef{{Kind: models.ContraptionToolgun, File: "Bus.excraft"}}
	if n, err := app.ExportContraptionsZip("frangfurd", picks); n != 1 || err != nil {
		t.Fatalf("exported %d, %v", n, err)
	}
	app.pickSaveFile = func(string, string) (string, error) { return "", nil }
	if n, err := app.ExportContraptionsZip("frangfurd", picks); n != 0 || err != nil {
		t.Errorf("a cancelled export: %d, %v", n, err)
	}
}

func TestOpenContraptionFolderMakesIt(t *testing.T) {
	app, game := contraptionApp(t)
	var opened string
	app.openFolder = func(p string) error { opened = p; return nil }
	if err := app.OpenContraptionFolder("frangfurd", models.ContraptionSchematic); err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(game, "schematics"); opened != want {
		t.Errorf("opened %q, want %q", opened, want)
	}
	if err := app.OpenContraptionFolder("frangfurd", "saves"); err == nil {
		t.Error("an unknown kind opened")
	}
}

func TestContraptionWritesAreRefusedUnderAPreview(t *testing.T) {
	app, _ := contraptionApp(t)
	mustStart(t, app, "frangfurd", services.PreviewNotInstalled)
	if _, err := app.DeleteContraption("frangfurd", models.ContraptionToolgun, "Bus.excraft"); err == nil ||
		!strings.Contains(err.Error(), "preview") {
		t.Fatalf("got %v", err)
	}
	if got, err := app.ListContraptions("frangfurd"); err != nil || len(got) != 1 {
		t.Errorf("the file went: %v, %v", got, err)
	}
}

func TestContraptionsOfAChapterThatIsNotThere(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.ListContraptions("atlantis"); err == nil || !strings.Contains(err.Error(), "no chapter") {
		t.Fatalf("got %v", err)
	}
	if _, err := app.ImportContraptions("atlantis"); err == nil {
		t.Fatal("imported into no chapter")
	}
}
