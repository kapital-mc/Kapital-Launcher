package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"

	"kapital/backend/models"
	"kapital/backend/services"
)

// A chapter's Contraptions page: the Tool Gun's saved contraptions and
// Create's schematics in its game folder (services/contraptions.go). The page
// names a chapter, a kind and a file name the listing gave; every path is built
// here from the instance chapterInstance resolves. A write is refused under a
// developer preview. The game reads these files only when the Tool Gun or the
// Schematic Table lists them, so writes are allowed while it runs.

// ListContraptions lists a chapter's saved contraptions, newest first, with
// the facts each file carries. A chapter that has not been started yet has
// none.
func (a *App) ListContraptions(chapterID string) ([]models.Contraption, error) {
	chapter, game, err := a.contraptionGame(chapterID)
	if err != nil {
		return nil, err
	}
	if game == "" {
		return []models.Contraption{}, nil
	}
	list, err := services.ListContraptions(game)
	if err != nil {
		slog.Error("list contraptions", "chapter", chapter.ID, "error", err)
		return nil, a.storageError(chapter, err)
	}
	return list, nil
}

// RenameContraption renames one file (name without its extension) and lists
// the chapter again.
func (a *App) RenameContraption(chapterID, kind, file, name string) ([]models.Contraption, error) {
	chapter, game, err := a.contraptionWrite(chapterID)
	if err != nil {
		return nil, err
	}
	renamed, err := services.RenameContraption(game, kind, file, name)
	if err != nil {
		slog.Warn("rename contraption", "chapter", chapter.ID, "kind", kind, "error", err)
		return nil, a.storageError(chapter, err)
	}
	slog.Info("contraption renamed", "chapter", chapter.ID, "kind", kind, "file", renamed)
	return a.ListContraptions(chapterID)
}

// DeleteContraption deletes one file and lists the chapter again.
func (a *App) DeleteContraption(chapterID, kind, file string) ([]models.Contraption, error) {
	chapter, game, err := a.contraptionWrite(chapterID)
	if err != nil {
		return nil, err
	}
	if err := services.DeleteContraption(game, kind, file); err != nil {
		slog.Warn("delete contraption", "chapter", chapter.ID, "kind", kind, "error", err)
		return nil, a.storageError(chapter, err)
	}
	slog.Info("contraption deleted", "chapter", chapter.ID, "kind", kind)
	return a.ListContraptions(chapterID)
}

// ImportContraptions copies the .excraft and .nbt files the player picks into
// the chapter and returns how many arrived; 0 when the dialog was cancelled.
// Each file is checked on its own; the first refusal is returned after the
// others are copied.
func (a *App) ImportContraptions(chapterID string) (int, error) {
	chapter, game, err := a.contraptionWrite(chapterID)
	if err != nil {
		return 0, err
	}
	picked, err := a.contraptionFiles(fmt.Sprintf("Import into %s", chapter.Name))
	if err != nil {
		return 0, err
	}
	imported := 0
	var first error
	for _, src := range picked {
		if _, err := services.ImportContraption(game, src); err != nil {
			slog.Warn("import contraption", "chapter", chapter.ID, "error", err)
			if first == nil {
				first = a.storageError(chapter, err)
			}
			continue
		}
		imported++
	}
	slog.Info("contraptions imported", "chapter", chapter.ID, "count", imported)
	return imported, first
}

// ExportContraption copies one file to where the player picks; false when the
// dialog was cancelled.
func (a *App) ExportContraption(chapterID, kind, file string) (bool, error) {
	chapter, game, err := a.contraptionGame(chapterID)
	if err != nil {
		return false, err
	}
	if game == "" {
		return false, fmt.Errorf("%s has no game folder yet", chapter.Name)
	}
	dest, err := a.contraptionSaveFile(fmt.Sprintf("Export %s", file), file)
	if err != nil || dest == "" {
		return false, err
	}
	if err := services.ExportContraption(game, kind, file, dest); err != nil {
		slog.Warn("export contraption", "chapter", chapter.ID, "kind", kind, "error", err)
		return false, a.storageError(chapter, err)
	}
	return true, nil
}

// OpenContraptionFolder shows a kind's folder in the file manager, made first
// when the game has not made it yet.
func (a *App) OpenContraptionFolder(chapterID, kind string) error {
	chapter, game, err := a.contraptionGame(chapterID)
	if err != nil {
		return err
	}
	if game == "" {
		return fmt.Errorf("%s has no game folder yet", chapter.Name)
	}
	dir, err := services.ContraptionFolder(game, kind)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return a.storageError(chapter, err)
	}
	if err := a.openFolder(dir); err != nil {
		slog.Error("open contraption folder", "chapter", chapter.ID, "kind", kind, "error", err)
		return fmt.Errorf("could not open the %s folder", chapter.Name)
	}
	return nil
}

// contraptionGame is a chapter's game folder, "" before its first start.
func (a *App) contraptionGame(chapterID string) (models.Chapter, string, error) {
	chapter, cfg, err := a.chapterInstance(chapterID)
	if err != nil {
		return models.Chapter{}, "", err
	}
	return chapter, services.GameFolder(filepath.Dir(cfg)), nil
}

// contraptionWrite is contraptionGame for a change: refused under a preview
// and before the game folder exists.
func (a *App) contraptionWrite(chapterID string) (models.Chapter, string, error) {
	chapter, ok := a.chapter(chapterID)
	if !ok {
		return models.Chapter{}, "", fmt.Errorf("no chapter %q", chapterID)
	}
	if err := a.refuseUnderPreview(chapter); err != nil {
		return models.Chapter{}, "", err
	}
	chapter, game, err := a.contraptionGame(chapterID)
	if err != nil {
		return models.Chapter{}, "", err
	}
	if game == "" {
		return models.Chapter{}, "", fmt.Errorf("%s has no game folder yet", chapter.Name)
	}
	return chapter, game, nil
}

func (a *App) contraptionFiles(title string) ([]string, error) {
	if a.pickFiles != nil {
		return a.pickFiles(title)
	}
	if a.ctx == nil {
		return nil, errors.New("window is not ready")
	}
	return wailsrt.OpenMultipleFilesDialog(a.ctx, wailsrt.OpenDialogOptions{
		Title:   title,
		Filters: []wailsrt.FileFilter{{DisplayName: "Contraptions (*.excraft, *.nbt)", Pattern: "*.excraft;*.nbt"}},
	})
}

func (a *App) contraptionSaveFile(title, name string) (string, error) {
	if a.pickSaveFile != nil {
		return a.pickSaveFile(title, name)
	}
	if a.ctx == nil {
		return "", errors.New("window is not ready")
	}
	return wailsrt.SaveFileDialog(a.ctx, wailsrt.SaveDialogOptions{Title: title, DefaultFilename: name})
}
