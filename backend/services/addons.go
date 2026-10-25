package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"kapital/backend/models"
)

// A chapter's add-ons (ADR-2, twenty-second amendment) are packwiz packs of their
// own, synced into the chapter's instance only for a player who turned one on:
// Camera Tools is Flashback and what it needs, 220 MB that nobody else should
// download. The pre-launch sync runs the same pinned packwiz-installer a second
// time for each one that is on, with `--meta-file` naming the add-on's own
// record, so the pack's record and the add-on's never see each other's files:
// v0.5.14 deletes only what its own record lists and the pack has since dropped
// (UpdateManager.kt:250-261), and leaves every other file alone. An add-on that
// is off is never downloaded; one turned off later keeps its jars, put away as
// ".jar.disabled" the way a disabled mod is, so turning it on again downloads
// nothing that did not change.

const (
	// maxAddons is how many add-ons a chapter may name.
	maxAddons = 4
	// addonRecordPrefix starts the file name of an add-on's record in the game
	// folder: "kapital-addon-camera.json".
	addonRecordPrefix = "kapital-addon-"
	// maxAddonRecordBytes bounds the record read back: a few entries per mod.
	maxAddonRecordBytes = 1 << 20
	maxAddonDescription = 160
)

// addonIDShape is what an add-on id may be: it goes into a file name.
var addonIDShape = regexp.MustCompile(`^[a-z][a-z0-9-]{1,23}$`)

// AddonID reports whether id is the shape of an add-on id.
func AddonID(id string) bool { return addonIDShape.MatchString(id) }

// addonRecordName is the add-on's packwiz-installer record, relative to the
// game folder, which is also the installer's pack folder.
func addonRecordName(id string) string { return addonRecordPrefix + id + ".json" }

// validateAddons holds a chapter's add-ons to what a manifest may say about one:
// an id of the file-name shape, a short plain name, a line of description, and a
// pack URL under the pack URL's own rules (allowlisted https, safe for a command
// line). Ids, names and URLs are unique, no add-on is the chapter's own pack, and
// a chapter with add-ons is a packwiz chapter: they are synced beside it.
func validateAddons(where string, pack models.Pack) error {
	if len(pack.Addons) == 0 {
		return nil
	}
	if len(pack.Addons) > maxAddons {
		return fmt.Errorf("%s: pack.addons has %d entries, the limit is %d", where, len(pack.Addons), maxAddons)
	}
	if pack.Packwiz == nil {
		return fmt.Errorf("%s: pack.addons needs a packwiz pack to sync beside", where)
	}
	for i, a := range pack.Addons {
		at := fmt.Sprintf("%s.pack.addons[%d]", where, i)
		if !AddonID(a.ID) {
			return fmt.Errorf("%s: id %q is not lower case letters, digits and dashes", at, a.ID)
		}
		if !plainLine(a.Name, 40) {
			return fmt.Errorf("%s: name %q is not a short plain name", at, a.Name)
		}
		if !plainLine(a.Description, maxAddonDescription) {
			return fmt.Errorf("%s: description is not one plain line of at most %d characters", at, maxAddonDescription)
		}
		if err := checkURL(at+".packwiz", a.Packwiz); err != nil {
			return err
		}
		if !commandSafeURL.MatchString(a.Packwiz) {
			return fmt.Errorf("%s: packwiz %q carries a character a command line would read", at, a.Packwiz)
		}
		if a.Packwiz == *pack.Packwiz {
			return fmt.Errorf("%s: packwiz is the chapter's own pack", at)
		}
		for j, other := range pack.Addons[:i] {
			switch {
			case other.ID == a.ID:
				return fmt.Errorf("%s: id %q is used by addons[%d] too", at, a.ID, j)
			case other.Name == a.Name:
				return fmt.Errorf("%s: name %q is used by addons[%d] too", at, a.Name, j)
			case other.Packwiz == a.Packwiz:
				return fmt.Errorf("%s: packwiz is addons[%d]'s too", at, j)
			}
		}
	}
	return nil
}

// plainLine reports whether s is non-empty, trimmed, at most max runes and free
// of control characters.
func plainLine(s string, max int) bool {
	t := strings.TrimSpace(s)
	return t != "" && t == s && len([]rune(s)) <= max && !strings.ContainsFunc(s, unicode.IsControl)
}

// addonRecord is the part of packwiz-installer's record the launcher reads:
// where each file it installed was put, relative to the pack folder
// (ManifestFile.kt, v0.5.14: "cachedLocation": "mods/Flashback-0.39.10.jar").
type addonRecord struct {
	CachedFiles map[string]struct {
		CachedLocation string `json:"cachedLocation"`
	} `json:"cachedFiles"`
}

// AddonJars is the base names of the jars an add-on's record says it put in the
// mods folder, sorted: only entries directly inside "mods/" by a name the jar
// shape accepts, so a record cannot point the launcher anywhere else. None when
// the add-on was never synced.
func AddonJars(gameDir, id string) ([]string, error) {
	if !AddonID(id) {
		return nil, fmt.Errorf("%q is not an add-on id", id)
	}
	path := filepath.Join(gameDir, addonRecordName(id))
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxAddonRecordBytes {
		return nil, errors.New("the add-on's record is not a file the installer wrote")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec addonRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, fmt.Errorf("the add-on's record cannot be read: %w", err)
	}
	var jars []string
	for _, f := range rec.CachedFiles {
		name, ok := strings.CutPrefix(f.CachedLocation, modsFolderName+"/")
		if ok && ModJarName(name) {
			jars = append(jars, name)
		}
	}
	slices.Sort(jars)
	return slices.Compact(jars), nil
}

// AddonEnabled reports whether the player turned the chapter's add-on on.
func AddonEnabled(settings models.AppSettings, chapterID, addonID string) bool {
	return slices.Contains(settings.EnabledAddons[chapterID], addonID)
}

// AddonStates resolves a chapter's add-ons against the player's choices and the
// game folder ("" before the pack's first sync). An add-on whose record cannot
// be read shows no jars.
func AddonStates(chapterID string, addons []models.PackAddon, settings models.AppSettings, gameDir string) []models.AddonState {
	states := make([]models.AddonState, 0, len(addons))
	for _, a := range addons {
		st := models.AddonState{
			ID: a.ID, Name: a.Name, Description: a.Description,
			Enabled: AddonEnabled(settings, chapterID, a.ID),
			Jars:    []string{},
		}
		if gameDir != "" {
			if jars, err := AddonJars(gameDir, a.ID); err == nil && jars != nil {
				st.Jars = jars
			}
		}
		states = append(states, st)
	}
	return states
}

// AddonJarSet is every jar of every add-on of the chapter, and the jars of the
// ones that are off: the mods page leaves the first out of its list, and the
// second is what a disabled list is applied with.
func AddonJarSet(states []models.AddonState) (all map[string]bool, off []string) {
	all = map[string]bool{}
	for _, st := range states {
		for _, jar := range st.Jars {
			all[jar] = true
			if !st.Enabled {
				off = append(off, jar)
			}
		}
	}
	return all, off
}

// SetAddonJars puts an add-on's jars away (".jar.disabled") or back. keepOff
// names jars that stay away whatever is asked: the player's own disabled list.
// It returns how many files it renamed; a jar it could not rename is in the
// error and does not stop the others.
func SetAddonJars(gameDir string, jars []string, enabled bool, keepOff []string) (int, error) {
	if len(jars) == 0 {
		return 0, nil
	}
	m, err := openModsFolder(gameDir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("mods folder: %w", err)
	}
	defer m.release()
	changed := 0
	var failed []error
	for _, jar := range jars {
		off := !enabled || slices.Contains(keepOff, jar)
		change, err := m.setDisabled(jar, off)
		if err != nil {
			failed = append(failed, fmt.Errorf("%s: %w", jar, err))
			continue
		}
		if change == changeRenamed || change == changeReplaced {
			changed++
		}
	}
	return changed, errors.Join(failed...)
}
