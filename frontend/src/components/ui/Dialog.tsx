import { useEffect, useId, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { X } from '../../lib/icons'
import { useEscapeClose } from '../../lib/useEscapeClose'
import { IconButton } from './IconButton'

interface Props {
  /** The heading, which is also the dialog's accessible name. */
  title: string
  /** A small line under the title, in the data face: versions, sizes, a chapter's name. */
  facts?: string
  /** Escape, a click on the cover, the X when it has one, and whatever the body's own cancel does. */
  onCancel: () => void
  /**
   * The rectangle of what opened it. The box then grows out of it, and shrinks
   * back into it before `onCancel` runs; without one it rises into place and
   * fades, and goes at once.
   */
  origin?: DOMRect
  /** The rectangle to centre the box on, such as the chapter card; the window without one. */
  frame?: DOMRect
  /** An X in the top right corner that closes it, for a popup with no question to answer. */
  closable?: boolean
  children: ReactNode
}

/** A rectangle as the custom properties the frame layer and the grow read, in px. */
function setRect(el: HTMLElement, prefix: string, r: { left: number; top: number }) {
  el.style.setProperty(`--${prefix}-x`, `${r.left}px`)
  el.style.setProperty(`--${prefix}-y`, `${r.top}px`)
}

/**
 * A popup over the whole window: a modal dialog on a scrim, for a question the
 * player answers before carrying on (getting Prism, a newer launcher), or a
 * column of the chapter card shown whole. It is portalled to the body, so no
 * transformed ancestor clips it and nothing under it moves; the accent and
 * theme are on the root element and still apply.
 * Focus moves into it when it opens and back to where it was when it closes.
 */
export function Dialog({ title, facts, onCancel, origin, frame, closable, children }: Props) {
  const layer = useRef<HTMLDivElement>(null)
  const box = useRef<HTMLDivElement>(null)
  const titleId = useId()
  const [leaving, setLeaving] = useState(false)
  // The grow class goes on only once its custom properties are set: an animation that
  // started without them would not pick them up.
  const [measured, setMeasured] = useState(false)
  const close = origin ? () => setLeaving(true) : onCancel
  useEscapeClose(close)
  useEffect(() => {
    const before = document.activeElement
    box.current?.focus()
    return () => {
      if (before instanceof HTMLElement) before.focus()
    }
  }, [])
  // The frame first, so the box is measured where it will stand; then the origin's
  // offset and scale from the box's own rectangle, all before the first paint.
  useLayoutEffect(() => {
    const el = box.current
    if (frame && layer.current) {
      setRect(layer.current, 'frame', frame)
      layer.current.style.setProperty('--frame-w', `${frame.width}px`)
      layer.current.style.setProperty('--frame-h', `${frame.height}px`)
    }
    if (!el || !origin) return
    const r = el.getBoundingClientRect()
    if (r.width > 0 && r.height > 0) {
      setRect(el, 'grow', { left: origin.left - r.left, top: origin.top - r.top })
      el.style.setProperty('--grow-sx', `${origin.width / r.width}`)
      el.style.setProperty('--grow-sy', `${origin.height / r.height}`)
    }
    setMeasured(true)
  }, [origin, frame])

  // Without an origin the box rises into place as a page's body does, and the cover fades in.
  const motion = !origin ? 'reveal' : !measured ? 'opacity-0' : leaving ? 'grow-out' : 'grow-in'
  const place = frame ? 'left-(--frame-x) top-(--frame-y) h-(--frame-h) w-(--frame-w)' : 'inset-0'
  return createPortal(
    <div
      className={`scrim-wash fixed inset-0 z-20 ${leaving ? 'fade-out' : 'fade-in'}`}
      onClick={(e) => {
        if (e.target === e.currentTarget) close()
      }}
    >
      <div
        ref={layer}
        className={`pointer-events-none absolute flex items-center justify-center p-10 ${place}`}
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
          className={`bg-raised-2 border-line-strong pointer-events-auto flex w-full max-w-120 flex-col gap-4 rounded-lg border p-6 text-sm outline-none ${motion}`}
        >
          <div className="flex items-start justify-between gap-4">
            <div className="flex min-w-0 flex-col gap-1">
              <h2 id={titleId} className="font-display m-0 text-xl font-semibold">
                {title}
              </h2>
              {facts && <span className="text-fg-muted font-mono text-xs">{facts}</span>}
            </div>
            {closable && (
              <div className="-mt-1 -mr-2">
                <IconButton icon={X} size="sm" title="Close" onClick={close} />
              </div>
            )}
          </div>
          {children}
        </div>
      </div>
    </div>,
    document.body,
  )
}
