package main

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"kapital/backend/models"
	"kapital/backend/services"
)

// The game settings of a chapter's Instance section (ADR-2, twenty-fifth
// amendment): client-side settings files reset one at a time. The caller names
// a chapter and an item of services' fixed list, never a path.

// GetGameSettings lists the game settings a chapter's instance can reset, each
// with the files it holds now. It reads directory entries, options.txt and the
// sync's record, and writes nothing.
func (a *App) GetGameSettings(chapterID string) (models.ChapterGameSettings, error) {
	chapter, cfg, err := a.chapterInstance(chapterID)
	if err != nil {
		return models.ChapterGameSettings{}, err
	}
	return a.gameSettings(chapter, filepath.Dir(cfg))
}

func (a *App) gameSettings(chapter models.Chapter, instanceDir string) (models.ChapterGameSettings, error) {
	items, err := services.ListGameSettings(instanceDir)
	if err != nil {
		slog.Error("list game settings", "chapter", chapter.ID, "error", err)
		return models.ChapterGameSettings{}, a.storageError(chapter, err)
	}
	tracker, guess := a.chapterRunning(chapter, instanceDir)
	return models.ChapterGameSettings{ChapterID: chapter.ID, Items: items, Running: tracker || guess}, nil
}

// ResetGameSetting resets one of a chapter's game settings and answers the
// list read again. It is refused under a developer preview and while the game
// runs: the game writes these files when it closes, which would put the old
// settings back.
func (a *App) ResetGameSetting(chapterID, id string) (models.ChapterGameSettings, error) {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return models.ChapterGameSettings{}, fmt.Errorf("no chapter %q", chapterID)
	}
	if err := a.refuseUnderPreview(chapter); err != nil {
		return models.ChapterGameSettings{}, err
	}
	chapter, cfg, err := a.chapterInstance(chapterID)
	if err != nil {
		return models.ChapterGameSettings{}, err
	}
	instanceDir := filepath.Dir(cfg)
	if err := a.refuseIfRunning(chapter, instanceDir); err != nil {
		return models.ChapterGameSettings{}, err
	}
	if err := services.ResetGameSetting(instanceDir, id); err != nil {
		slog.Warn("reset game setting", "chapter", chapter.ID, "setting", id, "error", err)
		return models.ChapterGameSettings{}, a.storageError(chapter, err)
	}
	slog.Info("game setting reset", "chapter", chapter.ID, "setting", id)
	return a.gameSettings(chapter, instanceDir)
}
