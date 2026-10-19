import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { morphFrom, smoothResize } from './smoothResize'

// jsdom lays nothing out and has no ResizeObserver: heights are stubbed, and the observer's
// callback is called by hand as the browser would after a layout.
let report: () => void
class FakeResizeObserver {
  constructor(cb: () => void) {
    report = cb
  }
  observe() {}
  disconnect() {}
}

function box(height: () => number) {
  const el = document.createElement('div')
  el.getBoundingClientRect = () => ({ height: height(), width: 100 }) as DOMRect
  return el
}

function transition(el: Element, type: string, propertyName: string) {
  const e = new Event(type, { bubbles: true })
  Object.assign(e, { propertyName })
  el.dispatchEvent(e)
}

describe('smoothResize', () => {
  let content = 40
  let outer: HTMLElement
  let inner: HTMLElement
  let stop: (() => void) | undefined

  beforeEach(() => {
    vi.stubGlobal('ResizeObserver', FakeResizeObserver)
    content = 40
    outer = box(() => (outer.style.height ? Number.parseFloat(outer.style.height) : content))
    inner = box(() => content)
    outer.append(inner)
    document.body.append(outer)
    stop = smoothResize(outer, inner)
  })
  afterEach(() => {
    stop?.()
    outer.remove()
    vi.unstubAllGlobals()
  })

  it('moves the box from the height it showed to the new one, then lets it go', () => {
    content = 90
    report()
    expect(outer).toHaveClass('smooth-resizing')
    expect(outer.style.height).toBe('90px')

    // A change mid-way moves on from where it is.
    content = 120
    report()
    expect(outer.style.height).toBe('120px')

    transition(outer, 'transitionend', 'height')
    expect(outer).not.toHaveClass('smooth-resizing')
    expect(outer.style.height).toBe('')
  })

  it('follows a part that runs its own height transition instead of chasing it', () => {
    const fold = document.createElement('div')
    inner.append(fold)
    transition(fold, 'transitionrun', 'grid-template-rows')
    content = 90
    report()
    expect(outer).not.toHaveClass('smooth-resizing')
    expect(outer.style.height).toBe('')

    transition(fold, 'transitionend', 'grid-template-rows')
    content = 60
    report()
    expect(outer).toHaveClass('smooth-resizing')
  })

  it('fades in what is added later, but not a part that runs its own entrance', async () => {
    const line = document.createElement('p')
    const own = document.createElement('li')
    own.dataset.smooth = 'self'
    inner.append(line, own)
    await Promise.resolve()
    expect(line).toHaveClass('smooth-enter')
    expect(own).not.toHaveClass('smooth-enter')

    line.dispatchEvent(new Event('animationend'))
    expect(line).not.toHaveClass('smooth-enter')
  })
})

describe('morphFrom', () => {
  it('grows an element from the rectangle of what it replaces and fades its content in', () => {
    const el = document.createElement('li')
    el.append(document.createElement('input'))
    el.getBoundingClientRect = () => ({ width: 400, height: 36 }) as DOMRect
    const grow = { onfinish: null as null | (() => void) }
    const animate = vi.fn(() => grow)
    el.animate = animate as never
    ;(el.firstElementChild as HTMLElement).animate = vi.fn() as never

    morphFrom(el, { width: 140, height: 32 } as DOMRect)
    expect(animate).toHaveBeenCalledWith(
      [
        { width: '140px', height: '32px' },
        { width: '400px', height: '36px' },
      ],
      expect.anything(),
    )
    expect(el.style.overflow).toBe('clip')
    grow.onfinish?.()
    expect(el.style.overflow).toBe('')
    expect((el.firstElementChild as HTMLElement).animate).toHaveBeenCalled()
  })
})
