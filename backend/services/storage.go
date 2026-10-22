package services

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"kapital/backend/models"
)

// The storage section of a chapter's settings: what the instance holds on disk
// by kind, and the few kinds the player may delete. Measuring reads directory
// entries, and of the files only each world's icon.png: links are not followed,
// and the walk stops at maxSizeWalkEntries. Every kind has fixed folders (storageTargets), built
// here from the kind and, for terrain and saves, a plain folder name. Deleting removes
// those of the six deletable kinds through an os.Root on the instance folder,
// and only a real directory: a link or a file in its place is refused untouched.
// Opening shows the first of them that is a real directory.

const (
	xaeroWorldMap  = "world-map"
	xaeroMinimap   = "minimap"
	xaeroFolder    = "xaero"
	screenshotsDir = "screenshots"
	logsDir        = "logs"
	crashDir       = "crash-reports"
	modsDir        = "mods"
	savesDir       = "saves"
	worldIcon      = "icon.png"
	// maxWorldIcon bounds the icon read. The game writes a 64x64 PNG, a few KiB.
	maxWorldIcon = 64 << 10
)

// storageOrder is the order the kinds after terrain are listed in. Saves, one
// item per world, are listed largest first in their place.
var storageOrder = []string{
	models.StorageTerrainDownload,
	models.StorageWorldMap,
	models.StorageMinimap,
	models.StorageSaves,
	models.StorageScreenshots,
	models.StorageLogs,
	models.StorageMods,
	models.StorageOther,
}

// MeasureStorage walks an instance folder and sums its files by kind. The
// free space is the instance disk's, -1 when it cannot be read.
func MeasureStorage(instanceDir string) (models.ChapterStorage, error) {
	game := gameFolderName(instanceDir)
	sums := map[string]*models.StorageItem{}
	report := models.ChapterStorage{FreeBytes: -1}
	entries := 0
	err := filepath.WalkDir(instanceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == instanceDir {
				return err
			}
			return nil
		}
		entries++
		if entries > maxSizeWalkEntries {
			report.Partial = true
			return fs.SkipAll
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(instanceDir, path)
		if err != nil {
			return nil
		}
		kind, name := storageKind(game, strings.Split(rel, string(filepath.Separator)))
		key := kind + "/" + name
		item := sums[key]
		if item == nil {
			item = &models.StorageItem{Kind: kind, Name: name}
			if kind == models.StorageTerrain {
				item.Label = terrainLabel(name)
			}
			sums[key] = item
		}
		item.Bytes += info.Size()
		item.Files++
		if at := info.ModTime().UTC().Format(time.RFC3339); at > item.LastModified {
			item.LastModified = at
		}
		report.TotalBytes += info.Size()
		return nil
	})
	if err != nil {
		return models.ChapterStorage{}, err
	}
	report.Items = sortStorage(sums)
	addWorldIcons(instanceDir, game, report.Items)
	if free, err := diskFree(instanceDir); err == nil {
		report.FreeBytes = free
	}
	return report, nil
}

// storageKind files a path inside the instance, split into its segments, under
// a kind and, for terrain, the server's folder, for saves, the world's. game is the game folder's name,
// "" when the instance has none yet.
func storageKind(game string, parts []string) (kind, name string) {
	if len(parts) > 1 && parts[0] == dhStaging {
		return models.StorageTerrainDownload, ""
	}
	if game == "" || len(parts) < 3 || parts[0] != game {
		return models.StorageOther, ""
	}
	switch parts[1] {
	case dhServerData:
		if len(parts) > 3 {
			return models.StorageTerrain, parts[2]
		}
	case xaeroFolder:
		if len(parts) > 3 && parts[2] == xaeroWorldMap {
			return models.StorageWorldMap, ""
		}
		if len(parts) > 3 && parts[2] == xaeroMinimap {
			return models.StorageMinimap, ""
		}
	case savesDir:
		if len(parts) > 3 {
			return models.StorageSaves, parts[2]
		}
	case screenshotsDir:
		return models.StorageScreenshots, ""
	case logsDir, crashDir:
		return models.StorageLogs, ""
	case modsDir:
		return models.StorageMods, ""
	}
	return models.StorageOther, ""
}

// sortStorage lists terrain largest first, then the other kinds in
// storageOrder, with saves largest first in their place.
func sortStorage(sums map[string]*models.StorageItem) []models.StorageItem {
	items := byKind(sums, models.StorageTerrain)
	for _, kind := range storageOrder {
		if kind == models.StorageSaves {
			items = append(items, byKind(sums, kind)...)
		} else if it := sums[kind+"/"]; it != nil {
			items = append(items, *it)
		}
	}
	return items
}

// byKind is every item of one kind, largest first, then by name.
func byKind(sums map[string]*models.StorageItem, kind string) []models.StorageItem {
	items := []models.StorageItem{}
	for _, it := range sums {
		if it.Kind == kind {
			items = append(items, *it)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Bytes != items[j].Bytes {
			return items[i].Bytes > items[j].Bytes
		}
		return items[i].Name < items[j].Name
	})
	return items
}

// addWorldIcons sets each save's Icon to its icon.png as a data URL, read
// through an os.Root on the instance folder. A world without one, or with a
// link, an oversized file or anything but a PNG in its place, keeps "".
func addWorldIcons(instanceDir, game string, items []models.StorageItem) {
	if game == "" {
		return
	}
	root, err := os.OpenRoot(instanceDir)
	if err != nil {
		return
	}
	defer root.Close() //nolint:errcheck // a read-only handle
	for i := range items {
		if items[i].Kind == models.StorageSaves {
			items[i].Icon = readWorldIcon(root, filepath.Join(game, savesDir, items[i].Name, worldIcon))
		}
	}
}

func readWorldIcon(root *os.Root, path string) string {
	info, err := root.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxWorldIcon {
		return ""
	}
	body, err := root.ReadFile(path)
	if err != nil || len(body) > maxWorldIcon || http.DetectContentType(body) != "image/png" {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(body)
}

// terrainLabel is a server folder's name as Distant Horizons wrote it: the
// server's name, form-encoded ("Frangfurd+%28Germany%29"). A name that does not
// decode is shown as it is.
func terrainLabel(folder string) string {
	if s, err := url.QueryUnescape(folder); err == nil && s != "" {
		return s
	}
	return folder
}

// validFolderName is the shape of a server or world folder name ClearStorage
// and StorageFolder take: one plain path segment.
func validFolderName(name string) bool {
	return name != "" && len(name) <= 255 && name != "." && name != ".." &&
		!strings.ContainsAny(name, "/\\:\x00")
}

// deletable are the kinds ClearStorage removes. The minimap (waypoints), the
// mods (put back by the next sync) and everything else (options, config,
// Essential's folder) are only measured and opened.
var deletable = map[string]bool{
	models.StorageTerrain:         true,
	models.StorageTerrainDownload: true,
	models.StorageWorldMap:        true,
	models.StorageSaves:           true,
	models.StorageScreenshots:     true,
	models.StorageLogs:            true,
}

// storageTargets are a kind's folders, relative to the instance folder; game is
// the game folder's name. Everything else (".") is the game folder itself, or
// the instance folder when there is none yet.
func storageTargets(game, kind, name string) ([]string, error) {
	if kind == models.StorageTerrainDownload {
		return []string{dhStaging}, nil
	}
	if kind == models.StorageOther && game == "" {
		return []string{"."}, nil
	}
	if game == "" {
		return nil, errors.New("the instance has no game folder yet")
	}
	switch kind {
	case models.StorageTerrain:
		if !validFolderName(name) {
			return nil, fmt.Errorf("%q is not a terrain folder", name)
		}
		return []string{filepath.Join(game, dhServerData, name)}, nil
	case models.StorageSaves:
		if !validFolderName(name) {
			return nil, fmt.Errorf("%q is not a world folder", name)
		}
		return []string{filepath.Join(game, savesDir, name)}, nil
	case models.StorageWorldMap:
		return []string{filepath.Join(game, xaeroFolder, xaeroWorldMap)}, nil
	case models.StorageMinimap:
		return []string{filepath.Join(game, xaeroFolder, xaeroMinimap)}, nil
	case models.StorageScreenshots:
		return []string{filepath.Join(game, screenshotsDir)}, nil
	case models.StorageLogs:
		return []string{filepath.Join(game, logsDir), filepath.Join(game, crashDir)}, nil
	case models.StorageMods:
		return []string{filepath.Join(game, modsDir)}, nil
	case models.StorageOther:
		return []string{game}, nil
	}
	return nil, fmt.Errorf("%q is not a kind of storage", kind)
}

// gameFolderName is the instance's game folder's name, "" when it has none.
func gameFolderName(instanceDir string) string {
	if dir := gameFolder(instanceDir); dir != "" {
		return filepath.Base(dir)
	}
	return ""
}

// ClearStorage deletes one kind of an instance's storage: one server's terrain
// (name is its folder), the unfinished terrain download, the Xaero world map,
// one world (name is its folder under saves), the screenshots, or the logs and
// crash reports. The caller has made sure the
// game is closed and no terrain download runs. A folder that is not there is
// nothing to delete.
func ClearStorage(instanceDir, kind, name string) error {
	if !deletable[kind] {
		return fmt.Errorf("%q cannot be deleted", kind)
	}
	targets, err := storageTargets(gameFolderName(instanceDir), kind, name)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(instanceDir)
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck // nothing was written through the handle itself
	for _, target := range targets {
		info, err := root.Lstat(target)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a folder; left as it is", filepath.ToSlash(target))
		}
		if err := root.RemoveAll(target); err != nil {
			return err
		}
	}
	return nil
}

// StorageFolder is the folder a kind of an instance's storage is in, to show in
// the file manager: the first of its targets that is a real directory inside
// the instance folder. A kind with none yet is an error.
func StorageFolder(instanceDir, kind, name string) (string, error) {
	targets, err := storageTargets(gameFolderName(instanceDir), kind, name)
	if err != nil {
		return "", err
	}
	root, err := os.OpenRoot(instanceDir)
	if err != nil {
		return "", err
	}
	defer root.Close() //nolint:errcheck // a read-only handle
	for _, target := range targets {
		if info, err := root.Lstat(target); err == nil && info.IsDir() {
			return filepath.Join(instanceDir, target), nil
		}
	}
	return "", errors.New("that folder is not there yet")
}
