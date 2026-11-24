import type { LucideIcon } from '../../lib/icons'
import { Fold } from './Fold'
import { Icon } from './Icon'

interface Props {
  icon: LucideIcon
  /** The action's name, the button's own while it is not asking: Delete, Reset. */
  action: string
  /** The tooltip before the first press, naming what it acts on. */
  title: string
  confirming: boolean
  disabled: boolean
  /** The first press: the caller shows Cancel and sets `confirming`. */
  onStart: () => void
  onConfirm: () => void
}

/**
 * An outlined icon button the size of a folder button (`size-8`) that asks in place. Pressed, it
 * grows to the right of its glyph into Confirm while the caller folds a Cancel in before it, and a
 * second press acts. The second click of a double click is not a confirmation.
 */
export function ConfirmButton({
  icon,
  action,
  title,
  confirming,
  disabled,
  onStart,
  onConfirm,
}: Props) {
  return (
    <button
      type="button"
      aria-label={confirming ? 'Confirm' : action}
      title={confirming ? undefined : title}
      disabled={disabled}
      onClick={(e) => {
        if (!confirming) onStart()
        else if (e.detail <= 1) onConfirm()
      }}
      className={`${confirming ? 'text-fg' : 'text-fg-muted'} border-line-strong hover:bg-hover enabled:hover:text-fg duration-fast ease-standard inline-flex h-8 items-center rounded-md border px-1.75 text-sm font-semibold transition-colors disabled:opacity-50 motion-reduce:transition-none`}
    >
      <Icon icon={icon} size="sm" />
      <Fold open={confirming} axis="x">
        <span className="pl-1.5 whitespace-nowrap">Confirm</span>
      </Fold>
    </button>
  )
}
