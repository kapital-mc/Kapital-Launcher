import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as Bindings from '../../wailsjs/go/main/App'
import { useGameSettingsStore } from './useGameSettingsStore'
import type { ChapterGameSettings } from '../types'

vi.mock('../../wailsjs/go/main/App')

const view = (files: string[]): ChapterGameSettings => ({
  chapterId: 'frangfurd',
  running: false,
  items: [{ id: 'distantHorizons', name: 'Distant Horizons', files, keys: 0 }],
})

describe('useGameSettingsStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(window, { go: {} })
    useGameSettingsStore.setState({
      byChapter: {},
      unavailable: false,
      readError: {},
      resetting: null,
    })
  })
  afterEach(() => Reflect.deleteProperty(window, 'go'))

  it('reads a chapter, and says so when there is no bridge', async () => {
    vi.mocked(Bindings.GetGameSettings).mockResolvedValue(view(['a']) as never)
    await useGameSettingsStore.getState().load('frangfurd')
    expect(useGameSettingsStore.getState().byChapter.frangfurd).toEqual(view(['a']))
    Reflect.deleteProperty(window, 'go')
    await useGameSettingsStore.getState().load('frangfurd')
    expect(useGameSettingsStore.getState().unavailable).toBe(true)
    await useGameSettingsStore.getState().reset('frangfurd', 'options')
    expect(Bindings.ResetGameSetting).not.toHaveBeenCalled()
  })

  it('keeps the reason a read was refused, and clears it on the next good read', async () => {
    vi.mocked(Bindings.GetGameSettings).mockRejectedValue('Frangfurd is not installed')
    await useGameSettingsStore.getState().load('frangfurd')
    expect(useGameSettingsStore.getState().readError.frangfurd).toBe('Frangfurd is not installed')
    vi.mocked(Bindings.GetGameSettings).mockResolvedValue(view([]) as never)
    await useGameSettingsStore.getState().load('frangfurd')
    expect(useGameSettingsStore.getState().readError.frangfurd).toBeUndefined()
  })

  it('takes Go’s list after a reset, and rethrows a refusal', async () => {
    vi.mocked(Bindings.ResetGameSetting).mockResolvedValue(view([]) as never)
    const done = useGameSettingsStore.getState().reset('frangfurd', 'distantHorizons')
    expect(useGameSettingsStore.getState().resetting).toBe('distantHorizons')
    await done
    expect(Bindings.ResetGameSetting).toHaveBeenCalledWith('frangfurd', 'distantHorizons')
    expect(useGameSettingsStore.getState().byChapter.frangfurd).toEqual(view([]))
    expect(useGameSettingsStore.getState().resetting).toBeNull()

    vi.mocked(Bindings.ResetGameSetting).mockRejectedValue('Frangfurd is starting or running')
    await expect(useGameSettingsStore.getState().reset('frangfurd', 'options')).rejects.toBe(
      'Frangfurd is starting or running',
    )
    expect(useGameSettingsStore.getState().resetting).toBeNull()
  })
})
