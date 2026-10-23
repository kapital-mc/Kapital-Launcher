// Opens the next chapter in the strip every --duration-cycle, so the page
// shows all three without a click. Choosing a chapter starts the wait again
// from that moment. It waits while the pointer or the keyboard focus is on
// the window, while a popup is open and while the tab is hidden, and never
// runs for a visitor who asks for reduced motion. Without the script the strip
// stays on the chapter it opened with and still switches on a click.
//
// The download popup's banner cycles too, while the popup is open: it starts
// at the open chapter and shows the next every --duration-banner, through
// data-banner on its frame (src/showcase.css sweeps the change in from the
// right). Under reduced motion it stays on the open chapter.
//
// Like detect.js it runs as part of one classic script, so nothing is
// declared at the top level.
document.addEventListener(
  'DOMContentLoaded',
  () => {
    const radios = [...document.querySelectorAll('input[name="chapter"]')]
    const strip = document.querySelector('.showcase-strip')
    // The browser may give a token back in seconds ("5s") or as written.
    const duration = (name) => {
      const token = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
      return parseFloat(token) * (token.endsWith('ms') ? 1 : token.endsWith('s') ? 1000 : NaN)
    }
    const wait = duration('--duration-cycle')
    if (radios.length < 2 || !strip || !(wait > 0)) return
    if (matchMedia('(prefers-reduced-motion: reduce)').matches) return

    const chapters = radios.map((radio) => radio.id.replace(/^chapter-/, ''))
    const frame = document.querySelector('.showcase-banner-frame')
    const popup = frame?.closest('[popover]')
    const every = duration('--duration-banner')
    if (frame && popup && every > 0) {
      let banner = 0
      popup.addEventListener('beforetoggle', (event) => {
        clearInterval(banner)
        if (event.newState !== 'open') return
        // Set before the popup is drawn, so the first picture does not sweep in.
        frame.dataset.banner = chapters[Math.max(0, radios.findIndex((radio) => radio.checked))]
        banner = setInterval(() => {
          frame.dataset.banner = chapters[(chapters.indexOf(frame.dataset.banner) + 1) % chapters.length]
        }, every)
      })
    }

    let timer = 0
    let hovered = false
    let focused = false

    const popupOpen = () => document.querySelector(':popover-open') !== null
    const held = () => hovered || focused || document.hidden || popupOpen()

    const next = () => {
      const open = radios.findIndex((radio) => radio.checked)
      const radio = radios[(open + 1) % radios.length]
      radio.checked = true
      // As a click would, so the cards follow (cards.js) and the wait restarts.
      radio.dispatchEvent(new Event('change', { bubbles: true }))
    }

    function restart() {
      clearTimeout(timer)
      timer = held() ? 0 : setTimeout(next, wait)
    }

    for (const radio of radios) radio.addEventListener('change', restart)
    const track = (on, off, set) => {
      strip.addEventListener(on, () => (set(true), restart()))
      strip.addEventListener(off, () => (set(false), restart()))
    }
    track('pointerenter', 'pointerleave', (value) => (hovered = value))
    track('focusin', 'focusout', (value) => (focused = value))
    document.addEventListener('visibilitychange', restart)
    for (const popup of document.querySelectorAll('[popover]')) popup.addEventListener('toggle', restart)

    restart()
  },
  { once: true },
)
