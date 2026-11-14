// The status page's one script. It asks the site's own /api/status
// (functions/api/status.js) on load, again every 30 s while the page is in
// view, and when it comes back into view with an answer older than that. Each
// answer sets data-up on every row (online or offline), the row's state text
// and the summary above them. Nothing a service says reaches the page but
// what the function passes on, and every value is checked again here and
// written as text, never markup.
//
// The build makes it a classic script in the head (stylesheetFirst in
// vite.config.js), so it waits for the document and declares nothing at the
// top level.
document.addEventListener(
  'DOMContentLoaded',
  () => {
    const rows = [...document.querySelectorAll('[data-service]')]
    const summary = document.querySelector('[data-summary]')
    const summaryText = document.querySelector('[data-summary-text]')
    const checked = document.querySelector('[data-checked]')
    if (rows.length === 0) return

    // functions/api/status.js caches its answer for 30 s.
    const EVERY = 30_000
    const VERSION = /^[0-9A-Za-z.+-]{1,32}$/
    const DAY = /^\d{4}-\d{2}-\d{2}$/
    const count = (n) => Number.isInteger(n) && n >= 0
    const day = new Intl.DateTimeFormat('en-GB', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'UTC' })
    const time = new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' })
    let asked = 0
    let timer = 0

    const stateOf = (answer, id) => answer?.servers?.[id] ?? answer?.[id]

    const show = (answer) => {
      let up = 0
      for (const row of rows) {
        const state = stateOf(answer, row.dataset.service)
        const online = state?.online === true
        if (online) up++
        row.dataset.up = online ? 'online' : 'offline'
        let text = online ? 'Online' : 'Offline'
        if (online && count(state.players) && count(state.max)) text += ` · ${state.players}/${state.max}`
        row.querySelector('[data-state]').textContent = text

        const version = row.querySelector('[data-pack-version]')
        if (version && typeof state?.version === 'string' && VERSION.test(state.version)) {
          version.textContent = state.version
        }
        const detail = row.querySelector('[data-detail]')
        if (row.dataset.service === 'terrain' && detail && typeof state?.date === 'string' && DAY.test(state.date)) {
          detail.textContent = `Distant Horizons · ${day.format(new Date(`${state.date}T00:00:00Z`))}`
        }
      }
      const all = rows.length
      summary.dataset.up = up === all ? 'online' : up === 0 ? 'offline' : 'partial'
      summaryText.textContent = up === all ? 'All online' : up === 0 ? 'Offline' : `${up} of ${all} online`
      const when = new Date(answer?.checked)
      checked.textContent = `Checked ${time.format(Number.isNaN(when.getTime()) ? new Date() : when)}`
    }

    const ask = async () => {
      asked = Date.now()
      clearTimeout(timer)
      try {
        const response = await fetch('/api/status', { headers: { accept: 'application/json' } })
        if (response.ok) show(await response.json())
      } catch {
        // No answer: the rows stay as they are.
      }
      if (document.visibilityState === 'visible') timer = setTimeout(ask, EVERY)
    }

    document.addEventListener('visibilitychange', () => {
      if (document.visibilityState !== 'visible') clearTimeout(timer)
      else if (Date.now() - asked >= EVERY) ask()
      else timer = setTimeout(ask, EVERY - (Date.now() - asked))
    })

    ask()
  },
  { once: true },
)
