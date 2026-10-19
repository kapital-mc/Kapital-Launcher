import { create } from 'zustand'
import type { PreviewSituation } from '../types'
import { hasWailsBridge, readOr } from '../lib/ipc'
import { ClearPreviews, GetPreviewSituations, StartPreview } from '../../wailsjs/go/main/App'
import { useEngineStore } from './useEngineStore'
import { useGameStore } from './useGameStore'
import { useTailscaleStore } from './useTailscaleStore'
import { useUpdateStore } from './useUpdateStore'

/**
 * The developer previews (#124): screens a run only shows when something goes
 * wrong, faked from the Developer section of Settings. Go pushes synthetic
 * state through the real paths (a `game:state` event, the loading card, the
 * pack, instance, engine, release and Tailscale reads, `prism:install` and
 * `tailscale:install`), so the real
 * components render their real copy. This store asks for one and reads what the
 * screens show again; the state itself lives in the stores that own it.
 *
 * Only the Developer section uses it, which is loaded on demand, so the store
 * is loaded with it and stays out of the launcher's first paint.
 */
interface PreviewStore {
  /** The fixed list of situations, from Go; empty without a bridge. */
  situations: PreviewSituation[]
  load: () => Promise<void>
  /** Shows a situation for a chapter. A write: a refusal is rethrown for the section to show. */
  start: (chapterId: string, situation: PreviewSituation) => Promise<void>
  /** Ends every preview and reads the real state again. */
  clear: () => Promise<void>
}

/** The install outcomes a preview leaves behind, which are not news after it. */
const FINISHED = ['done', 'failed']

/**
 * Reads again what a preview changes that no event carries: the engine, the
 * release, the instances with the pack state, the launcher's update and the
 * player's Tailscale. The release goes first, so a
 * made-up one never outlives its preview; it is read again where the launcher
 * would read it (App does the same on the engine).
 */
async function readViews() {
  useEngineStore.setState((s) => ({
    release: null,
    install: s.install && FINISHED.includes(s.install.phase) ? null : s.install,
  }))
  await useEngineStore.getState().load()
  const { engine, loadRelease } = useEngineStore.getState()
  if (engine && (!engine.found || engine.source === 'managed')) await loadRelease()
  // The launcher's own update notice, and the player's Tailscale, which its
  // previews make up; a made-up installer is dropped like Prism's release.
  await useUpdateStore.getState().load()
  useTailscaleStore.setState((s) => ({
    release: null,
    install: s.install && FINISHED.includes(s.install.phase) ? null : s.install,
  }))
  await useTailscaleStore.getState().load()
}

export const usePreviewStore = create<PreviewStore>((set) => ({
  situations: [],

  load: async () => {
    const raw: unknown = await readOr(GetPreviewSituations, [])
    set({ situations: Array.isArray(raw) ? (raw as PreviewSituation[]) : [] })
  },

  start: async (chapterId, situation) => {
    if (!hasWailsBridge()) throw new Error('Previews need the desktop app.')
    await StartPreview(chapterId, situation.id)
    await readViews()
    // The made-up install is the situation: it plays on its own, over
    // prism:install or tailscale:install, and fails.
    if (!situation.playsInstall) return
    if (situation.scope === 'launcher') void useTailscaleStore.getState().installTailscale()
    else void useEngineStore.getState().installPrism()
  },

  clear: async () => {
    if (!hasWailsBridge()) throw new Error('Previews need the desktop app.')
    await ClearPreviews()
    await readViews()
    await useGameStore.getState().load()
  },
}))
