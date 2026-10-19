/**
 * The motion behind `ui/Smooth`, on plain elements: a box (`outer`) that
 * follows its content's height (`inner`) over the reveal's 200 ms instead of
 * jumping, and content added to it later that fades in as it opens. It is one
 * mechanism for every card and popup, so a part that grows (a licence shown, a
 * field in place of a button, a line of progress) needs nothing of its own.
 *
 * The content is measured by a ResizeObserver, which reports after layout and
 * before paint: the box is pinned at the height it showed, then moved to the new
 * one by a CSS transition (`smooth-resizing`), and let go when it ends. A
 * change mid-way moves on from where it is. While a part inside runs its own
 * height transition (`ui/Grow`, `ui/Fold`) or morphs (`morphFrom`, `morphTo`),
 * the box simply follows it, so the two never chase each other. Added elements get `smooth-enter`, a fade a
 * little behind the growth. With reduced motion, or where the browser lacks a
 * ResizeObserver (the tests), it does nothing and has nothing to undo.
 */
const SELF = '[data-smooth="self"]'

const reducedMotion = () => window.matchMedia?.('(prefers-reduced-motion: reduce)').matches === true

/** A motion token's value, read from the root: a duration in ms, or an easing as written. */
function token(name: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
}
function ms(name: string): number {
  const v = token(name)
  const n = Number.parseFloat(v)
  return v.endsWith('ms') ? n : n * 1000
}

/**
 * An element that takes the place of a control, such as the field that opens
 * where Add argument was: it grows from the control's rectangle to its own
 * over the page's 320 ms, clipped meanwhile, and its content fades in behind. Mark
 * the element `data-smooth="self"` so the card it is in does not fade it
 * too. Nothing happens with reduced motion or without the Web Animations API.
 */
export function morphFrom(el: HTMLElement, from: DOMRect): void {
  if (typeof el.animate !== 'function' || reducedMotion()) return
  // A morph already running (an effect run twice) would hold the size being measured.
  for (const a of el.getAnimations?.({ subtree: true }) ?? []) a.cancel()
  const to = el.getBoundingClientRect()
  // Out quickly and settling, over the page's longer clock: an ease-in start reads as a
  // jump at the end on so short a move.
  const timing = { duration: ms('--duration-page'), easing: token('--ease-standard') }
  el.style.overflow = 'clip'
  el.animate(
    [
      { width: `${from.width}px`, height: `${from.height}px` },
      { width: `${to.width}px`, height: `${to.height}px` },
    ],
    timing,
  ).onfinish = () => el.style.removeProperty('overflow')
  fadeIn(el, { ...timing, delay: ms('--duration-fast') / 2 })
}

/**
 * Fades the content of `el` in, its box shown at once: a control back where a morph ended (Add
 * argument once its field has shrunk into it). Nothing happens with reduced motion.
 */
export function fadeIn(el: HTMLElement | null, timing?: KeyframeAnimationOptions): void {
  if (!el || typeof el.animate !== 'function' || reducedMotion()) return
  for (const child of el.children) {
    child.animate([{ opacity: 0 }, { opacity: 1 }], {
      duration: ms('--duration-fast'),
      easing: token('--ease-standard'),
      ...timing,
      fill: 'backwards',
    })
  }
}

/**
 * The way back: `el` shrinks to `to` (the rectangle of the control it took the place of) over
 * the reveal's 200 ms while its content fades out, then `done` removes it and puts the control
 * back where it ends. Without the Web Animations API, or with reduced motion, `done` runs at once.
 */
export function morphTo(el: HTMLElement, to: DOMRect, done: () => void): void {
  if (typeof el.animate !== 'function' || reducedMotion()) return done()
  for (const a of el.getAnimations?.({ subtree: true }) ?? []) a.cancel()
  const from = el.getBoundingClientRect()
  const easing = token('--ease-standard')
  el.style.overflow = 'clip'
  for (const child of el.children) {
    child.animate([{ opacity: 1 }, { opacity: 0 }], {
      duration: ms('--duration-fast'),
      easing,
      fill: 'forwards',
    })
  }
  el.animate(
    [
      { width: `${from.width}px`, height: `${from.height}px` },
      { width: `${to.width}px`, height: `${to.height}px` },
    ],
    { duration: ms('--duration-reveal'), easing, fill: 'forwards' },
  ).onfinish = done
}

/** A part under `inner` is growing or shrinking by itself (`morphFrom`, `morphTo`). */
function morphing(inner: HTMLElement): boolean {
  for (const el of inner.querySelectorAll<HTMLElement>(SELF)) {
    if (el.getAnimations?.().some((a) => a.playState === 'running')) return true
  }
  return false
}

export function smoothResize(outer: HTMLElement, inner: HTMLElement): (() => void) | undefined {
  if (typeof ResizeObserver === 'undefined' || typeof MutationObserver === 'undefined') {
    return undefined
  }
  const reduced = reducedMotion
  // The height the box showed after its last change: where a new change starts from.
  let shown = outer.getBoundingClientRect().height
  let moving = false
  // Height transitions running on parts inside, which the box follows as they go.
  let following = 0
  // Lets the box go even if its transition never reports an end (WebKit can skip
  // transitionend, for one when the box was not being drawn), so it is never left
  // pinned short and clipping its content.
  let failsafe: ReturnType<typeof setTimeout> | undefined

  const settle = () => {
    clearTimeout(failsafe)
    moving = false
    outer.classList.remove('smooth-resizing')
    outer.style.removeProperty('height')
  }
  const chrome = () => {
    const s = getComputedStyle(outer)
    const px = (v: string) => Number.parseFloat(v) || 0
    return px(s.paddingTop) + px(s.paddingBottom) + px(s.borderTopWidth) + px(s.borderBottomWidth)
  }

  const resize = new ResizeObserver(() => {
    const target = inner.getBoundingClientRect().height + chrome()
    if (Math.abs(target - shown) < 0.5) return
    if (following > 0 || reduced() || morphing(inner)) {
      if (moving) settle()
      shown = target
      return
    }
    if (!moving) {
      // Pinned where it was, then moved: the transition needs a height to start from.
      outer.style.height = `${shown}px`
      outer.classList.add('smooth-resizing')
      void outer.offsetHeight
      moving = true
    }
    outer.style.height = `${target}px`
    shown = target
    clearTimeout(failsafe)
    failsafe = setTimeout(settle, ms('--duration-reveal') + ms('--duration-fast'))
  })
  resize.observe(inner)

  const layout = (e: TransitionEvent) =>
    e.target !== outer &&
    (e.propertyName.startsWith('grid-template') || e.propertyName === 'height')
  const onRun = (e: TransitionEvent) => {
    if (layout(e)) following++
  }
  const onEnd = (e: TransitionEvent) => {
    if (e.target === outer && e.propertyName === 'height') settle()
    else if (layout(e)) following = Math.max(0, following - 1)
  }
  outer.addEventListener('transitionrun', onRun)
  outer.addEventListener('transitionend', onEnd)
  outer.addEventListener('transitioncancel', onEnd)

  const enter = fadeInAdded(inner)

  return () => {
    resize.disconnect()
    enter()
    outer.removeEventListener('transitionrun', onRun)
    outer.removeEventListener('transitionend', onEnd)
    outer.removeEventListener('transitioncancel', onEnd)
    settle()
  }
}

/** Gives `smooth-enter` to each element added under `inner` from now on; returns the stop. */
function fadeInAdded(inner: HTMLElement): () => void {
  const enter = new MutationObserver((records) => {
    if (reducedMotion()) return
    for (const r of records) {
      for (const node of r.addedNodes) {
        if (!(node instanceof HTMLElement)) continue
        // A part that runs its own entrance (`morphFrom`) is left to it.
        if (node.matches(SELF) || node.querySelector(SELF)) continue
        node.classList.add('smooth-enter')
        const done = (e: AnimationEvent) => {
          if (e.target !== node) return
          node.classList.remove('smooth-enter')
          node.removeEventListener('animationend', done)
        }
        node.addEventListener('animationend', done)
      }
    }
  })
  enter.observe(inner, { childList: true, subtree: true })
  return () => enter.disconnect()
}
