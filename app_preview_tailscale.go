package main

import (
	"errors"

	"kapital/backend/models"
	"kapital/backend/services"
)

// The launcher's Tailscale previews (#124, services/preview_tailscale.go): the
// settings row as Install, Connect and the switch, and the install popup
// failing. Under one, every Tailscale method answers from the preview: no CLI
// runs, no page opens, nothing is downloaded, and a switch is not saved
// (checkTailscaleSave).

// startLauncherPreview makes the situation the launcher's preview and, for a
// Tailscale one, tells the view what the row now shows.
func (a *App) startLauncherPreview(situation string) {
	a.previews.SetLauncher(situation)
	if info, ok := a.previewTailscale(); ok {
		a.emitTailscaleState(info)
	}
}

// clearLauncherPreview ends the launcher's preview, and after a Tailscale one
// looks at the real Tailscale again, which it then emits: the last look Go
// kept is forgotten, so the real state is news even when it did not change.
func (a *App) clearLauncherPreview() {
	was := a.previews.ClearLauncher()
	if _, ok := services.PreviewTailscale(was, nil); !ok {
		return
	}
	ts := a.tailscale
	ts.mu.Lock()
	ts.last = models.TailscaleInfo{}
	ts.mu.Unlock()
	a.lookAtTailscale(a.context())
}

// previewTailscale is the Tailscale of a launcher preview, if one holds.
func (a *App) previewTailscale() (models.TailscaleInfo, bool) {
	return services.PreviewTailscale(a.previews.Launcher(), a.manifest.Chapters)
}

func (a *App) underTailscalePreview() bool {
	_, ok := a.previewTailscale()
	return ok
}

// connectPreviewTailscale is Connect under the not-connected preview: the
// preview becomes the connected one, as accepting the share link would make
// it, and nothing opens.
func (a *App) connectPreviewTailscale() {
	if a.previews.Launcher() != services.PreviewTailscaleDisconnected {
		return
	}
	a.startLauncherPreview(services.PreviewTailscaleConnected)
}

// playPreviewTailscaleInstall is the made-up Tailscale install: the steps of
// services.PreviewTailscaleInstallProgress as tailscale:install events, a pause
// apart, then the failure it ends in. The installer is never reached.
func (a *App) playPreviewTailscaleInstall() error {
	steps := services.PreviewTailscaleInstallProgress()
	for i, step := range steps {
		if i > 0 && !a.previewPause() {
			return a.context().Err()
		}
		a.emitTailscaleInstall(step)
	}
	return errors.New(steps[len(steps)-1].Error)
}
