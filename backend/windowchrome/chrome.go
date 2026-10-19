// Package windowchrome is where macOS draws its title bar in the launcher's
// window, measured from AppKit rather than assumed (docs/adr/0010-window-chrome.md,
// 2026-10-19 amendment). The header bar is drawn by the page, so it takes its
// height and its left inset from Measure.
package windowchrome
