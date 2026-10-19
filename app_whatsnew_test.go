package main

import (
	"context"
	"testing"

	"kapital/backend/models"
	"kapital/backend/services"
)

// withVersion runs the test as a release build of version, with whatsNew as
// the bundled highlights.
func withVersion(t *testing.T, app *App, version string, whatsNew ...string) {
	t.Helper()
	previous := Version
	Version = version
	t.Cleanup(func() { Version = previous })
	app.whatsNew = nil
	for _, v := range whatsNew {
		app.whatsNew = append(app.whatsNew, models.WhatsNewRelease{
			Version:    v,
			Highlights: []models.WhatsNewHighlight{{Title: "A change", Text: "What it is."}},
		})
	}
}

func seenVersion(t *testing.T, app *App) string {
	t.Helper()
	s, err := app.settings.Load()
	if err != nil {
		t.Fatal(err)
	}
	return s.SeenVersion
}

// An update shows what came since the version seen, once.
func TestWhatsNewShowsReleasesSinceTheSeenOneOnce(t *testing.T) {
	app := newTestApp(t)
	withVersion(t, app, "v1.0.0", "v1.0.0", "v1.0.0-beta.6", "v1.0.0-beta.5")
	if err := app.settings.Save(models.AppSettings{Theme: "dark", SeenVersion: "v1.0.0-beta.5"}); err != nil {
		t.Fatal(err)
	}
	got, err := app.GetWhatsNew()
	if err != nil || len(got.Releases) != 2 || got.Releases[0].Version != "v1.0.0" || got.Version != "v1.0.0" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if wrote, err := app.MarkWhatsNewSeen(); err != nil || !wrote {
		t.Fatalf("mark: %v, %v", wrote, err)
	}
	if got, err := app.GetWhatsNew(); err != nil || len(got.Releases) != 0 {
		t.Fatalf("after seen: %+v, %v", got, err)
	}
}

// A fresh install has nothing new: it records the version and shows nothing.
func TestWhatsNewOnAFreshInstallShowsNothing(t *testing.T) {
	app := newTestApp(t)
	withVersion(t, app, "v1.0.0", "v1.0.0")
	if !app.freshInstall {
		t.Fatal("a new data dir is a fresh install")
	}
	got, err := app.GetWhatsNew()
	if err != nil || len(got.Releases) != 0 || seenVersion(t, app) != "v1.0.0" {
		t.Fatalf("got %+v, %v, seen %q", got, err, seenVersion(t, app))
	}
}

// A launcher from before What's new left settings but no seen version: this
// build's own entry shows.
func TestWhatsNewAfterALauncherWithoutIt(t *testing.T) {
	app := newTestApp(t)
	withVersion(t, app, "v1.0.0", "v1.0.0", "v0.9.0")
	app.freshInstall = false
	got, err := app.GetWhatsNew()
	if err != nil || len(got.Releases) != 1 || got.Releases[0].Version != "v1.0.0" || seenVersion(t, app) != "" {
		t.Fatalf("got %+v, %v, seen %q", got, err, seenVersion(t, app))
	}
}

// A release with no entry records itself, so the next update compares against it.
func TestWhatsNewWithNothingToShowRecordsTheVersion(t *testing.T) {
	app := newTestApp(t)
	withVersion(t, app, "v1.0.1", "v1.0.0")
	if err := app.settings.Save(models.AppSettings{Theme: "dark", SeenVersion: "v1.0.0"}); err != nil {
		t.Fatal(err)
	}
	if got, err := app.GetWhatsNew(); err != nil || len(got.Releases) != 0 || seenVersion(t, app) != "v1.0.1" {
		t.Fatalf("got %+v, %v, seen %q", got, err, seenVersion(t, app))
	}
}

// A dev build shows and writes nothing; the preview shows the newest entry
// and writes nothing either.
func TestWhatsNewDevBuildAndPreview(t *testing.T) {
	app := newTestApp(t)
	withVersion(t, app, "1.0.0-dev", "v1.0.0", "v0.9.0")
	got, err := app.GetWhatsNew()
	if err != nil || len(got.Releases) != 0 {
		t.Fatalf("dev: %+v, %v", got, err)
	}
	if _, err := app.StartPreview("", services.PreviewLauncherWhatsNew); err != nil {
		t.Fatal(err)
	}
	got, err = app.GetWhatsNew()
	if err != nil || len(got.Releases) != 1 || got.Releases[0].Version != "v1.0.0" {
		t.Fatalf("preview: %+v, %v", got, err)
	}
	if wrote, err := app.MarkWhatsNewSeen(); err != nil || wrote || seenVersion(t, app) != "" {
		t.Fatalf("preview mark: %v, %v", wrote, err)
	}
}

// A save from the settings screen never changes the version seen.
func TestSaveSettingsKeepsTheSeenVersion(t *testing.T) {
	app := newTestApp(t)
	if err := app.settings.Save(models.AppSettings{Theme: "dark", SeenVersion: "v1.0.0"}); err != nil {
		t.Fatal(err)
	}
	if err := app.SaveSettings(models.AppSettings{Theme: "light", SeenVersion: "v9.0.0"}); err != nil {
		t.Fatal(err)
	}
	if got := seenVersion(t, app); got != "v1.0.0" {
		t.Fatalf("seen %q", got)
	}
}

// The full changelog is read once a run, from the version seen before it; a
// failure is read again on the next open.
func TestWhatsNewChangelogIsReadOnceFromTheSeenVersion(t *testing.T) {
	app := newTestApp(t)
	withVersion(t, app, "v1.0.0", "v1.0.0")
	if err := app.settings.Save(models.AppSettings{Theme: "dark", SeenVersion: "v0.9.0"}); err != nil {
		t.Fatal(err)
	}
	var asked []string
	reason := "no answer"
	previous := fetchChangelog
	fetchChangelog = func(_ context.Context, seen, current string) models.Changelog {
		asked = append(asked, seen+".."+current)
		return models.Changelog{Reason: reason}
	}
	t.Cleanup(func() { fetchChangelog = previous })
	if _, err := app.GetWhatsNew(); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := app.GetWhatsNewChangelog(); err != nil {
			t.Fatal(err)
		}
	}
	reason = ""
	for range 2 {
		if _, err := app.GetWhatsNewChangelog(); err != nil {
			t.Fatal(err)
		}
	}
	if len(asked) != 3 || asked[0] != "v0.9.0..v1.0.0" {
		t.Fatalf("asked %v", asked)
	}
}

// Under the preview the changelog is made up and nothing is read.
func TestWhatsNewChangelogUnderThePreviewReadsNothing(t *testing.T) {
	app := newTestApp(t)
	previous := fetchChangelog
	fetchChangelog = func(context.Context, string, string) models.Changelog {
		t.Fatal("read under the preview")
		return models.Changelog{}
	}
	t.Cleanup(func() { fetchChangelog = previous })
	if _, err := app.StartPreview("", services.PreviewLauncherWhatsNew); err != nil {
		t.Fatal(err)
	}
	got, err := app.GetWhatsNewChangelog()
	if err != nil || len(got.Releases) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
}
