package services

import (
	"slices"
	"strings"
	"testing"

	"kapital/backend/models"
)

func TestPreviewTailscaleSituations(t *testing.T) {
	chapters := []models.Chapter{withTailscale(), {ID: "luxemburg"}}
	for _, id := range []string{PreviewTailscaleMissing, PreviewTailscaleInstall} {
		info, ok := PreviewTailscale(id, chapters)
		if !ok || info.State != TailscaleMissing || !info.CanInstall || info.Connected == nil {
			t.Errorf("%s: %+v", id, info)
		}
	}
	off, ok := PreviewTailscale(PreviewTailscaleDisconnected, chapters)
	if !ok || off.State != TailscaleRunning || off.Connected == nil || len(off.Connected) != 0 {
		t.Errorf("not connected: %+v", off)
	}
	on, ok := PreviewTailscale(PreviewTailscaleConnected, chapters)
	if !ok || !slices.Equal(on.Connected, []string{"frangfurd"}) {
		t.Errorf("connected: %+v", on)
	}
	for _, id := range []string{PreviewLauncherUpdate, PreviewPrismMissing, "nope", ""} {
		if _, ok := PreviewTailscale(id, chapters); ok {
			t.Errorf("%q is not Tailscale's", id)
		}
		if _, ok := PreviewTailscaleRelease(id); ok {
			t.Errorf("%q has a Tailscale release", id)
		}
	}
	rel, ok := PreviewTailscaleRelease(PreviewTailscaleInstall)
	if !ok || !strings.HasPrefix(rel.URL, tailscalePackages) || !strings.Contains(rel.File, "preview") {
		t.Errorf("release: %+v", rel)
	}
	steps := PreviewTailscaleInstallProgress()
	if last := steps[len(steps)-1]; last.Phase != "failed" || !strings.Contains(last.Error, "preview") {
		t.Errorf("the install ends in %+v", last)
	}
}
