package models

// WindowChrome is where macOS drew its title bar in the launcher's window, in
// points (a CSS pixel in the WebView). The zero value means it was not
// measured: another OS, or no window yet; the header then keeps its token
// sizes (ADR-10, 2026-10-19 amendment).
type WindowChrome struct {
	// TitlebarHeight is the bar macOS centres the traffic lights in.
	TitlebarHeight float64 `json:"titlebarHeight"`
	// ControlsEnd is the right edge of the zoom button, from the window's left.
	ControlsEnd float64 `json:"controlsEnd"`
}
