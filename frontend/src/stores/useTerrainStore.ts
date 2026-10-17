import { create } from 'zustand'
import type { DHDownloadProgress, DHSnapshotInfo } from '../types'
import { errMsg, hasWailsBridge } from '../lib/ipc'
import { isSettled } from '../lib/terrain'
import { CancelDHSnapshot, DownloadDHSnapshot, GetDHSnapshot } from '../../wailsjs/go/main/App'
import { EventsOff, EventsOn } from '../../wailsjs/runtime/runtime'

/** Go's event name for a terrain download's progress (services.EventDHDownload). */
export const EVENT_DH_DOWNLOAD = 'dh:download'

/**
 * The server's Distant Horizons terrain per chapter (issue 136): what is published and what the
 * instance holds, read when the chapter's Server settings open, and the progress of a download
 * the player started. Go owns the download and the files; the promise settles its outcome and
 * dh:download events fill in the bar while it runs, heard only while the card is on screen (the
 * card subscribes and unsubscribes, as the logs page does for log:live). Only the lazy settings
 * page imports it, so none of it is in the launcher's first paint.
 */
interface TerrainStore {
  /** The last answer per chapter, absent before the first read. */
  byChapter: Record<string, DHSnapshotInfo | undefined>
  /** The latest progress per chapter, absent before a download. */
  progress: Record<string, DHDownloadProgress | undefined>
  /** True when there is no Wails bridge, so nothing can be read. */
  unavailable: boolean
  /** Go's refusal of the last read, per chapter. */
  readError: Record<string, string | undefined>
  /** Per chapter, whether the player cancelled the download that is running. */
  cancelling: Record<string, boolean | undefined>
  /** Reads the chapter's snapshot; a refusal is kept in readError, never thrown. */
  load: (chapterId: string) => Promise<void>
  /** Downloads the chapter's terrain; rejects with Go's reason, except when the player cancelled. */
  download: (chapterId: string, replace: boolean) => Promise<void>
  cancel: (chapterId: string) => Promise<void>
  /** Subscribes to dh:download, returning the unsubscribe. */
  listen: () => () => void
  receive: (p: DHDownloadProgress) => void
}

export const useTerrainStore = create<TerrainStore>((set, get) => ({
  byChapter: {},
  progress: {},
  unavailable: false,
  readError: {},
  cancelling: {},

  load: async (chapterId) => {
    if (!hasWailsBridge()) {
      set({ unavailable: true })
      return
    }
    try {
      const info = (await GetDHSnapshot(chapterId)) as DHSnapshotInfo
      set((s) => ({
        unavailable: false,
        byChapter: { ...s.byChapter, [chapterId]: info },
        readError: { ...s.readError, [chapterId]: undefined },
      }))
    } catch (e) {
      set((s) => ({ readError: { ...s.readError, [chapterId]: errMsg(e) } }))
    }
  },

  download: async (chapterId, replace) => {
    const info = get().byChapter[chapterId]
    const start: DHDownloadProgress = {
      chapterId,
      phase: 'checking',
      received: info?.partialBytes ?? 0,
      total: info?.sizeBytes ?? 0,
    }
    set((s) => ({
      progress: { ...s.progress, [chapterId]: start },
      cancelling: { ...s.cancelling, [chapterId]: false },
    }))
    try {
      const after = (await DownloadDHSnapshot(chapterId, replace)) as DHSnapshotInfo
      set((s) => ({
        byChapter: { ...s.byChapter, [chapterId]: after },
        progress: { ...s.progress, [chapterId]: { ...start, phase: 'done', received: 0 } },
      }))
    } catch (e) {
      // Go answers a cancelled download with an error too; the player asked for that one.
      const cancelled = get().cancelling[chapterId] === true
      set((s) => ({
        progress: {
          ...s.progress,
          [chapterId]: cancelled
            ? { ...start, phase: 'cancelled', received: 0 }
            : { ...start, phase: 'failed', received: 0, error: errMsg(e) },
        },
        cancelling: { ...s.cancelling, [chapterId]: false },
      }))
      void get().load(chapterId)
      if (!cancelled) throw e
    }
  },

  cancel: async (chapterId) => {
    set((s) => ({ cancelling: { ...s.cancelling, [chapterId]: true } }))
    try {
      await CancelDHSnapshot(chapterId)
    } catch (e) {
      set((s) => ({ cancelling: { ...s.cancelling, [chapterId]: false } }))
      throw e
    }
  },

  listen: () => {
    if (!hasWailsBridge()) return () => undefined
    EventsOn(EVENT_DH_DOWNLOAD, (p: DHDownloadProgress) => get().receive(p))
    return () => EventsOff(EVENT_DH_DOWNLOAD)
  },

  // An event names its chapter, and a late one never reopens a settled download.
  receive: (p) => {
    if (!p?.chapterId || !p.phase) return
    const current = get().progress[p.chapterId]
    if (current && isSettled(current.phase) && p.phase !== 'checking') return
    if (isSettled(p.phase)) return
    set((s) => ({ progress: { ...s.progress, [p.chapterId]: p } }))
  },
}))
