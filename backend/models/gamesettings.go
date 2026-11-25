package models

// ChapterGameSettings is the game settings a chapter's instance can reset one
// at a time, for the Instance section of its settings.
type ChapterGameSettings struct {
	ChapterID string `json:"chapterId"`
	// Items are the settings in the page's order; one the instance has no
	// trace of is left out.
	Items []GameSettingsItem `json:"items"`
	// Running is whether the game looks to be running; a reset is refused
	// while it is.
	Running bool `json:"running"`
}

// GameSettingsItem is one resettable setting.
type GameSettingsItem struct {
	// ID is what ResetGameSetting takes back.
	ID   string `json:"id"`
	Name string `json:"name"`
	// Files are the item's files on disk now, relative to the game folder,
	// slash-separated. None after a reset, until the next Play or the game
	// writes them again.
	Files []string `json:"files"`
	// Keys is options.txt's key binding lines, for the key bindings item.
	Keys int `json:"keys"`
}
