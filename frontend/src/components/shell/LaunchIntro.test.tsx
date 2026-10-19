import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, cleanup, render, screen } from '@testing-library/react'
import { LaunchIntro } from './LaunchIntro'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('LaunchIntro', () => {
  it('is not there without the Web Animations API', () => {
    render(<LaunchIntro />)
    expect(screen.queryByTestId('launch-intro')).toBeNull()
  })

  it('draws every line and goes when its cover has faded', async () => {
    const finished: Animation[] = []
    const animate = vi.fn(function (this: Element) {
      const a = { cancel: vi.fn(), onfinish: null as (() => void) | null } as unknown as Animation
      finished.push(a)
      return a
    })
    Object.defineProperty(Element.prototype, 'animate', { value: animate, configurable: true })
    try {
      render(<LaunchIntro />)
      const intro = screen.getByTestId('launch-intro')
      await vi.waitFor(() => expect(intro.querySelectorAll('path')).toHaveLength(16))
      // A line each, the drawing, the picture and the cover.
      expect(animate).toHaveBeenCalledTimes(19)
      const cover = finished.at(-1)
      act(() => cover?.onfinish?.call(cover, new Event('finish') as AnimationPlaybackEvent))
      expect(screen.queryByTestId('launch-intro')).toBeNull()
    } finally {
      delete (Element.prototype as { animate?: unknown }).animate
    }
  })
})
