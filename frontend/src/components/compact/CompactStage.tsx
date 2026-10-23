import type { ReactNode, Ref } from 'react'
import type { Chapter } from '../../types'
import { CompactPanel } from './CompactPanel'

interface Props {
  chapters: readonly Chapter[]
  selectedId: string
  onSelect: (chapterId: string) => void
  onOpenSettings: (chapterId: string) => void
  onOpenLogs: (chapterId: string) => void
  onOpenMap: (chapterId: string) => void
  /** Whether the layout is the one shown. Hidden, it stays laid out, so the morph can measure it. */
  shown: boolean
  /** Whether it takes input: shown, with no morph running. */
  live: boolean
  /** Whether a page covers the panels, which leaves them inert beneath. */
  covered?: boolean
  /** The pages laid over the panels, each a `PageLayer`, while this is the layout shown. */
  children?: ReactNode
  ref?: Ref<HTMLDivElement>
}

/**
 * The compact launcher (docs/adr/0015-compact-launcher.md): the chapters as
 * panels side by side in the nav's order, the open one wider. It takes the
 * place of the sidebar and the chapter card, and a page opens over it as it
 * does over the card. Its code loads when it is first shown, which keeps it out
 * of the launcher's first paint and its bundle budget.
 */
export function CompactStage({
  chapters,
  selectedId,
  onSelect,
  onOpenSettings,
  onOpenLogs,
  onOpenMap,
  shown,
  live,
  covered = false,
  children,
  ref,
}: Props) {
  return (
    <div
      ref={ref}
      data-layout="compact"
      inert={!live}
      aria-hidden={!shown || undefined}
      className={`bg-sunken absolute inset-0 ${shown ? '' : 'invisible'}`}
    >
      <div
        inert={covered}
        aria-hidden={covered || undefined}
        className="compact-row flex size-full gap-(--layout-compact-gap) p-(--layout-compact-gap)"
      >
        {chapters.map((c) => (
          <CompactPanel
            key={c.id}
            chapter={c}
            active={c.id === selectedId}
            onSelect={() => onSelect(c.id)}
            onOpenSettings={onOpenSettings}
            onOpenLogs={onOpenLogs}
            onOpenMap={onOpenMap}
          />
        ))}
      </div>
      {/* The pages come in over the panels, within the same gap. */}
      <div
        data-popup-frame
        className="card-stage pointer-events-none absolute inset-0 m-(--layout-compact-gap)"
      >
        <div className="pointer-events-auto contents">{children}</div>
      </div>
    </div>
  )
}
