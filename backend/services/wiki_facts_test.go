package services

import (
	"encoding/json"
	"strings"
	"testing"

	"kapital/backend/models"
)

func TestWikiFactsReadsTheSchemaFieldsAsPlainText(t *testing.T) {
	var fields map[string]json.RawMessage
	raw := `{
		"kind": "Mining company",
		"founded": "2566",
		"hq": "[[The Citadel]], [[Frangfurd|the city]]",
		"died": {"date": "1362", "place": "[[Bellum Castle]] tunnels", "cause": "stabbed"},
		"born": {"date": "[PLACEHOLDER]", "place": "[[Old Darkwood]]"},
		"also_known_as": ["NPW", "The Power"],
		"state": "Unknown",
		"period": "[PLACEHOLDER: after 2623]",
		"relations": [{"page": "[[Rajesh Masala]]", "kind": "founded-by"}],
		"coords": {"Frangfurd": [1, 2]},
		"facts": [{"label": "Abbreviation", "value": "NPW"}, {"label": "Bad\nlabel", "value": "x"}]
	}`
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		t.Fatal(err)
	}
	got := wikiFacts(fields, "[[Warzow Peninsula]]")
	want := []models.WikiFact{
		{Label: "Within", Value: "Warzow Peninsula"},
		{Label: "Type", Value: "Mining company"},
		{Label: "Also known as", Value: "NPW, The Power"},
		{Label: "Born", Value: "Old Darkwood"},
		{Label: "Died", Value: "1362, Bellum Castle tunnels"},
		{Label: "Founded", Value: "2566"},
		{Label: "HQ", Value: "The Citadel, the city"},
		{Label: "Abbreviation", Value: "NPW"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestWikiFactsCapsAndRefuses(t *testing.T) {
	fields := map[string]json.RawMessage{}
	for _, f := range wikiFactFields {
		fields[f.key] = json.RawMessage(`"x"`)
	}
	if got := wikiFacts(fields, ""); len(got) != maxWikiFacts {
		t.Fatalf("%d rows, want the cap of %d", len(got), maxWikiFacts)
	}
	long := strings.Repeat("a", maxWikiFactValue+1)
	for _, v := range []string{"", "  ", "Unknown", "[PLACEHOLDER]", "a‮b", long} {
		if got, ok := factText(v); ok {
			t.Errorf("factText(%q) = %q, want it dropped", v, got)
		}
	}
	if got := wikiFacts(nil, ""); got == nil || len(got) != 0 {
		t.Fatalf("no fields is an empty list, not nil: %#v", got)
	}
}

func TestParseWikiExportCarriesTypeCategoryAndFacts(t *testing.T) {
	raw := `{"pages":[{"name":"N","url":"/wiki/factions/n","type":"faction","category":"economic","era":["Frangfurd"],"status":"draft","excerpt":"A company.","fields":{"founded":"2566"}},
		{"name":"M","url":"/wiki/factions/m","type":"Faction <b>","category":"x y","era":["Frangfurd"],"status":"draft","excerpt":"Another."}]}`
	pages := parseWikiExport([]byte(raw), "https://w")
	if len(pages) != 2 {
		t.Fatalf("%+v", pages)
	}
	if p := pages[0]; p.Type != "faction" || p.Category != "economic" || len(p.Facts) != 1 || p.Facts[0].Value != "2566" {
		t.Fatalf("%+v", p)
	}
	if p := pages[1]; p.Type != "" || p.Category != "" || p.Facts == nil {
		t.Fatalf("a type that is not a taxonomy word is dropped, facts stay a list: %+v", p)
	}
}
