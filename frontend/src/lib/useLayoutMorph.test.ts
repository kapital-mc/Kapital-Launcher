import { describe, expect, it } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useLayoutMorph } from './useLayoutMorph'

const refs = () => ({ current: document.createElement('div') })

describe('useLayoutMorph', () => {
  it('starts where the window is, and takes a change before the settings are read at once', () => {
    const area = refs()
    const layer = refs()
    const { result, rerender } = renderHook(
      ({ compact, ready }) => useLayoutMorph(compact, ready, area, layer, null),
      { initialProps: { compact: true, ready: false } },
    )
    expect(result.current).toEqual({ mode: 'compact', morphTo: null })
    rerender({ compact: false, ready: false })
    expect(result.current).toEqual({ mode: 'full', morphTo: null })
  })

  it('morphs once ready, waits for the compact layout to be in the page, then swaps', async () => {
    const area = refs()
    const layer = refs()
    const stage = document.createElement('div')
    const { result, rerender } = renderHook(
      ({ compact, stage }: { compact: boolean; stage: HTMLElement | null }) =>
        useLayoutMorph(compact, true, area, layer, stage),
      { initialProps: { compact: false, stage: null as HTMLElement | null } },
    )
    rerender({ compact: true, stage: null })
    // The compact layout is mounted for the morph, and nothing swaps until it is there.
    expect(result.current).toEqual({ mode: 'full', morphTo: 'compact' })
    // jsdom has no Web Animations: the layout swaps as soon as it can.
    await act(async () => rerender({ compact: true, stage }))
    expect(result.current).toEqual({ mode: 'compact', morphTo: null })
  })
})
