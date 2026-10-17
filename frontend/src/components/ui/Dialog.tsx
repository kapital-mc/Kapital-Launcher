import { useEffect, useId, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { useEscapeClose } from '../../lib/useEscapeClose'

interface Props {
  /** The heading, which is also the dialog's accessible name. */
  title: string
  /** A small line under the title, in the data face: versions, sizes. */
  facts?: string
  /** Escape, a click on the cover, and whatever the body's own cancel does. */
  onCancel: () => void
  /**
   * The rectangle of what opened it. The box then grows out of it, and shrinks
   * back into it before `onCancel` runs; without one it simply appears.
   */
  origin?: DOMRect
  /** The body, or a function given the close that animates first, for its own Close button. */
  children: ReactNode | ((close: () => void) => ReactNode)
}

/**
 * A popup over the whole window: a modal dialog on a scrim, for a question the
 * player answers before carrying on (getting Prism, a newer launcher), or a
 * column of the chapter card shown whole. It is portalled to the body, so no
 * transformed ancestor clips it and nothing under it moves; the accent and
 * theme are on the root element and still apply.
 * Focus moves into it when it opens and back to where it was when it closes.
 */
export function Dialog({ title, facts, onCancel, origin, children }: Props) {
  const box = useRef<HTMLDivElement>(null)
  const titleId = useId()
  const [leaving, setLeaving] = useState(false)
  const close = origin ? () => setLeaving(true) : onCancel
  useEscapeClose(close)
  useEffect(() => {
    const before = document.activeElement
    box.current?.focus()
    return () => {
      if (before instanceof HTMLElement) before.focus()
    }
  }, [])
  // Where the grow starts and ends: the origin's offset and scale from the box's
  // own rectangle, set before the first paint so the box never shows full size first.
  useLayoutEffect(() => {
    const el = box.current
    if (!el || !origin) return
    const r = el.getBoundingClientRect()
    if (r.width === 0 || r.height === 0) return
    el.style.setProperty('--grow-x', `${origin.left - r.left}px`)
    el.style.setProperty('--grow-y', `${origin.top - r.top}px`)
    el.style.setProperty('--grow-sx', `${origin.width / r.width}`)
    el.style.setProperty('--grow-sy', `${origin.height / r.height}`)
  }, [origin])

  const motion = origin ? (leaving ? 'grow-out' : 'grow-in') : ''
  return createPortal(
    <div
      className={`scrim-wash fixed inset-0 z-20 flex items-center justify-center p-10 ${
        origin ? (leaving ? 'fade-out' : 'fade-in') : ''
      }`}
      onClick={(e) => {
        if (e.target === e.currentTarget) close()
      }}
    >
      <div
        ref={box}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        onAnimationEnd={(e) => {
          if (leaving && e.target === e.currentTarget) onCancel()
        }}
        className={`bg-raised-2 border-line-strong flex w-full max-w-120 flex-col gap-4 rounded-lg border p-6 text-sm outline-none ${motion}`}
      >
        <div className="flex flex-col gap-1">
          <h2 id={titleId} className="font-display m-0 text-xl font-semibold">
            {title}
          </h2>
          {facts && <span className="text-fg-muted font-mono text-xs">{facts}</span>}
        </div>
        {typeof children === 'function' ? children(close) : children}
      </div>
    </div>,
    document.body,
  )
}
