import { useEffect, useId, useRef } from 'react'
import type { ReactNode } from 'react'
import { Check, Minus } from '../../lib/icons'
import { Icon } from './Icon'

interface Props {
  checked: boolean
  /** Some of what the box stands for is ticked: drawn with a dash, and read as mixed. */
  mixed?: boolean
  onChange: (checked: boolean) => void
  children: ReactNode
}

/** A checkbox and its label, drawn in the theme's colours: a native box stays light in the dark theme. */
export function Checkbox({ checked, mixed = false, onChange, children }: Props) {
  const id = useId()
  const box = useRef<HTMLInputElement>(null)
  useEffect(() => {
    if (box.current) box.current.indeterminate = mixed
  }, [mixed])
  return (
    <label htmlFor={id} className="text-fg-soft flex w-fit min-w-0 items-center gap-2 text-sm">
      <span className="relative flex size-4 shrink-0">
        <input
          ref={box}
          id={id}
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          className="peer border-line-strong bg-sunken checked:border-accent checked:bg-accent indeterminate:border-accent indeterminate:bg-accent duration-fast ease-standard size-4 appearance-none rounded-sm border transition-colors motion-reduce:transition-none"
        />
        <Icon
          icon={mixed ? Minus : Check}
          size="xs"
          className="text-canvas pointer-events-none absolute inset-0 m-auto opacity-0 peer-checked:opacity-100 peer-indeterminate:opacity-100"
        />
      </span>
      {children}
    </label>
  )
}
