import type { LucideIcon } from '../../lib/icons'
import { Icon } from './Icon'

interface Props {
  icon: LucideIcon
  /** The accessible name; the icon carries none, and no tooltip is drawn (the author, 2026-09-30). */
  title: string
  onClick: () => void
  /**
   * `md` is the 44px box the hero tools use; `sm` the 32px one in the account card; `xs`
   * the 28px one in a column's corner, with the smaller glyph.
   */
  size?: 'xs' | 'sm' | 'md'
  disabled?: boolean
  /**
   * One cell of a joined group of tools (the hero's): square, so its hover
   * fills the whole cell, and the group's own frame rounds the outer corners.
   */
  grouped?: boolean
  /** `accent` draws the glyph in the chapter's colour, for a control that has news, such as an update. */
  tone?: 'accent'
}

const BOX = {
  xs: 'size-7',
  sm: 'size-8',
  md: 'size-11',
} as const

/** One square, centred box for every icon control, so a column of them lines up. */
export function IconButton({ icon, title, onClick, size = 'md', disabled, grouped, tone }: Props) {
  const colour = tone === 'accent' ? 'text-accent hover:text-accent' : 'text-fg-muted hover:text-fg'
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={title}
      disabled={disabled}
      className={`${BOX[size]} ${colour} hover:bg-hover duration-fast ease-standard inline-flex shrink-0 cursor-pointer items-center justify-center ${grouped ? '' : 'rounded-md'} transition-colors disabled:cursor-default disabled:opacity-50 disabled:hover:bg-transparent`}
    >
      <Icon icon={icon} size={size} />
    </button>
  )
}
