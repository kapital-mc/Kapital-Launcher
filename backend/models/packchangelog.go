package models

// PackChangelogEntry is one release in a pack's own changelog.json (issue
// 164), shaped for the Changelog panel, whose own entries are a version and a
// summary: the file's lines are one to ten of plain text, short by the pack
// repo's rule ("Fixed Map Teleport"), and the panel shows them on one line
// joined with "; " and lists them in its expanded view.
type PackChangelogEntry struct {
	Version string `json:"version"`
	// Date is YYYY-MM-DD.
	Date string `json:"date"`
	// Summary is every line, joined with "; ".
	Summary string `json:"summary"`
	// Details is every line, one to a line.
	Details string `json:"details"`
}

// PackChangelog is what a chapter's pack source says changed, newest first.
type PackChangelog struct {
	ChapterID string `json:"chapterId"`
	// Checked is whether the source could be asked: true with no entries is a
	// pack that publishes no changelog; false is a file that could not be read
	// or did not validate, which the panel treats like no file.
	Checked bool                 `json:"checked"`
	Entries []PackChangelogEntry `json:"entries"`
}
