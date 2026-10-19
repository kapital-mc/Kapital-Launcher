import { useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import type { PreviewSituation } from '../../types'
import type { ChapterPageSpec } from '../../lib/usePages'
import { useChapterStore } from '../../stores/useChapterStore'
import { usePreviewStore } from '../../stores/usePreviewStore'
import { useSettingsStore } from '../../stores/useSettingsStore'
import { errMsg } from '../../lib/ipc'
import { ErrorLine, Hint } from '../ui/Notes'
import { Segmented } from '../ui/Segmented'
import { TextLink } from '../ui/TextLink'

interface Props {
  /**
   * Called once a preview is on, with the chapter whose screen it is, and that
   * chapter's own page when the screen is there (a Tailscale one is on its settings).
   */
  onStarted: (chapterId: string, page?: ChapterPageSpec['kind']) => void
}

/**
 * Previews (issue 124), in the Developer section: one link per situation a run
 * only shows when something goes wrong, in three groups: the launcher's own
 * (its update, the player's Tailscale), Prism's, and an instance's, for the
 * picked chapter. Go pushes made-up state through the real paths, so the bar,
 * the notices, the card, the sidebar and the settings rows render their real
 * copy. The list is Go's.
 *
 * A start brings the chapter up, closing every page over it, because the
 * screen under review is not this one; a Tailscale situation opens the
 * settings of the first chapter with a Tailscale address, where its row is.
 * It sits closed under its Preview heading (SettingsPanel), and is loaded on
 * demand when that opens, to stay out of the entry bundle.
 */
export function PreviewSection({ onStarted }: Props) {
  const chapters = useChapterStore((s) => s.manifest.chapters)
  const selectedId = useChapterStore((s) => s.selectedId)
  const situations = usePreviewStore((s) => s.situations)
  const load = usePreviewStore((s) => s.load)
  const start = usePreviewStore((s) => s.start)
  const clear = usePreviewStore((s) => s.clear)
  const splashOn = useSettingsStore((s) => s.settings.loadingSplashOn)
  // The chapter the instance previews are for; the open one until another is picked.
  const [picked, setPicked] = useState<string | null>(null)
  const chapterId = picked ?? selectedId
  const tailscaleChapter =
    chapters.find((c) => c.server?.addresses.some((a) => a.tailscale))?.id ?? selectedId
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    void load()
  }, [load])

  const run = async (act: () => Promise<void>, then?: () => void) => {
    setBusy(true)
    setError(null)
    try {
      await act()
      then?.()
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setBusy(false)
    }
  }

  const link = (s: PreviewSituation) => {
    const target = s.scope === 'chapter' ? chapterId : s.page ? tailscaleChapter : selectedId
    return (
      <TextLink
        key={s.id}
        disabled={busy}
        onClick={() =>
          void run(
            () => start(target, s),
            () => (s.page ? onStarted(target, s.page) : onStarted(target)),
          )
        }
      >
        {s.label}
      </TextLink>
    )
  }
  const scoped = (scope: PreviewSituation['scope']) =>
    situations.filter((s) => s.scope === scope).map(link)

  return (
    <div className="flex flex-col gap-3">
      <Hint>
        Shows a screen that a run only has when something goes wrong, so it can be looked at. A
        preview is made up: it writes nothing, starts nothing and reaches no network, and a real
        launch ends it.
      </Hint>
      <PreviewGroup name="Launcher">
        <Links>{scoped('launcher')}</Links>
      </PreviewGroup>
      <PreviewGroup name="Prism">
        <Links>{scoped('prism')}</Links>
      </PreviewGroup>
      <PreviewGroup name="Instance">
        <Segmented
          label="Chapter"
          value={chapterId}
          options={chapters.map((c) => ({ value: c.id, label: c.name }))}
          onChange={setPicked}
        />
        {splashOn === false && (
          <Hint>
            The loading splash is off here, so previews that have a card show the bar and the notice
            only.
          </Hint>
        )}
        <Links>{scoped('chapter')}</Links>
      </PreviewGroup>
      <div className="flex flex-wrap items-center gap-3">
        <TextLink disabled={busy} onClick={() => void run(clear)}>
          Clear previews
        </TextLink>
        {error && <ErrorLine>{error}</ErrorLine>}
      </div>
    </div>
  )
}

/** One group of previews: its name over what it holds. */
function PreviewGroup({ name, children }: { name: string; children: ReactNode }) {
  return (
    <div role="group" aria-label={`${name} previews`} className="flex flex-col gap-1.5">
      <span className="text-fg-muted text-xs font-medium">{name}</span>
      {children}
    </div>
  )
}

/** A group's links, wrapped onto as many lines as they need. */
function Links({ children }: { children: ReactNode }) {
  return <div className="flex flex-wrap gap-x-4 gap-y-1">{children}</div>
}
