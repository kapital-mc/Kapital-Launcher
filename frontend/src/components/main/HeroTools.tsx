import { Ellipsis, MapIcon, Pencil, Plane, SquareTerminal } from '../../lib/icons'
import { useMenu } from '../../lib/useMenu'
import { IconButton } from '../ui/IconButton'
import { Menu, MenuItem } from '../ui/Menu'

interface Props {
  /** The chapter's name, which names the group and each tool. */
  name: string
  onOpenSettings: () => void
  onOpenLogs: () => void
  onOpenMap: () => void
  /** Opens the chapter's saved contraptions; undefined where the chapter has none. */
  onOpenContraptions?: () => void
  /** Where the group sits: the hero's corner, or a compact panel's. */
  className: string
  /** Marks the group for the compact morph (lib/compactMorph). */
  morph: string
}

/**
 * A chapter's tools: its settings, its logs and its map. Where its pack has
 * saved contraptions, the map and the contraptions share a ⋯ menu that opens
 * downward. A wash behind them, so they read on bright art too (#77). They are
 * one group (issue 162): each cell's hover fills it edge to edge, with no gap
 * between cells, and only the group's outer corners round.
 */
export function HeroTools({
  name,
  onOpenSettings,
  onOpenLogs,
  onOpenMap,
  onOpenContraptions,
  className,
  morph,
}: Props) {
  const menu = useMenu()
  const pick = (open: () => void) => () => {
    menu.close()
    open()
  }
  return (
    <div ref={menu.wrap} data-morph={morph} className={className}>
      <div
        role="group"
        aria-label={`${name} tools`}
        className="bg-sunken/55 flex overflow-hidden rounded-md backdrop-blur-sm"
      >
        <IconButton grouped icon={Pencil} title={`${name} settings`} onClick={onOpenSettings} />
        <IconButton grouped icon={SquareTerminal} title={`Logs for ${name}`} onClick={onOpenLogs} />
        {onOpenContraptions ? (
          <IconButton
            grouped
            icon={Ellipsis}
            title={`More for ${name}`}
            onClick={menu.toggle}
            menu={{ open: menu.open, id: menu.id }}
          />
        ) : (
          <IconButton grouped icon={MapIcon} title={`Map of ${name}`} onClick={onOpenMap} />
        )}
      </div>
      {onOpenContraptions && (
        <Menu state={menu} label={`More for ${name}`} align="right">
          <MenuItem icon={MapIcon} onClick={pick(onOpenMap)}>
            Map
          </MenuItem>
          <MenuItem icon={Plane} onClick={pick(onOpenContraptions)}>
            Contraptions
          </MenuItem>
        </Menu>
      )}
    </div>
  )
}
