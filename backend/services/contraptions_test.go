package services

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"kapital/backend/models"
)

func nbtCompoundOf(fields ...nbtField) nbtTag { return nbtTag{Type: nbtCompound, Fields: fields} }
func nbtListOf(elem byte, items ...nbtTag) nbtTag {
	return nbtTag{Type: nbtList, ListType: elem, List: items}
}
func nbtI(v int32) nbtTag { return nbtTag{Type: nbtInt, Value: v} }
func nbtL(v int64) nbtTag { return nbtTag{Type: nbtLong, Value: v} }

func gzipNBT(t *testing.T, root nbtTag) []byte {
	t.Helper()
	raw, err := writeNBT("", root)
	if err != nil {
		t.Fatal(err)
	}
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

func count(id string, n int64) nbtTag {
	return nbtCompoundOf(nbtField{"id", nbtText(id)}, nbtField{"count", nbtL(n)})
}

// toolgunFixture is the shape of a real Tool Gun 0.3.6 save, cut down.
func toolgunFixture(t *testing.T) []byte {
	return gzipNBT(t, nbtCompoundOf(
		nbtField{"format", nbtText("enxv_aeronautics_plot_print_v8")},
		nbtField{"name", nbtText("Bus")},
		nbtField{"sublevels", nbtListOf(nbtCompound, nbtCompoundOf(), nbtCompoundOf())},
		nbtField{"AST_MaterialSummary", nbtCompoundOf(
			nbtField{"total_blocks", nbtL(609)},
			nbtField{"unique_block_types", nbtI(2)},
			nbtField{"blocks", nbtListOf(nbtCompound, count("create:shaft", 9), count("minecraft:oak_planks", 600))},
			nbtField{"items", nbtListOf(nbtCompound, count("minecraft:coal", 64))},
		)},
	))
}

func structureOf(names []string, states ...int32) []nbtField {
	palette := nbtListOf(nbtCompound)
	for _, n := range names {
		palette.List = append(palette.List, nbtCompoundOf(nbtField{"Name", nbtText(n)}))
	}
	blocks := nbtListOf(nbtCompound)
	for _, s := range states {
		blocks.List = append(blocks.List, nbtCompoundOf(nbtField{"state", nbtI(s)}))
	}
	return []nbtField{{"palette", palette}, {"blocks", blocks}}
}

// schematicFixture is Create's shape: a structure with a sub-level of its own.
func schematicFixture(t *testing.T) []byte {
	top := structureOf([]string{"minecraft:air", "create:shaft"}, 0, 1, 1)
	sub := nbtCompoundOf(structureOf([]string{"minecraft:stone"}, 0)...)
	fields := append([]nbtField{{"size", nbtListOf(nbtInt, nbtI(3), nbtI(2), nbtI(1))}}, top...)
	fields = append(fields, nbtField{"sub_levels", nbtListOf(nbtCompound, sub)})
	return gzipNBT(t, nbtCompoundOf(fields...))
}

func writeContraption(t *testing.T, game, rel string, data []byte, at time.Time) {
	t.Helper()
	path := filepath.Join(game, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
}

func TestListContraptionsReadsBothKindsNewestFirst(t *testing.T) {
	game := t.TempDir()
	day := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	writeContraption(t, game, "enxv_aeronautics_structures/Bus.excraft", toolgunFixture(t), day)
	writeContraption(t, game, "schematics/bridge.nbt", schematicFixture(t), day.Add(time.Hour))
	writeContraption(t, game, "schematics/broken.nbt", []byte("not gzip"), day.Add(-time.Hour))
	writeContraption(t, game, "schematics/notes.txt", []byte("x"), day)
	if err := os.Mkdir(filepath.Join(game, "schematics", "uploaded.nbt"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := ListContraptions(game)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].File != "bridge.nbt" || got[1].File != "Bus.excraft" || got[2].File != "broken.nbt" {
		t.Fatalf("listed %+v", got)
	}
	bridge, bus, broken := got[0], got[1], got[2]
	if bridge.Kind != models.ContraptionSchematic || bridge.Blocks != 3 || bridge.BlockTypes != 2 ||
		bridge.SubLevels != 1 || len(bridge.Size) != 3 || bridge.Materials[0] != (models.MaterialCount{ID: "create:shaft", Count: 2}) {
		t.Errorf("schematic facts %+v", bridge)
	}
	if bus.Kind != models.ContraptionToolgun || bus.Name != "Bus" || bus.Blocks != 609 || bus.BlockTypes != 2 ||
		bus.SubLevels != 2 || bus.Materials[0].ID != "minecraft:oak_planks" || len(bus.Items) != 1 || bus.Unread {
		t.Errorf("tool gun facts %+v", bus)
	}
	if !broken.Unread || broken.Blocks != -1 {
		t.Errorf("a broken file is listed unread: %+v", broken)
	}
}

func TestListContraptionsWithNoFoldersIsEmpty(t *testing.T) {
	got, err := ListContraptions(t.TempDir())
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestContraptionNameFollowsTheToolGunsRule(t *testing.T) {
	for in, want := range map[string]string{"  Bus de Battalion ": "Bus de Battalion", "Zeppelin_2-b": "Zeppelin_2-b", "Ölfass": "Ölfass"} {
		if got, err := ContraptionName(in); err != nil || got != want {
			t.Errorf("%q: %q, %v", in, got, err)
		}
	}
	for _, bad := range []string{"", "   ", "../up", "a.b", "a/b", strings.Repeat("x", 65)} {
		if _, err := ContraptionName(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestRenameContraption(t *testing.T) {
	game := t.TempDir()
	now := time.Now()
	writeContraption(t, game, "enxv_aeronautics_structures/Bus.excraft", toolgunFixture(t), now)
	writeContraption(t, game, "enxv_aeronautics_structures/Taken.excraft", toolgunFixture(t), now)

	if _, err := RenameContraption(game, models.ContraptionToolgun, "Bus.excraft", "Taken"); err == nil {
		t.Error("renamed over another file")
	}
	for _, file := range []string{"../Bus.excraft", "Bus.nbt", "Missing.excraft", ".excraft"} {
		if _, err := RenameContraption(game, models.ContraptionToolgun, file, "New"); err == nil {
			t.Errorf("%q renamed", file)
		}
	}
	if _, err := RenameContraption(game, "world", "Bus.excraft", "New"); err == nil {
		t.Error("an unknown kind renamed")
	}
	got, err := RenameContraption(game, models.ContraptionToolgun, "Bus.excraft", " Bus Mk2 ")
	if err != nil || got != "Bus Mk2.excraft" {
		t.Fatalf("got %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(game, toolgunDir, "Bus Mk2.excraft")); err != nil {
		t.Error(err)
	}
	if got, err := RenameContraption(game, models.ContraptionToolgun, "Bus Mk2.excraft", "bus mk2"); err != nil || got != "bus mk2.excraft" {
		t.Errorf("a change of case: %q, %v", got, err)
	}
}

func TestDeleteContraptionRemovesOnlyARegularFile(t *testing.T) {
	game := t.TempDir()
	writeContraption(t, game, "schematics/bridge.nbt", schematicFixture(t), time.Now())
	outside := filepath.Join(t.TempDir(), "keep.nbt")
	if err := os.WriteFile(outside, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Symlink(outside, filepath.Join(game, schematicsDir, "link.nbt")); err != nil {
			t.Fatal(err)
		}
		if err := DeleteContraption(game, models.ContraptionSchematic, "link.nbt"); err == nil {
			t.Error("deleted through a link")
		}
	}
	if err := DeleteContraption(game, models.ContraptionSchematic, "../../keep.nbt"); err == nil {
		t.Error("deleted outside the folder")
	}
	if err := DeleteContraption(game, models.ContraptionSchematic, "bridge.nbt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(game, schematicsDir, "bridge.nbt")); !os.IsNotExist(err) {
		t.Error("still there")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Error("the file outside went", err)
	}
}

func TestImportContraptionChecksAndNumbers(t *testing.T) {
	game, src := t.TempDir(), t.TempDir()
	pick := func(name string, data []byte) string {
		path := filepath.Join(src, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	first, err := ImportContraption(game, pick("Air ship!.excraft", toolgunFixture(t)))
	if err != nil || first.File != "Air ship.excraft" || first.Kind != models.ContraptionToolgun || first.Blocks != 609 {
		t.Fatalf("got %+v, %v", first, err)
	}
	second, err := ImportContraption(game, pick("Air ship.EXCRAFT", toolgunFixture(t)))
	if err != nil || second.File != "Air ship 2.excraft" {
		t.Fatalf("a second import: %+v, %v", second, err)
	}
	if got, err := ImportContraption(game, pick("bridge.nbt", schematicFixture(t))); err != nil || got.Kind != models.ContraptionSchematic {
		t.Fatalf("schematic: %+v, %v", got, err)
	}
	for name, data := range map[string][]byte{
		"plain.nbt":     []byte("not gzip"),
		"wrong.excraft": schematicFixture(t),
		"wrong.nbt":     toolgunFixture(t),
		"world.zip":     toolgunFixture(t),
	} {
		if _, err := ImportContraption(game, pick(name, data)); err == nil {
			t.Errorf("%s imported", name)
		}
	}
	if got, _ := ListContraptions(game); len(got) != 3 {
		t.Errorf("%d files after the refusals", len(got))
	}
}

func TestExportContraptionCopiesTheFile(t *testing.T) {
	game := t.TempDir()
	data := toolgunFixture(t)
	writeContraption(t, game, "enxv_aeronautics_structures/Bus.excraft", data, time.Now())
	dest := filepath.Join(t.TempDir(), "Bus.excraft")
	if err := ExportContraption(game, models.ContraptionToolgun, "Bus.excraft", dest); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(got, data) {
		t.Errorf("copy differs: %v", err)
	}
}

func TestCleanContraptionName(t *testing.T) {
	for in, want := range map[string]string{"Air ship!.excraft": "Air ship", "...nbt": "Imported", "Ölfass (1).nbt": "Ölfass 1"} {
		if got := cleanContraptionName(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}
