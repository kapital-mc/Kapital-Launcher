import { useEffect, useId, useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { X } from '../../lib/icons'
import { useEscapeClose } from '../../lib/useEscapeClose'
import { IconButton } from './IconButton'
import { Smooth } from './Smooth'

interface Props {
  /** The heading, which is also the dialog's accessible name. */
  title: string
  /** A small line under the title, in the data face: versions, sizes, a chapter's name. */
  facts?: string
  /** Escape, a click on the cover, the X when it has one, and whatever the body's own cancel does. */
  onCancel: () => void
  /**
   * The rectangle of what opened it, such as a column of the chapter card. A
   * surface in the card's colour then rises from it to the box, as if the
   * column itself lifted and grew, while the cover dims; the box's content
   * arrives from top to bottom a moment after it lands. Closing lowers the
   * surface back onto the column at once while the content fades and the cover
   * clears, and `onCancel` runs when it is down. Without one the box rises into
   * place and fades, and goes at once.
   */
  origin?: DOMRect
  /**
   * A copy of what `origin` is the rectangle of, taken as it was pressed: the
   * surface carries it, fading as it rises and coming back as it lowers. The
   * opener empties the original while the popup is open, so the column is what
   * seems to lift.
   */
  lifting?: HTMLElement
  /** The rectangle to centre the box on, such as the chapter card; the window without one. */
  frame?: DOMRect
  /** An X in the top right corner that closes it, for a popup with no question to answer. */
  closable?: boolean
  children: ReactNode
}

/** A rectangle as the custom properties the frame layer and the rise read, in px. */
function setRect(
  el: HTMLElement,
  prefix: string,
  r: { left: number; top: number; width?: number; height?: number },
) {
  el.style.setProperty(`--${prefix}-x`, `${r.left}px`)
  el.style.setProperty(`--${prefix}-y`, `${r.top}px`)
  if (r.width !== undefined) el.style.setProperty(`--${prefix}-w`, `${r.width}px`)
  if (r.height !== undefined) el.style.setProperty(`--${prefix}-h`, `${r.height}px`)
}

/**
 * A popup over the whole window: a modal dialog on a scrim, for a question the
 * player answers before carrying on (getting Prism, a newer launcher), or a
 * column of the chapter card shown whole. It is portalled to the body, so no
 * transformed ancestor clips it and nothing under it moves; the accent and
 * theme are on the root element and still apply.
 * Focus moves into it when it opens and back to where it was when it closes.
 */
export function Dialog({
  title,
  facts,
  onCancel,
  origin,
  lifting,
  frame,
  closable,
  children,
}: Props) {
  const layer = useRef<HTMLDivElement>(null)
  const ghost = useRef<HTMLDivElement>(null)
  const box = useRef<HTMLDivElement>(null)
  const surface = useRef<HTMLDivElement>(null)
  const titleId = useId()
  const [leaving, setLeaving] = useState(false)
  // The rise goes on only once its custom properties are set: an animation that
  // started without them would not pick them up.
  const [measured, setMeasured] = useState(false)
  // The surface has reached the box, which now stands alone.
  const [risen, setRisen] = useState(false)
  const close = origin
    ? () => {
        // The box may have grown since it opened (a body that loaded late): the surface
        // shrinks from where it is now.
        if (box.current && surface.current) {
          setRect(surface.current, 'rise-to', box.current.getBoundingClientRect())
        }
        setRisen(false)
        setLeaving(true)
      }
    : onCancel
  useEscapeClose(close)
  useEffect(() => {
    const before = document.activeElement
    box.current?.focus()
    return () => {
      if (before instanceof HTMLElement) before.focus()
    }
  }, [])
  // The frame first, so the box is measured where it will stand; then the surface's two
  // rectangles, the column's and the box's, all before the first paint.
  useLayoutEffect(() => {
    if (frame && layer.current) {
      setRect(layer.current, 'frame', frame)
    }
    if (!box.current || !surface.current || !origin) return
    setRect(surface.current, 'rise-from', origin)
    setRect(surface.current, 'rise-to', box.current.getBoundingClientRect())
    setMeasured(true)
  }, [origin, frame])
  // The copy of the column: a picture of it, inert and unnamed, so nothing in it can be
  // reached or found twice.
  useLayoutEffect(() => {
    const host = ghost.current
    if (!host || !lifting) return
    for (const el of [lifting, ...lifting.querySelectorAll('[id]')]) el.removeAttribute('id')
    host.replaceChildren(lifting)
    return () => host.replaceChildren()
  }, [lifting])

  // Without an origin the box rises into place as a page's body does. With one, the
  // surface does the moving and the box's content fades in once it is there.
  const motion = !origin
    ? 'reveal'
    : !measured
      ? 'opacity-0'
      : leaving
        ? 'fade-out'
        : 'rise-content-in'
  const lift = !measured ? 'opacity-0' : leaving ? 'rise-out' : risen ? 'invisible' : 'rise-in'
  // A column's popup is the chapter card's own surface lifted, so it keeps the card's colour.
  const tone = origin ? 'bg-raised border-line-strong' : 'bg-raised-2 border-line-strong'
  const place = frame ? 'left-(--frame-x) top-(--frame-y) h-(--frame-h) w-(--frame-w)' : 'inset-0'
  return createPortal(
    <div className="fixed inset-0 z-20">
      <div
        className={`scrim-wash absolute inset-0 ${
          !origin ? (leaving ? 'fade-out' : 'fade-in') : leaving ? 'wash-out' : 'wash-in'
        }`}
        onClick={close}
      />
      {origin && (
        <div
          ref={surface}
          aria-hidden
          onAnimationEnd={(e) => {
            if (e.target === e.currentTarget && leaving) onCancel()
          }}
          className={`${tone} pointer-events-none fixed overflow-hidden rounded-lg border ${lift}`}
        >
          {/* The copy keeps the column's size and sits where the column's content sat, so it
              moves with the surface without reflowing; the column's own divider stays behind. */}
          <div
            ref={ghost}
            inert
            className={`absolute -top-px -left-px h-(--rise-from-h) w-(--rise-from-w) *:h-full *:border-0 ${
              leaving ? 'lift-ghost-in' : 'lift-ghost-out'
            }`}
          />
        </div>
      )}
      <div
        ref={layer}
        className={`pointer-events-none absolute flex items-center justify-center p-10 ${place}`}
      >
        <Smooth
          ref={box}
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          tabIndex={-1}
          onAnimationEnd={(e) => {
            // The content is all there: the box now stands alone over the page.
            if (origin && e.target === e.currentTarget && !leaving) setRisen(true)
          }}
          className={`${tone} pointer-events-auto w-full max-w-120 rounded-lg border p-6 text-sm outline-none ${motion}`}
          inner="flex flex-col gap-4"
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
        </Smooth>
      </div>
    </div>,
    document.body,
  )
}
