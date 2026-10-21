package main

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"kapital/backend/models"
	"kapital/backend/services"
)

// The storage section of a chapter's settings: what the instance holds on disk
// by kind, a delete for the kinds that come back or are the player's to drop,
// and a way to the screenshots. The caller names a chapter and, for terrain, a
// server folder the listing gave; every path is built in Go from the instance
// chapterInstance resolves.

// GetChapterStorage measures a chapter's instance folder by kind: each
// server's Distant Horizons terrain, the unfinished terrain download, the world
// map, screenshots, logs, mods and the rest, with the disk's free space. It
// reads directory entries only and writes nothing.
func (a *App) GetChapterStorage(chapterID string) (models.ChapterStorage, error) {
	chapter, cfg, err := a.chapterInstance(chapterID)
	if err != nil {
		return models.ChapterStorage{}, err
	}
	return a.chapterStorage(chapter, filepath.Dir(cfg))
}

func (a *App) chapterStorage(chapter models.Chapter, instanceDir string) (models.ChapterStorage, error) {
	report, err := services.MeasureStorage(instanceDir)
	if err != nil {
		slog.Error("measure storage", "chapter", chapter.ID, "error", err)
		return models.ChapterStorage{}, a.storageError(chapter, err)
	}
	report.ChapterID = chapter.ID
	tracker, guess := a.chapterRunning(chapter, instanceDir)
	report.Running = tracker || guess
	return report, nil
}

// ClearChapterStorage deletes one kind of a chapter's storage and returns the
// instance measured again. kind is terrain (name is a server folder under
// Distant_Horizons_server_data), terrainDownload, worldMap or logs; anything
// else is refused. It is refused under a developer preview, while the game runs
// (Distant Horizons and the game hold these files open) and, for terrain, while
// a terrain download of the chapter runs.
func (a *App) ClearChapterStorage(chapterID, kind, name string) (models.ChapterStorage, error) {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return models.ChapterStorage{}, fmt.Errorf("no chapter %q", chapterID)
	}
	if err := a.refuseUnderPreview(chapter); err != nil {
		return models.ChapterStorage{}, err
	}
	chapter, cfg, err := a.chapterInstance(chapterID)
	if err != nil {
		return models.ChapterStorage{}, err
	}
	instanceDir := filepath.Dir(cfg)
	if err := a.refuseIfRunning(chapter, instanceDir); err != nil {
		return models.ChapterStorage{}, err
	}
	if (kind == models.StorageTerrain || kind == models.StorageTerrainDownload) && a.terrain.busy(chapter.ID) {
		return models.ChapterStorage{}, errors.New("a terrain download is under way; cancel it first")
	}
	if err := services.ClearStorage(instanceDir, kind, name); err != nil {
		slog.Warn("clear storage", "chapter", chapter.ID, "kind", kind, "error", err)
		return models.ChapterStorage{}, a.storageError(chapter, err)
	}
	slog.Info("storage cleared", "chapter", chapter.ID, "kind", kind)
	return a.chapterStorage(chapter, instanceDir)
}

// OpenScreenshotsFolder shows a chapter's screenshots folder in the file
// manager. Like OpenInstanceFolder, the caller names a chapter and never a
// path; an instance with no screenshots folder yet is refused.
func (a *App) OpenScreenshotsFolder(chapterID string) error {
	chapter, cfg, err := a.chapterInstance(chapterID)
	if err != nil {
		return err
	}
	dir := services.ScreenshotsFolder(filepath.Dir(cfg))
	if dir == "" {
		return fmt.Errorf("%s has no screenshots yet", chapter.Name)
	}
	if err := a.openFolder(dir); err != nil {
		slog.Error("open screenshots folder", "chapter", chapter.ID, "error", err)
		return fmt.Errorf("could not open the %s screenshots folder", chapter.Name)
	}
	return nil
}

// storageError is a failure as the page shows it: a file system error names
// the instance's absolute path, which carries the player's home folder, so its
// text passes the redactor.
func (a *App) storageError(chapter models.Chapter, err error) error {
	redactor, rerr := a.redactor()
	if rerr != nil {
		return maskedErr{msg: chapter.Name + ": the instance folder could not be read", err: err}
	}
	return maskErr(chapter, redactor, err)
}
