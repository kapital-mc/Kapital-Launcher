package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"kapital/backend/services"
)

func TestInstallCancelEndsOnlyADownload(t *testing.T) {
	var c installCancel
	if c.stop() {
		t.Fatal("cancelled with no install running")
	}
	ctx, end := c.begin(context.Background())
	if c.stop() {
		t.Fatal("cancelled before the download began")
	}
	c.step("downloading")
	if !c.stop() || ctx.Err() == nil {
		t.Fatal("a running download was not cancelled")
	}
	if c.step("failed") {
		t.Fatal("the failure a cancel causes was passed on")
	}
	if !end() {
		t.Fatal("end does not report the cancel")
	}

	ctx, end = c.begin(context.Background())
	c.step("downloading")
	c.step("verifying")
	if c.stop() || ctx.Err() != nil {
		t.Fatal("a cancel reached the step after the download")
	}
	if !c.step("failed") || end() {
		t.Fatal("a real failure is news and no cancel")
	}
}

func TestInstallCancelLeavesASecondInstallAlone(t *testing.T) {
	var c installCancel
	_, end := c.begin(context.Background())
	other, otherEnd := c.begin(context.Background())
	if otherEnd() {
		t.Fatal("the refused second install reports a cancel")
	}
	c.step("downloading")
	if !c.stop() || other.Err() != nil {
		t.Fatal("the cancel is the first install's")
	}
	if !end() {
		t.Fatal("the first install was not cancelled")
	}
}

func TestCancelEndsAMadeUpPrismDownload(t *testing.T) {
	app, _, installs := previewApp(t)
	app.previewStep = time.Hour
	mustStart(t, app, "", services.PreviewPrismInstall)
	done := make(chan error, 1)
	go func() { done <- app.InstallPrism() }()
	waitFor(t, "the first step", func() bool { return len(installs.all()) == 1 })
	if ok, err := app.CancelPrismInstall(); err != nil || !ok {
		t.Fatalf("%v %v", ok, err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, errInstallCancelled) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the install went on after Cancel")
	}
	steps := installs.all()
	if last := steps[len(steps)-1]; last.Phase != "cancelled" {
		t.Fatalf("%+v", steps)
	}
	if ok, _ := app.CancelPrismInstall(); ok {
		t.Fatal("cancelled an install that had ended")
	}
}
