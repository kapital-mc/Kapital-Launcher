import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as Bindings from '../../wailsjs/go/main/App'
import { useStorageStore } from './useStorageStore'
import type { ChapterStorage } from '../types'

vi.mock('../../wailsjs/go/main/App')

const view = (bytes: number): ChapterStorage => ({
  chapterId: 'frangfurd',
  totalBytes: bytes,
  freeBytes: 1000,
  partial: false,
  running: false,
  items: [{ kind: 'worldMap', name: '', label: '', bytes, files: 1, lastModified: '' }],
})

describe('useStorageStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(window, { go: {} })
    useStorageStore.setState({ byChapter: {}, unavailable: false, readError: {}, clearing: null })
  })
  afterEach(() => Reflect.deleteProperty(window, 'go'))

  it('reads a chapter, and says so when there is no bridge', async () => {
    vi.mocked(Bindings.GetChapterStorage).mockResolvedValue(view(10) as never)
    await useStorageStore.getState().load('frangfurd')
    expect(useStorageStore.getState().byChapter.frangfurd).toEqual(view(10))
    Reflect.deleteProperty(window, 'go')
    await useStorageStore.getState().load('frangfurd')
    expect(useStorageStore.getState().unavailable).toBe(true)
  })

  it('keeps the reason a read was refused, and clears it on the next good read', async () => {
    vi.mocked(Bindings.GetChapterStorage).mockRejectedValue('Frangfurd is not installed')
    await useStorageStore.getState().load('frangfurd')
    expect(useStorageStore.getState().readError.frangfurd).toBe('Frangfurd is not installed')
    vi.mocked(Bindings.GetChapterStorage).mockResolvedValue(view(10) as never)
    await useStorageStore.getState().load('frangfurd')
    expect(useStorageStore.getState().readError.frangfurd).toBeUndefined()
  })

  it('takes Go’s measure after a delete, and rethrows a refusal', async () => {
    vi.mocked(Bindings.ClearChapterStorage).mockResolvedValue(view(0) as never)
    const done = useStorageStore.getState().clear('frangfurd', 'worldMap', '')
    expect(useStorageStore.getState().clearing).toBe('worldMap/')
    await done
    expect(Bindings.ClearChapterStorage).toHaveBeenCalledWith('frangfurd', 'worldMap', '')
    expect(useStorageStore.getState().byChapter.frangfurd).toEqual(view(0))
    expect(useStorageStore.getState().clearing).toBeNull()

    vi.mocked(Bindings.ClearChapterStorage).mockRejectedValue('Frangfurd is starting or running')
    await expect(useStorageStore.getState().clear('frangfurd', 'logs', '')).rejects.toBe(
      'Frangfurd is starting or running',
    )
    expect(useStorageStore.getState().clearing).toBeNull()
    expect(useStorageStore.getState().byChapter.frangfurd).toEqual(view(0))
  })

  it('opens the screenshots through Go, and does nothing without a bridge', async () => {
    await useStorageStore.getState().openScreenshots('frangfurd')
    expect(Bindings.OpenScreenshotsFolder).toHaveBeenCalledWith('frangfurd')
    Reflect.deleteProperty(window, 'go')
    await useStorageStore.getState().openScreenshots('frangfurd')
    await useStorageStore.getState().clear('frangfurd', 'logs', '')
    expect(Bindings.OpenScreenshotsFolder).toHaveBeenCalledTimes(1)
    expect(Bindings.ClearChapterStorage).not.toHaveBeenCalled()
  })
})
