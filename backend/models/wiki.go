package models

// WikiPage is one page of the Kapitel Kapital wiki as its lore export lists
// it (#58): enough for the "From the wiki" panel to show a title and a line
// and to open the page, and for its expanded view to list the page's facts. The launcher never renders wiki content; it shows
// these two strings as text and hands the URL to the system browser.
type WikiPage struct {
	// Title is the page's name.
	Title string `json:"title"`
	// Line is the page's opening line, the export's excerpt.
	Line string `json:"line"`
	// URL is the page's absolute address on the wiki's host.
	URL string `json:"url"`
	// Eras names the chapters the page belongs to, by the wiki's era ids,
	// which are the chapter names ("Luxemburg", "Lichdenstein", "Frangfurd").
	Eras []string `json:"eras"`
	// ID is the page's id in the export ("locations/Bellum Castle.md"), which
	// a screenshot's Subject and Related name.
	ID string `json:"id"`
	// Related is the ids of the listed pages the wiki relates to this one, in
	// either direction (#141): the posts a picture of this page may come with.
	Related []string `json:"related"`
	// Type is the wiki's page type ("character", "location"), and Category
	// its sub-kind ("settlement"), "" when the page has none.
	Type     string `json:"type"`
	Category string `json:"category"`
	// Facts are the page's settled facts in the wiki's infobox order, as
	// plain text, at most eight; empty when it states none.
	Facts []WikiFact `json:"facts"`
}

// WikiFact is one row of a page's facts: "Founded", "2566".
type WikiFact struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// WikiShot is one of the wiki's pictures, a screenshot (#141) or a picture a
// page embeds (issue 172), downloaded and cached by the launcher, for the
// chapter art and the post that goes with it.
type WikiShot struct {
	// Era is the wiki's era id the picture belongs to, the chapter's name.
	Era string `json:"era"`
	// Src is where the page loads it: the launcher's own /wiki-art/ path,
	// served from the cache, never the wiki's address.
	Src string `json:"src"`
	// Subject is the id of the page the picture shows, "" when none.
	Subject string `json:"subject"`
}

// Panorama is a chapter's title-screen panorama (issue 195), read from the
// resources pack in the chapter's own instance and cached by the launcher.
type Panorama struct {
	// ChapterID is the manifest's chapter the six faces belong to.
	ChapterID string `json:"chapterId"`
	// Faces are the six pictures in the game's order (0 front, 1 right, 2 back,
	// 3 left, 4 top, 5 bottom), each the launcher's own /panorama/ path, served
	// from the cache.
	Faces [6]string `json:"faces"`
}

// WikiArtStats is what the settings screen needs to say how much room a number
// of wiki pictures per chapter takes (issue 172).
type WikiArtStats struct {
	// AvgBytes is the mean size of the pictures in the cache, a default of about
	// 100 KB while it holds none.
	AvgBytes int64 `json:"avgBytes"`
	// Pools is how many pictures each chapter can draw from, by the wiki's era
	// id (the chapter's name); empty when the wiki's export could not be read.
	Pools map[string]int `json:"pools"`
}
