package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"slices"
	"strings"

	"kapital/backend/models"
)

// The game settings a chapter's settings page can reset one at a time
// (ADR-2, twenty-fifth amendment): a fixed list of client-side files in the
// game folder, built here and never named by the page or the manifest. A reset
// deletes the item's files. The next Play's sync puts the pack's own copy back
// of every file it installed (packwiz-installer re-downloads a file its record
// lists that is missing, UpdateManager.kt:134-157, v0.5.14), and a mod writes
// its defaults for a file the pack does not ship. Key bindings are the one
// item that edits a file instead: their lines are taken out of options.txt and
// the rest of the file is kept, so the game falls back to its default keys,
// which Default Options sets from the pack's keybindings.txt.

const (
	// GameSettingKeys is the key bindings item: the key_ lines of options.txt.
	GameSettingKeys = "keys"
	// GameSettingShaders is every shader pack's settings, shaderpacks/*.txt.
	GameSettingShaders = "shaderSettings"

	optionsFile    = "options.txt"
	keyLinePrefix  = "key_"
	shaderPacksDir = "shaderpacks"
	// maxOptionsBytes bounds the options.txt the key bindings reset reads: the
	// author's Frangfurd copy is about 8 KB.
	maxOptionsBytes = 1 << 20
	optionsTemp     = "options.txt.kapital-reset"
)

// gameSetting is one item of the list: its id, the name the page shows, and
// its files relative to the game folder, slash-separated.
type gameSetting struct {
	id, name string
	files    []string
}

// gameSettings is the list, in the order the page shows it. An item whose
// files neither exist nor are in the sync's record is not shown, so each
// chapter lists only what its pack and mods use: Iris in Frangfurd and
// Lichdenstein, Oculus and Embeddium in Luxemburg.
var gameSettings = []gameSetting{
	{id: "options", name: "Game options", files: []string{optionsFile}},
	{id: GameSettingKeys, name: "Key bindings"},
	{id: "video", name: "Video settings", files: []string{
		"config/sodium-options.json",
		"config/sodium-extra-options.json",
		"config/embeddium-options.json",
		"config/embeddium++.toml",
		"config/rubidium_extras.toml",
	}},
	{id: "iris", name: "Iris", files: []string{"config/iris.properties"}},
	{id: "oculus", name: "Oculus", files: []string{"config/oculus.properties"}},
	{id: GameSettingShaders, name: "Shader settings"},
	{id: "distantHorizons", name: "Distant Horizons", files: []string{"config/DistantHorizons.toml"}},
	{id: "flywheel", name: "Flywheel", files: []string{"config/flywheel-client.toml"}},
	{id: "create", name: "Create", files: []string{"config/create-client.toml"}},
	{id: "sable", name: "Sable", files: []string{"config/sable-client.toml"}},
}

// ListGameSettings answers the items an instance's game folder has, each with
// the files it holds now. An item is listed when one of its files is there or
// the sync's record lists one, so an item just reset stays on the page, with
// no files, until the next Play puts the pack's copy back. Key bindings are
// listed with options.txt and count its key lines. An instance with no game
// folder yet has none.
func ListGameSettings(instanceDir string) ([]models.GameSettingsItem, error) {
	items := []models.GameSettingsItem{}
	game := gameFolderName(instanceDir)
	if game == "" {
		return items, nil
	}
	root, err := os.OpenRoot(instanceDir)
	if err != nil {
		return nil, err
	}
	defer root.Close() //nolint:errcheck // a read-only handle
	record := syncedFiles(root, game)
	shaders, err := shaderSettingsFiles(root, game)
	if err != nil {
		// Listed with no files; the reset says why.
		slog.Warn("list shader settings", "error", err)
	}
	for _, s := range gameSettings {
		item := models.GameSettingsItem{ID: s.id, Name: s.name, Files: []string{}}
		listed := false
		switch s.id {
		case GameSettingKeys:
			n, err := countKeyLines(root, game)
			if err != nil {
				// Counted as none; the reset says why.
				slog.Warn("count key bindings", "error", err)
			}
			item.Keys = n
			if n > 0 {
				item.Files = append(item.Files, optionsFile)
			}
			listed = n > 0 || regularFile(root, path.Join(game, optionsFile)) || record[optionsFile]
		case GameSettingShaders:
			item.Files = append(item.Files, shaders...)
			listed = len(shaders) > 0 || slices.ContainsFunc(recordNames(record), isShaderSettings)
		default:
			for _, f := range s.files {
				if regularFile(root, path.Join(game, f)) {
					item.Files = append(item.Files, f)
				}
				listed = listed || record[f]
			}
			listed = listed || len(item.Files) > 0
		}
		if listed {
			items = append(items, item)
		}
	}
	return items, nil
}

// ResetGameSetting resets one item of the list in an instance's game folder:
// it deletes the item's files, or for key bindings takes the key lines out of
// options.txt. An id not on the list is refused, and so is a file of the item
// that is a link or a folder, left as it is. A file that is not there is
// nothing to reset. The caller has made sure the game is closed.
func ResetGameSetting(instanceDir, id string) error {
	i := slices.IndexFunc(gameSettings, func(s gameSetting) bool { return s.id == id })
	if i < 0 {
		return fmt.Errorf("%q is not a game setting", id)
	}
	game := gameFolderName(instanceDir)
	if game == "" {
		return errors.New("the instance has no game folder yet")
	}
	root, err := os.OpenRoot(instanceDir)
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck // every write is checked on its own
	switch id {
	case GameSettingKeys:
		return dropKeyLines(root, game)
	case GameSettingShaders:
		files, err := shaderSettingsFiles(root, game)
		if err != nil {
			return err
		}
		return removeFiles(root, game, files)
	}
	return removeFiles(root, game, gameSettings[i].files)
}

// removeFiles deletes each file under the game folder that is there, refusing
// one that is not a regular file.
func removeFiles(root *os.Root, game string, files []string) error {
	for _, f := range files {
		name := path.Join(game, f)
		info, err := root.Lstat(name)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a file; left as it is", f)
		}
		if err := root.Remove(name); err != nil {
			return err
		}
	}
	return nil
}

// regularFile reports whether name is a regular file through root.
func regularFile(root *os.Root, name string) bool {
	info, err := root.Lstat(name)
	return err == nil && info.Mode().IsRegular()
}

// isShaderSettings reports whether a path relative to the game folder is a
// shader pack's settings: a .txt directly in shaderpacks.
func isShaderSettings(rel string) bool {
	dir, name := path.Split(rel)
	return dir == shaderPacksDir+"/" && strings.HasSuffix(name, ".txt") && len(name) > len(".txt")
}

// shaderSettingsFiles lists the shader packs' settings files on disk, sorted:
// regular files named *.txt directly in shaderpacks. Iris and Oculus write one
// beside each pack as <pack>.txt.
func shaderSettingsFiles(root *os.Root, game string) ([]string, error) {
	dir, err := root.Open(path.Join(game, shaderPacksDir))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer dir.Close() //nolint:errcheck // a read-only handle
	entries, err := dir.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		rel := shaderPacksDir + "/" + e.Name()
		if e.Type().IsRegular() && isShaderSettings(rel) {
			files = append(files, rel)
		}
	}
	slices.Sort(files)
	return files, nil
}

// syncedFiles is the set of files packwiz-installer's record in the game folder
// says it installed, by their path in the pack. A record that is missing, too
// large or unreadable lists none.
func syncedFiles(root *os.Root, game string) map[string]bool {
	files := map[string]bool{}
	name := path.Join(game, packwizStateName)
	info, err := root.Lstat(name)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxPackwizState {
		return files
	}
	raw, err := root.ReadFile(name)
	if err != nil {
		return files
	}
	var rec struct {
		CachedFiles map[string]json.RawMessage `json:"cachedFiles"`
	}
	if json.Unmarshal(raw, &rec) != nil {
		return files
	}
	for f := range rec.CachedFiles {
		files[f] = true
	}
	return files
}

func recordNames(record map[string]bool) []string {
	names := make([]string, 0, len(record))
	for f := range record {
		names = append(names, f)
	}
	return names
}

// readOptions reads options.txt from the game folder, nil when it is not
// there. Anything but a regular file of at most maxOptionsBytes is refused.
func readOptions(root *os.Root, game string) ([]byte, error) {
	name := path.Join(game, optionsFile)
	info, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxOptionsBytes {
		return nil, errors.New("options.txt is not a file the game wrote; left as it is")
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck // a read-only handle
	return io.ReadAll(io.LimitReader(f, maxOptionsBytes+1))
}

// countKeyLines counts options.txt's key binding lines, 0 when it is not there.
func countKeyLines(root *os.Root, game string) (int, error) {
	raw, err := readOptions(root, game)
	if err != nil {
		return 0, err
	}
	_, n := withoutKeyLines(raw)
	return n, nil
}

// withoutKeyLines is options.txt with every line that starts with key_ taken
// out, and how many were. Every other line is kept byte for byte, its line end
// included.
func withoutKeyLines(raw []byte) ([]byte, int) {
	var out bytes.Buffer
	n := 0
	for line := range bytes.Lines(raw) {
		if bytes.HasPrefix(line, []byte(keyLinePrefix)) {
			n++
			continue
		}
		out.Write(line)
	}
	return out.Bytes(), n
}

// dropKeyLines writes options.txt again without its key lines: to a new file
// beside it, then renamed over it, so a failure leaves the old one whole.
func dropKeyLines(root *os.Root, game string) error {
	raw, err := readOptions(root, game)
	if err != nil || raw == nil {
		return err
	}
	kept, n := withoutKeyLines(raw)
	if n == 0 {
		return nil
	}
	temp := path.Join(game, optionsTemp)
	if err := root.Remove(temp); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	f, err := root.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(kept); err != nil {
		f.Close()         //nolint:errcheck // the write already failed
		root.Remove(temp) //nolint:errcheck // best effort; the original is untouched
		return err
	}
	if err := f.Close(); err != nil {
		root.Remove(temp) //nolint:errcheck // best effort; the original is untouched
		return err
	}
	if err := root.Rename(temp, path.Join(game, optionsFile)); err != nil {
		root.Remove(temp) //nolint:errcheck // best effort; the original is untouched
		return err
	}
	return nil
}
