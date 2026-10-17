import { useEffect, type ReactNode } from 'react'
import type { Chapter, PackChangelogEntry, WikiPage } from '../../types'
import { ExternalLink } from '../../lib/icons'
import { PLACEHOLDER } from '../../lib/manifest'
import { serverSoftware } from '../../lib/serverLine'
import { memoryLabel } from '../../lib/chapterSettings'
import { disabledMods } from '../../lib/mods'
import { useEngineStore } from '../../stores/useEngineStore'
import { useModStore } from '../../stores/useModStore'
import { selectStatus, useServerStore } from '../../stores/useServerStore'
import { Button } from '../ui/Button'
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

/** The body of a popup that may run long: it scrolls inside the dialog rather than past it. */
function Body({ children }: { children: ReactNode }) {
  return (
    <Scrollable className="flex max-h-96 flex-col gap-5 overflow-y-auto pr-3">
      {children}
    </Scrollable>
  )
}

/**
 * The Pack column's popup body: the chapter's blurb, the column's facts, and
 * what the player set for it: the memory Prism starts it with and the mods
 * switched off. Both are read from the instance when the popup opens; a
 * chapter that is not installed has neither and says so.
 */
export function PackDetails({
  chapter,
  facts,
  unset,
}: {
  chapter: Chapter
  facts: PanelFact[]
  /** What the instance's own facts say while there is no answer: "Not installed", or "Unknown". */
  unset: string
}) {
  const settings = useEngineStore((s) => s.chapterSettings[chapter.id])
  const loadSettings = useEngineStore((s) => s.loadChapterSettings)
  const mods = useModStore((s) => s.byChapter[chapter.id])
  const status = useServerStore(selectStatus(chapter.id))
  const loadMods = useModStore((s) => s.load)
  useEffect(() => {
    void loadSettings(chapter.id)
    void loadMods(chapter.id)
  }, [chapter.id, loadSettings, loadMods])
  const off = mods ? disabledMods(mods) : undefined
  const memory = settings ? memoryLabel(settings.settings.maxMemoryMb) : PLACEHOLDER

  return (
    <Body>
      <p className="text-fg-soft m-0 leading-normal">{chapter.blurb}</p>
      <FactList facts={facts} loose>
        <Fact label="Memory" value={memory} unset={unset} />
        <Fact
          label="Server"
          value={serverSoftware(chapter, status)}
          unset={status?.checked && !status.online ? 'Offline' : 'Unknown'}
          plain
        />
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
  )
}

/** The Changelog column's popup body: every entry, each with its date and its full text. */
export function ChangelogDetails({ entries }: { entries: PanelEntry[] }) {
  return (
    <Body>
      {entries.length === 0 ? (
        <span className="text-fg-faint">No entries yet.</span>
      ) : (
        entries.map((entry) => (
          <div key={entry.version} className="flex flex-col gap-1.5">
            <div className="flex items-baseline gap-3">
              <span className="text-accent font-mono text-xs">{entry.version}</span>
              {entry.date && <span className="text-fg-faint font-mono text-xs">{entry.date}</span>}
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
  )
}

/** An entry's lines: the source's details one per line, else the summary split at its semicolons. */
function lines(entry: PanelEntry): string[] {
  const text = entry.details?.trim() ? entry.details : entry.summary.replaceAll('; ', '\n')
  return [...new Set(text.split('\n').map((l) => l.trim()))].filter(Boolean)
}

/**
 * The wiki column's popup body: the page's whole lead, the facts the wiki
 * states about it, and the way to the page. The manifest's teaser has only a
 * line.
 */
export function WikiDetails({
  page,
  line,
  onOpenWiki,
}: {
  page: WikiPage | undefined
  /** The manifest teaser's line, for when there is no page. */
  line: string
  onOpenWiki: () => void
}) {
  const facts = page?.facts ?? []
  return (
    <>
      <Body>
        <p className="text-fg-soft m-0 leading-normal">{page?.line ?? line}</p>
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
      </div>
    </>
  )
}
