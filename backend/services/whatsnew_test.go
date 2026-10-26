package services

import (
	"os"
	"slices"
	"strings"
	"testing"

	"kapital/backend/models"
)

func releaseOf(version string) string {
	return `{"version":"` + version + `","highlights":[{"title":"A change","text":"What it is."}]}`
}

func versions(rs []models.WhatsNewRelease) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Version
	}
	return out
}

// The file in the repository is the one built into the app.
func TestBundledWhatsNewParses(t *testing.T) {
	data, err := os.ReadFile("../../data/whatsnew.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseWhatsNew(data); err != nil {
		t.Fatal(err)
	}
}

func TestParseWhatsNewSortsNewestFirst(t *testing.T) {
	data := `{"releases":[` + releaseOf("v1.0.0-beta.5") + `,` + releaseOf("v1.0.0") + `,` + releaseOf("v1.0.0-beta.10") + `]}`
	got, err := ParseWhatsNew([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"v1.0.0", "v1.0.0-beta.10", "v1.0.0-beta.5"}; !slices.Equal(versions(got), want) {
		t.Fatalf("got %v, want %v", versions(got), want)
	}
}

func TestParseWhatsNewRefusesTheFileWhole(t *testing.T) {
	long := strings.Repeat("x", maxHighlightText+1)
	cases := map[string]string{
		"unknown key":   `{"releases":[],"extra":1}`,
		"not a tag":     `{"releases":[` + releaseOf("1.0.0-dev") + `]}`,
		"twice":         `{"releases":[` + releaseOf("v1.0.0") + `,` + releaseOf("v1.0.0") + `]}`,
		"no highlights": `{"releases":[{"version":"v1.0.0","highlights":[]}]}`,
		"no title":      `{"releases":[{"version":"v1.0.0","highlights":[{"title":" ","text":"B"}]}]}`,
		"too long":      `{"releases":[{"version":"v1.0.0","highlights":[{"title":"A","text":"` + long + `"}]}]}`,
		"em dash":       `{"releases":[{"version":"v1.0.0","highlights":[{"title":"A","text":"This — that"}]}]}`,
		"highlight key": `{"releases":[{"version":"v1.0.0","highlights":[{"title":"A","text":"B","icon":"x"}]}]}`,
		"empty item":    `{"releases":[{"version":"v1.0.0","highlights":[{"title":"Fixes","items":["A fix",""]}]}]}`,
		"item em dash":  `{"releases":[{"version":"v1.0.0","highlights":[{"title":"Fixes","items":["This — that"]}]}]}`,
	}
	for name, data := range cases {
		if _, err := ParseWhatsNew([]byte(data)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// A group of small changes has a title and its items, and no text.
func TestParseWhatsNewTakesAGroupOfItems(t *testing.T) {
	data := `{"releases":[{"version":"v1.0.0","highlights":[{"title":"Bug fixes","items":["A fix","Another fix"]}]}]}`
	got, err := ParseWhatsNew([]byte(data))
	if err != nil || len(got[0].Highlights[0].Items) != 2 {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// A title alone names small changes without listing them.
func TestParseWhatsNewTakesATitleAlone(t *testing.T) {
	data := `{"releases":[{"version":"v1.0.0","highlights":[{"title":"UI and bug fixes"}]}]}`
	got, err := ParseWhatsNew([]byte(data))
	if err != nil || got[0].Highlights[0].Title != "UI and bug fixes" {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestWhatsNewSince(t *testing.T) {
	all := []models.WhatsNewRelease{{Version: "v1.1.0"}, {Version: "v1.0.0"}, {Version: "v1.0.0-beta.6"}, {Version: "v1.0.0-beta.5"}}
	cases := []struct {
		seen, current string
		want          []string
	}{
		{"v1.0.0-beta.5", "v1.0.0", []string{"v1.0.0", "v1.0.0-beta.6"}},
		{"v1.0.0-beta.4", "v1.0.0-beta.5", []string{"v1.0.0-beta.5"}},
		{"v1.0.0", "v1.0.0", nil},
		{"v1.1.0", "v1.0.0", nil},
		// From a launcher before What's new: this build's own only.
		{"", "v1.0.0", []string{"v1.0.0"}},
		{"", "v1.0.1", nil},
		{"v1.0.0", "1.0.0-dev", nil},
	}
	for _, c := range cases {
		if got := versions(WhatsNewSince(all, c.seen, c.current)); !slices.Equal(got, c.want) && len(got)+len(c.want) > 0 {
			t.Errorf("seen %q, running %q: got %v, want %v", c.seen, c.current, got, c.want)
		}
	}
}
