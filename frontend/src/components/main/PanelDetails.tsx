import { useEffect, type ReactNode } from 'react'
import type { Chapter, PackChangelogEntry, WikiPage } from '../../types'
import { ExternalLink } from '../../lib/icons'
import { PLACEHOLDER } from '../../lib/manifest'
import { memoryLabel } from '../../lib/chapterSettings'
import { disabledMods } from '../../lib/mods'
import { useEngineStore } from '../../stores/useEngineStore'
import { useModStore } from '../../stores/useModStore'
import { Button } from '../ui/Button'
import { Dialog } from '../ui/Dialog'
import { Fact, FactList } from '../ui/Fact'
import { Icon } from '../ui/Icon'
import { Pill } from '../ui/Pill'
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

/** What every popup takes: the column's rectangle it grows out of, and how it closes. */
interface Opened {
  origin: DOMRect
  onClose: () => void
}

/** The body of a popup that may run long: it scrolls inside the dialog rather than past it. */
function Body({ children }: { children: ReactNode }) {
  return (
    <Scrollable className="flex max-h-96 flex-col gap-5 overflow-y-auto pr-3">
      {children}
    </Scrollable>
  )
}

/**
 * The Pack column, expanded: the chapter's blurb, the column's facts, and what
 * the player set for it: the memory Prism starts it with and the mods switched
 * off. Both are read from the instance when the popup opens; a chapter that is
 * not installed has neither and says so.
 */
export function PackDetails({
  chapter,
  facts,
  unset,
  origin,
  onClose,
}: Opened & {
  chapter: Chapter
  facts: PanelFact[]
  /** What the instance's own facts say while there is no answer: "Not installed", or "Unknown". */
  unset: string
}) {
  const settings = useEngineStore((s) => s.chapterSettings[chapter.id])
  const loadSettings = useEngineStore((s) => s.loadChapterSettings)
  const mods = useModStore((s) => s.byChapter[chapter.id])
  const loadMods = useModStore((s) => s.load)
  useEffect(() => {
    void loadSettings(chapter.id)
    void loadMods(chapter.id)
  }, [chapter.id, loadSettings, loadMods])
  const off = mods ? disabledMods(mods) : undefined
  const memory = settings ? memoryLabel(settings.settings.maxMemoryMb) : PLACEHOLDER

  return (
    <Dialog title={`${chapter.name} pack`} origin={origin} onCancel={onClose}>
      {(close) => (
        <>
          <Body>
            <p className="text-fg-soft m-0 leading-normal">{chapter.blurb}</p>
            <FactList facts={facts} loose>
              <Fact label="Memory" value={memory} unset={unset} />
              <Fact label="Server" value={chapter.server?.software ?? 'None'} plain />
              <Fact label="Disabled mods" value={off ? 'None' : PLACEHOLDER} unset={unset} plain>
                {off && off.length > 0 && (
                  <span className="flex flex-wrap gap-1.5">
                    {off.map((name) => (
                      <Pill key={name}>{name}</Pill>
                    ))}
                  </span>
                )}
              </Fact>
            </FactList>
          </Body>
          <Close onClose={close} />
        </>
      )}
    </Dialog>
  )
}

/** The Changelog column, expanded: every entry, each with its date and its full text. */
export function ChangelogDetails({
  chapter,
  entries,
  origin,
  onClose,
}: Opened & { chapter: Chapter; entries: PanelEntry[] }) {
  return (
    <Dialog title={`${chapter.name} changelog`} origin={origin} onCancel={onClose}>
      {(close) => (
        <>
          <Body>
            {entries.length === 0 ? (
              <span className="text-fg-faint">No entries yet.</span>
            ) : (
              entries.map((entry) => (
                <div key={entry.version} className="flex flex-col gap-1.5">
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
          <Close onClose={close} />
        </>
      )}
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
  origin,
  onClose,
}: Opened & {
  page: WikiPage | undefined
  teaser: { title: string; line: string }
  onOpenWiki: () => void
}) {
  const kind = page ? [page.category || page.type, ...page.eras].filter(Boolean) : []
  const facts = page?.facts ?? []
  return (
    <Dialog
      title={page?.title ?? teaser.title}
      facts={kind.length ? kind.map(capital).join(' · ') : undefined}
      origin={origin}
      onCancel={onClose}
    >
      {(close) => (
        <>
          <Body>
            <p className="text-fg-soft m-0 leading-normal">{page?.line ?? teaser.line}</p>
            {facts.length > 0 && (
              <FactList loose>
                {facts.map((f, i) => (
                  <Fact key={i} label={f.label} value={f.value} plain />
                ))}
              </FactList>
            )}
          </Body>
          <div className="flex flex-wrap items-center gap-3">
            <Button onClick={onOpenWiki}>
              <Icon icon={ExternalLink} size="sm" />
              <span>Read on the wiki</span>
            </Button>
            <Button onClick={close}>Close</Button>
          </div>
        </>
      )}
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
