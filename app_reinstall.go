package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"kapital/backend/models"
	"kapital/backend/services"
)

// ReinstallChapter replaces a chapter's instance that the launcher did not set
// up (issue 244, ADR-2's fourteenth amendment): one under the launcher's name
// with no pre-launch command of its own, made by hand or by another tool, which
// Play would start as it is and never sync. The old folder is moved aside to
// <id>-stashed-<day> in the same instances folder, nothing in it deleted, and
// the chapter is installed as InstallChapter does. An install that fails puts
// the old folder back. The answer names the stash, which ShowStashedInstance
// opens in Prism.
//
// It refuses an instance that is the launcher's own (one with a pack URL in
// the report: SetPackSource is its way to another pack), a chapter that is not
// installed, and everything InstallChapter and a write to an instance refuse:
// a preview, an active or running game, an open Prism, no Prism, no pack.
func (a *App) ReinstallChapter(chapterID string) (models.ReinstallResult, error) {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return models.ReinstallResult{}, fmt.Errorf("no chapter %q", chapterID)
	}
	if err := a.refuseUnderPreview(chapter); err != nil {
		return models.ReinstallResult{}, err
	}
	engine := a.realEngine()
	if a.games.Active(chapterID) {
		return models.ReinstallResult{}, errGameActive(chapter)
	}
	if !engine.Found {
		return models.ReinstallResult{}, services.ErrPrismNotFound
	}
	settings, err := a.settings.Load()
	if err != nil {
		return models.ReinstallResult{}, err
	}
	override := settings.PackOverrides[chapter.ID]
	if chapter.Pack.Packwiz == nil && override == "" {
		return models.ReinstallResult{}, fmt.Errorf("%s has no hosted pack to install", chapter.Name)
	}
	report := a.prism.Instances(settings, engine, a.manifest.Chapters)
	if report.Dir == "" || !report.Present[chapterID] {
		return models.ReinstallResult{}, fmt.Errorf("%s is not installed", chapter.Name)
	}
	if report.PackURL[chapterID] != "" {
		return models.ReinstallResult{}, fmt.Errorf("%s is already set up by Kapital Launcher", chapter.Name)
	}
	// Renaming a folder Prism has loaded, or one the game writes to, would
	// leave Prism and the game on a path that is gone.
	if err := a.refuseIfRunning(chapter, filepath.Join(report.Dir, chapter.Instance.ID)); err != nil {
		return models.ReinstallResult{}, err
	}
	if err := a.refuseIfPrismOpen(chapter, engine.Executable); err != nil {
		return models.ReinstallResult{}, err
	}
	stashed, err := services.StashInstance(report.Dir, chapter.Instance.ID, time.Now())
	if err != nil {
		slog.Error("reinstall chapter", "chapter", chapterID, "error", err)
		return models.ReinstallResult{}, fmt.Errorf("%s could not be moved aside: %w", chapter.Name, err)
	}
	slog.Info("instance stashed", "chapter", chapterID, "stash", stashed)
	report.Present[chapterID] = false
	if err := a.creator.Install(a.context(), chapter, report, override); err != nil {
		slog.Error("reinstall chapter", "chapter", chapterID, "error", err)
		if back := services.UnstashInstance(report.Dir, chapter.Instance.ID, stashed); back != nil {
			slog.Error("instance not put back", "chapter", chapterID, "stash", stashed, "error", back)
			return models.ReinstallResult{}, fmt.Errorf("%w; the old instance is kept in Prism as %s", err, stashed)
		}
		slog.Info("instance put back", "chapter", chapterID)
		return models.ReinstallResult{}, err
	}
	slog.Info("reinstalled", "chapter", chapterID, "instance", chapter.Instance.ID, "stash", stashed, "local pack", override != "")
	a.seenMu.Lock()
	if a.stashes == nil {
		a.stashes = map[string]string{}
	}
	a.stashes[chapterID] = stashed
	a.seenMu.Unlock()
	instances, err := a.GetInstances()
	return models.ReinstallResult{Instances: instances, Stash: stashed}, err
}

// ShowStashedInstance opens, in Prism's own window, the old instance the
// chapter's last Reinstall in this app run moved aside (issue 244), so the
// player can play, copy from or delete it there. The folder name is the one Go
// recorded, never the caller's, and must still be in the instances folder.
func (a *App) ShowStashedInstance(chapterID string) error {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return fmt.Errorf("no chapter %q", chapterID)
	}
	a.seenMu.Lock()
	stashed := a.stashes[chapterID]
	a.seenMu.Unlock()
	if stashed == "" {
		return fmt.Errorf("%s has no backed-up instance from this session", chapter.Name)
	}
	report, err := a.realInstances()
	if err != nil {
		return err
	}
	if report.Dir == "" {
		return errors.New("cannot work out where Prism keeps its instances")
	}
	if _, err := os.Stat(filepath.Join(report.Dir, stashed, "instance.cfg")); err != nil {
		return fmt.Errorf("the backed-up %s instance is no longer in Prism", chapter.Name)
	}
	engine := a.realEngine()
	if err := a.showInPrism(engine, stashed, engine.Root); err != nil {
		slog.Error("show stashed instance in prism", "chapter", chapter.ID, "error", err)
		return fmt.Errorf("could not open the backed-up %s instance in Prism: %w", chapter.Name, err)
	}
	slog.Info("opened stash in prism", "chapter", chapter.ID, "stash", stashed)
	return nil
}
