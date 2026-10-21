import { create } from 'zustand'
import type { ChapterStorage } from '../types'
import { errMsg, hasWailsBridge } from '../lib/ipc'
import {
  ClearChapterStorage,
  GetChapterStorage,
  OpenScreenshotsFolder,
} from '../../wailsjs/go/main/App'

/**
 * A chapter's instance on disk by kind, as the storage section shows it. Go measures the folder and
 * does every delete; this is the last answer, read when the section opens, when the game ends and
 * when the window regains focus. Only the lazy section imports it, so none of it is in the
 * launcher's first paint.
 *
 * A delete is not drawn ahead of Go: the answer is the folder measured again, and a refusal is
 * rethrown for the section to show. With no bridge (the browser-only preview) nothing can be read.
 */
interface StorageStore {
  /** The last answer per chapter, absent before the first read. */
  byChapter: Record<string, ChapterStorage | undefined>
  /** True when there is no Wails bridge, so there is nothing to measure. */
  unavailable: boolean
  /** The reason a read was refused, by chapter. */
  readError: Record<string, string | undefined>
  /** The row whose delete is in flight, by its key (lib/storage's storageKey). */
  clearing: string | null
  load: (chapterId: string) => Promise<void>
  /** Deletes one kind of the chapter's storage; rejects with Go's reason. */
  clear: (chapterId: string, kind: string, name: string) => Promise<void>
  /** Shows the chapter's screenshots folder; rejects with Go's reason. */
  openScreenshots: (chapterId: string) => Promise<void>
}

export const useStorageStore = create<StorageStore>((set) => ({
  byChapter: {},
  unavailable: false,
  readError: {},
  clearing: null,

  load: async (chapterId) => {
    if (!hasWailsBridge()) {
      set({ unavailable: true })
      return
    }
    try {
      const storage = (await GetChapterStorage(chapterId)) as ChapterStorage
      set((s) => ({
        unavailable: false,
        byChapter: { ...s.byChapter, [chapterId]: storage },
        readError: { ...s.readError, [chapterId]: undefined },
      }))
    } catch (e) {
      set((s) => ({ readError: { ...s.readError, [chapterId]: errMsg(e) } }))
    }
  },

  clear: async (chapterId, kind, name) => {
    if (!hasWailsBridge()) return
    set({ clearing: `${kind}/${name}` })
    try {
      const storage = (await ClearChapterStorage(chapterId, kind, name)) as ChapterStorage
      set((s) => ({ clearing: null, byChapter: { ...s.byChapter, [chapterId]: storage } }))
    } catch (e) {
      set({ clearing: null })
      throw e
    }
  },

  openScreenshots: async (chapterId) => {
    if (!hasWailsBridge()) return
    await OpenScreenshotsFolder(chapterId)
  },
}))
