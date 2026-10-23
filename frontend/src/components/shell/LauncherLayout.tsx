import { Suspense, lazy, useEffect, useRef, useState } from 'react'
import type { Chapter } from '../../types'
import { Sidebar } from '../sidebar/Sidebar'
import { ChapterStage } from '../main/ChapterStage'
import { PageLayer } from '../main/PageLayer'
import { Scrollable } from '../ui/Scrollable'
import { useSettingsStore } from '../../stores/useSettingsStore'
import type { usePages } from '../../lib/usePages'
import { useLayoutMorph } from '../../lib/useLayoutMorph'

// The pages that slide over the chapter card are opened on demand, so they
// load when asked for and stay out of the launcher's first paint and its
// bundle budget (scripts/check-bundle-size.mjs): the settings, and a chapter's
// own pages (components/main/ChapterPage), its settings, its logs, its map and
// the run report, opened rarely, from Details beside a game that went wrong.
const SettingsPanel = lazy(() =>
  import('../settings/SettingsPanel').then((m) => ({ default: m.SettingsPanel })),
)
const AccountPanel = lazy(() =>
  import('../settings/AccountPanel').then((m) => ({ default: m.AccountPanel })),
)
const ChapterPage = lazy(() =>
  import('../main/ChapterPage').then((m) => ({ default: m.ChapterPage })),
)
// So does the compact launcher, which most windows never show.
const CompactStage = lazy(() =>
  import('../compact/CompactStage').then((m) => ({ default: m.CompactStage })),
)

interface Props {
  chapter: Chapter
  chapters: readonly Chapter[]
  select: (chapterId: string) => void
  pages: ReturnType<typeof usePages>
  /** Whether the compact launcher is the layout to show (lib/compactMode). */
  compact: boolean
}

/**
 * Everything under the header bar: the full launcher (the sidebar and the
 * chapter card) or the compact one (the chapters as panels), and the morph
 * between them (docs/adr/0015-compact-launcher.md). The full layout is always
 * laid out, hidden while compact shows; the compact one is in the page while
 * it shows or a morph runs. Until a morph's first frame is up, the layout
 * being left stays on screen. The pages open over whichever is shown.
 */
export function LauncherLayout({ chapter, chapters, select, pages, compact }: Props) {
  const { settings, account, chapterPage } = pages
  const openedChapter =
    chapterPage.slot && chapters.find((c) => c.id === chapterPage.slot?.page.chapterId)

  const area = useRef<HTMLDivElement>(null)
  const layer = useRef<HTMLDivElement>(null)
  const [stage, setStage] = useState<HTMLDivElement | null>(null)
  // A change that arrives with the saved settings is where the window starts, not a morph.
  const loaded = useSettingsStore((s) => s.loaded)
  const [ready, setReady] = useState(false)
  useEffect(() => {
    if (!loaded) return
    const frame = requestAnimationFrame(() => setReady(true))
    return () => cancelAnimationFrame(frame)
  }, [loaded])
  const { mode, morphTo } = useLayoutMorph(compact, ready, area, layer, stage)
  // Each layout shows while it is the mode, until the morph's actors take over
  // from it (lib/compactMorphRun), and takes input only when no morph runs.
  const fullLive = mode === 'full' && morphTo === null
  const compactLive = mode === 'compact' && morphTo === null

  const layers = (
    <>
      {chapterPage.slot && openedChapter && (
        <PageLayer
          key={`${chapterPage.slot.page.kind}:${openedChapter.id}`}
          edge="right"
          open={chapterPage.slot.open}
          onExited={chapterPage.done}
        >
          <ChapterPage
            page={chapterPage.slot.page}
            chapter={openedChapter}
            onClose={chapterPage.hide}
          />
        </PageLayer>
      )}
      {settings.slot && (
        <PageLayer edge="top" open={settings.slot.open} onExited={settings.done}>
          <SettingsPanel onClose={settings.hide} onShowChapter={pages.showChapter} />
        </PageLayer>
      )}
      {account.slot && (
        <PageLayer edge="bottom" open={account.slot.open} onExited={account.done}>
          <AccountPanel onClose={account.hide} />
        </PageLayer>
      )}
    </>
  )

  return (
    <div ref={area} className="relative flex min-h-0 grow overflow-clip">
      <div
        data-layout="full"
        inert={!fullLive}
        aria-hidden={mode !== 'full' || undefined}
        className={`flex min-h-0 grow ${mode === 'full' ? '' : 'invisible'}`}
      >
        <Sidebar onOpenAccount={pages.toggleAccount} accountOpen={account.slot?.open === true} />
        {/* The chapter card scrolls under the header bar and beside the
            sidebar when the window is shorter than it (issue 56). A page slides
            over the card in the same stage, and scrolls inside its own frame. */}
        <Scrollable as="main" className="flex flex-col">
          <ChapterStage
            chapter={chapter}
            chapters={chapters}
            onOpenSettings={pages.openChapterSettings}
            onOpenLogs={pages.openLogs}
            onOpenMap={pages.openMap}
            covered={pages.covered}
          >
            {mode === 'full' && layers}
          </ChapterStage>
        </Scrollable>
      </div>
      {(mode === 'compact' || morphTo !== null) && (
        <Suspense fallback={null}>
          <CompactStage
            ref={setStage}
            chapters={chapters}
            selectedId={chapter.id}
            onSelect={select}
            onOpenSettings={pages.openChapterSettings}
            onOpenLogs={pages.openLogs}
            onOpenMap={pages.openMap}
            shown={mode === 'compact'}
            live={compactLive}
            covered={pages.covered}
          >
            {mode === 'compact' && layers}
          </CompactStage>
        </Suspense>
      )}
      {/* The morph's actors (lib/compactMorphRun), over both layouts while it runs. */}
      <div ref={layer} aria-hidden className="morph-actors" />
    </div>
  )
}
