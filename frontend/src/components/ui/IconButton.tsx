import type { MouseEvent } from 'react'
import type { LucideIcon } from '../../lib/icons'
import { Icon } from './Icon'

interface Props {
  icon: LucideIcon
  /**
   * The accessible name; the icon carries none. No tooltip is drawn unless `tooltip` asks for one
   * (the author, 2026-09-30; the header's tools, 2026-10-19).
   */
  title: string
  onClick: () => void
  /** A right-click, or the keyboard's menu key: the header's sidebar button opens a menu with it. */
  onContextMenu?: (e: MouseEvent<HTMLButtonElement>) => void
  /** Whether the button has a menu, and whether it is open. */
  menu?: { open: boolean; id: string }
  /**
   * `md` is the 44px box the hero tools use; `sm` the 32px one in the account card; `xs`
   * the 28px one in a column's corner, with the smaller glyph; `2xs` the 20px one on a line of
   * small text, such as a download's step, with that glyph too.
   */
  size?: '2xs' | 'xs' | 'sm' | 'md'
  disabled?: boolean
  /**
   * One cell of a joined group of tools (the hero's): square, so its hover
   * fills the whole cell, and the group's own frame rounds the outer corners.
   */
  grouped?: boolean
  /** `accent` draws the glyph in the chapter's colour, for a control that has news, such as an update. */
  tone?: 'accent'
  /**
   * Shows `title` in a tooltip under the button after a long hover or on keyboard focus: the
   * header's tools, where nothing else names them. Square and in the mono face, drawn here
   * rather than by the browser.
   */
  tooltip?: boolean
}

const BOX = {
  '2xs': 'size-5',
  xs: 'size-7',
  sm: 'size-8',
  md: 'size-11',
} as const

/** One square, centred box for every icon control, so a column of them lines up. */
export function IconButton({
  icon,
  title,
  onClick,
  onContextMenu,
  menu,
  size = 'md',
  disabled,
  grouped,
  tone,
  tooltip,
}: Props) {
  const colour = tone === 'accent' ? 'text-accent hover:text-accent' : 'text-fg-muted hover:text-fg'
  return (
    <button
      type="button"
      onClick={onClick}
      onContextMenu={onContextMenu}
      aria-haspopup={menu ? 'menu' : undefined}
      aria-expanded={menu?.open}
      aria-controls={menu?.open ? menu.id : undefined}
      aria-label={title}
      disabled={disabled}
      className={`${BOX[size]} ${colour} hover:bg-hover duration-fast ease-standard ${tooltip ? 'group relative' : ''} inline-flex shrink-0 items-center justify-center ${grouped ? '' : 'rounded-md'} transition-colors disabled:opacity-50 disabled:hover:bg-transparent`}
    >
      <Icon icon={icon} size={size === '2xs' ? 'xs' : size} />
      {tooltip && (
        <span
          aria-hidden
          className="border-line-strong bg-raised-2 text-fg duration-fast ease-standard text-2xs pointer-events-none absolute top-full left-0 z-50 mt-1.5 border px-2 py-1 font-mono whitespace-nowrap opacity-0 transition-opacity group-hover:opacity-100 group-hover:delay-(--duration-tooltip) group-focus-visible:opacity-100 group-aria-expanded:hidden motion-reduce:transition-none"
        >
          {title}
        </span>
      )}
    </button>
  )
}
