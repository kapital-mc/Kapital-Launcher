import { useEffect, useId, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'

/**
 * A menu under its button: the header's sidebar button's and the chapter tools' ⋯ column. Its first
 * item takes the focus when it opens; Escape closes it and gives the focus back to the button, Tab
 * or a press outside it closes it. `M` is the menu's element: a span where it sits in a `Fold`.
 */
export function useMenu<M extends HTMLElement = HTMLDivElement>() {
  const [open, setOpen] = useState(false)
  const wrap = useRef<HTMLDivElement>(null)
  const menu = useRef<M>(null)
  const id = useId()

  useEffect(() => {
    if (!open) return
    // Without preventScroll the browser scrolls a folded menu's clip to show the item, and the
    // column slides instead of unrolling.
    menu.current
      ?.querySelector<HTMLButtonElement>('[role="menuitem"]')
      ?.focus({ preventScroll: true })
    const onDown = (e: PointerEvent) => {
      if (!wrap.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('pointerdown', onDown)
    return () => document.removeEventListener('pointerdown', onDown)
  }, [open])

  const close = () => {
    setOpen(false)
    wrap.current?.querySelector<HTMLButtonElement>('[aria-haspopup="menu"]')?.focus()
  }
  const onKeyDown = (e: KeyboardEvent<M>) => {
    if (e.key === 'Escape') {
      // The menu closes, not a page.
      e.stopPropagation()
      close()
    } else if (e.key === 'Tab') setOpen(false)
  }
  return { open, toggle: () => setOpen((o) => !o), close, wrap, menu, id, onKeyDown }
}
