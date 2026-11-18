package models

// PlayTime is one installed chapter's time played and when it was last started.
type PlayTime struct {
	ChapterID string `json:"chapterId"`
	// TotalSeconds is the time in a world with AFK left out, counted from the
	// game's logs (issue 341); 0 when no log shows any. As ReadPlayTime returns
	// it, before GetPlayTime counts, it is Prism's totalTimePlayed.
	TotalSeconds int64 `json:"totalSeconds"`
	// LastLaunchMs is Prism's lastLaunchTime, milliseconds since the epoch;
	// 0 when the game was never started.
	LastLaunchMs int64 `json:"lastLaunchMs"`
}
