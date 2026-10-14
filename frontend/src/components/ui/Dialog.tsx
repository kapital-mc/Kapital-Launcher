import { useEffect, useId, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { useEscapeClose } from '../../lib/useEscapeClose'

interface Props {
  /** The heading, which is also the dialog's accessible name. */
  title: string
  /** A small line under the title, in the data face: versions, sizes. */
  facts?: string
  /** Escape, a click on the cover, and whatever the body's own cancel does. */
  onCancel: () => void
  children: ReactNode
}

/**
 * A popup over the whole window: a modal dialog on a scrim, for a question the
 * player answers before carrying on (getting Prism, a newer launcher). It is
 * portalled to the body, so no transformed ancestor clips it and nothing under
 * it moves; the accent and theme are on the root element and still apply.
 * Focus moves into it when it opens and back to where it was when it closes.
 */
export function Dialog({ title, facts, onCancel, children }: Props) {
  const box = useRef<HTMLDivElement>(null)
  const titleId = useId()
  useEscapeClose(onCancel)
  useEffect(() => {
    const before = document.activeElement
    box.current?.focus()
    return () => {
      if (before instanceof HTMLElement) before.focus()
    }
  }, [])

  return createPortal(
    <div
      className="scrim-wash fixed inset-0 z-20 flex items-center justify-center p-10"
      onClick={(e) => {
        if (e.target === e.currentTarget) onCancel()
      }}
    >
      <div
        ref={box}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className="bg-raised-2 border-line-strong flex w-full max-w-120 flex-col gap-4 rounded-lg border p-6 text-sm outline-none"
      >
        <div className="flex flex-col gap-1">
          <h2 id={titleId} className="font-display m-0 text-xl font-semibold">
            {title}
          </h2>
          {facts && <span className="text-fg-muted font-mono text-xs">{facts}</span>}
        </div>
        {children}
      </div>
    </div>,
    document.body,
  )
}
