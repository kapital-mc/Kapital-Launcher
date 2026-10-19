//go:build !darwin || !cgo

package windowchrome

import "kapital/backend/models"

// Measure has nothing to measure off macOS, where the header draws its own
// window buttons, nor in a build without cgo, which never ships for macOS.
func Measure() (models.WindowChrome, bool) {
	return models.WindowChrome{}, false
}
