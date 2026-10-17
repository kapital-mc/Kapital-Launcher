import type { ReactNode } from 'react'
import type { Chapter, PackChangelogEntry, WikiFact, WikiPage } from '../../types'
import { ExternalLink } from '../../lib/icons'
import { factValue } from '../../lib/manifest'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { Fact } from '../ui/Fact'
import { Icon } from '../ui/Icon'
import { Scrollable } from '../ui/Scrollable'

/** A changelog entry as the column has it: the pack source's, or the manifest's with no date. */
export type PanelEntry = Pick<PackChangelogEntry, 'version' | 'summary'> &
  Partial<Pick<PackChangelogEntry, 'date' | 'details'>>

/** A fact row; `plain` sets a word in the UI face rather than the data face. */
export interface PanelFact {
  label: string
  value: string
  /** What an unsettled value says: "Unknown", or "Not installed". */
  unset?: string
  plain?: boolean
}

/** The body of a popup that may run long: it scrolls inside the dialog rather than past it. */
function Body({ children }: { children: ReactNode }) {
  return (
    <Scrollable className="flex max-h-96 flex-col gap-4 overflow-y-auto pr-3">
      {children}
    </Scrollable>
  )
}

/**
 * The Pack column, expanded: the chapter's blurb, the column's facts and what
 * the column leaves out. Memory is the manifest's recommendation; what a player
 * set for the chapter is in its own settings.
 */
export function PackDetails({
  chapter,
  facts,
  onClose,
}: {
  chapter: Chapter
  facts: PanelFact[]
  onClose: () => void
}) {
  const { pack, server } = chapter
  const toggles = pack.toggles ?? []
  const more: PanelFact[] = [
    {
      label: 'Recommended memory',
      value: factValue(pack.memoryGb == null ? null : `${pack.memoryGb} GB`),
    },
    { label: 'Server', value: server?.software ?? 'None', plain: true },
    {
      label: 'Optional mods',
      value: toggles.length ? toggles.map((t) => t.name).join(', ') : 'None',
      plain: true,
    },
  ]
  return (
    <Dialog title={`${chapter.name} pack`} onCancel={onClose}>
      <Body>
        <p className="text-fg-soft m-0 leading-normal">{chapter.blurb}</p>
        <div className="flex flex-col gap-3">
          {[...facts, ...more].map((f) => (
            <Fact key={f.label} {...f} />
          ))}
        </div>
      </Body>
      <Close onClose={onClose} />
    </Dialog>
  )
}

/** The Changelog column, expanded: every entry, each with its date and its full text. */
export function ChangelogDetails({
  chapter,
  entries,
  onClose,
}: {
  chapter: Chapter
  entries: PanelEntry[]
  onClose: () => void
}) {
  return (
    <Dialog title={`${chapter.name} changelog`} onCancel={onClose}>
      <Body>
        {entries.length === 0 ? (
          <span className="text-fg-faint">No entries yet.</span>
        ) : (
          entries.map((entry) => (
            <div key={entry.version} className="flex flex-col gap-1">
              <div className="flex items-baseline gap-3">
                <span className="text-accent font-mono text-xs">{entry.version}</span>
                {entry.date && (
                  <span className="text-fg-faint font-mono text-xs">{entry.date}</span>
                )}
              </div>
              <ul className="text-fg-soft m-0 flex list-none flex-col gap-1 p-0 leading-normal">
                {lines(entry).map((line) => (
                  <li key={line}>{line}</li>
                ))}
              </ul>
            </div>
          ))
        )}
      </Body>
      <Close onClose={onClose} />
    </Dialog>
  )
}

/** An entry's lines: the source's details one per line, else the summary split at its semicolons. */
function lines(entry: PanelEntry): string[] {
  const text = entry.details?.trim() ? entry.details : entry.summary.replaceAll('; ', '\n')
  return [...new Set(text.split('\n').map((l) => l.trim()))].filter(Boolean)
}

/**
 * The wiki column, expanded: the whole line, and the facts the wiki states
 * about the page (type, eras, then its own rows). The manifest's teaser has
 * only a title and a line.
 */
export function WikiDetails({
  page,
  teaser,
  onOpenWiki,
  onClose,
}: {
  page: WikiPage | undefined
  teaser: { title: string; line: string }
  onOpenWiki: () => void
  onClose: () => void
}) {
  const kind = page ? [page.category || page.type, ...page.eras].filter(Boolean) : []
  const facts: WikiFact[] = page?.facts ?? []
  return (
    <Dialog
      title={page?.title ?? teaser.title}
      facts={kind.length ? kind.map(capital).join(' · ') : undefined}
      onCancel={onClose}
    >
      <Body>
        <p className="text-fg-soft m-0 leading-normal">{page?.line ?? teaser.line}</p>
        {facts.length > 0 && (
          <div className="flex flex-col gap-3">
            {facts.map((f, i) => (
              <Fact key={i} label={f.label} value={f.value} plain />
            ))}
          </div>
        )}
      </Body>
      <div className="flex flex-wrap items-center gap-3">
        <Button onClick={onOpenWiki}>
          <Icon icon={ExternalLink} size="sm" />
          <span>Read on the wiki</span>
        </Button>
        <Button onClick={onClose}>Close</Button>
      </div>
    </Dialog>
  )
}

function Close({ onClose }: { onClose: () => void }) {
  return (
    <div className="flex">
      <Button onClick={onClose}>Close</Button>
    </div>
  )
}

const capital = (word: string) => word.charAt(0).toUpperCase() + word.slice(1)
