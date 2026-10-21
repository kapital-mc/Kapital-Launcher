package main

import (
	"context"
	"log/slog"
	"path/filepath"

	"kapital/backend/services"
)

// adoptRunningGames picks up the chapters' games that are already running when
// the launcher starts, after a restart into an update or a launcher reopened
// while a game ran (ADR-2, nineteenth amendment), so their chapters show the
// game and hold Play. Only the detected Prism's games are looked for, in the
// manifest's instance folders. It runs once, off the startup path.
func (a *App) adoptRunningGames(ctx context.Context) {
	engine := a.realEngine()
	if a.games == nil || !engine.Found || engine.Executable == "" {
		return
	}
	settings, err := a.settings.Load()
	if err != nil {
		slog.Info("running games not looked for", "error", err)
		return
	}
	report := a.prism.Instances(settings, engine, a.manifest.Chapters)
	if report.Dir == "" {
		return
	}
	candidates := make([]services.AdoptCandidate, 0, len(a.manifest.Chapters))
	for _, c := range a.manifest.Chapters {
		candidates = append(candidates, services.AdoptCandidate{ChapterID: c.ID, InstanceDir: filepath.Join(report.Dir, c.Instance.ID)})
	}
	a.games.Adopt(ctx, engine.Executable, candidates)
}
