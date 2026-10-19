package services

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// The in-game server list is the game folder's servers.dat. The packs ship
// one in config/defaultoptions/, which the Default Options mod copies in on
// the first start only, so a player's list is theirs from then on: the
// launcher adds or removes the one entry a chapter's Tailscale switch is
// about, matched by its address, and leaves every other entry as it was.

const maxServersDat = 4 << 20

// ServerListEntry is the entry a Tailscale switch puts in the list.
type ServerListEntry struct {
	Name    string
	Address string
}

// SyncServerList makes the instance's server list hold the entry, or not,
// as want says. It reports whether it wrote the file. An instance with no
// game folder yet is left alone: its first start writes the list.
func SyncServerList(instanceDir string, entry ServerListEntry, want bool) (bool, error) {
	game := gameFolder(instanceDir)
	if game == "" {
		return false, nil
	}
	path := filepath.Join(game, "servers.dat")
	data, err := readCapped(path, maxServersDat)
	if errors.Is(err, os.ErrNotExist) {
		if !want {
			return false, nil
		}
		// The defaults first, as Default Options would copy them, so this
		// entry is added to the pack's list rather than standing in for it.
		data, err = readCapped(filepath.Join(game, "config", "defaultoptions", "servers.dat"), maxServersDat)
		if errors.Is(err, os.ErrNotExist) {
			data, err = writeNBT("", nbtTag{Type: nbtCompound})
		}
	}
	if err != nil {
		return false, fmt.Errorf("server list: %w", err)
	}
	name, root, err := readNBT(data)
	if err != nil {
		return false, fmt.Errorf("server list: %w", err)
	}
	if !editServerList(&root, entry, want) {
		return false, nil
	}
	out, err := writeNBT(name, root)
	if err != nil {
		return false, fmt.Errorf("server list: %w", err)
	}
	if err := writeFileAtomic(path, out, 0o644); err != nil {
		return false, fmt.Errorf("server list: %w", err)
	}
	return true, nil
}

// editServerList adds the entry at the end of the root's "servers" list, or
// removes every entry with its address, and reports whether anything changed.
func editServerList(root *nbtTag, entry ServerListEntry, want bool) bool {
	servers, ok := root.field("servers")
	if !ok || servers.Type != nbtList {
		if !want {
			return false
		}
		root.Fields = append(root.Fields, nbtField{Name: "servers", Tag: nbtTag{Type: nbtList, ListType: nbtCompound}})
		servers = &root.Fields[len(root.Fields)-1].Tag
	}
	if len(servers.List) == 0 {
		// An empty list may be written with element type End.
		servers.ListType = nbtCompound
	}
	if servers.ListType != nbtCompound {
		return false
	}
	kept := servers.List[:0:0]
	for _, s := range servers.List {
		if !sameServerAddress(s.stringField("ip"), entry.Address) {
			kept = append(kept, s)
		}
	}
	found := len(kept) != len(servers.List)
	if want == found {
		return false
	}
	if want {
		kept = append(kept, nbtTag{Type: nbtCompound, Fields: []nbtField{
			{Name: "name", Tag: nbtText(entry.Name)},
			{Name: "ip", Tag: nbtText(entry.Address)},
			{Name: "hidden", Tag: nbtTag{Type: nbtByte, Value: int8(0)}},
		}})
	}
	servers.List = kept
	return true
}

func sameServerAddress(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// readCapped reads a regular file of at most limit bytes.
func readCapped(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, fmt.Errorf("%s is not a file of at most %d bytes", filepath.Base(path), limit)
	}
	return io.ReadAll(io.LimitReader(f, limit))
}
