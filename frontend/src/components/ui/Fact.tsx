import type { ReactNode } from 'react'
import { isPlaceholder } from '../../lib/manifest'

interface Props {
  label: string
  value: string
  /** What an unsettled value says, faint: "Unknown", or "Not installed". */
  unset?: string
  /** The value is a word, not data, so it is set in the UI face. */
  plain?: boolean
  /** Drawn in place of the value when it is settled: the pills of a list, say. */
  children?: ReactNode
}

/**
 * One row of a fact list: a label in the UI face, a value in the data face
 * when it is a number or a version. An unsettled value is shown faint, in
 * words, rather than hidden, because an unsettled fact is information too.
 * It is a term and its description, so it lives inside a FactList.
 */
export function Fact({ label, value, unset = 'Unknown', plain, children }: Props) {
  const pending = isPlaceholder(value)
  return (
    <div className="contents">
      <dt className="text-fg-muted">{label}</dt>
      {pending ? (
        <dd className="text-fg-faint m-0">{unset}</dd>
      ) : (
        <dd className={`m-0 min-w-0 ${plain ? '' : 'font-mono'}`}>{children ?? value}</dd>
      )}
    </div>
  )
}

/**
 * Facts as two columns: the labels, and each value set just after the longest
 * label, so a label and its value read as one rather than from the two edges.
 * `spread` sets the values against the right edge instead, for a short list of
 * short values in a wide column (the chapter card's Pack column, by the
 * author's choice, 2026-10-19).
 */
export function FactList({
  facts,
  loose,
  spread,
  children,
}: {
  facts?: Props[]
  /** The popup's wider spacing; the chapter card's columns keep the tighter one. */
  loose?: boolean
  /** The values at the right edge rather than after the labels. */
  spread?: boolean
  children?: ReactNode
}) {
  return (
    <dl
      className={`m-0 grid grid-cols-[auto_minmax(0,1fr)] items-baseline gap-x-6 text-sm ${
        loose ? 'gap-y-4' : 'gap-y-3.5'
      } ${spread ? '[&_dd]:text-right' : ''}`}
    >
      {facts?.map((f) => (
        <Fact key={f.label} {...f} />
      ))}
      {children}
    </dl>
  )
}
