package main

import (
	"sync"

	"kapital/backend/models"
	"kapital/backend/services"
)

// whatsNewState is what one run knows of What's new: the version seen before
// it, which the full changelog is measured from, and that changelog once read.
type whatsNewState struct {
	mu        sync.Mutex
	from      string
	changelog *models.Changelog
}

// fetchChangelog is the read of the site's changelog, a variable so the tests
// need no network.
var fetchChangelog = services.FetchChangelog

// GetWhatsNew is what the first start after an update shows: the highlights of
// every release since the version the player last saw, built into the app
// (data/whatsnew.json). A fresh install has seen nothing to call new, so it
// records this version and shows nothing; a dev build shows nothing. With
// nothing to show the version is recorded at once, so a later update compares
// against it. Under the launcher's What's new preview it is this build's
// newest entry, whatever was seen, and nothing is recorded.
func (a *App) GetWhatsNew() (models.WhatsNew, error) {
	out := models.WhatsNew{Version: Version}
	if a.previews.Launcher() == services.PreviewLauncherWhatsNew {
		out.Releases = services.PreviewWhatsNew(a.whatsNew)
		return out, nil
	}
	if !services.IsReleaseTag(Version) {
		return out, nil
	}
	settings, err := a.settings.Load()
	if err != nil {
		return out, err
	}
	if settings.SeenVersion == Version {
		return out, nil
	}
	if settings.SeenVersion != "" || !a.freshInstall {
		out.Releases = services.WhatsNewSince(a.whatsNew, settings.SeenVersion, Version)
	}
	a.whatsNewRun.mu.Lock()
	a.whatsNewRun.from = settings.SeenVersion
	a.whatsNewRun.mu.Unlock()
	if len(out.Releases) == 0 {
		_, err = a.MarkWhatsNewSeen()
	}
	return out, err
}

// GetWhatsNewChangelog is the full changelog What's new folds away, read from
// the download site when the player opens it (services.FetchChangelog): the
// releases since the version seen before this run, up to this build. Read once
// a run when it was reached; under the preview it is made up and nothing is
// fetched.
func (a *App) GetWhatsNewChangelog() (models.Changelog, error) {
	if a.previews.Launcher() == services.PreviewLauncherWhatsNew {
		return services.PreviewChangelog(), nil
	}
	a.whatsNewRun.mu.Lock()
	defer a.whatsNewRun.mu.Unlock()
	if a.whatsNewRun.changelog != nil {
		return *a.whatsNewRun.changelog, nil
	}
	got := fetchChangelog(a.context(), a.whatsNewRun.from, Version)
	if got.Reason == "" {
		a.whatsNewRun.changelog = &got
	}
	return got, nil
}

// MarkWhatsNewSeen records this version as seen, when the player closes What's
// new. It reports whether it wrote; a dev build and a preview write nothing.
func (a *App) MarkWhatsNewSeen() (bool, error) {
	if a.previews.Launcher() == services.PreviewLauncherWhatsNew || !services.IsReleaseTag(Version) {
		return false, nil
	}
	err := a.settings.Update(func(s *models.AppSettings) error {
		s.SeenVersion = Version
		return nil
	})
	return err == nil, err
}
