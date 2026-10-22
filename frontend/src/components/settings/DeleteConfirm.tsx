import { useId, useState } from 'react'
import { Check } from '../../lib/icons'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { Icon } from '../ui/Icon'

interface Props {
  /** The row and its size, under the title. */
  facts: string
  /** Called with whether the player ticked "Don't ask me again". */
  onConfirm: (dontAsk: boolean) => void
  onCancel: () => void
  /** Whether the popup offers "Don't ask me again"; a world's delete always asks. */
  skippable: boolean
}

/**
 * The second question before the storage section deletes a world, terrain, a map or the
 * screenshots. Confirm deletes, Cancel (or Escape, or the cover) keeps them, and the box turns the
 * popup off for later deletes but a world's.
 */
export function DeleteConfirm({ facts, onConfirm, onCancel, skippable }: Props) {
  const [dontAsk, setDontAsk] = useState(false)
  const id = useId()
  return (
    <Dialog title="Are you sure?" facts={facts} onCancel={onCancel}>
      <p className="text-fg-soft m-0 leading-normal">This will permanently delete your files.</p>
      {skippable && (
        <label htmlFor={id} className="text-fg-soft flex w-fit items-center gap-2 text-sm">
          {/* Drawn in the theme's colours: a native box stays light in the dark theme. */}
          <span className="relative flex size-4 shrink-0">
            <input
              id={id}
              type="checkbox"
              checked={dontAsk}
              onChange={(e) => setDontAsk(e.target.checked)}
              className="peer border-line-strong bg-sunken checked:border-accent checked:bg-accent duration-fast ease-standard size-4 appearance-none rounded-sm border transition-colors motion-reduce:transition-none"
            />
            <Icon
              icon={Check}
              size="xs"
              className="text-canvas pointer-events-none absolute inset-0 m-auto opacity-0 peer-checked:opacity-100"
            />
          </span>
          Don&apos;t ask me again
        </label>
      )}
      <div className="flex flex-wrap items-center gap-3">
        <Button onClick={() => onConfirm(dontAsk)}>Confirm</Button>
        <Button variant="quiet" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </Dialog>
  )
}
