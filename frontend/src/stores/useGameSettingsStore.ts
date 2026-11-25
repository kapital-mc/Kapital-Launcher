import { create } from 'zustand'
import type { ChapterGameSettings } from '../types'
import { errMsg, hasWailsBridge } from '../lib/ipc'
import { GetGameSettings, ResetGameSetting } from '../../wailsjs/go/main/App'

/**
 * A chapter's resettable game settings (ADR-2, twenty-fifth amendment), as the Instance section
 * shows them above Reinstall. Go lists the items and does every reset; this is the last answer,
 * read when the section opens, when the game ends and when the window regains focus. Only the lazy
 * settings page imports it, so none of it is in the launcher's first paint.
 *
 * A reset is not drawn ahead of Go: the answer is the list read again, and a refusal is rethrown
 * for the section to show. With no bridge (the browser-only preview) nothing can be read.
 */
interface GameSettingsStore {
  /** The last answer per chapter, absent before the first read. */
  byChapter: Record<string, ChapterGameSettings | undefined>
  /** True when there is no Wails bridge, so there is nothing to read. */
  unavailable: boolean
  /** The reason a read was refused, by chapter. */
  readError: Record<string, string | undefined>
  /** The item whose reset is in flight. */
  resetting: string | null
  load: (chapterId: string) => Promise<void>
  /** Resets one item; rejects with Go's reason. */
  reset: (chapterId: string, id: string) => Promise<void>
}

export const useGameSettingsStore = create<GameSettingsStore>((set) => ({
  byChapter: {},
  unavailable: false,
  readError: {},
  resetting: null,

  load: async (chapterId) => {
    if (!hasWailsBridge()) {
      set({ unavailable: true })
      return
    }
    try {
      const view = (await GetGameSettings(chapterId)) as ChapterGameSettings
      set((s) => ({
        unavailable: false,
        byChapter: { ...s.byChapter, [chapterId]: view },
        readError: { ...s.readError, [chapterId]: undefined },
      }))
    } catch (e) {
      set((s) => ({ readError: { ...s.readError, [chapterId]: errMsg(e) } }))
    }
  },

  reset: async (chapterId, id) => {
    if (!hasWailsBridge()) return
    set({ resetting: id })
    try {
      const view = (await ResetGameSetting(chapterId, id)) as ChapterGameSettings
      set((s) => ({ resetting: null, byChapter: { ...s.byChapter, [chapterId]: view } }))
    } catch (e) {
      set({ resetting: null })
      throw e
    }
  },
}))
