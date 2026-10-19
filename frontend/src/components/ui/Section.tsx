import { useId, useState } from 'react'
import type { ReactNode } from 'react'
import { ChevronDown } from '../../lib/icons'
import { Icon } from './Icon'

interface SectionProps {
  title: string
  children: ReactNode
  /**
   * Starts closed behind its title, which opens and closes it: for a section
   * most players never need, such as Developer. Its body is not rendered
   * while closed, so what it loads on demand waits until it is opened.
   */
  collapsible?: boolean
}

const TITLE = 'font-display text-fg m-0 text-lg font-semibold'

/**
 * A titled section of a page: the title in the display face, the body a
 * column at the standard gap. Pages stack sections ten steps apart; what
 * sits inside a section is five.
 */
export function Section({ title, children, collapsible = false }: SectionProps) {
  const [open, setOpen] = useState(!collapsible)
  const bodyId = useId()
  if (!collapsible) {
    return (
      <div className="flex flex-col gap-5">
        <h2 className={TITLE}>{title}</h2>
        {children}
      </div>
    )
  }
  return (
    <div className="flex flex-col gap-5">
      <h2 className={TITLE}>
        <Disclosure
          open={open}
          controls={bodyId}
          onToggle={() => setOpen((o) => !o)}
          className="hover:text-accent duration-fast ease-standard transition-colors"
        >
          {title}
        </Disclosure>
      </h2>
      <div id={bodyId} className={open ? 'flex flex-col gap-5' : 'hidden'}>
        {open && children}
      </div>
    </div>
  )
}

/** The heading of a block inside a section or a page, one style wherever it is. */
export function SubHeading({ id, children }: { id?: string; children: ReactNode }) {
  return (
    <h3 id={id} className="text-fg m-0 text-sm font-semibold">
      {children}
    </h3>
  )
}

/**
 * A block inside a section that starts closed behind its sub-heading, which
 * opens and closes it, like a collapsible Section one level down. Its body is
 * not rendered while closed, so what it loads on demand waits until it is opened.
 */
export function SubSection({ title, children }: { title: string; children: ReactNode }) {
  const [open, setOpen] = useState(false)
  const bodyId = useId()
  return (
    <div className="flex flex-col gap-3">
      <h3 className="m-0">
        <Disclosure
          open={open}
          controls={bodyId}
          onToggle={() => setOpen((o) => !o)}
          className="group text-fg text-sm font-semibold"
        >
          {title}
        </Disclosure>
      </h3>
      <div id={bodyId} className={open ? 'flex flex-col gap-3' : 'hidden'}>
        {open && children}
      </div>
    </div>
  )
}

/** A heading's button that opens and closes the body it controls, with its chevron. */
function Disclosure({
  open,
  controls,
  onToggle,
  className,
  children,
}: {
  open: boolean
  controls: string
  onToggle: () => void
  className: string
  children: ReactNode
}) {
  return (
    <button
      type="button"
      aria-expanded={open}
      aria-controls={controls}
      onClick={onToggle}
      className={`flex items-center gap-2 ${className}`}
    >
      {children}
      <Icon
        icon={ChevronDown}
        size="sm"
        className={`text-fg-muted group-hover:text-fg duration-fast ease-standard transition-transform motion-reduce:transition-none ${
          open ? 'rotate-180' : ''
        }`}
      />
    </button>
  )
}
