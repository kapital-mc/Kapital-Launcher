//go:build !darwin || !cgo

package services

// activateGame is macOS's (gameforeground_darwin.go). Windows gives the game
// the foreground through its window hold, and a macOS build without cgo has no
// Objective-C to call, so here it does nothing.
func activateGame(int) bool { return false }
