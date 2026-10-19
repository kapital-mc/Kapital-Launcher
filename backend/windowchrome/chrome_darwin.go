//go:build darwin && cgo

package windowchrome

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include "chrome_darwin.h"
*/
import "C"

import (
	"math"

	"kapital/backend/models"
)

// Measure gives the launcher's window the unified toolbar style and reads
// where macOS put the traffic lights. AppKit is asked on the main thread, so
// it must not be called from there with that thread waiting on the caller. It
// reports false until Wails has made the window.
func Measure() (models.WindowChrome, bool) {
	var height, end C.double
	if C.kapitalWindowChrome(&height, &end) == 0 {
		return models.WindowChrome{}, false
	}
	return models.WindowChrome{
		TitlebarHeight: math.Round(float64(height)),
		ControlsEnd:    math.Round(float64(end)),
	}, true
}
