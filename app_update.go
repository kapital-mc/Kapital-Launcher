package main

import (
	"sync"

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

// GetUpdate says whether the download site offers a newer Kapital Launcher
// (ADR-9 amendment): one GET of the site's version.json per start, at a fixed
// address, sending nothing about the player or the build. A dev build asks
// nothing. Not reaching the site is an answer with a reason, never an error,
// so the launcher simply shows no notice. Under the launcher update preview
// (#124) the answer is made up and nothing is fetched.
func (a *App) GetUpdate() (models.UpdateInfo, error) {
	if info, ok := services.PreviewUpdate(a.previews.Launcher(), Version); ok {
		return info, nil
	}
	a.update.once.Do(func() {
		a.update.info = checkUpdate(a.context(), Version)
	})
	return a.update.info, nil
}
