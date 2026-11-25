import { Ellipsis, MapIcon, Pencil, Plane, SquareTerminal } from '../../lib/icons'
import { useMenu } from '../../lib/useMenu'
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
 * A chapter's tools: its settings, its logs and its map. A wash behind them,
 * so they read on bright art too (#77). They are one group (issue 162): each
 * cell's hover fills it edge to edge, with no gap between cells, and only the
 * group's outer corners round. Where the pack has saved contraptions, the map
 * and the contraptions fold down under a ⋯ cell as a column of the same cells
 * on a darker wash, joined to it with no gap. The group keeps all four round
 * corners while it is open (the author, 2026-11-25).
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
  const menu = useMenu<HTMLSpanElement>()
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
        // Not ui/Fold: its clip reaches 4px past the content for focus rings, which would
        // leave a sliver of the column showing under the ⋯ while folded. This one unrolls the
        // column from behind the group's rounded corner (tuck-under), solid from the first frame.
        <span
          inert={!menu.open}
          className={`duration-slow ease-glide absolute top-[calc(100%-var(--radius-md))] right-0 grid transition-[grid-template-rows] motion-reduce:transition-none ${menu.open ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'}`}
        >
          <span className="min-h-0 overflow-hidden rounded-b-md">
            <span
              ref={menu.menu}
              id={menu.id}
              role="menu"
              aria-label={`More for ${name}`}
              onKeyDown={menu.onKeyDown}
              className="bg-sunken/70 tuck-under flex flex-col backdrop-blur-sm"
            >
              <IconButton
                grouped
                role="menuitem"
                icon={MapIcon}
                title={`Map of ${name}`}
                onClick={pick(onOpenMap)}
              />
              <IconButton
                grouped
                role="menuitem"
                icon={Plane}
                title={`Contraptions of ${name}`}
                onClick={pick(onOpenContraptions)}
              />
            </span>
          </span>
        </span>
      )}
    </div>
  )
}
