package services

import (
	"errors"
	"fmt"
	"io/fs"
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
// entries only: no file is opened, links are not followed, and the walk stops
// at maxSizeWalkEntries. Deleting removes one of four fixed folders, or one
// server's terrain folder, through an os.Root on the instance folder, and only
// a real directory: a link or a file in its place is refused untouched.

const (
	xaeroWorldMap  = "world-map"
	xaeroFolder    = "xaero"
	screenshotsDir = "screenshots"
	logsDir        = "logs"
	crashDir       = "crash-reports"
	modsDir        = "mods"
)

// storageOrder is the order the kinds after terrain are listed in.
var storageOrder = []string{
	models.StorageTerrainDownload,
	models.StorageWorldMap,
	models.StorageScreenshots,
	models.StorageLogs,
	models.StorageMods,
	models.StorageOther,
}

// MeasureStorage walks an instance folder and sums its files by kind. The
// free space is the instance disk's, -1 when it cannot be read.
func MeasureStorage(instanceDir string) (models.ChapterStorage, error) {
	game := ""
	if dir := gameFolder(instanceDir); dir != "" {
		game = filepath.Base(dir)
	}
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
	if free, err := diskFree(instanceDir); err == nil {
		report.FreeBytes = free
	}
	return report, nil
}

// storageKind files a path inside the instance, split into its segments, under
// a kind and, for terrain, the server's folder. game is the game folder's name,
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
// storageOrder.
func sortStorage(sums map[string]*models.StorageItem) []models.StorageItem {
	items := []models.StorageItem{}
	for _, it := range sums {
		if it.Kind == models.StorageTerrain {
			items = append(items, *it)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Bytes != items[j].Bytes {
			return items[i].Bytes > items[j].Bytes
		}
		return items[i].Name < items[j].Name
	})
	for _, kind := range storageOrder {
		if it := sums[kind+"/"]; it != nil {
			items = append(items, *it)
		}
	}
	return items
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

// validTerrainFolder is the shape of a server folder name ClearStorage takes:
// one plain path segment.
func validTerrainFolder(name string) bool {
	return name != "" && len(name) <= 255 && name != "." && name != ".." &&
		!strings.ContainsAny(name, "/\\:\x00")
}

// ClearStorage deletes one kind of an instance's storage: one server's terrain
// (name is its folder), the unfinished terrain download, the Xaero world map, or
// the logs and crash reports. The caller has made sure the game is closed and
// no terrain download runs. A folder that is not there is nothing to delete.
func ClearStorage(instanceDir, kind, name string) error {
	game := ""
	if dir := gameFolder(instanceDir); dir != "" {
		game = filepath.Base(dir)
	}
	var targets []string
	switch kind {
	case models.StorageTerrainDownload:
		targets = []string{dhStaging}
	case models.StorageTerrain:
		if !validTerrainFolder(name) {
			return fmt.Errorf("%q is not a terrain folder", name)
		}
		targets = []string{filepath.Join(game, dhServerData, name)}
	case models.StorageWorldMap:
		targets = []string{filepath.Join(game, xaeroFolder, xaeroWorldMap)}
	case models.StorageLogs:
		targets = []string{filepath.Join(game, logsDir), filepath.Join(game, crashDir)}
	default:
		return fmt.Errorf("%q cannot be deleted", kind)
	}
	if game == "" && kind != models.StorageTerrainDownload {
		return errors.New("the instance has no game folder yet")
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

// ScreenshotsFolder is the instance's screenshots folder, or "" when there is
// none yet.
func ScreenshotsFolder(instanceDir string) string {
	game := gameFolder(instanceDir)
	if game == "" {
		return ""
	}
	dir := filepath.Join(game, screenshotsDir)
	if info, err := os.Lstat(dir); err != nil || !info.IsDir() {
		return ""
	}
	return dir
}
