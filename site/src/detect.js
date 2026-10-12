// Checks the download popup's radio for the visitor's operating system, and
// does nothing else: no network, no storage. Without this script, or on any
// other system, Windows stays checked and the switch still works.
//
// The build makes this a classic script in the head, after the stylesheet
// (stylesheetFirst in vite.config.js), so it runs before the popup exists and
// waits for the document. Nothing is declared at the top level: as a classic
// script, a top-level name would be a global.
document.addEventListener(
  'DOMContentLoaded',
  () => {
    const platform = (
      navigator.userAgentData?.platform ||
      navigator.platform ||
      navigator.userAgent ||
      ''
    ).toLowerCase()

    // iPhone, iPad and Android are neither, nor is Linux. iPadOS may report a
    // Mac platform, which touch points give away.
    const touchMac = /mac/.test(platform) && navigator.maxTouchPoints > 1
    const other = /iphone|ipad|ipod|android|linux|cros/.test(platform) || touchMac

    let id = null
    if (!other) {
      if (/win/.test(platform) && !/darwin/.test(platform)) id = 'os-windows'
      else if (/mac/.test(platform)) id = 'os-macos'
    }
    const radio = id && document.getElementById(id)
    if (radio instanceof HTMLInputElement) radio.checked = true
  },
  { once: true },
)
