package services

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
)

const servedChangelog = `{"releases":[
  {"tag":"v1.0.0","date":"2026-11-01","sections":[{"title":"Features","items":["A feature"]},{"title":"Maintenance","items":["Dropped"]}]},
  {"tag":"v1.0.0-beta.6","date":"2026-10-25","sections":[{"title":"Fixes","items":["A fix"," "]}]},
  {"tag":"v1.0.0-beta.5","date":"2026-10-20","sections":[]},
  {"tag":"not a tag","date":"2026-10-20","sections":[]},
  {"tag":"v0.9.0","date":"20 October","sections":[]}
]}`

// The changelog keeps the releases What's new covers, their known sections and
// lines, and drops what is not the site's shape.
func TestFetchChangelogKeepsTheReleasesSinceTheSeenOne(t *testing.T) {
	url, hits := feed(t, http.StatusOK, servedChangelog)
	got := fetchChangelog(context.Background(), newMapClient(time.Second), url, "v1.0.0-beta.5", "v1.0.0")
	tags := make([]string, len(got.Releases))
	for i, r := range got.Releases {
		tags[i] = r.Tag
	}
	if got.Reason != "" || !slices.Equal(tags, []string{"v1.0.0", "v1.0.0-beta.6"}) || hits.Load() != 1 {
		t.Fatalf("got %+v", got)
	}
	if s := got.Releases[0].Sections; len(s) != 1 || s[0].Title != "Features" {
		t.Errorf("sections %+v", s)
	}
	if s := got.Releases[1].Sections; len(s) != 1 || !slices.Equal(s[0].Items, []string{"A fix"}) {
		t.Errorf("sections %+v", s)
	}
	if got.Page != DownloadPage {
		t.Errorf("page %q", got.Page)
	}
}

func TestFetchChangelogFailuresAreReasons(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		reason string
	}{
		"missing":   {http.StatusNotFound, "", "no answer"},
		"not json":  {http.StatusOK, "<html>", "not a changelog"},
		"too large": {http.StatusOK, `{"releases":[` + strings.Repeat(" ", maxChangelogBody) + `]}`, "not a changelog"},
	}
	for name, tc := range cases {
		url, _ := feed(t, tc.status, tc.body)
		if got := fetchChangelog(context.Background(), newMapClient(time.Second), url, "", "v1.0.0"); got.Reason != tc.reason {
			t.Errorf("%s: got %+v", name, got)
		}
	}
}

// A line longer than the limit is cut, not dropped.
func TestChangelogCutsALongLine(t *testing.T) {
	url, _ := feed(t, http.StatusOK, `{"releases":[{"tag":"v1.0.0","date":"2026-11-01","sections":[{"title":"Fixes","items":["`+strings.Repeat("x", 400)+`"]}]}]}`)
	got := fetchChangelog(context.Background(), newMapClient(time.Second), url, "", "v1.0.0")
	if len(got.Releases) != 1 || len([]rune(got.Releases[0].Sections[0].Items[0])) != maxNotesItem {
		t.Fatalf("got %+v", got)
	}
}
