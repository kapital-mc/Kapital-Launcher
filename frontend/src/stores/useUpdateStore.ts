import { create } from 'zustand'
import type { UpdateInfo } from '../types'
import { errMsg, readOr } from '../lib/ipc'
import { GetUpdate, OpenExternal } from '../../wailsjs/go/main/App'

/**
 * Whether a newer Kapital Launcher is out (ADR-9 amendment). Go reads the
 * download site's version.json once a start and this store asks it once on
 * mount; a preview of the notice (#124) asks again. Not now hides the popup
 * for the rest of the run only, so it opens again on the next start until the
 * launcher is updated.
 */
interface UpdateStore {
  /** Go's answer; null until asked, and without a bridge. */
  update: UpdateInfo | null
  /** Whether the player put the popup away this run. */
  dismissed: boolean
  /** Why the download page did not open, for the popup to show. */
  error: string | null
  load: () => Promise<void>
  dismiss: () => void
  /** Opens the download site in the browser, through Go's link checks. */
  openPage: () => Promise<void>
}

export const useUpdateStore = create<UpdateStore>((set, get) => ({
  update: null,
  dismissed: false,
  error: null,

  load: async () => {
    const update = await readOr(GetUpdate, null)
    set({ update, dismissed: false, error: null })
  },

  dismiss: () => set({ dismissed: true, error: null }),

  openPage: async () => {
    const page = get().update?.page
    if (!page) return
    try {
      await OpenExternal(page)
      set({ dismissed: true, error: null })
    } catch (e) {
      set({ error: errMsg(e) })
    }
  },
}))

/** Whether the popup is up: a newer build, not put away this run. */
export const selectUpdateShown = (s: UpdateStore) => !!s.update?.available && !s.dismissed
