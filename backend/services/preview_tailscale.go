package services

import (
	"strings"

	"kapital/backend/models"
)

// The launcher's Tailscale situations: the settings row's three faces
// (Install, Connect, the switch), the install popup on its offer, and the
// install popup failing. Like the rest
// of preview.go, everything here is state and copy: no CLI runs, nothing is
// downloaded and no page opens.
const (
	PreviewTailscaleMissing      = "tailscale-missing"
	PreviewTailscaleOffer        = "tailscale-offer"
	PreviewTailscaleInstall      = "tailscale-install"
	PreviewTailscaleDisconnected = "tailscale-disconnected"
	PreviewTailscaleConnected    = "tailscale-connected"
)

// previewTailscalePage is where the Tailscale situations' screen is: a
// chapter's settings page, where its Tailscale row is.
const previewTailscalePage = "settings"

const (
	previewTailscaleVersion = "9.9.9"
	previewTailscaleSize    = 30 << 20
)

// PreviewTailscale is the synthetic Tailscale of a situation, or false for one
// that is not Tailscale's. Installed and connected reaches every chapter with a
// Tailscale address; installed and not connected reaches none.
func PreviewTailscale(situation string, chapters []models.Chapter) (models.TailscaleInfo, bool) {
	switch situation {
	case PreviewTailscaleMissing, PreviewTailscaleOffer, PreviewTailscaleInstall:
		return models.TailscaleInfo{State: TailscaleMissing, CanInstall: true, Connected: []string{}}, true
	case PreviewTailscaleDisconnected:
		return models.TailscaleInfo{State: TailscaleRunning, Version: previewTailscaleVersion, CanInstall: true, Connected: []string{}}, true
	case PreviewTailscaleConnected:
		connected := []string{}
		for _, c := range chapters {
			if _, ok := TailscaleAddress(c); ok {
				connected = append(connected, c.ID)
			}
		}
		return models.TailscaleInfo{State: TailscaleRunning, Version: previewTailscaleVersion, CanInstall: true, Connected: connected}, true
	}
	return models.TailscaleInfo{}, false
}

// PreviewTailscaleRelease is the made-up installer the popup shows under a
// Tailscale situation. Its address is never fetched.
func PreviewTailscaleRelease(situation string) (models.TailscaleRelease, bool) {
	if _, ok := PreviewTailscale(situation, nil); !ok {
		return models.TailscaleRelease{}, false
	}
	file := "tailscale-setup-" + previewTailscaleVersion + "-preview.msi"
	return models.TailscaleRelease{
		Version: previewTailscaleVersion,
		File:    file,
		URL:     tailscalePackages + file,
		Size:    previewTailscaleSize,
		Digest:  "sha256:" + strings.Repeat("0", 64),
	}, true
}

// previewTailscaleError is what the made-up Tailscale install fails with.
const previewTailscaleError = "The download failed (preview: nothing was downloaded)"

// PreviewTailscaleInstallProgress is the tailscale:install steps a preview
// plays: a download that fills, the check of it, and the failure.
func PreviewTailscaleInstallProgress() []models.TailscaleInstallProgress {
	const total = previewTailscaleSize
	return []models.TailscaleInstallProgress{
		{Phase: "downloading", Received: 0, Total: total},
		{Phase: "downloading", Received: total * 40 / 100, Total: total},
		{Phase: "downloading", Received: total * 80 / 100, Total: total},
		{Phase: "downloading", Received: total, Total: total},
		{Phase: "verifying", Received: total, Total: total},
		{Phase: "failed", Error: previewTailscaleError},
	}
}
