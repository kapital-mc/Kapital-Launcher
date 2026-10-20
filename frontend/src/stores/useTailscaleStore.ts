import { create } from 'zustand'
import type { TailscaleInfo, TailscaleInstallProgress, TailscaleRelease } from '../types/tailscale'
import { errMsg, hasWailsBridge, readOr } from '../lib/ipc'
import {
  CancelTailscaleInstall,
  ConnectTailscale,
  GetTailscale,
  GetTailscaleRelease,
  InstallTailscale,
  OpenTailscale,
} from '../../wailsjs/go/main/App'
import { EventsOff, EventsOn } from '../../wailsjs/runtime/runtime'

/** Same strings as services.EventTailscaleState and services.EventTailscaleInstall. */
export const EVENT_TAILSCALE_STATE = 'tailscale:state'
export const EVENT_TAILSCALE_INSTALL = 'tailscale:install'

/** The install phases in which something is still happening. */
export const TAILSCALE_WORKING = ['downloading', 'verifying', 'installing']

/**
 * The player's Tailscale: whether it is installed and connected, the installer
 * the launcher would download, and an install in progress. The state is read
 * when the settings page opens; after an install, a sign-in, Connect or Open
 * Tailscale Go watches it and emits tailscale:state, so nothing here polls.
 */
interface TailscaleStore {
  info: TailscaleInfo | null
  release: TailscaleRelease | null
  install: TailscaleInstallProgress | null
  error: string | null
  load: () => Promise<void>
  loadRelease: () => Promise<void>
  /** A write the player approved; progress arrives as events, the outcome from the promise. */
  installTailscale: () => Promise<void>
  /** Cancels the install while it downloads; it goes back to being an offer. */
  cancelInstall: () => Promise<void>
  /**
   * A developer preview asks for the install popup (#124): the row opens it on its next render
   * and takes the request back.
   */
  offerPending: boolean
  /** The next step towards reaching the chapter's server; Go picks it, the state follows as events. */
  connect: (chapterId: string) => Promise<void>
  open: () => Promise<void>
  listen: () => () => void
  receiveInstall: (p: TailscaleInstallProgress) => void
}

/** Whether the player cancelled the install that runs, which then fails as no failure. */
let cancelled = false

export const useTailscaleStore = create<TailscaleStore>((set, get) => ({
  info: null,
  release: null,
  install: null,
  error: null,
  offerPending: false,

  load: async () => {
    set({ info: (await readOr(GetTailscale, null)) ?? null })
  },

  loadRelease: async () => {
    set({ release: (await readOr(GetTailscaleRelease, null)) ?? null })
  },

  installTailscale: async () => {
    set({
      install: { phase: 'downloading', received: 0, total: get().release?.size ?? 0, error: '' },
    })
    cancelled = false
    try {
      await InstallTailscale()
      set({ install: { phase: 'done', received: 0, total: 0, error: '' } })
      await get().load()
    } catch (e) {
      set({
        install: cancelled ? null : { phase: 'failed', received: 0, total: 0, error: errMsg(e) },
      })
    }
  },

  // Marked before Go is asked, as the install's own answer can come back first.
  cancelInstall: async () => {
    cancelled = true
    try {
      cancelled = await CancelTailscaleInstall()
    } catch (e) {
      cancelled = false
      set({ error: errMsg(e) })
    }
  },

  connect: async (chapterId) => {
    set({ error: null })
    try {
      await ConnectTailscale(chapterId)
      await get().load()
    } catch (e) {
      set({ error: errMsg(e) })
    }
  },

  open: async () => {
    set({ error: null })
    try {
      await OpenTailscale()
    } catch (e) {
      set({ error: errMsg(e) })
    }
  },

  // One pair of listeners for the app's lifetime, like prism:install.
  listen: () => {
    if (!hasWailsBridge()) return () => undefined
    EventsOn(EVENT_TAILSCALE_STATE, (info: TailscaleInfo) => set({ info }))
    EventsOn(EVENT_TAILSCALE_INSTALL, (p: TailscaleInstallProgress) => get().receiveInstall(p))
    return () => {
      EventsOff(EVENT_TAILSCALE_STATE)
      EventsOff(EVENT_TAILSCALE_INSTALL)
    }
  },

  // A late event must not reopen a finished install, and a cancelled one is an offer again.
  receiveInstall: (p) => {
    const current = get().install
    if (!p?.phase || current === null || current.phase === 'done' || current.phase === 'failed')
      return
    set({ install: p.phase === 'cancelled' ? null : p })
  },
}))

/** Whether the store's install is still going. */
export const tailscaleWorking = (p: TailscaleInstallProgress | null) =>
  p !== null && TAILSCALE_WORKING.includes(p.phase)
