package services

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Reinstall of the launcher's own instance (ADR-2's twenty-third amendment):
// the old folder is stashed, a fresh instance is installed in its place, and
// what the player made or would wait long to fetch again is moved across from
// the stash. The rest of the stash (mods, configs, options.txt, shader and
// resource packs, caches, logs) is then removed, and the next Play's sync
// fetches the pack whole. Uninstall removes the whole folder. Both work inside
// the instances folder through an os.Root, so no path leaves it.

// keptEntry is a path Reinstall moves from the old instance into the new one.
type keptEntry struct {
	// path is relative to the game folder, or to the instance folder when
	// instanceLevel is set.
	path          string
	instanceLevel bool
}

// keptEntries is what Reinstall keeps. Everything here is the player's own or
// costly to fetch again; nothing here is written by the pack, so a broken
// setting cannot survive in it.
var keptEntries = []keptEntry{
	{path: savesDir},
	{path: screenshotsDir},
	{path: schematicsDir},
	// The Tool Gun's saved contraptions (create_aeronautics_toolgun).
	{path: toolgunDir},
	{path: dhServerData},
	{path: xaeroFolder},
	{path: "flashback"},
	// The server list: the launcher's own entry is written again on Play.
	{path: "servers.dat"},
	// Essential keeps its own account data here; the launcher never touches it.
	{path: "essential"},
	{path: dhStaging, instanceLevel: true},
}

// removingName is the name an instance folder gets just before it is
// removed, so a removal that stops half way leaves no folder under the
// chapter's id.
func removingName(id string, now time.Time) string {
	return fmt.Sprintf("%s-removing-%d", id, now.UnixNano())
}

// RemoveInstance removes <instancesDir>/<name> and everything in it. The
// folder is renamed aside first and its instance.cfg removed next, so Prism
// stops listing it before the long part. A folder that is not there is
// nothing to remove. It refuses a name that is not a plain folder name and a
// link or file in the folder's place. The error of a removal that stopped half
// way names the folder left behind.
func RemoveInstance(instancesDir, name string, now time.Time) error {
	if !prismInstanceID.MatchString(name) {
		return fmt.Errorf("instance %q is not a plain folder name", name)
	}
	root, err := os.OpenRoot(instancesDir)
	if err != nil {
		return fmt.Errorf("remove instance %s: %w", name, err)
	}
	defer root.Close() //nolint:errcheck // nothing was written through the handle itself
	info, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove instance %s: %w", name, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("remove instance %s: not a folder", name)
	}
	aside := removingName(name, now)
	if !prismInstanceID.MatchString(aside) {
		aside = name
	} else if err := root.Rename(name, aside); err != nil {
		return fmt.Errorf("remove instance %s: %w", name, err)
	}
	if err := root.Remove(filepath.Join(aside, "instance.cfg")); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove instance %s: %w; left as %s", name, err, aside)
	}
	if err := root.RemoveAll(aside); err != nil {
		return fmt.Errorf("remove instance %s: %w; left as %s", name, err, aside)
	}
	return nil
}

// CarryOver moves the kept entries from the stashed instance into the new one
// with the same id, and the play time in instance.cfg with them. An entry the
// stash lacks is skipped; one the new instance already has stops the carry,
// since nothing is put over a file. It returns how many entries it moved. On
// an error the entries moved so far stay in the new instance and the rest in
// the stash.
func CarryOver(instancesDir, stashed, id string) (int, error) {
	if !prismInstanceID.MatchString(id) || !prismInstanceID.MatchString(stashed) {
		return 0, fmt.Errorf("carry over %s: not a plain folder name", id)
	}
	oldGame := gameFolderName(filepath.Join(instancesDir, stashed))
	newGame := gameFolderName(filepath.Join(instancesDir, id))
	if newGame == "" {
		newGame = "minecraft"
	}
	root, err := os.OpenRoot(instancesDir)
	if err != nil {
		return 0, fmt.Errorf("carry over %s: %w", id, err)
	}
	defer root.Close() //nolint:errcheck // renames only, nothing written through the handle
	moved := 0
	for _, e := range keptEntries {
		from, to := filepath.Join(stashed, e.path), filepath.Join(id, e.path)
		if !e.instanceLevel {
			if oldGame == "" {
				continue
			}
			from, to = filepath.Join(stashed, oldGame, e.path), filepath.Join(id, newGame, e.path)
		}
		if _, err := root.Lstat(from); errors.Is(err, fs.ErrNotExist) {
			continue
		} else if err != nil {
			return moved, fmt.Errorf("carry over %s: %w", filepath.ToSlash(e.path), err)
		}
		if _, err := root.Lstat(to); err == nil {
			return moved, fmt.Errorf("carry over %s: already in the new instance", filepath.ToSlash(e.path))
		}
		if err := root.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return moved, fmt.Errorf("carry over %s: %w", filepath.ToSlash(e.path), err)
		}
		if err := root.Rename(from, to); err != nil {
			return moved, fmt.Errorf("carry over %s: %w", filepath.ToSlash(e.path), err)
		}
		moved++
	}
	if err := carryPlayTime(root, filepath.Join(stashed, "instance.cfg"), filepath.Join(id, "instance.cfg")); err != nil {
		return moved, err
	}
	return moved, nil
}

// carryPlayTime writes the old instance.cfg's play time keys into the new one.
// An old file with neither leaves the new one as Install wrote it.
func carryPlayTime(root *os.Root, fromCfg, toCfg string) error {
	f, err := root.Open(fromCfg)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("carry play time: %w", err)
	}
	keys, err := scanINIKeys(io.LimitReader(f, maxPrismConfigLen), playTimeTotalKey, playTimeLastKey)
	f.Close() //nolint:errcheck // read-only file, nothing to flush
	if err != nil {
		return fmt.Errorf("carry play time: %w", err)
	}
	set := map[string]string{}
	for _, key := range []string{playTimeTotalKey, playTimeLastKey} {
		if n := nonNegativeInt(keys[key]); n > 0 {
			set[key] = strconv.FormatInt(n, 10)
		}
	}
	if len(set) == 0 {
		return nil
	}
	raw, err := root.ReadFile(toCfg)
	if err != nil {
		return fmt.Errorf("carry play time: %w", err)
	}
	out, err := rewriteINIKeys(raw, set)
	if err != nil {
		return fmt.Errorf("carry play time: %w", err)
	}
	info, err := root.Stat(toCfg)
	if err != nil {
		return fmt.Errorf("carry play time: %w", err)
	}
	return writeFileAtomic(filepath.Join(root.Name(), toCfg), out, info.Mode().Perm())
}
