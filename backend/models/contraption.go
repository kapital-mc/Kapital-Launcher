package models

// The kinds of saved contraption a chapter's game folder holds: the Tool Gun's
// blueprints (create_aeronautics_toolgun, .excraft) and Create's schematics
// (.nbt), which may carry Sable sub-levels.
const (
	ContraptionToolgun   = "toolgun"
	ContraptionSchematic = "schematic"
)

// Contraption is one saved contraption file and the facts read from it.
type Contraption struct {
	Kind string `json:"kind"`
	// File is the file's name in its kind's folder, the one handle the page
	// passes back.
	File string `json:"file"`
	// Name is the file name without its extension, which is what the Tool Gun
	// and Create list it as.
	Name     string `json:"name"`
	Bytes    int64  `json:"bytes"`
	Modified int64  `json:"modified"` // Unix milliseconds
	// Blocks, BlockTypes and SubLevels are -1 when the file does not say.
	Blocks     int64 `json:"blocks"`
	BlockTypes int   `json:"blockTypes"`
	SubLevels  int   `json:"subLevels"`
	// Size is a schematic's bounding box (x, y, z); a Tool Gun file has none.
	Size []int `json:"size,omitempty"`
	// Materials are the blocks it is built from, most first; Items are the
	// items its containers hold, from the Tool Gun's own summary.
	Materials []MaterialCount `json:"materials"`
	Items     []MaterialCount `json:"items"`
	// Unread is set when the facts could not be read; the file is still listed.
	Unread bool `json:"unread"`
}

// MaterialCount is one block or item id and how many.
type MaterialCount struct {
	ID    string `json:"id"`
	Count int64  `json:"count"`
}
