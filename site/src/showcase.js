// Makes the launcher window's chapter rows clickable. A click shows that
// chapter with the app's own motion and restarts the cycle from it, so the
// chosen chapter stays for a full dwell before the next one slides in.
//
// Without this script the window cycles on CSS animations alone
// (src/showcase.css). With it, the script sets data-showcase="live" on the
// root, which turns those animations off, and runs the same cycle itself: it
// only moves attributes (which chapter is open, which card enters or leaves
// and which way), and every length, duration and easing stays a token in the
// stylesheet. The step is read from the same tokens the CSS cycle uses.
//
// Like detect.js it runs as part of one classic script (stylesheetFirst in
// vite.config.js), so nothing is declared at the top level.
document.addEventListener(
  'DOMContentLoaded',
  () => {
    const root = document.documentElement
    const rows = [...document.querySelectorAll('.showcase-rows > button')]
    const cards = [...document.querySelectorAll('.showcase-stage > .showcase-card')]
    if (rows.length === 0 || rows.length !== cards.length) return

    // A duration token as milliseconds: "3780ms" or "3.78s".
    const css = getComputedStyle(root)
    const ms = (name) => {
      const value = css.getPropertyValue(name).trim()
      return parseFloat(value) * (value.endsWith('ms') ? 1 : 1000)
    }
    const step = ms('--duration-dwell') + ms('--duration-slow')
    if (!(step > 0)) return

    let open = 0
    let timer = 0

    const mark = (index) => {
      open = index
      root.dataset.showcaseOpen = String(index)
      rows.forEach((row, i) => row.setAttribute('aria-current', String(i === index)))
      cards.forEach((card, i) => card.toggleAttribute('data-open', i === index))
    }

    // Down the list the new card comes up from below and the old one leaves
    // upward; up the list, and on the wrap from the last to the first, the
    // other way, as in the app. Clearing data-motion and reading a layout
    // value restarts an animation that was still running.
    const show = (index) => {
      if (index === open) return
      const down = index > open
      const leaving = cards[open]
      const entering = cards[index]
      for (const card of cards) delete card.dataset.motion
      void entering.offsetWidth
      leaving.dataset.motion = down ? 'leave-up' : 'leave-down'
      entering.dataset.motion = down ? 'enter-up' : 'enter-down'
      mark(index)
    }

    const schedule = () => {
      clearTimeout(timer)
      timer = setTimeout(() => {
        show((open + 1) % cards.length)
        schedule()
      }, step)
    }

    rows.forEach((row, i) =>
      row.addEventListener('click', () => {
        show(i)
        schedule()
      }),
    )

    mark(0)
    root.dataset.showcase = 'live'
    schedule()
  },
  { once: true },
)
