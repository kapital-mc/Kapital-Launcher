package main

import (
	"log/slog"

	"kapital/backend/models"
	"kapital/backend/windowchrome"
)

// GetWindowChrome is where macOS drew its title bar, for the header bar to
// take its height and its left inset from (ADR-10, 2026-10-19 amendment). The
// zero value off macOS, where the header draws its own window buttons.
func (a *App) GetWindowChrome() (models.WindowChrome, error) {
	if c, ok := windowchrome.Measure(); ok {
		return c, nil
	}
	return models.WindowChrome{}, nil
}

// logWindowChrome sets the bar's style as the window comes up and logs what
// macOS made of it, the one line a report from a Mac needs to place the header.
func logWindowChrome() {
	if c, ok := windowchrome.Measure(); ok {
		slog.Info("window chrome", "titlebar", c.TitlebarHeight, "controlsEnd", c.ControlsEnd)
	}
}
