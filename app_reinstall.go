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

// ReinstallChapter puts a chapter's instance back to a fresh copy of its pack.
//
// The launcher's own instance (one with a pack URL in the report) is reset
// (ADR-2's twenty-third amendment): the old folder is moved aside, a new one
// installed from the same pack source, the player's worlds, screenshots,
// schematics, Distant Horizons terrain, maps, replays, server list and
// Essential's data moved across with the play time, and the rest of the old
// folder removed. The next Play's sync fetches mods and configs whole. An
// install that fails puts the old folder back; a carry that fails leaves the
// old folder in Prism under its stash name and says so.
//
// An instance the launcher did not set up (issue 244, ADR-2's fourteenth
// amendment), one under the launcher's name with no pre-launch command of its
// own, is moved aside to <id>-stashed-<day> with nothing in it deleted, and the
// answer names the stash, which ShowStashedInstance opens in Prism.
//
// It refuses a chapter that is not installed, a terrain download under way, and
// everything InstallChapter and a write to an instance refuse: a preview, an
// active or running game, an open Prism, no Prism, no pack.
func (a *App) ReinstallChapter(chapterID string) (models.ReinstallResult, error) {
	chapter, report, override, err := a.reinstallTarget(chapterID)
	if err != nil {
		return models.ReinstallResult{}, err
	}
	own := report.PackURL[chapterID] != ""
	stashed, err := services.StashInstance(report.Dir, chapter.Instance.ID, time.Now())
	if err != nil {
		slog.Error("reinstall chapter", "chapter", chapterID, "error", err)
		return models.ReinstallResult{}, fmt.Errorf("%s could not be moved aside: %w", chapter.Name, err)
	}
	slog.Info("instance stashed", "chapter", chapterID, "stash", stashed, "own", own)
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
	return a.finishReinstall(chapter, report.Dir, stashed, own)
}

// finishReinstall answers a reinstall whose new instance is installed. For the
// launcher's own instance it resets, which clears the stash when the carry
// succeeds; a carry that fails keeps the stash, which still holds its
// instance.cfg and what was not moved. A stash still there is recorded for
// ShowStashedInstance and named in the answer.
func (a *App) finishReinstall(chapter models.Chapter, instancesDir, stashed string, own bool) (models.ReinstallResult, error) {
	if own {
		kept, err := a.finishReset(chapter, instancesDir, stashed)
		if err != nil {
			return models.ReinstallResult{}, err
		}
		if !kept {
			stashed = ""
		}
	}
	slog.Info("reinstalled", "chapter", chapter.ID, "instance", chapter.Instance.ID, "stash", stashed)
	if stashed != "" {
		a.seenMu.Lock()
		if a.stashes == nil {
			a.stashes = map[string]string{}
		}
		a.stashes[chapter.ID] = stashed
		a.seenMu.Unlock()
	}
	instances, err := a.GetInstances()
	return models.ReinstallResult{Instances: instances, Stash: stashed}, err
}

// reinstallTarget runs Reinstall's refusals and answers the chapter, the
// instances report and the pack override the new instance is installed from:
// the local pack when the launcher's own instance syncs from it now, otherwise
// the published one; for an instance the launcher did not set up, the local
// pack whenever settings name one, as Install does.
func (a *App) reinstallTarget(chapterID string) (models.Chapter, models.InstanceReport, string, error) {
	chapter, report, err := a.instanceWriteTarget(chapterID)
	if err != nil {
		return chapter, report, "", err
	}
	settings, err := a.settings.Load()
	if err != nil {
		return chapter, report, "", err
	}
	override := settings.PackOverrides[chapter.ID]
	if own := report.PackURL[chapterID]; own != "" && own != override {
		override = ""
	}
	if chapter.Pack.Packwiz == nil && override == "" {
		return chapter, report, "", fmt.Errorf("%s has no hosted pack to install", chapter.Name)
	}
	return chapter, report, override, nil
}

// instanceWriteTarget is the refusals Reinstall and Uninstall share before
// they move or remove a chapter's instance folder: a preview, an active or
// running game, no Prism, a chapter that is not installed, a terrain download
// under way and an open Prism. Renaming or removing a folder Prism has loaded,
// or one the game writes to, would leave Prism and the game on a path that is
// gone.
func (a *App) instanceWriteTarget(chapterID string) (models.Chapter, models.InstanceReport, error) {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return chapter, models.InstanceReport{}, fmt.Errorf("no chapter %q", chapterID)
	}
	if err := a.refuseUnderPreview(chapter); err != nil {
		return chapter, models.InstanceReport{}, err
	}
	if a.games.Active(chapterID) {
		return chapter, models.InstanceReport{}, errGameActive(chapter)
	}
	engine := a.realEngine()
	if !engine.Found {
		return chapter, models.InstanceReport{}, services.ErrPrismNotFound
	}
	settings, err := a.settings.Load()
	if err != nil {
		return chapter, models.InstanceReport{}, err
	}
	report := a.prism.Instances(settings, engine, a.manifest.Chapters)
	if report.Dir == "" || !report.Present[chapterID] {
		return chapter, report, fmt.Errorf("%s is not installed", chapter.Name)
	}
	if a.terrain != nil && a.terrain.busy(chapterID) {
		return chapter, report, errors.New("a terrain download is under way; cancel it first")
	}
	if err := a.refuseIfRunning(chapter, filepath.Join(report.Dir, chapter.Instance.ID)); err != nil {
		return chapter, report, err
	}
	if err := a.refuseIfPrismOpen(chapter, engine.Executable); err != nil {
		return chapter, report, err
	}
	return chapter, report, nil
}

// finishReset moves what Reinstall keeps from the stash into the new instance
// and removes the rest of the stash. It answers true when the carry failed and
// the stash is kept as an instance in Prism; an error is a stash that lost its
// instance.cfg to a removal that failed.
func (a *App) finishReset(chapter models.Chapter, instancesDir, stashed string) (bool, error) {
	moved, err := services.CarryOver(instancesDir, stashed, chapter.Instance.ID)
	if err != nil {
		slog.Error("reinstall carry over", "chapter", chapter.ID, "stash", stashed, "moved", moved, "error", err)
		return true, nil
	}
	if err := services.RemoveInstance(instancesDir, stashed, time.Now()); err != nil {
		slog.Error("reinstall remove old instance", "chapter", chapter.ID, "stash", stashed, "error", err)
		return false, fmt.Errorf("%s was reinstalled, but the old files were not all removed: %w", chapter.Name, err)
	}
	slog.Info("instance reset", "chapter", chapter.ID, "kept", moved)
	return false, nil
}

// UninstallChapter removes the chapter's whole instance folder from Prism's
// instances folder, worlds and terrain included (ADR-2's twenty-third
// amendment). The launcher's settings for the chapter stay, so Install brings
// back the same mod switches and server choice. A stash an earlier Reinstall
// made is a folder of its own and stays. It refuses what Reinstall refuses,
// but needs no pack.
func (a *App) UninstallChapter(chapterID string) (models.InstanceReport, error) {
	chapter, report, err := a.instanceWriteTarget(chapterID)
	if err != nil {
		return models.InstanceReport{}, err
	}
	if err := services.RemoveInstance(report.Dir, chapter.Instance.ID, time.Now()); err != nil {
		slog.Error("uninstall chapter", "chapter", chapterID, "error", err)
		return models.InstanceReport{}, fmt.Errorf("%s could not be removed: %w", chapter.Name, err)
	}
	slog.Info("uninstalled", "chapter", chapterID, "instance", chapter.Instance.ID)
	return a.GetInstances()
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
