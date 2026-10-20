package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"kapital/backend/models"
)

// The full changelog under What's new: the download site's build writes
// changelog.json beside version.json, from the same releases feed its
// Changelog popup reads (site/changelog.js). The launcher reads it only when
// the player opens the list, from this constant address, sending nothing but
// the request, and keeps the releases What's new covers. Like the update
// check, not reaching it is a reason and never an error.
const (
	// changelogFeed is the site's changelog file.
	changelogFeed = DownloadPage + "changelog.json"
	// maxChangelogBody is well over ten releases of notes.
	maxChangelogBody = 256 << 10
	// maxNotesItem is in characters; a longer line is cut.
	maxNotesItem = 300
)

// notesSections are the headings the site keeps (site/changelog.js, KEPT).
var notesSections = []string{"Features", "Fixes", "Other changes"}

// notesDate is the shape of a release's date.
var notesDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// FetchChangelog reads the site's changelog and keeps the releases What's new
// covers between seen and current (Shown).
func FetchChangelog(ctx context.Context, seen, current string) models.Changelog {
	return fetchChangelog(ctx, newMapClient(updateCheckTimeout), changelogFeed, seen, current)
}

// fetchChangelog is FetchChangelog with the client and the address, so a test
// can point it at a local server.
func fetchChangelog(ctx context.Context, client *http.Client, feed, seen, current string) models.Changelog {
	out := models.Changelog{Page: DownloadPage}
	all, reason := readChangelog(ctx, client, feed)
	if reason != "" {
		out.Reason = reason
		slog.Info("changelog", "reason", reason)
		return out
	}
	for _, r := range all {
		if Shown(r.Tag, seen, current) {
			out.Releases = append(out.Releases, r)
		}
	}
	return out
}

// readChangelog fetches the changelog file and returns its releases, each held
// to the site's shape, or the reason there are none. No redirect is followed,
// only a 200 counts, and at most maxChangelogBody bytes are read.
func readChangelog(ctx context.Context, client *http.Client, feed string) ([]models.ReleaseNotes, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feed, nil)
	if err != nil {
		return nil, "no answer"
	}
	req.Header.Set("User-Agent", prismUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
			return nil, "timed out"
		}
		// The error's text is left out: it holds the address and nothing more useful.
		return nil, "no answer"
	}
	defer resp.Body.Close() //nolint:errcheck // read-only response
	if resp.StatusCode != http.StatusOK {
		return nil, "no answer"
	}
	var body struct {
		Releases []models.ReleaseNotes `json:"releases"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxChangelogBody)).Decode(&body); err != nil {
		return nil, "not a changelog"
	}
	var releases []models.ReleaseNotes
	for _, r := range body.Releases {
		if clean, ok := cleanNotes(r); ok {
			releases = append(releases, clean)
		}
	}
	return releases, ""
}

// cleanNotes keeps a release with a real tag and date, its known sections and
// their non-empty lines, each cut to maxNotesItem characters. A release that
// fails is dropped alone.
func cleanNotes(r models.ReleaseNotes) (models.ReleaseNotes, bool) {
	if _, ok := parseRelease(r.Tag); !ok || !strings.HasPrefix(r.Tag, "v") || !notesDate.MatchString(r.Date) {
		return models.ReleaseNotes{}, false
	}
	out := models.ReleaseNotes{Tag: r.Tag, Date: r.Date}
	for _, s := range r.Sections {
		if !slices.Contains(notesSections, s.Title) {
			continue
		}
		section := models.ReleaseNotesSection{Title: s.Title}
		for _, item := range s.Items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if utf8.RuneCountInString(item) > maxNotesItem {
				item = string([]rune(item)[:maxNotesItem-1]) + "…"
			}
			section.Items = append(section.Items, item)
		}
		if len(section.Items) > 0 {
			out.Sections = append(out.Sections, section)
		}
	}
	return out, true
}
