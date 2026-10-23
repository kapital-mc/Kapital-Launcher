import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import type { RefObject } from 'react'
import type { Morph } from './compactMorphRun'
import { reducedMotion } from './smoothResize'

export type LayoutMode = 'full' | 'compact'

interface Layout {
  /** The layout on screen, which changes when a morph has arrived. */
  mode: LayoutMode
  /** The layout a morph is carrying the window into; null when none runs. */
  morphTo: LayoutMode | null
}

/** Every picture in `root` decoded, so the morph measures them at their own size and drops no frame. */
const decoded = (root: HTMLElement) =>
  Promise.all([...root.querySelectorAll('img')].map((i) => i.decode().catch(() => undefined)))

/**
 * Which layout the window shows, and the morph between them
 * (lib/compactMorph). A change of `compact` plays the morph once the compact
 * layout is in the page (`stage`, which loads on demand) and its pictures are
 * decoded; the layout swaps when it has arrived, and the actors are taken down
 * in the same frame. A change while one runs waits for it, then plays from
 * there. Before `ready` (the settings not yet read and painted), with reduced
 * motion, in a hidden window or where a piece is missing, the layout swaps at
 * once.
 */
export function useLayoutMorph(
  compact: boolean,
  ready: boolean,
  area: RefObject<HTMLElement | null>,
  layer: RefObject<HTMLElement | null>,
  stage: HTMLElement | null,
): Layout {
  const target: LayoutMode = compact ? 'compact' : 'full'
  const [mode, setMode] = useState<LayoutMode>(target)
  const [morphTo, setMorphTo] = useState<LayoutMode | null>(null)
  const running = useRef<Morph | null>(null)

  // Before the first paint with the saved settings, a change is where the window starts.
  if (!ready && morphTo === null && mode !== target) setMode(target)

  useEffect(() => {
    if (ready && morphTo === null && mode !== target) setMorphTo(target)
  }, [ready, morphTo, mode, target])

  useEffect(() => {
    const root = area.current
    const actors = layer.current
    if (morphTo === null || !stage || !root || !actors) return
    let live = true
    const arrive = () => {
      if (!live) return
      setMode(morphTo)
      setMorphTo(null)
    }
    const instant =
      reducedMotion() || document.visibilityState === 'hidden' || typeof root.animate !== 'function'
    if (instant) {
      arrive()
      return
    }
    // The morph's code loads with its first run, out of the launcher's first paint.
    void Promise.all([import('./compactMorphRun'), decoded(stage)])
      .then(([run]) => new Promise<typeof run>((r) => requestAnimationFrame(() => r(run))))
      .then(({ runMorph }) => {
        if (!live) return
        const m = runMorph(root, actors, morphTo === 'compact')
        if (!m) return arrive()
        running.current = m
        // A morph cut short (the app unmounting) rejects; there is nothing left to swap.
        return m.finished.then(arrive, () => undefined)
      })
    return () => {
      live = false
    }
  }, [morphTo, stage, area, layer])

  // The layout has swapped: the actors go before this frame paints.
  useLayoutEffect(() => {
    running.current?.end()
    running.current = null
  }, [mode])
  useEffect(() => () => running.current?.end(), [])

  return { mode, morphTo }
}
