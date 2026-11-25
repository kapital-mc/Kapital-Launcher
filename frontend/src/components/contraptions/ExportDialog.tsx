import { useState } from 'react'
import type { Contraption } from '../../types'
import { CONTRAPTION_KINDS, contraptionKey } from '../../lib/contraptions'
import { ChevronDown } from '../../lib/icons'
import { Button } from '../ui/Button'
import { Checkbox } from '../ui/Checkbox'
import { Dialog } from '../ui/Dialog'
import { Fold } from '../ui/Fold'
import { Icon } from '../ui/Icon'

interface Props {
  list: readonly Contraption[]
  onExport: (picks: Contraption[]) => void
  onCancel: () => void
}

/**
 * What goes into the zip: a box per kind, and under each, folded at first, a box per file. Every
 * file is ticked when it opens; a kind's box ticks or clears all of its files and shows a dash
 * when only some are ticked.
 */
export function ExportDialog({ list, onExport, onCancel }: Props) {
  const [picked, setPicked] = useState(() => new Set(list.map(contraptionKey)))
  const [open, setOpen] = useState<string | null>(null)
  const set = (keys: string[], on: boolean) =>
    setPicked((was) => {
      const next = new Set(was)
      for (const k of keys) {
        if (on) next.add(k)
        else next.delete(k)
      }
      return next
    })
  const picks = list.filter((c) => picked.has(contraptionKey(c)))

  return (
    <Dialog
      title="Export contraptions"
      facts={`${picks.length} of ${list.length} ${list.length === 1 ? 'file' : 'files'}`}
      onCancel={onCancel}
    >
      <div className="flex flex-col gap-3">
        {CONTRAPTION_KINDS.map(({ kind, title }) => {
          const items = list.filter((c) => c.kind === kind)
          if (items.length === 0) return null
          const keys = items.map(contraptionKey)
          const on = keys.filter((k) => picked.has(k)).length
          const shown = open === kind
          return (
            <div key={kind} className="flex flex-col">
              <div className="flex items-center gap-2.5">
                <Checkbox
                  checked={on === keys.length}
                  mixed={on > 0 && on < keys.length}
                  onChange={(v) => set(keys, v)}
                >
                  {title}
                </Checkbox>
                <span className="text-fg-muted text-xs tabular-nums">
                  {on} of {keys.length}
                </span>
                <button
                  type="button"
                  aria-expanded={shown}
                  aria-label={`${title} files`}
                  onClick={() => setOpen(shown ? null : kind)}
                  className="text-fg-muted hover:text-fg duration-fast ease-standard ml-auto flex size-8 items-center justify-center transition-colors"
                >
                  <Icon
                    icon={ChevronDown}
                    size="sm"
                    className={`duration-fast ease-standard transition-transform motion-reduce:transition-none ${shown ? 'rotate-180' : ''}`}
                  />
                </button>
              </div>
              <Fold open={shown} axis="y">
                <span className="flex max-h-60 flex-col gap-2 overflow-y-auto py-2 pl-6">
                  {items.map((c) => {
                    const key = contraptionKey(c)
                    return (
                      <Checkbox key={key} checked={picked.has(key)} onChange={(v) => set([key], v)}>
                        <span className="truncate">{c.name}</span>
                      </Checkbox>
                    )
                  })}
                </span>
              </Fold>
            </div>
          )
        })}
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <Button onClick={() => onExport(picks)} disabled={picks.length === 0}>
          Export
        </Button>
        <Button variant="quiet" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </Dialog>
  )
}
