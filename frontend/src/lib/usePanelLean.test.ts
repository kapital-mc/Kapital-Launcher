import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { usePanelLean } from './usePanelLean'

describe('usePanelLean', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    document.documentElement.style.setProperty('--duration-lean', '700ms')
  })
  afterEach(() => {
    vi.useRealTimers()
    document.documentElement.style.removeProperty('--duration-lean')
  })

  it('leans a panel only once the pointer has rested on it', () => {
    const { result } = renderHook(() => usePanelLean('luxemburg'))
    act(() => result.current.enter('frangfurd'))
    act(() => vi.advanceTimersByTime(600))
    expect(result.current.leaning).toBeNull()
    act(() => vi.advanceTimersByTime(100))
    expect(result.current.leaning).toBe('frangfurd')
  })

  it('lets the leaning panel go at once when the pointer moves to the next one', () => {
    const { result } = renderHook(() => usePanelLean('luxemburg'))
    act(() => result.current.enter('lichdenstein'))
    act(() => vi.advanceTimersByTime(700))
    act(() => {
      result.current.leave('lichdenstein')
      result.current.enter('frangfurd')
    })
    expect(result.current.leaning).toBeNull()
    act(() => vi.advanceTimersByTime(700))
    expect(result.current.leaning).toBe('frangfurd')
  })

  it('never leans the open panel, and a pointer passing over starts nothing', () => {
    const { result } = renderHook(() => usePanelLean('luxemburg'))
    act(() => result.current.enter('luxemburg'))
    act(() => vi.advanceTimersByTime(700))
    expect(result.current.leaning).toBeNull()
    act(() => result.current.enter('frangfurd'))
    act(() => result.current.leave('frangfurd'))
    act(() => vi.advanceTimersByTime(700))
    expect(result.current.leaning).toBeNull()
  })
})
