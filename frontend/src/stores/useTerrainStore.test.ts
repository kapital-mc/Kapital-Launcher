import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import * as Bindings from '../../wailsjs/go/main/App'
import * as Runtime from '../../wailsjs/runtime/runtime'
import { EVENT_DH_DOWNLOAD, useTerrainStore } from './useTerrainStore'
import type { DHSnapshotInfo } from '../types'

vi.mock('../../wailsjs/go/main/App')
vi.mock('../../wailsjs/runtime/runtime')

const published: DHSnapshotInfo = {
  chapterId: 'frangfurd',
  available: true,
  reason: '',
  createdAt: '2026-10-13T12:03:41Z',
  sizeBytes: 6_000_000_000,
  dimensions: 11,
  serverDhVersion: '3.3.3',
  clientDhVersion: '3.3.3',
  existingBytes: 0,
  partialBytes: 0,
  running: false,
  downloading: false,
}

const reset = () =>
  useTerrainStore.setState({
    byChapter: {},
    progress: {},
    unavailable: false,
    readError: {},
    cancelling: {},
  })

describe('useTerrainStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.assign(window, { go: {} })
    reset()
  })
  afterEach(() => Reflect.deleteProperty(window, 'go'))

  it('reads what is published, and keeps a refusal without throwing', async () => {
    vi.mocked(Bindings.GetDHSnapshot).mockResolvedValueOnce(published as never)
    await useTerrainStore.getState().load('frangfurd')
    expect(useTerrainStore.getState().byChapter.frangfurd).toEqual(published)
    vi.mocked(Bindings.GetDHSnapshot).mockRejectedValueOnce(new Error('no chapter "x"'))
    await useTerrainStore.getState().load('x')
    expect(useTerrainStore.getState().readError.x).toBe('no chapter "x"')
  })

  it('reads nothing without a bridge', async () => {
    Reflect.deleteProperty(window, 'go')
    await useTerrainStore.getState().load('frangfurd')
    expect(useTerrainStore.getState().unavailable).toBe(true)
    expect(Bindings.GetDHSnapshot).not.toHaveBeenCalled()
    expect(useTerrainStore.getState().listen()).toBeTypeOf('function')
    expect(Runtime.EventsOn).not.toHaveBeenCalled()
  })

  it('ends a download done, with the answer Go gave after it', async () => {
    useTerrainStore.setState({ byChapter: { frangfurd: published } })
    const after = { ...published, existingBytes: published.sizeBytes }
    vi.mocked(Bindings.DownloadDHSnapshot).mockResolvedValueOnce(after as never)
    await useTerrainStore.getState().download('frangfurd', false)
    expect(Bindings.DownloadDHSnapshot).toHaveBeenCalledWith('frangfurd', false)
    expect(useTerrainStore.getState().progress.frangfurd?.phase).toBe('done')
    expect(useTerrainStore.getState().byChapter.frangfurd).toEqual(after)
  })

  it('rethrows a failure and records it', async () => {
    vi.mocked(Bindings.DownloadDHSnapshot).mockRejectedValueOnce(new Error('not enough free space'))
    vi.mocked(Bindings.GetDHSnapshot).mockResolvedValue(published as never)
    await expect(useTerrainStore.getState().download('frangfurd', false)).rejects.toThrow(
      'not enough free space',
    )
    const p = useTerrainStore.getState().progress.frangfurd
    expect(p?.phase).toBe('failed')
    expect(p?.error).toBe('not enough free space')
  })

  it('settles a download the player cancelled as cancelled, without throwing', async () => {
    let refuse!: (e: Error) => void
    vi.mocked(Bindings.DownloadDHSnapshot).mockReturnValueOnce(
      new Promise((_, reject) => (refuse = reject)) as never,
    )
    vi.mocked(Bindings.CancelDHSnapshot).mockResolvedValueOnce(true)
    vi.mocked(Bindings.GetDHSnapshot).mockResolvedValue(published as never)
    const running = useTerrainStore.getState().download('frangfurd', false)
    await useTerrainStore.getState().cancel('frangfurd')
    refuse(new Error('the download was cancelled'))
    await running
    expect(useTerrainStore.getState().progress.frangfurd?.phase).toBe('cancelled')
    expect(useTerrainStore.getState().cancelling.frangfurd).toBe(false)
  })

  it('forgets the cancel when Go refuses it', async () => {
    vi.mocked(Bindings.CancelDHSnapshot).mockRejectedValueOnce(new Error('no chapter'))
    await expect(useTerrainStore.getState().cancel('frangfurd')).rejects.toThrow('no chapter')
    expect(useTerrainStore.getState().cancelling.frangfurd).toBe(false)
  })

  it('files events by chapter and never reopens a settled download', () => {
    let hear!: (p: unknown) => void
    vi.mocked(Runtime.EventsOn).mockImplementation(((_: string, cb: (p: unknown) => void) => {
      hear = cb
      return () => undefined
    }) as never)
    const off = useTerrainStore.getState().listen()
    expect(Runtime.EventsOn).toHaveBeenCalledWith(EVENT_DH_DOWNLOAD, expect.any(Function))
    hear({ chapterId: 'frangfurd', phase: 'downloading', received: 5, total: 10 })
    expect(useTerrainStore.getState().progress.frangfurd?.received).toBe(5)
    hear({ phase: 'downloading', received: 6, total: 10 })
    expect(useTerrainStore.getState().progress.frangfurd?.received).toBe(5)
    // The promise, not the event, settles the outcome.
    hear({ chapterId: 'frangfurd', phase: 'done', received: 10, total: 10 })
    expect(useTerrainStore.getState().progress.frangfurd?.phase).toBe('downloading')
    useTerrainStore.setState({
      progress: { frangfurd: { chapterId: 'frangfurd', phase: 'done', received: 0, total: 10 } },
    })
    hear({ chapterId: 'frangfurd', phase: 'downloading', received: 7, total: 10 })
    expect(useTerrainStore.getState().progress.frangfurd?.phase).toBe('done')
    hear({ chapterId: 'frangfurd', phase: 'checking', received: 0, total: 10 })
    expect(useTerrainStore.getState().progress.frangfurd?.phase).toBe('checking')
    off()
    expect(Runtime.EventsOff).toHaveBeenCalledWith(EVENT_DH_DOWNLOAD)
  })
})
