import { Blocks, MapIcon, Pencil, SquareTerminal } from '../../lib/icons'
import { IconButton } from '../ui/IconButton'

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
 * A chapter's tools: its settings, its logs, its map and, where its pack
 * has them, its contraptions. A wash behind them,
 * so they read on bright art too (#77). They are one group (issue 162): each
 * cell's hover fills it edge to edge, with no gap between cells, and only the
 * group's outer corners round.
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
  return (
    <div
      role="group"
      aria-label={`${name} tools`}
      data-morph={morph}
      className={`bg-sunken/55 flex overflow-hidden rounded-md backdrop-blur-sm ${className}`}
    >
      <IconButton grouped icon={Pencil} title={`${name} settings`} onClick={onOpenSettings} />
      <IconButton grouped icon={SquareTerminal} title={`Logs for ${name}`} onClick={onOpenLogs} />
      <IconButton grouped icon={MapIcon} title={`Map of ${name}`} onClick={onOpenMap} />
      {onOpenContraptions && (
        <IconButton
          grouped
          icon={Blocks}
          title={`Contraptions of ${name}`}
          onClick={onOpenContraptions}
        />
      )}
    </div>
  )
}
