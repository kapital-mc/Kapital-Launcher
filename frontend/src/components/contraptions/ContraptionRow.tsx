import { useId } from 'react'
import type { Contraption, MaterialCount } from '../../types'
import { contraptionDetail, materialName } from '../../lib/contraptions'
import { ChevronDown, Download, Trash2 } from '../../lib/icons'
import { Button } from '../ui/Button'
import { Grow } from '../ui/Grow'
import { Icon } from '../ui/Icon'
import { TextField } from '../ui/TextField'

interface Props {
  item: Contraption
  open: boolean
  busy: boolean
  renameError: string | null
  onToggle: () => void
  onRename: (name: string) => void
  onExport: () => void
  onDelete: () => void
}

/**
 * One saved contraption: its name and facts on a line that opens it, and under it, opened, its
 * name to edit, what it is built from and what it carries, and Export and Delete.
 */
export function ContraptionRow({
  item,
  open,
  busy,
  renameError,
  onToggle,
  onRename,
  onExport,
  onDelete,
}: Props) {
  const bodyId = useId()
  return (
    <div className="flex flex-col">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={bodyId}
        onClick={onToggle}
        className="group -mx-4 -my-3.5 flex items-center gap-3 px-4 py-3.5 text-left"
      >
        <span className="flex min-w-0 grow flex-col gap-1">
          <span className="text-fg text-sm font-medium break-words">{item.name}</span>
          <span className="text-fg-muted text-xs tabular-nums">{contraptionDetail(item)}</span>
        </span>
        <span className="text-fg-muted group-hover:text-fg duration-fast ease-standard flex size-8 shrink-0 items-center justify-center transition-colors">
          <Icon
            icon={ChevronDown}
            size="sm"
            className={`duration-fast ease-standard transition-transform motion-reduce:transition-none ${open ? 'rotate-180' : ''}`}
          />
        </span>
      </button>
      <div id={bodyId}>
        <Grow inner="pt-4">
          {open && (
            <div className="flex flex-col gap-5">
              <TextField label="Name" value={item.name} error={renameError} onCommit={onRename} />
              {item.unread ? (
                <p className="text-fg-muted m-0 text-sm">The file could not be read.</p>
              ) : (
                <>
                  <Counts title="Blocks" counts={item.materials} />
                  <Counts title="Items" counts={item.items} />
                </>
              )}
              <div className="flex flex-wrap items-center gap-3">
                <Button onClick={onExport} disabled={busy}>
                  <Icon icon={Download} size="sm" />
                  <span>Export</span>
                </Button>
                <Button onClick={onDelete} disabled={busy}>
                  <Icon icon={Trash2} size="sm" />
                  <span>Delete</span>
                </Button>
              </div>
            </div>
          )}
        </Grow>
      </div>
    </div>
  )
}

/** A list of ids and counts, most first, in two columns where there is room. */
function Counts({ title, counts }: { title: string; counts: readonly MaterialCount[] }) {
  if (counts.length === 0) return null
  return (
    <div className="flex flex-col gap-2">
      <span className="text-fg text-sm font-medium">{title}</span>
      <ul className="m-0 grid list-none grid-cols-1 gap-x-6 gap-y-1 p-0 sm:grid-cols-2">
        {counts.map((m) => {
          const { name, mod } = materialName(m.id)
          return (
            <li key={m.id} className="flex min-w-0 items-baseline gap-2 text-xs">
              <span className="text-fg w-10 shrink-0 text-right tabular-nums">
                {m.count.toLocaleString()}
              </span>
              <span className="text-fg-soft truncate" title={m.id}>
                {name}
              </span>
              {mod && <span className="text-fg-faint shrink-0">{mod}</span>}
            </li>
          )
        })}
      </ul>
    </div>
  )
}
