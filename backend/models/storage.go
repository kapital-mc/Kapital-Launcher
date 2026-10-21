package models

// The kinds of storage a chapter's instance is measured by, for the storage
// section of its settings. Terrain, the unfinished terrain download, the world
// map, the screenshots and the logs can be deleted; the minimap (waypoints), the
// mods and the rest are only shown. Every kind's folder can be opened.
const (
	StorageTerrain         = "terrain"
	StorageTerrainDownload = "terrainDownload"
	StorageWorldMap        = "worldMap"
	StorageMinimap         = "minimap"
	StorageScreenshots     = "screenshots"
	StorageLogs            = "logs"
	StorageMods            = "mods"
	StorageOther           = "other"
)

// ChapterStorage is what a chapter's instance holds on disk, sorted by what the
// player can do about it.
type ChapterStorage struct {
	ChapterID string `json:"chapterId"`
	// TotalBytes is the whole instance folder.
	TotalBytes int64 `json:"totalBytes"`
	// FreeBytes is the space left on the instance's disk, -1 when it could not
	// be read.
	FreeBytes int64 `json:"freeBytes"`
	// Items are the instance's storage by kind, largest terrain first, then the
	// other kinds in a fixed order. A kind with nothing in it is left out.
	Items []StorageItem `json:"items"`
	// Partial is true when the walk stopped at its ceiling of entries, so the
	// sizes are lower than the disk's.
	Partial bool `json:"partial"`
	// Running is whether the game looks to be running; a delete is refused
	// while it is.
	Running bool `json:"running"`
}

// StorageItem is one kind of storage in an instance: for terrain, one server's
// Distant Horizons folder.
type StorageItem struct {
	// Kind is one of the Storage constants.
	Kind string `json:"kind"`
	// Name is the server's folder under Distant_Horizons_server_data for
	// terrain, what ClearChapterStorage takes back, and "" for every other kind.
	Name string `json:"name"`
	// Label is the server's name as Distant Horizons wrote it in the folder
	// name, decoded, for terrain; "" for every other kind.
	Label string `json:"label"`
	Bytes int64  `json:"bytes"`
	Files int    `json:"files"`
	// LastModified is the newest file's time, RFC 3339 in UTC, or "".
	LastModified string `json:"lastModified"`
}
