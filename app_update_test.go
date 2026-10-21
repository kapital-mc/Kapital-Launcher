package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"kapital/backend/models"
	"kapital/backend/services"
)

// stubCheckUpdate answers the update check with answer and counts the asks.
func stubCheckUpdate(t *testing.T, answer models.UpdateInfo) *int {
	t.Helper()
	asked := 0
	previous := checkUpdate
	checkUpdate = func(_ context.Context, current string) models.UpdateInfo {
		asked++
		answer.Current = current
		return answer
	}
	t.Cleanup(func() { checkUpdate = previous })
	return &asked
}

// The site is asked once a start, however often the page asks.
func TestGetUpdateAsksTheSiteOncePerStart(t *testing.T) {
	app := newTestApp(t)
	asked := stubCheckUpdate(t, models.UpdateInfo{Latest: "v9.0.0", Available: true, Page: services.DownloadPage})
	for range 3 {
		got, err := app.GetUpdate()
		if err != nil {
			t.Fatal(err)
		}
		if !got.Available || got.Latest != "v9.0.0" || got.Current != Version {
			t.Fatalf("got %+v", got)
		}
	}
	if *asked != 1 {
		t.Fatalf("asked %d times", *asked)
	}
}

// Under the launcher update preview the answer is made up and nothing is asked;
// clearing it gives the real answer back.
func TestGetUpdateUnderThePreviewAsksNothing(t *testing.T) {
	app := newTestApp(t)
	asked := stubCheckUpdate(t, models.UpdateInfo{Reason: "no answer"})
	if _, err := app.StartPreview("", services.PreviewLauncherUpdate); err != nil {
		t.Fatal(err)
	}
	got, err := app.GetUpdate()
	if err != nil || !got.Available || got.Page != services.DownloadPage || *asked != 0 {
		t.Fatalf("got %+v, %v after %d asks", got, err, *asked)
	}
	if err := app.ClearPreviews(); err != nil {
		t.Fatal(err)
	}
	got, err = app.GetUpdate()
	if err != nil || got.Available || got.Reason != "no answer" || *asked != 1 {
		t.Fatalf("got %+v, %v after %d asks", got, err, *asked)
	}
}

// A build with no update key offers the site only, and says why.
func TestGetUpdateWithoutAKeyIsNotInstallable(t *testing.T) {
	saved := services.UpdatePublicKeys
	t.Cleanup(func() { services.UpdatePublicKeys = saved })
	services.UpdatePublicKeys = nil
	app := newTestApp(t)
	app.self.updater = services.NewLauncherUpdater()
	stubCheckUpdate(t, models.UpdateInfo{Latest: "v9.0.0", Available: true, Page: services.DownloadPage})
	got, err := app.GetUpdate()
	if err != nil || got.Installable || got.Blocked == "" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if err := app.DownloadUpdate(); err == nil {
		t.Fatal("DownloadUpdate went ahead without a key")
	}
}

// Under the preview the download is made up and ends ready; nothing is fetched.
func TestDownloadUpdateUnderThePreviewEndsReady(t *testing.T) {
	app := newTestApp(t)
	app.previewStep = 0
	var steps []models.LauncherUpdateProgress
	app.self.emit = func(p models.LauncherUpdateProgress) { steps = append(steps, p) }
	if _, err := app.StartPreview("", services.PreviewLauncherUpdate); err != nil {
		t.Fatal(err)
	}
	if err := app.DownloadUpdate(); err != nil {
		t.Fatal(err)
	}
	if len(steps) == 0 || steps[len(steps)-1].Phase != "ready" {
		t.Fatalf("steps = %+v", steps)
	}
}

// Cancel ends the made-up download, which then reports cancelled.
func TestCancelUpdateDownload(t *testing.T) {
	app := newTestApp(t)
	app.previewStep = time.Hour
	steps := make(chan models.LauncherUpdateProgress, 16)
	app.self.emit = func(p models.LauncherUpdateProgress) { steps <- p }
	if _, err := app.StartPreview("", services.PreviewLauncherUpdate); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- app.DownloadUpdate() }()
	<-steps // the first step is out, so the download is running
	if ok, err := app.CancelUpdateDownload(); !ok || err != nil {
		t.Fatalf("CancelUpdateDownload() = %v, %v", ok, err)
	}
	if err := <-done; !errors.Is(err, errInstallCancelled) {
		t.Fatalf("DownloadUpdate() = %v", err)
	}
	for range len(steps) {
		if p := <-steps; p.Phase == "failed" {
			t.Fatal("a cancel was reported as a failure")
		}
	}
}

// A restart waits for a download, which quitting would cut.
func TestRestartToUpdateWaitsForDownloads(t *testing.T) {
	app := newTestApp(t)
	_, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.terrain.cancel = cancel
	if err := app.RestartToUpdate(); err == nil {
		t.Fatal("RestartToUpdate went ahead during a download")
	}
	if app.self.relaunch != "" {
		t.Fatal("a relaunch was set")
	}
}

// Without a staged update there is nothing to restart into.
func TestRestartToUpdateWithNothingStaged(t *testing.T) {
	app := newTestApp(t)
	if err := app.RestartToUpdate(); err == nil {
		t.Fatal("RestartToUpdate went ahead with nothing staged")
	}
	app.relaunchAfterQuit() // nothing to start
}
