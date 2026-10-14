package main

import (
	"errors"
	"log/slog"
	"time"

	wailsrt "github.com/wailsapp/wails/v2/pkg/runtime"

	"kapital/backend/models"
	"kapital/backend/services"
)

// The Prism the launcher runs: what detection found, re-detecting it, and
// getting or updating the launcher-managed copy (ADR-11). Each answer the view
// reads may come from a developer preview (#124, app_preview.go); the app's own
// work reads realEngine.

// GetEngine returns what is known about the Prism install, from the last
// detection. RefreshEngine re-runs detection. Under a Prism preview (#124) it
// answers with the synthetic engine; the app's own work reads realEngine.
//
// On Windows Wails runs startup beside the page load, so the page can ask
// before the first detection has finished, and it asks once (issue 260). The
// answer waits for that detection, up to engineWait.
func (a *App) GetEngine() (models.EngineInfo, error) {
	if info, ok := a.previewEngine(); ok {
		return info, nil
	}
	if a.engineReady != nil {
		select {
		case <-a.engineReady:
		case <-time.After(engineWait):
			slog.Warn("engine asked for before detection finished", "waited", engineWait)
		}
	}
	return a.realEngine(), nil
}

// engineWait bounds GetEngine's wait for startup's detection, which is itself
// bounded: Flatpak's check (5 s) and Prism's --version (15 s) at most.
const engineWait = 30 * time.Second

// detectStartupEngine is startup's detection: it keeps the result and then
// lets GetEngine answer, whether or not detection could read the settings.
func (a *App) detectStartupEngine() {
	if a.engineReady != nil {
		defer close(a.engineReady)
	}
	if _, err := a.detectEngine(); err != nil {
		slog.Warn("engine detection", "error", err)
	}
}

// realEngine is the Prism the last detection found, which every action of the
// app uses whatever a preview shows the view.
func (a *App) realEngine() models.EngineInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.engine
}

// RefreshEngine re-detects Prism with the current settings and returns the
// result. Called on startup and after the settings change. Under a Prism
// preview it runs nothing and answers with the synthetic engine.
func (a *App) RefreshEngine() (models.EngineInfo, error) {
	if info, ok := a.previewEngine(); ok {
		return info, nil
	}
	return a.detectEngine()
}

// detectEngine is the detection itself, and keeps the result.
func (a *App) detectEngine() (models.EngineInfo, error) {
	settings, err := a.settings.Load()
	if err != nil {
		return models.EngineInfo{}, err
	}
	info := a.prism.Detect(a.context(), settings)
	a.mu.Lock()
	a.engine = info
	a.mu.Unlock()
	slog.Info("engine", "found", info.Found, "source", info.Source, "version", info.Version)
	return info, nil
}

// GetPrismRelease reads Prism's latest release: what installing Prism would
// download, and whether the launcher-managed copy has an update. Nothing is
// downloaded; the approval card shows this before InstallPrism is called.
func (a *App) GetPrismRelease() (models.PrismRelease, error) {
	// Under a Prism preview (#124) the release is a made-up one: no request.
	if rel, ok := a.previewRelease(); ok {
		return rel, nil
	}
	rel, err := a.managed.Latest(a.context())
	if err != nil {
		return rel, err
	}
	// An update is only worth offering for the Prism the launcher runs, and
	// measured against the version it runs, which Prism's own updater may have
	// changed (#221).
	engine := a.realEngine()
	if engine.Source != "managed" {
		rel.UpdateAvailable = false
		return rel, nil
	}
	a.managed.Reconcile(a.context(), &rel, engine.Version)
	return rel, nil
}

// InstallPrism installs or updates the launcher-managed Prism from Prism's
// latest release, after the player approved it. The release is read again
// here rather than taken from the caller, so what is downloaded is always
// what Prism published. Progress arrives as prism:install events; detection
// re-runs once it is in place.
func (a *App) InstallPrism() error {
	// Under a Prism preview (#124) the install is a made-up one that fails: it
	// never reaches the installer, so nothing is downloaded or unpacked.
	if a.previews.Prism() != "" {
		return a.playPreviewInstall()
	}
	rel, err := a.managed.Latest(a.context())
	if err != nil {
		a.emitPrismInstall(models.PrismInstallProgress{Phase: "failed", Error: err.Error()})
		return err
	}
	if err := a.refusePrismUpdateWhileOpen(rel); err != nil {
		a.emitPrismInstall(models.PrismInstallProgress{Phase: "failed", Error: err.Error()})
		return err
	}
	if err := a.managed.Install(a.context(), rel, a.emitPrismInstall); err != nil {
		slog.Error("install prism", "version", rel.Version, "error", err)
		return err
	}
	_, err = a.detectEngine()
	return err
}

// refusePrismUpdateWhileOpen refuses an update of the managed Prism while a
// process runs from its executable (#227): an update removes the previous
// version's folder, and on macOS that delete succeeds under a running Prism and
// the game it started. The look is the one the instance.cfg writes take
// (OtherPrismOpen, by name and image path), and it covers the launcher's own
// Prism too, which stays up while its game runs. A first install, or the
// version already in place, has nothing to remove. Where the processes cannot
// be listed the update goes on, as before. Nothing is ended.
func (a *App) refusePrismUpdateWhileOpen(rel models.PrismRelease) error {
	installed, exe, ok := a.managed.Installed()
	if !ok || installed == rel.Version {
		return nil
	}
	open, err := a.otherPrismOpen("", exe)
	if err != nil {
		slog.Info("could not tell whether the managed Prism is open", "error", err)
		return nil
	}
	if open {
		return errPrismUpdateOpen
	}
	return nil
}

// errPrismUpdateOpen is what an update refused for an open Prism says.
var errPrismUpdateOpen = errors.New("Prism is open; close it, and any game it started, then update")

// emitPrismInstall is one prism:install event.
func (a *App) emitPrismInstall(p models.PrismInstallProgress) {
	if a.emitInstall != nil {
		a.emitInstall(p)
		return
	}
	if a.ctx != nil {
		wailsrt.EventsEmit(a.ctx, services.EventPrismInstall, p)
	}
}
