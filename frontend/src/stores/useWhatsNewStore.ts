import { create } from 'zustand'
import type { Changelog, WhatsNew } from '../types'
import { errMsg, readOr } from '../lib/ipc'
import {
  GetWhatsNew,
  GetWhatsNewChangelog,
  MarkWhatsNewSeen,
  OpenExternal,
} from '../../wailsjs/go/main/App'

/**
 * What's new on the first start after an update. Go picks the releases from
 * the highlights built into the app and the version last seen; this store asks
 * once on mount, and again under the launcher's What's new preview (#124). The
 * full changelog is read from the download site only when the player opens it.
 * Loaded with the popup's own chunk, so it stays out of the first paint.
 */
interface WhatsNewStore {
  /** Go's answer; null until asked, without a bridge, and once closed. */
  whatsNew: WhatsNew | null
  /** The full changelog; null until opened. */
  changelog: Changelog | null
  /** Why closing or opening the download page failed, for the popup to show. */
  error: string | null
  load: () => Promise<void>
  /** Reads the full changelog, once it is opened. */
  loadChangelog: () => Promise<void>
  /** Records this version as seen and puts the popup away. Stays up, with the error, if Go refuses. */
  close: () => Promise<void>
  /** Opens the download site, whose Changelog has every release, when the changelog was not read. */
  openPage: () => Promise<void>
}

export const useWhatsNewStore = create<WhatsNewStore>((set, get) => ({
  whatsNew: null,
  changelog: null,
  error: null,

  load: async () => {
    const whatsNew = await readOr(GetWhatsNew, null)
    set({ whatsNew, changelog: null, error: null })
  },

  loadChangelog: async () => {
    if (get().changelog?.releases) return
    const changelog = await readOr(GetWhatsNewChangelog, null)
    set({ changelog: changelog ?? { releases: null, reason: 'no answer', page: '' } })
  },

  close: async () => {
    try {
      await MarkWhatsNewSeen()
      set({ whatsNew: null, error: null })
    } catch (e) {
      set({ error: errMsg(e) })
    }
  },

  openPage: async () => {
    const page = get().changelog?.page
    if (!page) return
    try {
      await OpenExternal(page)
    } catch (e) {
      set({ error: errMsg(e) })
    }
  },
}))

/** Whether the popup is up: something new, not put away. */
export const selectWhatsNewShown = (s: WhatsNewStore) => (s.whatsNew?.releases?.length ?? 0) > 0
