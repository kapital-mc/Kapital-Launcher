import { useEffect, useId, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'

/**
 * A small menu under its button: the header's sidebar button and the chapter tools' ⋯. Its first
 * item takes the focus when it opens; Escape closes it and gives the focus back to the button, Tab
 * or a press outside it closes it.
 */
export function useMenu() {
  const [open, setOpen] = useState(false)
  const wrap = useRef<HTMLDivElement>(null)
  const menu = useRef<HTMLDivElement>(null)
  const id = useId()

  useEffect(() => {
    if (!open) return
    menu.current?.querySelector<HTMLButtonElement>('[role="menuitem"]')?.focus()
    const onDown = (e: PointerEvent) => {
      if (!wrap.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('pointerdown', onDown)
    return () => document.removeEventListener('pointerdown', onDown)
  }, [open])

  const close = () => {
    setOpen(false)
    wrap.current?.querySelector<HTMLButtonElement>('button')?.focus()
  }
  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Escape') {
      // The menu closes, not a page.
      e.stopPropagation()
      close()
    } else if (e.key === 'Tab') setOpen(false)
  }
  return { open, toggle: () => setOpen((o) => !o), close, wrap, menu, id, onKeyDown }
}
