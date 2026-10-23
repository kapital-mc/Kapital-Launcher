import { useCallback, useEffect, useRef, useState } from 'react'
import { ms } from './smoothResize'

interface PanelLean {
  /** The panel leaning toward the open one, or null. Never the open one itself. */
  leaning: string | null
  /** The pointer came onto a panel: it leans once it has rested there for `duration.lean`. */
  enter: (id: string) => void
  /** The pointer left a panel: it stops leaning at once. */
  leave: (id: string) => void
}

/**
 * Which compact panel leans toward the open one (docs/adr/0015-compact-launcher.md).
 * One state for the whole row, so the panel that leans and the open panel that
 * gives it the room change together: moving from one panel to the next lets
 * the first go before the second leans, and the others never move.
 */
export function usePanelLean(selectedId: string): PanelLean {
  const [resting, setResting] = useState<string | null>(null)
  const timer = useRef<number | undefined>(undefined)

  const enter = useCallback((id: string) => {
    window.clearTimeout(timer.current)
    setResting(null)
    timer.current = window.setTimeout(() => setResting(id), ms('--duration-lean'))
  }, [])
  const leave = useCallback((id: string) => {
    window.clearTimeout(timer.current)
    setResting((r) => (r === id ? null : r))
  }, [])
  useEffect(() => () => window.clearTimeout(timer.current), [])

  return { leaning: resting === selectedId ? null : resting, enter, leave }
}
