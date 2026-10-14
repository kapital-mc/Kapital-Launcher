package models

// UpdateInfo is what the launcher knows about a newer build of itself after one
// look at the download site's version file (ADR-9 amendment). The popup that
// offers the download page shows only when Available is true.
type UpdateInfo struct {
	// Current is the version stamped into this build.
	Current string `json:"current"`
	// Latest is the release tag the download site offers, or "" when it was
	// not read.
	Latest string `json:"latest"`
	// Available is whether Latest is newer than Current.
	Available bool `json:"available"`
	// Page is the download site's address, the one the popup opens.
	Page string `json:"page"`
	// Reason is a short plain-text cause when nothing was compared: "dev
	// build", "no answer", "timed out" or "not a version". Empty when it was.
	Reason string `json:"reason"`
}
