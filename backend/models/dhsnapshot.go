package models

// DHSnapshotInfo is what a chapter's settings page shows about the server's
// Distant Horizons terrain (issue 136): the published snapshot, what this
// instance already has, and whether a download can start. Go reads the snapshot
// from the address the manifest names (Pack.DHSnapshot); nothing here names a
// path the caller may send back.
type DHSnapshotInfo struct {
	ChapterID string `json:"chapterId"`
	// Available is whether a download can start now. When false, Reason says why
	// in a sentence the page shows as it is.
	Available bool   `json:"available"`
	Reason    string `json:"reason"`
	// CreatedAt is when the server's snapshot was taken, RFC 3339 in UTC, or ""
	// when it could not be read.
	CreatedAt string `json:"createdAt"`
	// SizeBytes is the whole snapshot, every dimension's database together.
	SizeBytes int64 `json:"sizeBytes"`
	// Dimensions is how many databases the snapshot holds.
	Dimensions int `json:"dimensions"`
	// ServerDHVersion is the server's Distant Horizons; ClientDHVersion is the
	// one in this instance's mods folder, "" when none was found.
	ServerDHVersion string `json:"serverDhVersion"`
	ClientDHVersion string `json:"clientDhVersion"`
	// ExistingBytes is what the instance already holds for this server's
	// dimensions: terrain the player gathered in game, or an earlier download.
	// A download replaces it, so the page asks first.
	ExistingBytes int64 `json:"existingBytes"`
	// PartialBytes is what an earlier download of this same snapshot left
	// verified, which the next one continues from.
	PartialBytes int64 `json:"partialBytes"`
	// Running is whether the game looks to be running; a download is refused
	// while it is, because Distant Horizons holds its database open.
	Running bool `json:"running"`
	// Downloading is whether a download of this chapter's snapshot is under way.
	Downloading bool `json:"downloading"`
}

// DHDownloadProgress is one dh:download event (issue 136). Phase is
// "checking", "downloading", "verifying", "placing", "done", "cancelled" or
// "failed"; Received and Total are bytes of the whole snapshot, Received
// counting what an earlier, interrupted download already left verified.
type DHDownloadProgress struct {
	ChapterID string `json:"chapterId"`
	Phase     string `json:"phase"`
	Received  int64  `json:"received"`
	Total     int64  `json:"total"`
	Error     string `json:"error,omitempty"`
}
