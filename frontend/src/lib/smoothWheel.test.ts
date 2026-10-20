import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { smoothWheel } from './smoothWheel'

/** A 100px box over 1000px of content, as jsdom has no layout. */
function box(scrollTop = 0) {
  const el = document.createElement('div')
  el.style.overflowY = 'auto'
  Object.defineProperty(el, 'clientHeight', { value: 100 })
  Object.defineProperty(el, 'scrollHeight', { value: 1000 })
  el.scrollTop = scrollTop
  document.body.append(el)
  return el
}

function wheel(el: Element, deltaY: number, init: WheelEventInit = {}) {
  const e = new WheelEvent('wheel', { deltaY, bubbles: true, cancelable: true, ...init })
  el.dispatchEvent(e)
  return e
}

describe('smoothWheel', () => {
  let frames: FrameRequestCallback[] = []
  let now = 0
  const run = (n: number) => {
    for (let i = 0; i < n; i++) {
      now += 16
      const due = frames
      frames = []
      due.forEach((f) => f(now))
    }
  }
  beforeEach(() => {
    vi.spyOn(performance, 'now').mockImplementation(() => now)
    vi.stubGlobal('requestAnimationFrame', (f: FrameRequestCallback) => frames.push(f))
    vi.stubGlobal('cancelAnimationFrame', () => (frames = []))
    document.documentElement.style.setProperty('--duration-fast', '140ms')
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
    document.body.replaceChildren()
  })

  it('glides to where the wheel points instead of jumping there', () => {
    const el = box()
    smoothWheel(el)
    expect(wheel(el, 100).defaultPrevented).toBe(true)
    run(1)
    expect(el.scrollTop).toBeGreaterThan(0)
    expect(el.scrollTop).toBeLessThan(100)
    run(60)
    expect(el.scrollTop).toBe(100)
  })

  it('adds steps together, in lines when the wheel counts lines', () => {
    const el = box()
    smoothWheel(el)
    wheel(el, 3, { deltaMode: 1 })
    wheel(el, 3, { deltaMode: 1 })
    run(60)
    expect(el.scrollTop).toBe(96)
  })

  it('stops at the ends and leaves a wheel past them to the browser', () => {
    const el = box(900)
    smoothWheel(el)
    expect(wheel(el, 100).defaultPrevented).toBe(false)
    wheel(el, -2000)
    run(120)
    expect(el.scrollTop).toBe(0)
  })

  it('leaves pinches, sideways swipes and inner scrollers alone', () => {
    const el = box()
    const inner = box()
    el.append(inner)
    smoothWheel(el)
    expect(wheel(el, 100, { ctrlKey: true }).defaultPrevented).toBe(false)
    expect(wheel(el, 10, { deltaX: 50 }).defaultPrevented).toBe(false)
    expect(wheel(inner, 100).defaultPrevented).toBe(false)
  })

  it('lets a scroll from elsewhere win over the glide', () => {
    const el = box()
    smoothWheel(el)
    wheel(el, 300)
    run(2)
    el.scrollTop = 700
    run(60)
    expect(el.scrollTop).toBe(700)
  })
})
