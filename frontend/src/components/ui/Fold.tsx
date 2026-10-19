import type { ReactNode } from 'react'
import { fade } from './fade'

interface Props {
  /** Shown, or folded away to nothing. */
  open: boolean
  /**
   * `y` folds the height away. `x` folds the width, and the height with it, so content taller
   * than its row's other items leaves the row as short as they are.
   */
  axis: 'x' | 'y'
  /** For the fold's place in its parent, such as `min-w-0 grow` in a row. */
  className?: string
  children: ReactNode
}

/**
 * Folds its content to nothing along one axis by moving a one-track grid
 * between 1fr and 0fr, so content of any size folds smoothly without being
 * measured. The content fades as it goes (`fade`). Folded, it is inert: out
 * of the tab order and the accessibility tree, so give a control that keeps
 * its place a name of its own. It is drawn with spans, so it may sit inside a
 * button.
 */
export function Fold({ open, axis, className = '', children }: Props) {
  const track =
    axis === 'x'
      ? `transition-[grid-template-columns,grid-template-rows] ${
          open ? 'grid-cols-[1fr] grid-rows-[1fr]' : 'grid-cols-[0fr] grid-rows-[0fr]'
        }`
      : `transition-[grid-template-rows] ${open ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'}`
  return (
    <span
      inert={!open}
      className={`duration-slow ease-glide grid motion-reduce:transition-none ${track} ${className}`}
    >
      <span className={`block min-h-0 min-w-0 overflow-hidden ${fade(open)}`}>{children}</span>
    </span>
  )
}
