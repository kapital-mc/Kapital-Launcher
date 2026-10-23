import { useEffect, useState } from 'react'
import { LAYOUT } from '../styles/tokens'
import { ms } from './smoothResize'

export interface WindowSize {
  width: number
  height: number
}

/**
 * Whether the window is too small for the full layout, so compact is held on
 * (docs/adr/0015-compact-launcher.md). Below the floor it holds; it lets go
 * only once the window is the hysteresis past both sides of it, so a size on
 * the line does not flip the layout back and forth.
 */
export function holdsCompact(
  size: WindowSize,
  held: boolean,
  floor: { width: number; height: number; hysteresis: number } = LAYOUT.compact,
): boolean {
  if (size.width < floor.width || size.height < floor.height) return true
  if (!held) return false
  return (
    size.width < floor.width + floor.hysteresis || size.height < floor.height + floor.hysteresis
  )
}

const windowSize = (): WindowSize => ({ width: window.innerWidth, height: window.innerHeight })

/**
 * The layout the window shows: compact when it was chosen or when the window
 * is too small for the full one. Shrinking past the floor takes compact at
 * once, mid-drag, so the full layout is never seen cut off; growing past it
 * waits for the drag to pause, so the layout does not change under the cursor
 * while the window is still moving.
 */
export function useCompactLayout(chosen: boolean): { compact: boolean; held: boolean } {
  const [held, setHeld] = useState(() => holdsCompact(windowSize(), false))
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined
    const onResize = () => {
      clearTimeout(timer)
      setHeld((was) => {
        const now = holdsCompact(windowSize(), was)
        if (now || !was) return now
        // Letting go waits for the size to settle.
        timer = setTimeout(() => setHeld(holdsCompact(windowSize(), true)), ms('--duration-reveal'))
        return was
      })
    }
    window.addEventListener('resize', onResize)
    return () => {
      clearTimeout(timer)
      window.removeEventListener('resize', onResize)
    }
  }, [])
  return { compact: chosen || held, held }
}
