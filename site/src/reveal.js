// Shows each section below the hero as it scrolls into view (src/intro.css):
// the root gets data-reveal-ready at once, which hides what is not yet shown, and a
// section gets data-shown the first time a fifth of it is in view. Never
// hidden without IntersectionObserver, and the CSS moves nothing under
// reduced motion.
//
// The build makes this a classic script in the head, after the stylesheet
// (stylesheetFirst in vite.config.js), so the root is marked before the body
// is painted. Nothing is declared at the top level: as a classic script, a
// top-level name would be a global.
if ('IntersectionObserver' in window) {
  document.documentElement.dataset.revealReady = ''
  document.addEventListener(
    'DOMContentLoaded',
    () => {
      const seen = new IntersectionObserver(
        (entries) => {
          for (const entry of entries) {
            if (!entry.isIntersecting) continue
            entry.target.dataset.shown = ''
            seen.unobserve(entry.target)
          }
        },
        { threshold: 0.2 },
      )
      for (const section of document.querySelectorAll('[data-reveal]')) seen.observe(section)
    },
    { once: true },
  )
}
