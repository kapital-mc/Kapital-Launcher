import { create } from 'zustand'
import type { UpdateInfo } from '../types'
import type { LauncherUpdateProgress } from '../types/update'
import { errMsg, readOr } from '../lib/ipc'
import {
  CancelUpdateDownload,
  DownloadUpdate,
  GetUpdate,
  OpenExternal,
  RestartToUpdate,
} from '../../wailsjs/go/main/App'
import { EventsOff, EventsOn } from '../../wailsjs/runtime/runtime'

/** Go's event for each step of the launcher's own update download. */
export const EVENT_LAUNCHER_UPDATE = 'launcher:update'

/** The steps in which the download is still going. */
export const UPDATE_WORKING = ['downloading', 'verifying']

/**
 * Whether a newer Kapital Launcher is out (ADR-9 amendment), and the launcher's
 * own update of itself (ADR-6 amendment, 2026-10-21). Go reads the download
 * site's version.json once a start and this store asks it once on mount; a
 * preview of the notice (#124) asks again. Not now hides the popup for the
 * rest of the run only, so it opens again on the next start until the
 * launcher is updated.
 *
 * Update downloads the new build in the background: the popup closes, the
 * sidebar's downloads show the bar, and once it is ready the header's button
 * turns to Restart. Where this copy cannot update itself (`blocked`), the
 * popup opens the download site as before.
 */
interface UpdateStore {
  /** Go's answer; null until asked, and without a bridge. */
  update: UpdateInfo | null
  /** Whether the player put the popup away this run. */
  dismissed: boolean
  /** Why the download page did not open, or the restart did not happen. */
  error: string | null
  /** The background download's latest step; null before Update and after a cancel. */
  progress: LauncherUpdateProgress | null
  load: () => Promise<void>
  dismiss: () => void
  /** Brings the popup back after Not now, from the header's update button. */
  reopen: () => void
  /** Opens the download site in the browser, through Go's link checks. */
  openPage: () => Promise<void>
  /** Starts the background download and puts the popup away. */
  download: () => Promise<void>
  /** Cancels the download while it runs. */
  cancelDownload: () => Promise<void>
  /** Swaps the downloaded build in and restarts into it. */
  restart: () => Promise<void>
  /** Follows Go's launcher:update events; returns the unsubscribe. */
  listen: () => () => void
  receive: (p: LauncherUpdateProgress) => void
}

/** Whether the player cancelled the download that runs, which then fails as no failure. */
let cancelling = false

export const useUpdateStore = create<UpdateStore>((set, get) => ({
  update: null,
  dismissed: false,
  error: null,
  progress: null,

  load: async () => {
    const update = await readOr(GetUpdate, null)
    set({ update, dismissed: false, error: null })
  },

  dismiss: () => set({ dismissed: true, error: null }),

  reopen: () => set({ dismissed: false, error: null }),

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

  download: async () => {
    const version = get().update?.latest ?? ''
    cancelling = false
    set({
      dismissed: true,
      error: null,
      progress: { phase: 'downloading', received: 0, total: 0, version, error: '' },
    })
    try {
      await DownloadUpdate()
    } catch (e) {
      // A cancel is no failure: the button goes back to Update.
      if (cancelling) {
        cancelling = false
        set({ progress: null })
        return
      }
      set({ progress: { phase: 'failed', received: 0, total: 0, version, error: errMsg(e) } })
    }
  },

  // Marked before Go is asked, as the download's own answer can come back first.
  cancelDownload: async () => {
    cancelling = true
    if (!(await CancelUpdateDownload())) cancelling = false
  },

  restart: async () => {
    set({ error: null })
    try {
      await RestartToUpdate()
    } catch (e) {
      set({ error: errMsg(e), dismissed: false })
    }
  },

  // One listener for the app's lifetime, held by the header, which is always on screen.
  listen: () => {
    EventsOn(EVENT_LAUNCHER_UPDATE, (p: LauncherUpdateProgress) => get().receive(p))
    return () => EventsOff(EVENT_LAUNCHER_UPDATE)
  },

  // A late download step must not undo ready, and a cancelled download is an offer again.
  receive: (p) => {
    const current = get().progress
    if (current?.phase === 'ready' && p.phase !== 'ready') return
    set({ progress: p.phase === 'cancelled' ? null : p })
  },
}))

/** Whether the popup is up: a newer build, not put away this run. */
export const selectUpdateShown = (s: UpdateStore) => !!s.update?.available && !s.dismissed

/** Whether the download is still going. */
export const selectUpdateWorking = (s: UpdateStore) =>
  !!s.progress && UPDATE_WORKING.includes(s.progress.phase)

/** Whether the new build is staged and Restart would swap it in. */
export const selectUpdateReady = (s: UpdateStore) => s.progress?.phase === 'ready'
