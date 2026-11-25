import { useState } from 'react'
import { Button } from '../ui/Button'
import { Checkbox } from '../ui/Checkbox'
import { Dialog } from '../ui/Dialog'

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
  return (
    <Dialog title="Are you sure?" facts={facts} onCancel={onCancel}>
      <p className="text-fg-soft m-0 leading-normal">This will permanently delete your files.</p>
      {skippable && (
        <Checkbox checked={dontAsk} onChange={setDontAsk}>
          Don&apos;t ask me again
        </Checkbox>
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
