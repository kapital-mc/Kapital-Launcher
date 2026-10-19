package models

// TailscaleInfo is what the launcher knows of the player's Tailscale, from its
// own command line (services.Tailscale). Nothing about the account or the
// tailnet is kept: only whether Tailscale is there, and which chapters' servers
// it reaches.
type TailscaleInfo struct {
	// State is "missing" (not installed), "stopped" (installed, not running
	// or switched off), "needsLogin" (running, not signed in) or "running".
	State string `json:"state"`
	// Version is the installed Tailscale's version, "" when unknown.
	Version string `json:"version"`
	// CanInstall is whether the launcher can get Tailscale on this OS; where
	// it cannot, Get Tailscale opens Tailscale's download page.
	CanInstall bool `json:"canInstall"`
	// SigningIn is whether a sign-in started from the launcher is waiting on
	// the browser.
	SigningIn bool `json:"signingIn"`
	// Connected is the chapters whose Tailscale address this machine reaches:
	// Tailscale is running and the player accepted the server's share link.
	// Empty, never null, while it is not running.
	Connected []string `json:"connected"`
}

// TailscaleRelease is the Tailscale installer the launcher would download, as
// read from Tailscale's own package server. The popup shows it before
// anything is fetched.
type TailscaleRelease struct {
	// Version is the stable release, e.g. "1.104.1".
	Version string `json:"version"`
	// File is the installer's file name: an .msi on Windows, a .pkg on macOS.
	File string `json:"file"`
	// URL is where it is downloaded from, on pkgs.tailscale.com.
	URL string `json:"url"`
	// Size is its size in bytes.
	Size int64 `json:"size"`
	// Digest is the SHA-256 Tailscale publishes beside it, "sha256:<hex>".
	Digest string `json:"digest"`
}

// TailscaleInstallProgress is the payload of the tailscale:install event, one
// per step of an install.
type TailscaleInstallProgress struct {
	// Phase is "downloading", "verifying", "installing" (the installer runs),
	// "done" or "failed".
	Phase string `json:"phase"`
	// Received and Total are bytes of the download so far and in all.
	Received int64 `json:"received"`
	Total    int64 `json:"total"`
	// Error says what failed when Phase is "failed".
	Error string `json:"error"`
}
