import { useLayoutEffect, useRef } from 'react'

interface Props {
  /** How far along, from 0 to 1. */
  value: number
  /** What is progressing, for a screen reader. */
  label: string
}

/**
 * A thin bar in the accent, filled through a custom property: an inline `style` prop is refused.
 * The terrain download, the install popup and the sidebar's downloads share it.
 */
export function ProgressBar({ value, label }: Props) {
  const fill = useRef<HTMLDivElement>(null)
  const clamped = Math.min(1, Math.max(0, value))
  useLayoutEffect(() => {
    fill.current?.style.setProperty('--progress-fill', String(clamped))
  }, [clamped])
  return (
    <div
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={Math.floor(clamped * 100)}
      className="bg-line-strong rounded-pill h-1 overflow-hidden"
    >
      <div
        ref={fill}
        className="bg-accent duration-fast ease-standard size-full origin-left scale-x-(--progress-fill) transition-transform motion-reduce:transition-none"
      />
    </div>
  )
}
