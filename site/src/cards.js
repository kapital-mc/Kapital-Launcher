// The two floating cards beside the window, and the download popup's button
// word. On a chapter switch (or a system switch, for the word) their text
// changes the way the backdrop changes its squares: each letter that differs
// runs through block characters and lands on the new one at its own moment
// within --duration-slow. Every version sits in one cell (src/showcase.css,
// src/popup.css), so the script only shows the new one at once and scrambles
// its text from what the old one was showing. Without the script, or under
// reduced motion, the versions cross-fade instead.
//
// It also fills in the server card. Frangfurd's server is asked through the
// site's own /status (functions/status.js), once on load and again when
// Frangfurd opens after the answer is older than the function's cache; the
// answer sets data-server on the root, for the card's square, and the card's
// text. Without an answer the card stays at "Server".
//
// Like detect.js it runs as part of one classic script, so nothing is
// declared at the top level.
document.addEventListener(
  'DOMContentLoaded',
  () => {
    const cards = [...document.querySelectorAll('.showcase-card')]
    const radios = [...document.querySelectorAll('input[name="chapter"]')]
    const word = document.querySelector('.download-word')
    const systems = [...document.querySelectorAll('input[name="os"]')]
    if (cards.length === 0 || radios.length === 0) return
    // The browser may give the token back in seconds ("0.42s") or as written.
    const token = getComputedStyle(document.documentElement).getPropertyValue('--duration-slow').trim()
    const span = parseFloat(token) * (token.endsWith('ms') ? 1 : token.endsWith('s') ? 1000 : NaN)
    const motion = span > 0 && !matchMedia('(prefers-reduced-motion: reduce)').matches

    // A letter is redrawn this many times on its way to the new one, at most.
    const STEPS = 10
    // Shades and halves of a square, after the backdrop's: all in code page
    // 437, so every system's fallback font has them.
    const BLOCKS = '█▓▒░▀▄▌▐'
    const block = () => BLOCKS[Math.floor(Math.random() * BLOCKS.length)]

    // Every text node's own text, to land on and to put back, in page order.
    const finals = new Map()
    for (const group of word ? [...cards, word] : cards) {
      const walker = document.createTreeWalker(group, NodeFilter.SHOW_TEXT)
      while (walker.nextNode()) {
        if (walker.currentNode.data.trim()) finals.set(walker.currentNode, walker.currentNode.data)
      }
    }
    const textOf = (element) => [...finals.keys()].filter((node) => element.contains(node))
    const frames = new Map()

    const scramble = (node, from, to) => {
      cancelAnimationFrame(frames.get(node))
      if (!motion) {
        node.data = to
        return
      }
      const length = Math.max(from.length, to.length)
      const lands = Array.from({ length }, () => Math.random() * span)
      const steady = Array.from({ length }, (_, i) => from[i] === to[i])
      const shown = Array.from({ length }, (_, i) => from[i] ?? '')
      const begin = performance.now()
      let step = -1
      const frame = (now) => {
        const t = now - begin
        const next = Math.floor((t / span) * STEPS)
        if (next !== step) {
          step = next
          for (let i = 0; i < length; i++) {
            if (steady[i]) shown[i] = to[i]
            else if (t >= lands[i]) shown[i] = to[i] ?? ''
            else if (to[i] === ' ' || from[i] === ' ') shown[i] = ' '
            else shown[i] = block()
          }
          node.data = shown.join('')
        }
        if (t < span) frames.set(node, requestAnimationFrame(frame))
        else {
          node.data = to
          frames.delete(node)
        }
      }
      frames.set(node, requestAnimationFrame(frame))
    }

    const chapterOf = (radio) => radio.id.replace(/^chapter-/, '')
    let open = chapterOf(radios.find((radio) => radio.checked) ?? radios[0])

    // The server card's live line, and when its answer came.
    const live = document.querySelector('[data-status]')
    const liveNode = live && textOf(live)[0]
    const liveChapter = live?.closest('[data-for]')?.dataset.for
    let answered = 0
    // functions/status.js caches its answer for 30 s, so asking sooner gets the same one.
    const FRESH = 30_000

    const ask = async () => {
      answered = Date.now()
      try {
        const response = await fetch('/status', { headers: { accept: 'application/json' } })
        if (!response.ok) return
        const status = (await response.json())?.[liveChapter]
        if (typeof status?.online !== 'boolean') return
        const counts = Number.isInteger(status.players) && Number.isInteger(status.max)
        const text = status.online
          ? `Server online${counts ? ` · ${status.players}/${status.max}` : ''}`
          : 'Server offline'
        document.documentElement.dataset.server = status.online ? 'online' : 'offline'
        const was = liveNode.data
        finals.set(liveNode, text)
        if (open === liveChapter) scramble(liveNode, was, text)
        else liveNode.data = text
      } catch {
        // No answer, or not one this script reads: the card stays as it is.
      }
    }

    // Shows `after` in place of `before`, scrambling from what `before` showed.
    const swap = (before, after) => {
      if (!before || !after) return
      const shown = textOf(before).map((node) => {
        const text = node.data
        cancelAnimationFrame(frames.get(node))
        node.data = finals.get(node)
        return text
      })
      textOf(after).forEach((node, i) => scramble(node, shown[i] ?? '', finals.get(node)))
    }

    if (motion) for (const group of word ? [...cards, word] : cards) group.setAttribute('data-scramble', '')
    for (const radio of radios) {
      radio.addEventListener('change', () => {
        if (!radio.checked) return
        const from = open
        open = chapterOf(radio)
        if (from === open) return
        if (liveNode && open === liveChapter && Date.now() - answered > FRESH) ask()
        if (!motion) return
        for (const card of cards) {
          swap(card.querySelector(`[data-for="${from}"]`), card.querySelector(`[data-for="${open}"]`))
        }
      })
    }

    // Two systems, so the word coming in replaces the other one. detect.js
    // and the buttons that open the popup check a radio with no change event,
    // so there is no last system to keep.
    for (const radio of systems) {
      radio.addEventListener('change', () => {
        const from = systems.find((other) => other !== radio)?.value
        if (!radio.checked || !motion || !word) return
        swap(word.querySelector(`[data-word="${from}"]`), word.querySelector(`[data-word="${radio.value}"]`))
      })
    }

    if (liveNode && liveChapter) ask()
  },
  { once: true },
)
