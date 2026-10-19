import { useState } from 'react'
import type { ReactNode } from 'react'
import { fade } from './fade'

interface Props {
  /** What to show; nothing (null, undefined, false or '') folds it away. */
  children: ReactNode
  /**
   * For the place in a parent with a gap: the gap taken back, such as `-mt-3` in a `gap-3`
   * column, so the gap folds away with the content rather than staying behind.
   */
  className?: string
  /** The gap given back inside the fold, such as `pt-3`, so it opens and closes with the content. */
  inner?: string
}

/**
 * A line or a block that comes and goes under a control (a hint, a prompt, a
 * progress line, a field): the space opens, then the content fades in, and on
 * the way out it fades quickly while the space closes. It moves a one-track grid
 * between 1fr and 0fr as `Fold` does, over the reveal's 200 ms on a fast-out
 * curve, and keeps the last content while it folds, so a line that goes away is
 * seen leaving, then lets it go. What is there on the first render is simply
 * there.
 *
 * Folded, it is inert and hidden from assistive technology. A parent's gap does not fold on its own: give it back
 * with `className` and `inner`.
 */
export function Grow({ children, className = '', inner = '' }: Props) {
  const shown = children !== null && children !== undefined && children !== false && children !== ''
  // The last content shown, kept while it folds away.
  const [held, setHeld] = useState<ReactNode>(shown ? children : null)
  if (shown && held !== children) setHeld(children)
  return (
    <div
      inert={!shown}
      aria-hidden={!shown || undefined}
      onTransitionEnd={(e) => {
        // Folded all the way: what it held goes.
        if (!shown && e.target === e.currentTarget) setHeld(null)
      }}
      className={`duration-reveal grid transition-[grid-template-rows] ease-out motion-reduce:transition-none ${
        shown ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'
      } ${className}`}
    >
      {/* The clip reaches 4px past the content (the focus ring's width and offset), so a
          control's ring at the edge is drawn whole. */}
      <div className={`-m-1 min-h-0 overflow-hidden p-1 ${fade(shown)}`}>
        <div className={inner}>{shown ? children : held}</div>
      </div>
    </div>
  )
}
