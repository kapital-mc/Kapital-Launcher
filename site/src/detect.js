// Picks the visitor's operating system, and does nothing else: no network, no
// storage. It sets data-os on the root, which the Download button and the
// link under it read (data-when-os), and checks the download popup's radio
// for it. A button that opens the popup for one system (data-os-pick) checks
// that system's radio first. Without this script, or on any other system,
// Windows is shown and checked, and the switch in the popup still works.
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

    const check = (os) => {
      const radio = document.getElementById(`os-${os}`)
      if (radio instanceof HTMLInputElement) radio.checked = true
    }

    if (!other && /mac/.test(platform)) {
      document.documentElement.dataset.os = 'macos'
      check('macos')
    }

    for (const button of document.querySelectorAll('[data-os-pick]')) {
      button.addEventListener('click', () => check(button.dataset.osPick))
    }
  },
  { once: true },
)
