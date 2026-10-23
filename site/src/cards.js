// The two floating cards beside the window, and the download popup's button
// word. On a chapter switch (or a system switch, for the word) their text
// changes the way the backdrop changes its squares: each letter that differs
// runs through blocks (full, top half, bottom half) and lands on the new one
// at its own moment within --duration-slow. The blocks are drawn as boxes one
// letter wide and one capital tall (.scramble-block, src/showcase.css), not as
// characters: no face the page loads has them, and a fallback face's blocks
// are a full line tall. Every version sits in one cell (src/showcase.css,
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
    const BLOCKS = ['full', 'top', 'bottom']
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

    // A scrambling text node is emptied and drawn in a span before it: its
    // letters as text, its blocks as boxes. `shown` is what each node shows.
    const overlays = new Map()
    const shown = new Map()
    const textShown = (node) => shown.get(node) ?? node.data
    const draw = (node, letters) => {
      if (letters.every((letter) => typeof letter === 'string')) {
        overlays.get(node)?.remove()
        overlays.delete(node)
        node.data = letters.join('')
        shown.delete(node)
        return
      }
      let overlay = overlays.get(node)
      if (!overlay) {
        overlay = document.createElement('span')
        node.parentNode.insertBefore(overlay, node)
        overlays.set(node, overlay)
      }
      node.data = ''
      const parts = []
      let run = ''
      for (const letter of letters) {
        if (typeof letter === 'string') {
          run += letter
          continue
        }
        if (run) parts.push(run)
        run = ''
        const box = document.createElement('span')
        box.className = 'scramble-block'
        box.dataset.block = letter.block
        parts.push(box)
      }
      if (run) parts.push(run)
      overlay.replaceChildren(...parts)
      // Something in place of every block, for what swap() reads back.
      shown.set(node, letters.map((letter) => (typeof letter === 'string' ? letter : '\u2588')).join(''))
    }

    const scramble = (node, from, to) => {
      cancelAnimationFrame(frames.get(node))
      if (!motion) {
        draw(node, [to])
        return
      }
      const length = Math.max(from.length, to.length)
      const lands = Array.from({ length }, () => Math.random() * span)
      const steady = Array.from({ length }, (_, i) => from[i] === to[i])
      const letters = Array.from({ length }, (_, i) => from[i] ?? '')
      const begin = performance.now()
      let step = -1
      const frame = (now) => {
        const t = now - begin
        const next = Math.floor((t / span) * STEPS)
        if (next !== step) {
          step = next
          for (let i = 0; i < length; i++) {
            if (steady[i]) letters[i] = to[i]
            else if (t >= lands[i]) letters[i] = to[i] ?? ''
            else if (to[i] === ' ' || from[i] === ' ') letters[i] = ' '
            else letters[i] = { block: block() }
          }
          draw(node, letters)
        }
        if (t < span) frames.set(node, requestAnimationFrame(frame))
        else {
          draw(node, [to])
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
        const was = textShown(liveNode)
        finals.set(liveNode, text)
        if (open === liveChapter) scramble(liveNode, was, text)
        else draw(liveNode, [text])
      } catch {
        // No answer, or not one this script reads: the card stays as it is.
      }
    }

    // Shows `after` in place of `before`, scrambling from what `before` showed.
    const swap = (before, after) => {
      if (!before || !after) return
      const was = textOf(before).map((node) => {
        const text = textShown(node)
        cancelAnimationFrame(frames.get(node))
        draw(node, [finals.get(node)])
        return text
      })
      textOf(after).forEach((node, i) => scramble(node, was[i] ?? '', finals.get(node)))
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
