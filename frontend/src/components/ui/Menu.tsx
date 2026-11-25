import type { ReactNode } from 'react'
import type { LucideIcon } from '../../lib/icons'
import type { useMenu } from '../../lib/useMenu'
import { Icon } from './Icon'

type MenuState = ReturnType<typeof useMenu>

interface MenuProps {
  state: MenuState
  label: string
  /** Which edge of the button the menu lines up with. */
  align: 'left' | 'right'
  children: ReactNode
}

/** The menu `useMenu` opens, under its button. */
export function Menu({ state, label, align, children }: MenuProps) {
  if (!state.open) return null
  return (
    <div
      ref={state.menu}
      id={state.id}
      role="menu"
      aria-label={label}
      onKeyDown={state.onKeyDown}
      className={`bg-raised-2 border-line-strong drop-in absolute top-full ${align === 'left' ? 'left-0' : 'right-0'} z-50 mt-1 flex min-w-44 flex-col rounded-md border p-1`}
    >
      {children}
    </div>
  )
}

interface ItemProps {
  icon: LucideIcon
  onClick: () => void
  disabled?: boolean
  children: ReactNode
}

export function MenuItem({ icon, onClick, disabled, children }: ItemProps) {
  return (
    <button
      type="button"
      role="menuitem"
      onClick={onClick}
      aria-disabled={disabled || undefined}
      className="text-fg hover:bg-hover focus-visible:bg-hover flex items-center gap-2.5 rounded-sm px-2.5 py-1.5 text-left text-sm outline-none aria-disabled:opacity-50 aria-disabled:hover:bg-transparent"
    >
      <Icon icon={icon} size="sm" className="text-fg-muted" />
      {children}
    </button>
  )
}
