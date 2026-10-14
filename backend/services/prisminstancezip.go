package services

import (
	"archive/zip"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"time"

	"kapital/backend/models"
)

// A chapter's instance as a zip that Prism imports (Add Instance, Import), for
// a player who uses Prism without the launcher (#251). It holds what Install
// writes for a fresh instance (instanceFiles) and no mod: the first Play
// installs the pack through the pre-launch command and every later Play syncs
// it, so nothing third-party is inside and nothing is re-hosted (ADR-3). This
// is packwiz's own documented MultiMC setup, exported as an instance zip.
//
// Two things differ from Install:
//
//   - The pre-launch command is the first template, with packwiz-installer's
//     window (legacyPreLaunchCommand). The launcher runs it headless because
//     its splash covers the sync (#95); in Prism alone the window is the only
//     progress a player sees, and it is where packwiz-installer asks before it
//     moves mmc-pack.json to a pack's new loader version.
//   - The jars' MIT licence goes in beside them (packwizlicense.txt), since
//     this is a copy of them handed to someone else.
//
// Prism finds the instance by its instance.cfg, at the zip's root here, and
// names the instance from its dialog, which suggests the zip's own name
// (PrismZipName). instance.cfg is stored first, though Prism does not need it.

//go:embed packwizlicense.txt
var packwizLicense []byte

const packwizLicenseName = "packwiz-LICENSE.txt"

// prismZipTime is every entry's modified time, so the same inputs make the same
// bytes and a release's checksum says whether anything changed.
var prismZipTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

var prismZipChapterName = regexp.MustCompile(`^[A-Za-z0-9]{1,32}$`)

// PrismZipName is the release asset's file name for a chapter. Prism suggests
// it, less ".zip", as the instance name, so it is a name a player can keep:
// Kapitel-Kapital-Frangfurd. It is deliberately not kapital-<id>, the folder
// the launcher looks for (ADR-2), which a case-insensitive disk would also
// match. site/vite.config.js builds the same name for the download links.
func PrismZipName(chapter models.Chapter) (string, error) {
	if !prismZipChapterName.MatchString(chapter.Name) {
		return "", fmt.Errorf("%s: name %q cannot be a file name as it stands", chapter.ID, chapter.Name)
	}
	return "Kapitel-Kapital-" + chapter.Name + ".zip", nil
}

// PrismZip builds the chapter's instance zip from its hosted pack.
func (c *InstanceCreator) PrismZip(ctx context.Context, chapter models.Chapter) ([]byte, error) {
	if chapter.Pack.Packwiz == nil {
		return nil, fmt.Errorf("%s has no hosted pack to build a Prism instance from", chapter.ID)
	}
	packURL := *chapter.Pack.Packwiz
	if err := c.checkPackURL(chapter.ID+".pack.packwiz", packURL); err != nil {
		return nil, err
	}
	cfg, err := renderConfigWith(chapter, packURL, legacyPreLaunchCommand)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	files, err := c.instanceFiles(ctx, chapter, packURL, c.checkPackURL)
	if err != nil {
		return nil, err
	}
	files[packwizLicenseName] = packwizLicense
	return zipInstance(cfg, files)
}

// zipInstance writes instance.cfg and then files in name order (only
// minecraft/ has a separator, so the OS's does not change the order), with
// forward slashes whatever the OS, ordinary file modes and a fixed time.
func zipInstance(cfg []byte, files map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	add := func(name string, data []byte) error {
		h := &zip.FileHeader{Name: filepath.ToSlash(name), Method: zip.Deflate, Modified: prismZipTime}
		h.SetMode(0o644)
		f, err := w.CreateHeader(h)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	}
	if err := add(instanceConfig, cfg); err != nil {
		return nil, err
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := add(name, files[name]); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
