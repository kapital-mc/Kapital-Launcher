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
	// Installable is whether this copy can update itself in place (ADR-6
	// amendment, 2026-10-21): a release build on Windows or macOS, in a folder
	// the player can write, with an update key built in.
	Installable bool `json:"installable"`
	// Blocked is why it cannot, in a sentence for the player, when an update is
	// available and Installable is false. The popup then offers the site only.
	Blocked string `json:"blocked"`
}

// LauncherUpdateProgress is one step of the launcher's own update, sent as a
// launcher:update event while the new build downloads in the background.
type LauncherUpdateProgress struct {
	// Phase is "downloading", "verifying", "ready" (staged, waiting for
	// Restart), "failed" or "cancelled".
	Phase    string `json:"phase"`
	Received int64  `json:"received"`
	Total    int64  `json:"total"`
	// Version is the release being downloaded.
	Version string `json:"version"`
	Error   string `json:"error"`
}

// WhatsNewHighlight is one change a release puts in front of the player,
// written by hand for the release (data/whatsnew.json), not taken from the
// pull requests: a short name with a line about it, or with a list of small
// changes it groups ("Bug fixes", "UI improvements"), or both.
type WhatsNewHighlight struct {
	Title string   `json:"title"`
	Text  string   `json:"text,omitempty"`
	Items []string `json:"items,omitempty"`
}

// WhatsNewRelease is a release's highlights, under its tag.
type WhatsNewRelease struct {
	Version    string              `json:"version"`
	Highlights []WhatsNewHighlight `json:"highlights"`
}

// WhatsNew is what the first start after an update shows: this build's
// version and the highlights of every release since the one the player last
// saw, newest first. No releases means nothing to show.
type WhatsNew struct {
	Version  string            `json:"version"`
	Releases []WhatsNewRelease `json:"releases"`
}

// ReleaseNotes is one release's entry in the full changelog the download site
// publishes (changelog.json): its tag, its date and its sections.
type ReleaseNotes struct {
	Tag      string                `json:"tag"`
	Date     string                `json:"date"`
	Sections []ReleaseNotesSection `json:"sections"`
}

// ReleaseNotesSection is a heading of the notes ("Features", "Fixes", "Other
// changes") and its lines, plain text.
type ReleaseNotesSection struct {
	Title string   `json:"title"`
	Items []string `json:"items"`
}

// Changelog is the full changelog What's new folds away: the releases it
// covers, newest first, or the reason there are none ("no answer", "timed
// out", "not a changelog"), with the download site, whose page has every
// release.
type Changelog struct {
	Releases []ReleaseNotes `json:"releases"`
	Reason   string         `json:"reason"`
	Page     string         `json:"page"`
}
