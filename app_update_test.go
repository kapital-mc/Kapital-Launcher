package main

import (
	"context"
	"testing"

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
