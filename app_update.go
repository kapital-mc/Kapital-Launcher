package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"

	"kapital/backend/models"
	"kapital/backend/services"
)

// checkUpdate is the look at the download site, a variable so the tests can
// count the requests without a network.
var checkUpdate = services.CheckUpdate

// updateCheck holds the one answer a start gets, so the page asking again (a
// reload, a preview cleared) reaches the site only once per run.
type updateCheck struct {
	once sync.Once
	info models.UpdateInfo
}

// selfUpdate is the launcher's own update (ADR-6 amendment, 2026-10-21): the
// background download, Cancel on it, and the copy to start once this one has
// quit.
type selfUpdate struct {
	updater *services.LauncherUpdater
	// emit, when set, takes the launcher:update events in place of the
	// window; a test sets it.
	emit     func(models.LauncherUpdateProgress)
	download installCancel

	mu sync.Mutex
	// relaunch is the copy main starts after the window has closed, "" for none.
	relaunch string
}

// GetUpdate says whether the download site offers a newer Kapital Launcher
// (ADR-9 amendment): one GET of the site's version.json per start, at a fixed
// address, sending nothing about the player or the build. A dev build asks
// nothing. Not reaching the site is an answer with a reason, never an error,
// so the launcher simply shows no notice. Under the launcher update preview
// (#124) the answer is made up and nothing is fetched.
//
// Installable says whether this copy can download the update and restart into
// it; Blocked says why not, and the popup then offers the site alone.
func (a *App) GetUpdate() (models.UpdateInfo, error) {
	if info, ok := services.PreviewUpdate(a.previews.Launcher(), Version); ok {
		info.Installable = true
		return info, nil
	}
	a.update.once.Do(func() {
		a.update.info = checkUpdate(a.context(), Version)
	})
	info := a.update.info
	if info.Available && a.self.updater != nil {
		if _, err := a.self.updater.Target(); err != nil {
			info.Blocked = err.Error()
		} else {
			info.Installable = true
		}
	}
	return info, nil
}

// DownloadUpdate downloads the release the site offers in the background,
// checks it and stages it beside this copy, reporting each step as a
// launcher:update event; the last is "ready", after which RestartToUpdate
// swaps it in. Nothing of the running launcher changes before that.
func (a *App) DownloadUpdate() error {
	info, err := a.GetUpdate()
	if err != nil {
		return err
	}
	if !info.Available || !info.Installable {
		return errors.New("there is no update this launcher can install itself")
	}
	ctx, end := a.self.download.begin(a.context())
	emit := func(p models.LauncherUpdateProgress) {
		if a.self.download.step(p.Phase) {
			a.emitLauncherUpdate(p)
		}
	}
	if a.previews.Launcher() == services.PreviewLauncherUpdate {
		err = a.playPreviewUpdate(ctx, emit)
	} else {
		err = a.self.updater.Download(ctx, info.Latest, emit)
	}
	if end() {
		a.emitLauncherUpdate(models.LauncherUpdateProgress{Phase: "cancelled", Version: info.Latest})
		return errInstallCancelled
	}
	if err != nil {
		slog.Error("launcher update download", "version", info.Latest, "error", err)
	}
	return err
}

// CancelUpdateDownload cancels the update's download, reporting whether one ran.
func (a *App) CancelUpdateDownload() (bool, error) {
	return a.self.download.stop(), nil
}

// RestartToUpdate swaps the staged update in and quits; main starts the new
// copy once the window has closed. A game that is still loading or closing
// waits, as its window may be hidden behind the loading card; one that is
// running carries on, since quitting never closes a game. A download that
// runs waits too, since quitting would cut it. Under the preview nothing is
// swapped and the launcher restarts as it is.
func (a *App) RestartToUpdate() error {
	if err := a.refuseRestart(); err != nil {
		return err
	}
	var target string
	var err error
	if a.previews.Launcher() == services.PreviewLauncherUpdate {
		target, err = a.self.updater.Target()
	} else {
		target, err = a.self.updater.Apply()
	}
	if err != nil {
		return err
	}
	a.self.mu.Lock()
	a.self.relaunch = target
	a.self.mu.Unlock()
	if a.ctx != nil {
		wailsrt.Quit(a.ctx)
	}
	return nil
}

// loadingPhases are the phases in which a game's window may be hidden behind
// the loading card, or the game is between closing and gone.
var loadingPhases = []string{
	models.GamePhaseStarting, models.GamePhaseMods, models.GamePhaseWindow,
	models.GamePhaseResources, models.GamePhaseStopping,
}

// refuseRestart is nil when quitting now loses nothing.
func (a *App) refuseRestart() error {
	if a.games != nil {
		for _, s := range a.games.All() {
			if slices.Contains(loadingPhases, s.Phase) {
				return fmt.Errorf("Wait until %s has finished loading, then restart", a.chapterName(s.ChapterID))
			}
		}
	}
	if a.prismInstall.running() || a.tailscale.install.running() || a.terrain.running() {
		return errors.New("Wait until the downloads have finished, then restart")
	}
	return nil
}

// chapterName is the chapter's name from the manifest, or its id.
func (a *App) chapterName(id string) string {
	for _, c := range a.manifest.Chapters {
		if c.ID == id {
			return c.Name
		}
	}
	return id
}

// relaunchAfterQuit starts the copy RestartToUpdate chose, once the window has
// closed and shutdown has run; main calls it after wails.Run returns.
func (a *App) relaunchAfterQuit() {
	a.self.mu.Lock()
	target := a.self.relaunch
	a.self.mu.Unlock()
	if target == "" {
		return
	}
	if err := a.self.updater.Relaunch(target); err != nil {
		slog.Error("launcher update: relaunch", "target", target, "error", err)
		return
	}
	slog.Info("launcher update: relaunched", "target", target)
}

// cleanUpUpdate removes what an earlier update left beside this copy, off the
// startup path. A dev build has nothing there.
func (a *App) cleanUpUpdate(ctx context.Context) {
	if a.self.updater == nil || strings.HasSuffix(Version, "-dev") {
		return
	}
	a.self.updater.CleanUp(ctx)
}

// playPreviewUpdate is the made-up download of the update preview: it fills,
// is checked and is ready, and nothing is fetched or staged.
func (a *App) playPreviewUpdate(ctx context.Context, emit func(models.LauncherUpdateProgress)) error {
	for i, step := range services.PreviewLauncherUpdateProgress() {
		if i > 0 && !a.previewPause(ctx) {
			return ctx.Err()
		}
		emit(step)
	}
	return nil
}

func (a *App) emitLauncherUpdate(p models.LauncherUpdateProgress) {
	if a.self.emit != nil {
		a.self.emit(p)
		return
	}
	if a.ctx != nil {
		wailsrt.EventsEmit(a.ctx, services.EventLauncherUpdate, p)
	}
}
