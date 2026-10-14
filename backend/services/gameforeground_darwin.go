//go:build darwin && cgo

package services

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit
#include "gameforeground_darwin.h"
*/
import "C"

// activateGame asks macOS to bring the game's process to the front
// (gameforeground_darwin.m), and reports whether it accepted. It is the
// handover's foreground on macOS, where there is no window hold (#258).
func activateGame(pid int) bool { return C.kapitalActivateProcess(C.int(pid)) != 0 }
