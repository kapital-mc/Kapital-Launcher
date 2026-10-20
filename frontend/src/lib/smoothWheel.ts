import { ms, reducedMotion } from './smoothResize'

/** A wheel's line, for a wheel that reports in lines, in px. */
const LINE = 16

/** Whether `el` could scroll further by `dy` itself, as an overflowing box. */
function canScroll(el: Element, dy: number) {
  const { overflowY } = getComputedStyle(el)
  if (overflowY !== 'auto' && overflowY !== 'scroll') return false
  return dy < 0 ? el.scrollTop > 0 : el.scrollTop + el.clientHeight < el.scrollHeight - 1
}

/**
 * Wheel scrolling that glides: each wheel step moves a target, and the element
 * eases towards it every frame, so a notched wheel no longer jumps a step at a
 * time and a turn the other way bends the glide round rather than stopping it.
 * WebKit on macOS moves a mouse wheel's steps at once, and WebView2 smooths them
 * only when Windows' animations are on.
 *
 * The distance left shrinks by e each fast duration, so a step settles in about
 * three. A trackpad's own glide feeds the target the same way. What the element
 * does not take is left to the browser: a pinch (Ctrl), a sideways swipe, a
 * scroll inside a box of its own that can still move, a page at its end (so it
 * reaches an outer scroller), and everything with reduced motion. A scroll from
 * elsewhere (the thumb, the keys) stops the glide where it is.
 */
export function smoothWheel(el: HTMLElement): () => void {
  let target = 0
  let pos = 0
  let set = 0
  let frame = 0
  let last = 0
  let tau = 0

  const step = (now: number) => {
    // Something else moved it since the last frame: that wins.
    if (Math.abs(el.scrollTop - set) > 1) {
      frame = 0
      return
    }
    const k = 1 - Math.exp(-(now - last) / tau)
    last = now
    pos += (target - pos) * k
    if (Math.abs(target - pos) < 0.5) pos = target
    el.scrollTop = pos
    set = el.scrollTop
    frame = pos === target ? 0 : requestAnimationFrame(step)
  }

  const onWheel = (e: WheelEvent) => {
    if (e.defaultPrevented || e.ctrlKey || Math.abs(e.deltaX) > Math.abs(e.deltaY)) return
    if (reducedMotion()) return
    const dy = e.deltaY * (e.deltaMode === 1 ? LINE : e.deltaMode === 2 ? el.clientHeight : 1)
    for (let n = e.target as Element | null; n && n !== el; n = n.parentElement) {
      if (canScroll(n, dy)) return
    }
    if (!frame) target = pos = set = el.scrollTop
    const next = Math.min(Math.max(target + dy, 0), el.scrollHeight - el.clientHeight)
    if (next === target && !frame) return
    e.preventDefault()
    target = next
    if (!frame) {
      tau = ms('--duration-fast')
      last = performance.now()
      frame = requestAnimationFrame(step)
    }
  }

  el.addEventListener('wheel', onWheel, { passive: false })
  return () => {
    el.removeEventListener('wheel', onWheel)
    cancelAnimationFrame(frame)
  }
}
