import { Suspense, lazy, useEffect, useRef, useState, type ReactNode } from 'react'
import type { Chapter, PackChangelog, PackState, WikiPage } from '../../types'
import { Maximize2 } from '../../lib/icons'
import { factValue, sizeValue, unsetLabel } from '../../lib/manifest'
import { versionValue } from '../../lib/packState'
import { Drift } from '../ui/Drift'
import { FactList } from '../ui/Fact'
import { IconButton } from '../ui/IconButton'
import { TextLink } from '../ui/TextLink'
import type { PanelEntry, PanelFact } from './PanelDetails'

// The popups open only when a column is expanded, so they load then and stay out
// of the first paint and its bundle budget (scripts/check-bundle-size.mjs).
const details = () => import('./PanelDetails')
const preload = () => void details()
/** How long after the card shows the popups' chunk is fetched anyway, in ms. */
const PRELOAD_AFTER_MS = 1500
const PackDetails = lazy(() => details().then((m) => ({ default: m.PackDetails })))
const ChangelogDetails = lazy(() => details().then((m) => ({ default: m.ChangelogDetails })))
const WikiDetails = lazy(() => details().then((m) => ({ default: m.WikiDetails })))

interface Props {
  chapter: Chapter
  /** Whether the chapter's instance exists; undefined when that cannot be looked up. */
  installed: boolean | undefined
  /** The chapter's size on disk, from the instance report; undefined until it is installed. */
  sizeBytes: number | undefined
  /** Whether the installed pack is its source's current one (#71); undefined until checked. */
  packState: PackState | undefined
  /** What the pack source publishes as its changelog (issue 164); undefined until read. */
  changelog: PackChangelog | undefined
  /** The slide's wiki page (#58, issue 142); undefined shows the manifest's teaser. */
  wikiPage: WikiPage | undefined
  onOpenWiki: () => void
}

/** How many of the latest changelog entries the column shows, whatever the manifest holds. */
const CHANGELOG_SHOWN = 3

function PanelTitle({ children }: { children: string }) {
  return <h2 className="text-fg-faint m-0 text-xs font-medium">{children}</h2>
}

/** Where a popup opens: the column it grows out of, and the chapter card it is centred on. */
interface Opening {
  origin: DOMRect
  frame?: DOMRect
}

/**
 * One column, whose expand button shows in its corner on hover and on keyboard
 * focus and opens the column's popup, which grows out of the column. Hovering
 * the column loads the popups' chunk, so the first one opens without waiting.
 * A button that has focus back from a closed popup only shows while it is
 * keyboard focus, so the mouse moving away hides it as before.
 */
function Column({
  title,
  label,
  onExpand,
  last,
  children,
}: {
  title: string
  /** The expand button's accessible name. */
  label: string
  onExpand: (at: Opening) => void
  last?: boolean
  children: ReactNode
}) {
  const ref = useRef<HTMLDivElement>(null)
  return (
    <div
      ref={ref}
      onPointerEnter={preload}
      onFocus={preload}
      className={`group border-line relative flex flex-col gap-3 p-7 ${last ? '' : 'border-r'}`}
    >
      <PanelTitle>{title}</PanelTitle>
      <div className="duration-fast ease-standard absolute top-5 right-5 opacity-0 transition-opacity group-hover:opacity-100 has-focus-visible:opacity-100">
        <IconButton
          icon={Maximize2}
          size="xs"
          title={label}
          onClick={() => {
            const el = ref.current
            if (!el) return
            const frame = el.closest('[data-popup-frame]')?.getBoundingClientRect()
            onExpand({ origin: el.getBoundingClientRect(), frame })
          }}
        />
      </div>
      {children}
    </div>
  )
}

/**
 * The three columns under the actions: the pack's facts, its changelog, and
 * a line from the wiki. The facts are what a player compares packs by (#57):
 * the installed pack's version (#71), the loader, the game version, the mod
 * count and the size on disk. A chapter that is not installed has no version
 * and no size, and says so.
 * Memory is edited in the chapter's own settings (#36), and the server is on
 * the action bar's state line with its ping.
 *
 * The row keeps one height whatever it holds, because the hero takes what the
 * panels leave: the wiki post's title is one line and its text reserves four,
 * with its link pinned under them, and the changelog shows only its latest
 * entries, each clamped, so it can never be the tallest column. The entries
 * are the pack source's own changelog (issue 164), every line of each joined
 * by semicolons, and the manifest's when the source has none. Whatever a
 * column cuts, its popup (PanelDetails) shows whole: every changelog entry,
 * more pack facts, and the wiki page's own facts.
 *
 * The columns are equal, and each has the same 28 px on every side: the first
 * column's text starts as far from the card's edge as the others' from their
 * divider, and the space under the row matches its sides (the author, issue
 * 162, to see how it looks; the Play button and the hero keep their gutter).
 */
export function Panels({
  chapter,
  installed,
  sizeBytes,
  packState,
  changelog,
  wikiPage,
  onOpenWiki,
}: Props) {
  // The popups' chunk is fetched a moment after the card shows, so even a first
  // open that skips the hover preload does not wait for it.
  useEffect(() => {
    const t = setTimeout(preload, PRELOAD_AFTER_MS)
    return () => clearTimeout(t)
  }, [])
  const [open, setOpen] = useState<Opening & { kind: 'pack' | 'changelog' | 'wiki' }>()
  const close = () => setOpen(undefined)
  const expand = (kind: 'pack' | 'changelog' | 'wiki') => (at: Opening) => setOpen({ kind, ...at })
  const { pack } = chapter
  const wiki = wikiPage ?? chapter.wiki
  const unset = unsetLabel(installed)
  // The pack source's entries when it has some, else the manifest's, which have no details.
  const all: PanelEntry[] = changelog?.entries.length ? changelog.entries : chapter.changelog
  const entries = all.slice(0, CHANGELOG_SHOWN)
  const visuals = pack.type === 'client-visuals'
  const facts: PanelFact[] = [
    { label: 'Version', value: versionValue(packState), unset },
    { label: 'Loader', value: visuals ? 'Client visuals' : factValue(pack.loader), plain: visuals },
    { label: 'Minecraft', value: factValue(pack.minecraft) },
    { label: 'Mods', value: factValue(pack.mods) },
    { label: 'Size', value: sizeValue(sizeBytes), unset },
  ]

  return (
    <section className="grid shrink-0 grid-cols-3">
      <Column title="Pack" label="Pack details" onExpand={expand('pack')}>
        <FactList facts={facts} />
      </Column>
      <Column title="Changelog" label="Full changelog" onExpand={expand('changelog')}>
        <div className="flex flex-col gap-3">
          {entries.length === 0 ? (
            <span className="text-fg-faint text-sm">No entries yet.</span>
          ) : (
            entries.map((entry) => (
              <div
                key={entry.version}
                className="grid grid-cols-[64px_minmax(0,1fr)] items-baseline gap-3 text-sm"
              >
                <span className="text-accent text-2xs truncate font-mono">{entry.version}</span>
                <span className="text-fg-soft line-clamp-2" title={entry.details ?? entry.summary}>
                  {entry.summary}
                </span>
              </div>
            ))
          )}
        </div>
      </Column>
      <Column title="From the wiki" label="Wiki details" onExpand={expand('wiki')} last>
        {/* The post drifts to the next with the hero's picture (issue 142). */}
        <Drift
          id={wikiPage?.url ?? 'teaser'}
          className="relative flex grow flex-col"
          layer="flex grow flex-col gap-3"
        >
          <p className="font-display m-0 line-clamp-1 text-lg leading-snug" title={wiki.title}>
            {wiki.title}
          </p>
          <p className="text-fg-muted m-0 line-clamp-4 min-h-[4lh] text-sm leading-normal">
            {wiki.line}
          </p>
          <TextLink className="mt-auto self-start text-left" onClick={onOpenWiki}>
            Read the history →
          </TextLink>
        </Drift>
      </Column>
      <Suspense fallback={null}>
        {open?.kind === 'pack' && (
          <PackDetails
            chapter={chapter}
            facts={facts}
            unset={unset}
            origin={open.origin}
            frame={open.frame}
            onClose={close}
          />
        )}
        {open?.kind === 'changelog' && (
          <ChangelogDetails
            chapter={chapter}
            entries={all}
            origin={open.origin}
            frame={open.frame}
            onClose={close}
          />
        )}
        {open?.kind === 'wiki' && (
          <WikiDetails
            page={wikiPage}
            teaser={{ ...chapter.wiki, chapter: chapter.name }}
            origin={open.origin}
            frame={open.frame}
            onOpenWiki={onOpenWiki}
            onClose={close}
          />
        )}
      </Suspense>
    </section>
  )
}
