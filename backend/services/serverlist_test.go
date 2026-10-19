package services

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// The Frangfurd pack's own servers.dat, as Default Options copies it in: two
// entries with icons, written by the game.
func packServersDat(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "serverlist", "frangfurd-servers.dat"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestNBTRoundTripsTheGamesOwnFile(t *testing.T) {
	data := packServersDat(t)
	name, root, err := readNBT(data)
	if err != nil {
		t.Fatal(err)
	}
	out, err := writeNBT(name, root)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out, data) {
		t.Fatalf("round trip changed the file: %d bytes in, %d out", len(data), len(out))
	}
}

func TestNBTRoundTripsEveryTagType(t *testing.T) {
	root := nbtTag{Type: nbtCompound, Fields: []nbtField{
		{"b", nbtTag{Type: nbtByte, Value: int8(-1)}},
		{"s", nbtTag{Type: nbtShort, Value: int16(300)}},
		{"i", nbtTag{Type: nbtInt, Value: int32(-70000)}},
		{"l", nbtTag{Type: nbtLong, Value: int64(1) << 40}},
		{"f", nbtTag{Type: nbtFloat, Value: float32(1.5)}},
		{"d", nbtTag{Type: nbtDouble, Value: 2.25}},
		{"ba", nbtTag{Type: nbtByteArray, Value: []int8{1, 2}}},
		{"ia", nbtTag{Type: nbtIntArray, Value: []int32{3}}},
		{"la", nbtTag{Type: nbtLongArray, Value: []int64{4, 5}}},
		{"str", nbtText("Grüße")},
		{"empty", nbtTag{Type: nbtList, ListType: nbtEnd}},
		{"nested", nbtTag{Type: nbtList, ListType: nbtCompound, List: []nbtTag{{Type: nbtCompound}}}},
	}}
	first, err := writeNBT("root", root)
	if err != nil {
		t.Fatal(err)
	}
	name, back, err := readNBT(first)
	if err != nil || name != "root" {
		t.Fatalf("read back: %q %v", name, err)
	}
	second, err := writeNBT(name, back)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("the second write differs from the first")
	}
}

func TestNBTRefusesBrokenFiles(t *testing.T) {
	data := packServersDat(t)
	for name, bad := range map[string][]byte{
		"empty":            {},
		"not a compound":   {nbtList, 0, 0},
		"cut short":        data[:len(data)/2],
		"unknown type":     {nbtCompound, 0, 0, 99, 0, 1, 'x'},
		"negative length":  {nbtCompound, 0, 0, nbtIntArray, 0, 1, 'x', 0xff, 0xff, 0xff, 0xff},
		"oversized length": {nbtCompound, 0, 0, nbtIntArray, 0, 1, 'x', 0x7f, 0xff, 0xff, 0xff},
	} {
		if _, _, err := readNBT(bad); err == nil {
			t.Errorf("%s: read without an error", name)
		}
	}
}

func serverIPs(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, root, err := readNBT(data)
	if err != nil {
		t.Fatal(err)
	}
	servers, ok := root.field("servers")
	if !ok {
		return nil
	}
	var ips []string
	for _, s := range servers.List {
		ips = append(ips, s.stringField("name")+"="+s.stringField("ip"))
	}
	return ips
}

func gameDir(t *testing.T) (instance, game string) {
	t.Helper()
	instance = t.TempDir()
	game = filepath.Join(instance, "minecraft")
	if err := os.MkdirAll(game, 0o755); err != nil {
		t.Fatal(err)
	}
	return instance, game
}

var tailscaleEntry = ServerListEntry{Name: "Frangfurd (Tailscale)", Address: "ts.frangfurd.click"}

func TestSyncServerListAddsAndRemovesOnlyItsEntry(t *testing.T) {
	instance, game := gameDir(t)
	path := filepath.Join(game, "servers.dat")
	if err := os.WriteFile(path, packServersDat(t), 0o644); err != nil {
		t.Fatal(err)
	}
	wrote, err := SyncServerList(instance, tailscaleEntry, true)
	if err != nil || !wrote {
		t.Fatalf("add: %v %v", wrote, err)
	}
	got := serverIPs(t, path)
	want := []string{
		"Frangfurd (Global)=female-specified.gl.joinmc.link",
		"Frangfurd (Germany)=rails-enjoyed.tun.ply.gg",
		"Frangfurd (Tailscale)=ts.frangfurd.click",
	}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("after add: %v", got)
	}
	if wrote, err := SyncServerList(instance, tailscaleEntry, true); err != nil || wrote {
		t.Fatalf("a second add wrote: %v %v", wrote, err)
	}
	if wrote, err := SyncServerList(instance, tailscaleEntry, false); err != nil || !wrote {
		t.Fatalf("remove: %v %v", wrote, err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, packServersDat(t)) {
		t.Fatal("removing the entry did not give back the pack's file byte for byte")
	}
}

func TestSyncServerListRemovesAnEntryThePlayerRenamed(t *testing.T) {
	instance, game := gameDir(t)
	if _, err := SyncServerList(instance, ServerListEntry{Name: "Mine", Address: "TS.frangfurd.click "}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncServerList(instance, tailscaleEntry, false); err != nil {
		t.Fatal(err)
	}
	if got := serverIPs(t, filepath.Join(game, "servers.dat")); len(got) != 0 {
		t.Fatalf("left: %v", got)
	}
}

func TestSyncServerListSeedsFromThePacksDefaults(t *testing.T) {
	instance, game := gameDir(t)
	defaults := filepath.Join(game, "config", "defaultoptions")
	if err := os.MkdirAll(defaults, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(defaults, "servers.dat"), packServersDat(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncServerList(instance, tailscaleEntry, true); err != nil {
		t.Fatal(err)
	}
	if got := serverIPs(t, filepath.Join(game, "servers.dat")); len(got) != 3 {
		t.Fatalf("seeded list: %v", got)
	}
}

func TestSyncServerListWithNothingToDo(t *testing.T) {
	// No game folder: the first start writes the list.
	if wrote, err := SyncServerList(t.TempDir(), tailscaleEntry, true); err != nil || wrote {
		t.Fatalf("no game folder: %v %v", wrote, err)
	}
	// No list and nothing to add.
	instance, game := gameDir(t)
	if wrote, err := SyncServerList(instance, tailscaleEntry, false); err != nil || wrote {
		t.Fatalf("no list: %v %v", wrote, err)
	}
	// No list and no defaults: a list of one.
	if _, err := SyncServerList(instance, tailscaleEntry, true); err != nil {
		t.Fatal(err)
	}
	if got := serverIPs(t, filepath.Join(game, "servers.dat")); len(got) != 1 {
		t.Fatalf("new list: %v", got)
	}
}

func TestSyncServerListRefusesABrokenList(t *testing.T) {
	instance, game := gameDir(t)
	path := filepath.Join(game, "servers.dat")
	if err := os.WriteFile(path, []byte("not nbt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := SyncServerList(instance, tailscaleEntry, true); err == nil {
		t.Fatal("a broken list was overwritten")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "not nbt" {
		t.Fatalf("the broken list changed: %q %v", data, err)
	}
}
