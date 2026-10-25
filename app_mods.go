package main

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"

	"kapital/backend/models"
	"kapital/backend/services"
)

// chapterGameDir is the instance's game folder, the one that holds mods, or ""
// before the pack's first sync made it.
func chapterGameDir(cfg string) string {
	return services.GameFolder(filepath.Dir(cfg))
}

// GetChapterMods lists a chapter's mods for the settings page (issue 156): every
// jar in the instance's mods folder with whether it is switched off, the
// manifest's quick toggles resolved against them, and whether the game looks to
// be running. The chapter id resolves through the manifest and the folder is the
// one the instance report gives, so nothing the caller sends names a path.
//
// When the game is not running it first puts away what a sync run that was
// killed (Prism's cancel is a hard kill) left enabled, from its journal and
// only for mods still on the player's list, so the page shows what the player
// chose and not a half-finished run.
func (a *App) GetChapterMods(chapterID string) (models.ChapterMods, error) {
	chapter, cfg, err := a.chapterInstance(chapterID)
	if err != nil {
		return models.ChapterMods{}, err
	}
	return a.chapterMods(chapter, cfg, true)
}

// chapterMods builds the view. recoverJournal puts a killed run's mods away first,
// which only ever happens while the game is not running.
func (a *App) chapterMods(chapter models.Chapter, cfg string, recoverJournal bool) (models.ChapterMods, error) {
	tracker, guess := a.chapterRunning(chapter, filepath.Dir(cfg))
	settings, err := a.settings.Load()
	if err != nil {
		return models.ChapterMods{}, err
	}
	gameDir := chapterGameDir(cfg)
	view := models.ChapterMods{
		ChapterID: chapter.ID,
		Mods:      []models.ModFile{},
		Toggles:   services.ModToggleStates(chapter.Pack.Toggles, nil),
		Addons:    services.AddonStates(chapter.ID, chapter.Pack.Addons, settings, gameDir),
		Running:   tracker || guess,
	}
	if gameDir == "" {
		return view, nil
	}
	if recoverJournal && !view.Running {
		if n, err := services.RecoverDisabledMods(gameDir, settings.DisabledMods[chapter.ID]); err != nil {
			slog.Warn("mods journal not recovered", "chapter", chapter.ID, "error", err)
		} else if n > 0 {
			slog.Info("mods put away after an interrupted sync", "chapter", chapter.ID, "mods", n)
		}
	}
	mods, err := services.ListMods(gameDir)
	if err != nil {
		slog.Error("list mods", "chapter", chapter.ID, "error", err)
		return models.ChapterMods{}, a.maskedError(chapter, err)
	}
	// An add-on's jars have the add-on's switch and no other.
	addonJars, _ := services.AddonJarSet(view.Addons)
	view.Mods = slices.DeleteFunc(mods, func(m models.ModFile) bool { return addonJars[m.Name] })
	view.Toggles = services.ModToggleStates(chapter.Pack.Toggles, view.Mods)
	return view, nil
}

// SetModsDisabled saves which of a chapter's mods are switched off and applies
// it to the mods folder at once, so the folder matches before the next Play does
// anything. disabled is the whole list: every jar base name named is disabled,
// every other jar of the folder is enabled. Each name must be a plain jar name
// that is in the instance's mods folder now (services.ValidateDisabledMods), the
// chapter resolves through the manifest, a game that runs, or a developer
// preview of the chapter, refuses it, and the tracker's word and the game log's
// are both asked (refuseIfRunning).
//
// The list is the launcher's own, in its settings: the pre-launch sync that
// Prism runs reads it, puts these jars back for packwiz-installer so it
// downloads nothing, and puts them away again (ADR-2, ninth amendment). It is
// saved before the folder is changed, so a rename that fails now is made by the
// next Play. The page's answer is the folder read again.
func (a *App) SetModsDisabled(chapterID string, disabled []string) (models.ChapterMods, error) {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return models.ChapterMods{}, fmt.Errorf("no chapter %q", chapterID)
	}
	if err := a.refuseUnderPreview(chapter); err != nil {
		return models.ChapterMods{}, err
	}
	chapter, cfg, err := a.chapterInstance(chapterID)
	if err != nil {
		return models.ChapterMods{}, err
	}
	if err := a.refuseIfRunning(chapter, filepath.Dir(cfg)); err != nil {
		return models.ChapterMods{}, err
	}
	gameDir := chapterGameDir(cfg)
	if gameDir == "" {
		return models.ChapterMods{}, fmt.Errorf("%s has no mods yet: Play once so the pack is downloaded", chapter.Name)
	}
	// What a killed sync left is put away by the list that was kept, before the
	// list changes.
	before, err := a.settings.Load()
	if err != nil {
		return models.ChapterMods{}, err
	}
	if _, err := services.RecoverDisabledMods(gameDir, before.DisabledMods[chapter.ID]); err != nil {
		slog.Warn("mods journal not recovered", "chapter", chapter.ID, "error", err)
	}
	clean, err := services.ValidateDisabledMods(gameDir, disabled)
	if err != nil {
		return models.ChapterMods{}, fmt.Errorf("%s: %w", chapter.Name, err)
	}
	// A mod whose requirement is off goes off with it, whatever the page sent.
	present, err := services.ListMods(gameDir)
	if err != nil {
		return models.ChapterMods{}, fmt.Errorf("%s: %w", chapter.Name, err)
	}
	clean = services.WithRequiredOff(chapter.Pack.Toggles, present, clean)
	// An add-on's jars are not the list's: the add-on's switch is theirs, and the
	// folder keeps the ones of an add-on that is off put away.
	addonJars, addonsOff := services.AddonJarSet(services.AddonStates(chapter.ID, chapter.Pack.Addons, before, gameDir))
	clean = slices.DeleteFunc(clean, func(name string) bool { return addonJars[name] })
	if err := a.settings.Update(func(s *models.AppSettings) error {
		next := make(map[string][]string, len(s.DisabledMods)+1)
		for id, names := range s.DisabledMods {
			next[id] = names
		}
		if len(clean) == 0 {
			delete(next, chapter.ID)
		} else {
			next[chapter.ID] = clean
		}
		s.DisabledMods = next
		return nil
	}); err != nil {
		return models.ChapterMods{}, err
	}
	changed, applyErr := services.ApplyDisabledMods(gameDir, append(slices.Clip(clean), addonsOff...))
	slog.Info("mods disabled", "chapter", chapter.ID, "disabled", len(clean), "renamed", changed)
	if applyErr != nil {
		slog.Warn("mods not all renamed", "chapter", chapter.ID, "error", applyErr)
		return models.ChapterMods{}, fmt.Errorf("%s: the choice is saved, but not every mod could be renamed now (the next Play does it)", chapter.Name)
	}
	return a.chapterMods(chapter, cfg, false)
}

// SetAddonEnabled turns one of a chapter's add-ons on or off (ADR-2, twenty-second
// amendment). The choice is saved in the launcher's settings, where the
// pre-launch sync reads it: an add-on that is on is downloaded by the next Play
// with its own pack, one that is off is not. The add-on resolves through the
// manifest, so nothing the caller sends names a pack or a path, and the write is
// refused while the game runs and under a developer preview, as the mods' are.
//
// The folder follows at once where the add-on was synced before: its jars are
// put away, or back unless the player's own list disables one. Before the pack's
// first sync there is nothing to rename, and the choice waits for the first Play.
func (a *App) SetAddonEnabled(chapterID, addonID string, enabled bool) (models.ChapterMods, error) {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return models.ChapterMods{}, fmt.Errorf("no chapter %q", chapterID)
	}
	if !slices.ContainsFunc(chapter.Pack.Addons, func(ad models.PackAddon) bool { return ad.ID == addonID }) {
		return models.ChapterMods{}, fmt.Errorf("%s has no add-on %q", chapter.Name, addonID)
	}
	if err := a.refuseUnderPreview(chapter); err != nil {
		return models.ChapterMods{}, err
	}
	chapter, cfg, err := a.chapterInstance(chapterID)
	if err != nil {
		return models.ChapterMods{}, err
	}
	if err := a.refuseIfRunning(chapter, filepath.Dir(cfg)); err != nil {
		return models.ChapterMods{}, err
	}
	var saved models.AppSettings
	if err := a.settings.Update(func(s *models.AppSettings) error {
		next := make(map[string][]string, len(s.EnabledAddons)+1)
		for id, ids := range s.EnabledAddons {
			next[id] = ids
		}
		on := slices.DeleteFunc(slices.Clone(next[chapter.ID]), func(id string) bool { return id == addonID })
		if enabled {
			on = append(on, addonID)
		}
		next[chapter.ID] = on
		s.EnabledAddons = next
		saved = *s
		return nil
	}); err != nil {
		return models.ChapterMods{}, err
	}
	gameDir := chapterGameDir(cfg)
	if gameDir != "" {
		jars, err := services.AddonJars(gameDir, addonID)
		if err != nil {
			slog.Warn("add-on record", "chapter", chapter.ID, "addon", addonID, "error", err)
		}
		changed, applyErr := services.SetAddonJars(gameDir, jars, enabled, saved.DisabledMods[chapter.ID])
		slog.Info("add-on switched", "chapter", chapter.ID, "addon", addonID, "enabled", enabled, "renamed", changed)
		if applyErr != nil {
			slog.Warn("add-on jars not all renamed", "chapter", chapter.ID, "error", applyErr)
			return models.ChapterMods{}, fmt.Errorf("%s: the choice is saved, but not every jar could be renamed now (the next Play does it)", chapter.Name)
		}
	}
	return a.chapterMods(chapter, cfg, false)
}
