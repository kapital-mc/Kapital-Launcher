package services

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"kapital/backend/models"
)

// A page's typed facts, for the expanded "From the wiki" panel: the export's
// `fields` are the page's frontmatter as authored (the wiki's src/lib/schema.mjs,
// read 2026-10-17), so a date can be a string or an object and a link keeps its
// [[brackets]]. Like the rest of the export they are untrusted text: only the
// keys below are read, under the launcher's own labels, and every value is
// reduced to one line of plain text or dropped.

// wikiFactFields is the schema's infobox fields in its own order, with the
// label the wiki prints. `facts` (free label and value pairs) is read after them.
var wikiFactFields = []struct{ key, label string }{
	{"full_name", "Full name"},
	{"born_name", "Born name"},
	{"kind", "Type"},
	{"also_known_as", "Also known as"},
	{"born", "Born"},
	{"died", "Died"},
	{"fate", "Status"},
	{"origin", "Origin"},
	{"affiliation", "Affiliation"},
	{"date", "Date"},
	{"founded", "Founded"},
	{"built", "Built"},
	{"opened", "Opened"},
	{"formed", "Formed"},
	{"emerged", "Emerged"},
	{"unified", "Unified"},
	{"collapsed", "Collapsed"},
	{"state", "Status"},
	{"hq", "HQ"},
	{"location", "Location"},
	{"period", "Period"},
	{"source", "Source"},
}

const (
	// maxWikiFacts is how many rows the expanded panel lists.
	maxWikiFacts     = 8
	maxWikiFactLabel = 40
	maxWikiFactValue = 200
)

// wikiLink is a [[Page]], [[Page|Shown]] or [[Page#Section]] link.
var wikiLink = regexp.MustCompile(`\[\[([^\]|#]*)(?:#[^\]|]*)?(?:\|([^\]]*))?\]\]`)

// wikiFacts reads a page's facts from its export entry: Within (its region),
// then the schema's fields in order, then its own `facts` rows, at most
// maxWikiFacts. An undecided or unknown value is left out, not shown.
func wikiFacts(fields map[string]json.RawMessage, region string) []models.WikiFact {
	facts := []models.WikiFact{}
	add := func(label, value string) {
		if v, ok := factText(value); ok && len(facts) < maxWikiFacts {
			facts = append(facts, models.WikiFact{Label: label, Value: v})
		}
	}
	add("Within", region)
	for _, f := range wikiFactFields {
		if raw, ok := fields[f.key]; ok {
			add(f.label, factValue(raw))
		}
	}
	var rows []struct {
		Label string `json:"label"`
		Value string `json:"value"`
	}
	if raw, ok := fields["facts"]; ok && json.Unmarshal(raw, &rows) == nil {
		for _, r := range rows {
			if label, ok := factText(r.Label); ok && utf8.RuneCountInString(label) <= maxWikiFactLabel {
				add(label, r.Value)
			}
		}
	}
	return facts
}

// factValue flattens one field: a string as it is, a list of strings joined,
// and a date object as its date and place ("1362, Bellum Castle tunnels").
// Anything else is "".
func factValue(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return strings.Join(list, ", ")
	}
	var event struct {
		Date  string `json:"date"`
		Place string `json:"place"`
	}
	if json.Unmarshal(raw, &event) == nil {
		parts := []string{}
		for _, p := range []string{event.Date, event.Place} {
			if p = strings.TrimSpace(p); p != "" && !undecided(p) {
				parts = append(parts, p)
			}
		}
		return strings.Join(parts, ", ")
	}
	return ""
}

// factText is value as one line of plain text, links reduced to their words,
// and whether it is worth a row: not empty, not undecided or unknown, plain
// text only, and no longer than maxWikiFactValue.
func factText(value string) (string, bool) {
	v := wikiLink.ReplaceAllStringFunc(value, func(link string) string {
		m := wikiLink.FindStringSubmatch(link)
		if strings.TrimSpace(m[2]) != "" {
			return m[2]
		}
		return m[1]
	})
	v = strings.TrimSpace(v)
	if v == "" || undecided(v) || strings.IndexFunc(v, notPlainText) >= 0 {
		return "", false
	}
	return v, utf8.RuneCountInString(v) <= maxWikiFactValue
}

// undecided is the schema's two non-answers: `Unknown`, and a value that still
// carries a `[PLACEHOLDER]` anywhere in it.
func undecided(v string) bool {
	return v == "Unknown" || strings.Contains(v, "[PLACEHOLDER")
}
