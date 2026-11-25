import { Columns3, PanelLeftClose, PanelLeftOpen } from '../../lib/icons'
import { Icon } from '../ui/Icon'
import { IconButton } from '../ui/IconButton'
import { useMenu } from '../../lib/useMenu'

/** The compact launcher as the header's sidebar button sees it (lib/compactMode). */
export interface LauncherChoice {
  /** Whether the compact launcher is the layout to show. */
  compact: boolean
  /** Whether the window is too small for the full layout, which holds compact on. */
  held: boolean
  /** Saves the choice from the menu. */
  choose: (compact: boolean) => void
}

interface Props {
  size: 'xs' | 'sm'
  sidebarCompact: boolean
  onToggleSidebar?: () => void
  launcher?: LauncherChoice
}

/**
 * The header's first button. A click narrows the sidebar or widens it again,
 * and in the compact launcher goes back to the full one. A right-click, or the
 * keyboard's menu key, opens a menu of one item that switches between the two
 * launchers (docs/adr/0015-compact-launcher.md); the full launcher is greyed out
 * while the window is too small for it.
 */
export function SidebarButton({ size, sidebarCompact, onToggleSidebar, launcher }: Props) {
  const menu = useMenu()
  const compact = launcher?.compact === true

  const held = compact && launcher?.held === true
  const pick = () => {
    if (held) return
    menu.close()
    launcher?.choose(!compact)
  }

  return (
    <div ref={menu.wrap} className="relative">
      <IconButton
        icon={compact || sidebarCompact ? PanelLeftOpen : PanelLeftClose}
        title={compact ? 'Full Launcher' : sidebarCompact ? 'Expand Sidebar' : 'Compact Sidebar'}
        size={size}
        tooltip
        onClick={() => (compact ? launcher?.choose(false) : onToggleSidebar?.())}
        onContextMenu={(e) => {
          if (!launcher) return
          e.preventDefault()
          menu.toggle()
        }}
        menu={launcher && { open: menu.open, id: menu.id }}
        disabled={!onToggleSidebar && !compact}
      />
      {menu.open && launcher && (
        <div
          ref={menu.menu}
          id={menu.id}
          role="menu"
          aria-label="Layout"
          onKeyDown={menu.onKeyDown}
          className="bg-raised-2 border-line-strong drop-in absolute top-full left-0 z-50 mt-1 flex min-w-44 flex-col rounded-md border p-1"
        >
          <button
            type="button"
            role="menuitem"
            onClick={pick}
            aria-disabled={held || undefined}
            className="text-fg hover:bg-hover focus-visible:bg-hover flex items-center gap-2.5 rounded-sm px-2.5 py-1.5 text-left text-sm outline-none aria-disabled:opacity-50 aria-disabled:hover:bg-transparent"
          >
            <Icon icon={compact ? PanelLeftOpen : Columns3} size="sm" className="text-fg-muted" />
            {compact ? 'Full launcher' : 'Compact launcher'}
          </button>
        </div>
      )}
    </div>
  )
}
