package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"kapital/backend/models"
)

// What's new: the first start after an update shows the highlights of every
// release since the one the player last saw. They are written by hand for
// each release in data/whatsnew.json, built into the app, and read here once
// at startup. The file is the app's own, but it is held to the same care as a
// manifest: an unknown key, a tag that is not one, or copy outside the limits
// refuses it whole.
const (
	// maxHighlights is how many changes one release may put forward.
	maxHighlights = 6
	// maxHighlightItems is how many small changes one group may list.
	maxHighlightItems = 10
	// maxHighlightTitle, maxHighlightText and maxHighlightItem are in characters.
	maxHighlightTitle = 48
	maxHighlightText  = 240
	maxHighlightItem  = 120
)

// whatsNewFile is the shape of data/whatsnew.json.
type whatsNewFile struct {
	Releases []models.WhatsNewRelease `json:"releases"`
}

// ParseWhatsNew reads the release highlights, newest first, or refuses the
// file with the first fault it finds.
func ParseWhatsNew(data []byte) ([]models.WhatsNewRelease, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var f whatsNewFile
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("whatsnew: %w", err)
	}
	type tagged struct {
		release models.WhatsNewRelease
		version releaseVersion
	}
	var all []tagged
	seen := map[string]bool{}
	for _, r := range f.Releases {
		v, ok := parseRelease(r.Version)
		if !ok {
			return nil, fmt.Errorf("whatsnew: %q is not a release tag", r.Version)
		}
		if seen[r.Version] {
			return nil, fmt.Errorf("whatsnew: %s is listed twice", r.Version)
		}
		seen[r.Version] = true
		if len(r.Highlights) == 0 || len(r.Highlights) > maxHighlights {
			return nil, fmt.Errorf("whatsnew: %s has %d highlights, 1 to %d allowed", r.Version, len(r.Highlights), maxHighlights)
		}
		for _, h := range r.Highlights {
			if err := checkHighlight(h); err != nil {
				return nil, fmt.Errorf("whatsnew: %s: %w", r.Version, err)
			}
		}
		all = append(all, tagged{r, v})
	}
	slices.SortFunc(all, func(a, b tagged) int {
		switch {
		case a.version.newerThan(b.version):
			return -1
		case b.version.newerThan(a.version):
			return 1
		}
		return 0
	})
	releases := make([]models.WhatsNewRelease, len(all))
	for i, t := range all {
		releases[i] = t.release
	}
	return releases, nil
}

// checkHighlight holds one highlight to the copy rules: a title, with a text,
// a list of items, both or neither, each within its length, and no em dash
// (agent_docs/CLAUDE.md). A title alone ("UI and bug fixes") names a group of
// small changes without listing them.
func checkHighlight(h models.WhatsNewHighlight) error {
	title, text := strings.TrimSpace(h.Title), strings.TrimSpace(h.Text)
	switch {
	case title == "":
		return errors.New("a highlight needs a title")
	case len(h.Items) > maxHighlightItems:
		return fmt.Errorf("%q has %d items, %d allowed", title, len(h.Items), maxHighlightItems)
	}
	if err := checkCopy(title, title, maxHighlightTitle); err != nil {
		return err
	}
	if text != "" {
		if err := checkCopy(title, text, maxHighlightText); err != nil {
			return err
		}
	}
	for _, item := range h.Items {
		if strings.TrimSpace(item) == "" {
			return fmt.Errorf("%q has an empty item", title)
		}
		if err := checkCopy(title, item, maxHighlightItem); err != nil {
			return err
		}
	}
	return nil
}

// checkCopy holds one piece of a highlight's copy to its length and refuses
// an em dash.
func checkCopy(title, line string, limit int) error {
	if utf8.RuneCountInString(line) > limit {
		return fmt.Errorf("%q: %q is over %d characters", title, line, limit)
	}
	if strings.ContainsRune(line, '—') {
		return fmt.Errorf("%q: %q has an em dash", title, line)
	}
	return nil
}

// WhatsNewSince picks the releases to show a player who last saw seen and now
// runs current, newest first (Shown).
func WhatsNewSince(all []models.WhatsNewRelease, seen, current string) []models.WhatsNewRelease {
	var out []models.WhatsNewRelease
	for _, r := range all {
		if Shown(r.Version, seen, current) {
			out = append(out, r)
		}
	}
	return out
}

// Shown says whether What's new covers the release tag for a player who last
// saw seen and now runs current: newer than seen and not newer than current. A
// seen that is empty or not a tag (a launcher from before What's new) covers
// only current itself. A current or a tag that is not one (a dev build)
// covers nothing.
func Shown(tag, seen, current string) bool {
	now, ok := parseRelease(current)
	v, isTag := parseRelease(tag)
	if !ok || !isTag || v.newerThan(now) {
		return false
	}
	if last, known := parseRelease(seen); known {
		return v.newerThan(last)
	}
	return !now.newerThan(v)
}

// IsReleaseTag says whether a version is a release's, as a build stamped at
// release time is and a dev build is not.
func IsReleaseTag(version string) bool {
	_, ok := parseRelease(version)
	return ok
}

// PreviewChangelog is the full changelog under the What's new preview: one
// made-up release, read from nowhere.
func PreviewChangelog() models.Changelog {
	return models.Changelog{Page: DownloadPage, Releases: []models.ReleaseNotes{{
		Tag:  previewLauncherLatest,
		Date: "2026-01-01",
		Sections: []models.ReleaseNotesSection{
			{Title: "Features", Items: []string{"A made-up feature (preview: no changelog was read)", "Another made-up feature"}},
			{Title: "Fixes", Items: []string{"A made-up fix"}},
		},
	}}}
}

// PreviewWhatsNew is what the launcher's What's new preview shows: the newest
// entry, or none when the file has none.
func PreviewWhatsNew(all []models.WhatsNewRelease) []models.WhatsNewRelease {
	if len(all) == 0 {
		return nil
	}
	return all[:1]
}
